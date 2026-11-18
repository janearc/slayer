package omnipositor

import (
	"bytes"
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

// snapshotInputs are the score and cue texts the snapshot tests build from.
var (
	snapshotScore = []byte("score: {beats: {times: [0, 1, 2, 3]}}")
	snapshotCues  = []byte("cues: [{at_sec: 0, fire: 1}]")
)

// newSnapshot builds a snapshot from the test settings with fire frames, and
// stops the test if that fails.
func newSnapshot(t *testing.T) (*ExecutionSnapshot, settings.Settings) {
	t.Helper()
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	e, err := NewExecutionSnapshot(context.Background(), s, executionConfig(t), lights.DefaultConfig(), snapshotScore, snapshotCues, executionBinary(t), "cpu", executionDevice())
	if err != nil {
		t.Fatal(err)
	}
	return e, s
}

// TestNewExecutionSnapshotRefusals checks each input the snapshot refuses
// before it builds anything, and that each error names its cause.
func TestNewExecutionSnapshotRefusals(t *testing.T) {
	hash := executionBinary(t)
	build := func(ctx context.Context, s settings.Settings, c Config, l lights.Config, score, sheet []byte) (*ExecutionSnapshot, error) {
		return NewExecutionSnapshot(ctx, s, c, l, score, sheet, hash, "cpu", executionDevice())
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	var noContext context.Context
	if e, err := build(noContext, assetSettings(t), executionConfig(t), lights.DefaultConfig(), snapshotScore, snapshotCues); e != nil || err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Errorf("nil context: %v", err)
	}
	if e, err := build(ended, assetSettings(t), executionConfig(t), lights.DefaultConfig(), snapshotScore, snapshotCues); e != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
	big := bytes.Repeat([]byte(" "), MaxExecutionSourceBytes+1)
	for name, c := range map[string]struct{ score, sheet []byte }{
		"empty score":   {nil, snapshotCues},
		"empty cues":    {snapshotScore, nil},
		"score too big": {big, snapshotCues},
		"cues too big":  {snapshotScore, big},
	} {
		if e, err := build(context.Background(), assetSettings(t), executionConfig(t), lights.DefaultConfig(), c.score, c.sheet); e != nil || err == nil || !strings.Contains(err.Error(), "nonempty and at most") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := build(context.Background(), assetSettings(t), executionConfig(t), lights.DefaultConfig(), []byte("score: [unclosed"), snapshotCues); err == nil || !strings.Contains(err.Error(), "execution: show") {
		t.Errorf("bad score: %v", err)
	}
	wide := executionConfig(t)
	wide.Width = 3
	if _, err := build(context.Background(), assetSettings(t), wide, lights.DefaultConfig(), snapshotScore, snapshotCues); err == nil || !strings.Contains(err.Error(), "dimensions differ") {
		t.Errorf("dimensions: %v", err)
	}
	missing := assetSettings(t)
	if err := os.Remove(missing.Passes.Disc); err != nil {
		t.Fatal(err)
	}
	if _, err := build(context.Background(), missing, executionConfig(t), lights.DefaultConfig(), snapshotScore, snapshotCues); err == nil || !strings.Contains(err.Error(), "passes.disc") {
		t.Errorf("missing pass: %v", err)
	}
	broken := lights.DefaultConfig()
	broken.HazeGain = math.NaN()
	if _, err := build(context.Background(), assetSettings(t), executionConfig(t), broken, snapshotScore, snapshotCues); err == nil || !strings.Contains(err.Error(), "execution: config") {
		t.Errorf("config that cannot be written: %v", err)
	}
	flat := executionConfig(t)
	flat.MoonRadius = 0
	if _, err := build(context.Background(), assetSettings(t), flat, lights.DefaultConfig(), snapshotScore, snapshotCues); err == nil || !strings.Contains(err.Error(), "invalid geometry") {
		t.Errorf("geometry: %v", err)
	}
	still := executionConfig(t)
	still.FPS = 0
	if _, err := build(context.Background(), assetSettings(t), still, lights.DefaultConfig(), snapshotScore, snapshotCues); err == nil {
		t.Error("accepted a frame rate of zero")
	}
}

// TestPrepareFrameRefusals checks the frames a snapshot will not prepare: a
// snapshot that is not there, and a frame outside the range it was built for.
func TestPrepareFrameRefusals(t *testing.T) {
	e, _ := newSnapshot(t)
	for _, index := range []int{-1, 3, 100} {
		if _, err := e.PrepareFrame(context.Background(), index); err == nil || !strings.Contains(err.Error(), "outside range") {
			t.Errorf("frame %d: %v", index, err)
		}
	}
	var none *ExecutionSnapshot
	if _, err := none.PrepareFrame(context.Background(), 0); !errors.Is(err, ErrUnboundExecutionSnapshot) {
		t.Errorf("nil snapshot: %v", err)
	}
	if _, err := (&ExecutionSnapshot{}).PrepareFrame(context.Background(), 0); !errors.Is(err, ErrUnboundExecutionSnapshot) {
		t.Errorf("empty snapshot: %v", err)
	}
	keys := map[string]bool{}
	for _, index := range []int{0, 1, 2} {
		p, err := e.PrepareFrame(context.Background(), index)
		if err != nil || p.request.Index != index {
			t.Fatalf("frame %d: %v", index, err)
		}
		keys[p.Key().String()] = true
	}
	if len(keys) != 3 {
		t.Errorf("frames 0, 1 and 2 share keys: %v", keys)
	}
}

// TestPreparedFrameAccessors checks the key and cache hits of a frame that is
// not there, and of a frame with no fire.
func TestPreparedFrameAccessors(t *testing.T) {
	var none *PreparedFrame
	if key := none.Key(); key != (identity.Key{}) {
		t.Errorf("key of nil: %v", key)
	}
	if hits := none.CacheHits(); hits != nil {
		t.Errorf("hits of nil: %v", hits)
	}
	if hits := (&PreparedFrame{}).CacheHits(); hits != nil {
		t.Errorf("hits with no fire snapshot: %v", hits)
	}
	e, _ := newSnapshot(t)
	p, err := e.PrepareFrame(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits := p.CacheHits(); len(hits) != 0 {
		t.Errorf("first preparation reported hits: %v", hits)
	}
	if p.Key().String() == "" {
		t.Error("prepared frame has no key")
	}
}

// TestPreparedFrameRenderRefusals checks the refusals of Render that come
// before any drawing: no context, an ended context, and a frame whose owner
// is gone.
func TestPreparedFrameRenderRefusals(t *testing.T) {
	e, _ := newSnapshot(t)
	p, err := e.PrepareFrame(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	consume := func(Result) error { called = true; return nil }
	var noContext context.Context
	if err := p.Render(noContext, consume); err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Errorf("nil context: %v", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.Render(ended, consume); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
	var none *PreparedFrame
	if err := none.Render(context.Background(), consume); !errors.Is(err, ErrUnboundExecutionSnapshot) {
		t.Errorf("nil frame: %v", err)
	}
	orphan := &PreparedFrame{request: p.request}
	if err := orphan.Render(context.Background(), consume); !errors.Is(err, ErrUnboundExecutionSnapshot) {
		t.Errorf("frame with no owner: %v", err)
	}
	if called {
		t.Error("a refused render delivered a frame")
	}
}

// TestPreparedLightRendersItsOwnRequest checks that the prepared light stage
// accepts exactly the request the frame was prepared with and returns light
// of the frame's size.
func TestPreparedLightRendersItsOwnRequest(t *testing.T) {
	e, _ := newSnapshot(t)
	p, err := e.PrepareFrame(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	request := p.request
	request.FireSnapshot = nil
	got, err := (preparedLight{p: p}).Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sky.Width != 2 || got.Sky.Height != 2 || len(got.Sky.Light) != 12 || got.Front.Width != 2 || len(got.Front.Light) != 12 {
		t.Errorf("light of the wrong shape: sky %dx%d, %d values", got.Sky.Width, got.Sky.Height, len(got.Sky.Light))
	}
}
