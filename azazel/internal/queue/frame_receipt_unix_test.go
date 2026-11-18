//go:build darwin || linux

package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/azazel/internal/budget"
	"github.com/janearc/slayer/azazel/internal/footprint"
	"github.com/janearc/slayer/internal/observability"
)

func TestFrameReceiptDrainStopsUnderContinuousSender(t *testing.T) {
	stop := make(chan struct{})
	close(stop)
	r := frameReceipt{nonce: "0123456789abcdef0123456789abcdef", sequence: 3, attempt: 2, stop: stop}
	var calls uint64
	state := r.receive(func(buffer []byte) (int, int, error) {
		calls++
		packet := observability.FramePacket{Version: 1, Kind: "frame", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: calls}
		raw, _ := json.Marshal(packet)
		return copy(buffer, raw), 0, nil
	}, io.Discard)
	if calls != frameDrainLimit || !state.cutoff || state.complete() {
		t.Fatal(calls, state)
	}
}

func TestReceiptDiskGuardStopsBeforeFloorOrUnknown(t *testing.T) {
	for _, row := range []struct {
		name  string
		free  uint64
		err   error
		write bool
	}{
		{"safe", budget.QueueDiskFloorBytes + frameReceiptLimit, nil, true},
		{"near floor", budget.QueueDiskFloorBytes + frameReceiptLimit - 1, nil, false},
		{"unknown", 0, errors.New("stat unavailable"), false},
	} {
		t.Run(row.name, func(t *testing.T) {
			var output bytes.Buffer
			guard := receiptDiskWriter{file: &output, free: func() (uint64, error) { return row.free, row.err }}
			_, err := guard.Write([]byte("frame"))
			if (err == nil) != row.write || (output.String() == "frame") != row.write {
				t.Fatal(err, output.String())
			}
		})
	}
}

func TestUnavailableReceiptSetupDoesNotFailJob(t *testing.T) {
	store, job := executableFixture(t, "exit 0")
	job.Request.Environment = map[string]string{"SLAYER_OBSERVE_FRAMES": "on"}
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	result := store.executeUnderLockWithObservers(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(int) (footprint.Leader, bool) { return footprint.Leader{}, false }, func(string, Job) (*frameReceipt, error) { return nil, errors.New("disk unsafe") })
	if result.State != Succeeded || result.Uncertain {
		t.Fatal(result)
	}
}

func TestBlockedFrameReceiptSetupDoesNotBlockOrChangeJob(t *testing.T) {
	store, job := executableFixture(t, "exit 0")
	job.Request.Environment = map[string]string{"SLAYER_OBSERVE_FRAMES": "on"}
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	release := make(chan struct{})
	defer close(release)
	started := time.Now()
	result := store.executeUnderLockWithObservers(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(int) (footprint.Leader, bool) { return footprint.Leader{}, false }, func(string, Job) (*frameReceipt, error) { <-release; return nil, nil })
	if result.State != Succeeded || result.Uncertain || time.Since(started) > time.Second {
		t.Fatal(result, time.Since(started))
	}
}

func TestCancellationDuringReceiptSetupNeverLaunchesCommand(t *testing.T) {
	store, job := executableFixture(t, "touch should-not-exist")
	job.Request.Environment = map[string]string{"SLAYER_OBSERVE_FRAMES": "on"}
	lease, err := store.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	release, entered := make(chan struct{}), make(chan struct{})
	defer close(release)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { <-entered; cancel() }()
	result := store.executeUnderLockWithObservers(ctx, job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), lease, func(int) (footprint.Leader, bool) { return footprint.Leader{}, false }, func(string, Job) (*frameReceipt, error) { close(entered); <-release; return nil, nil })
	if result.State != Cancelled || result.Uncertain {
		t.Fatal(result)
	}
	if _, err := os.Stat(filepath.Join(job.Request.Directory, "should-not-exist")); !os.IsNotExist(err) {
		t.Fatal("command started", err)
	}
}

func TestFrameOptInSurvivesOldV3RequestRoundTrip(t *testing.T) {
	type oldRequest struct {
		Agent          string            `json:"agent"`
		Key            string            `json:"key"`
		Command        string            `json:"command"`
		Args           []string          `json:"args"`
		Directory      string            `json:"directory"`
		Environment    map[string]string `json:"environment,omitempty"`
		Inputs         []string          `json:"inputs,omitempty"`
		Lane           string            `json:"lane"`
		TimeoutSeconds int64             `json:"timeout_seconds"`
		MemoryBytes    uint64            `json:"memory_bytes,omitempty"`
		Pictures       *int64            `json:"pictures,omitempty"`
	}
	store, request := fixture(t)
	request.Environment = map[string]string{"SLAYER_OBSERVE_FRAMES": "on"}
	first, err := store.Submit(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.transact(t.Context(), false, func(snapshot *Snapshot) error {
		raw, e := json.Marshal(snapshot.Jobs[0].Request)
		if e != nil {
			return e
		}
		var old oldRequest
		if e = json.Unmarshal(raw, &old); e != nil {
			return e
		}
		raw, e = json.Marshal(old)
		if e != nil {
			return e
		}
		var rewritten Request
		if e = json.Unmarshal(raw, &rewritten); e != nil {
			return e
		}
		snapshot.Jobs[0].Request = rewritten
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	repeated, err := store.Submit(t.Context(), request)
	if err != nil || repeated.Sequence != first.Sequence || !observeFrames(repeated.Request) {
		t.Fatal(repeated, err)
	}
	// An older worker passes this already-supported environment entry normally.
	env := environment(repeated.Request)
	if !slices.Contains(env, "SLAYER_OBSERVE_FRAMES=on") {
		t.Fatal(env)
	}
}

func TestInheritedObserverEnvironmentCannotOptInOrLeakHandoff(t *testing.T) {
	t.Setenv("SLAYER_OBSERVE_FRAMES", "on")
	for _, name := range []string{"AZAZEL_FRAME_FD", "AZAZEL_FRAME_NONCE", "AZAZEL_FRAME_SEQUENCE", "AZAZEL_FRAME_ATTEMPT"} {
		t.Setenv(name, "stale")
	}
	request := Request{}
	if observeFrames(request) {
		t.Fatal("inherited env opted in")
	}
	for _, entry := range environment(request) {
		if strings.HasPrefix(entry, "AZAZEL_FRAME_") {
			t.Fatal("inherited handoff leaked", entry)
		}
	}
}

func TestFrameReceiptHelper(t *testing.T) {
	if os.Getenv("AZAZEL_FRAME_FD") == "" {
		return
	}
	stop, err := observability.StartFrameEventsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	observability.EmitFrame(observability.FrameEvent{Frame: 0, SceneTimeKnown: true, ObservedAt: time.Now(), DrawKnown: true})
	observability.EmitFrame(observability.FrameEvent{Frame: 1, SceneTimeKnown: true, ObservedAt: time.Now(), Kept: true})
	stop()
}

func TestFrameReceiptRunsThroughRealQueueAttempt(t *testing.T) {
	store, request := fixture(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	request.Command = self
	request.Args = []string{"-test.run=^TestFrameReceiptHelper$"}
	request.Environment = map[string]string{"SLAYER_OBSERVE_FRAMES": "on"}
	if _, err = store.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	job, err := store.claim(t.Context(), "frame-worker")
	if err != nil {
		t.Fatal(err)
	}
	result := store.execute(t.Context(), job, io.Discard, observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil))
	if result.State != Succeeded || result.Uncertain {
		t.Fatal(result)
	}
	path := filepath.Join(store.root, fmt.Sprintf("%020d.%04d.frames.jsonl", job.Sequence, job.Attempt))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 4 {
		t.Fatalf("receipt: %s", raw)
	}
	var first, second, end observability.FramePacket
	if json.Unmarshal([]byte(lines[0]), &first) != nil || json.Unmarshal([]byte(lines[1]), &second) != nil || json.Unmarshal([]byte(lines[2]), &end) != nil {
		t.Fatal(string(raw))
	}
	if first.Frame != 0 || first.Kept || second.Frame != 1 || !second.Kept || end.Kind != "end" || first.Nonce == "" || first.Nonce != end.Nonce || first.Sequence != job.Sequence || first.Attempt != job.Attempt || !strings.Contains(lines[3], `"complete":true`) {
		t.Fatal(string(raw))
	}
}

func TestFrameReceiptRejectsWrongAttemptAndTruncation(t *testing.T) {
	r := frameReceipt{nonce: "0123456789abcdef0123456789abcdef", sequence: 3, attempt: 2}
	raw, _ := json.Marshal(observability.FramePacket{Version: 1, Kind: "frame", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: 1})
	if _, err := r.packet(raw, 0, 0, false); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(raw), r.nonce, "1123456789abcdef0123456789abcdef", 1)
	if _, err := r.packet([]byte(bad), 0, 0, false); err == nil {
		t.Fatal("accepted wrong nonce")
	}
	if _, err := r.packet(raw, 0, 1, false); err == nil {
		t.Fatal("accepted duplicate ordinal")
	}
	if _, err := r.packet(raw, 0, 0, true); err == nil {
		t.Fatal("accepted after terminal")
	}
	if _, err := r.packet([]byte(`{"unknown":1}`), 0, 0, false); err == nil {
		t.Fatal("accepted malformed packet")
	}
}

func TestFrameReceiptFitsTwentyMinute24FPSAndMarksOverflow(t *testing.T) {
	r := frameReceipt{nonce: "0123456789abcdef0123456789abcdef", sequence: 3, attempt: 2, limit: frameReceiptLimit}
	var state receiptState
	const frames = 20 * 60 * 24
	for i := 0; i < frames; i++ {
		packet := observability.FramePacket{Version: 1, Kind: "frame", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: uint64(i + 1), FrameEvent: observability.FrameEvent{Frame: i, ObservedAt: time.Unix(100, 0), SceneTime: time.Duration(i) * time.Second / 24, SceneTimeKnown: true, DrawKnown: true}}
		raw, err := json.Marshal(packet)
		if err != nil {
			t.Fatal(err)
		}
		if err = r.accept(&state, raw, 0, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	terminal := observability.FramePacket{Version: 1, Kind: "end", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: frames, Attempted: frames}
	raw, _ := json.Marshal(terminal)
	if err := r.accept(&state, raw, 0, io.Discard); err != nil || !state.complete() {
		t.Fatal(state, err)
	}
	if state.size >= frameReceiptLimit-frameTrailerReserve {
		t.Fatal("20 minute receipt too large", state.size)
	}
	r.limit = 512
	var overflow receiptState
	for i := 0; i < 2; i++ {
		packet := observability.FramePacket{Version: 1, Kind: "frame", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: uint64(i + 1)}
		raw, _ := json.Marshal(packet)
		if err := r.accept(&overflow, raw, 0, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	terminal.Ordinal = 2
	terminal.Attempted = 2
	raw, _ = json.Marshal(terminal)
	if err := r.accept(&overflow, raw, 0, io.Discard); err != nil || !overflow.overflow || overflow.complete() {
		t.Fatal(overflow, err)
	}
}

func TestFrameReceiptCannotCompleteWithDroppedOrInconsistentTerminal(t *testing.T) {
	r := frameReceipt{nonce: "0123456789abcdef0123456789abcdef", sequence: 3, attempt: 2}
	for _, end := range []observability.FramePacket{
		{Version: 1, Kind: "end", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: 0, Attempted: 1, Dropped: 1},
		{Version: 1, Kind: "end", Nonce: r.nonce, Sequence: 3, Attempt: 2, Ordinal: 0, Attempted: 1, Dropped: 0},
	} {
		var state receiptState
		raw, _ := json.Marshal(end)
		if err := r.accept(&state, raw, 0, io.Discard); err != nil || state.complete() {
			t.Fatal(state, err)
		}
	}
}
