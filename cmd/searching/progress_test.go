package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestProgressColumn keeps an unknown estimate distinct from completion
// and replaces a longer previous snapshot without leaving trailing bytes.
func TestProgressColumn(t *testing.T) {
	remaining := 61.1
	reading := progressReading{Updated: time.Unix(1000, 0), Status: "observing",
		Written: 3, Total: 10, PerMinute: 12.6, Remaining: &remaining}
	if got := reading.column(); got != "3 10 13 62 1000\n" {
		t.Fatal(got)
	}
	for _, status := range []string{"no recent frame", "stopped without completion", "PNGs written; movie completion pending"} {
		reading.Status = status
		if got := reading.column(); got != "3 10 13 -1 1000\n" {
			t.Fatal(status, got)
		}
	}
	reading.Status, reading.Remaining = "observing", nil
	if got := reading.column(); got != "3 10 13 -1 1000\n" {
		t.Fatal(got)
	}
	reading.Status, reading.Written = "completed movie", 10
	path := filepath.Join(t.TempDir(), "render-progress.txt")
	if err := os.WriteFile(path, bytes.Repeat([]byte("old"), 100), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishProgress(path, []byte(reading.column())); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "10 10 13 0 1000\n" {
		t.Fatal(string(raw), err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatal("temporary snapshot left behind", entries, err)
	}
}

// TestProgressCountsFinishedPNGs excludes a currently written file and
// frames outside the excerpt even when they share the same directory.
func TestProgressCountsFinishedPNGs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "frames"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"frame-000010.png", "frame-000009.png", "frame-000011.png"} {
		if err := writePNG(filepath.Join(root, "frames", name), image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "frames", "frame-000011.png")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, info.Size()-12); err != nil {
		t.Fatal(err)
	}
	reading, err := scanProgress(root, 10, 2, time.Now())
	if err != nil || reading.Written != 1 || reading.Percent != 50 || reading.Frame != "frames/frame-000010.png" {
		t.Fatalf("partial or foreign frame counted: %+v, %v", reading, err)
	}
	if reading.Bytes == 0 || reading.ETA != nil {
		t.Fatal("bytes missing or premature ETA")
	}
}

// TestProgressRecentPace drops the cheap opening slate from the estimate
// once enough expensive frames have arrived, and withdraws a stale ETA.
func TestProgressRecentPace(t *testing.T) {
	start := time.Unix(1000, 0)
	var frames []completedPNG
	for index := range 40 {
		frames = append(frames, completedPNG{at: start.Add(time.Duration(index) * time.Millisecond)})
	}
	for index := range 30 {
		frames = append(frames, completedPNG{at: start.Add(time.Duration(index+1) * 4 * time.Second)})
	}
	now := start.Add(120 * time.Second)
	reading := measureProgress(progressReading{Total: 100}, frames, now)
	if reading.PerMinute != 15 || reading.SecondsFrame != 4 || reading.Remaining == nil || *reading.Remaining != 120 {
		t.Fatalf("title distorted the recent pace: %+v", reading)
	}
	if reading.ETA == nil || !reading.ETA.Equal(now.Add(120*time.Second)) {
		t.Fatal("wrong ETA")
	}
	stale := measureProgress(progressReading{Total: 100}, frames, now.Add(time.Minute))
	if stale.ETA != nil || stale.Remaining != nil || stale.Status != "no recent frame" {
		t.Fatalf("stale ETA: %+v", stale)
	}
}

// TestProgressCompletionNeedsReceipt prevents a directory full of PNGs
// from being reported as an accepted, complete movie.
func TestProgressCompletionNeedsReceipt(t *testing.T) {
	root := t.TempDir()
	if done, err := progressComplete(root, 10, 2); done || err != nil {
		t.Fatal(done, err)
	}
	if err := os.WriteFile(filepath.Join(root, "render.json"), []byte(`{"source_start_frame":10,"frames":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if done, err := progressComplete(root, 10, 2); done || err == nil {
		t.Fatal("missing movie accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "animation.mp4"), []byte("producer output"), 0600); err != nil {
		t.Fatal(err)
	}
	if done, err := progressComplete(root, 10, 2); !done || err != nil {
		t.Fatal(done, err)
	}
	if done, err := progressComplete(root, 0, 2); done || err == nil {
		t.Fatal("wrong excerpt accepted")
	}
}

// statusTransport is an in-memory Barkeep endpoint for client contracts.
func TestProgressFrameSetNeedsExactCompletedReceipt(t *testing.T) {
	root := t.TempDir()
	for _, row := range []struct {
		raw       string
		done, bad bool
	}{
		{`{"status":"running","frames":[]}`, false, false},
		{`{"status":"complete","frames":[]}`, false, true},
		{`{"status":"complete","frames":[{"file":"frame-000001.png"}]}`, false, true},
		{`{"status":"complete","frames":[{"file":"frame-000000.png"}]}`, true, false},
	} {
		if err := os.WriteFile(filepath.Join(root, "receipt.json"), []byte(row.raw), 0600); err != nil {
			t.Fatal(err)
		}
		done, err := progressFramesComplete(root, 0, 1)
		if done != row.done || (err != nil) != row.bad {
			t.Fatal(row, done, err)
		}
	}
	r := progressReading{Status: "completed frame set", Written: 1, Total: 1, Updated: time.Unix(1000, 0)}
	if r.column() != "1 1 0 0 1000\n" {
		t.Fatal(r.column())
	}
}

type statusTransport func(*http.Request) (*http.Response, error)

// RoundTrip handles requests without opening a socket or touching a bar.
func (transport statusTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

// TestBarkeepQueueDoesNotAccumulate proves repeated progress does not pile
// up behind a message awaiting presentation, while the final signal updates.
func TestBarkeepQueueDoesNotAccumulate(t *testing.T) {
	queued := 0
	var actions []string
	client := &http.Client{Transport: statusTransport(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodPost {
			if request.Header.Get("Content-Type") != "application/json" {
				t.Error("missing JSON content type")
			}
			var body map[string]string
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if request.URL.Path == "/say" {
				if !strings.Contains(body["text"], "1/10") || body["for"] != "5s" {
					t.Error(body)
				}
				actions = append(actions, "say")
				queued++
			} else {
				actions = append(actions, body["state"])
			}
		}
		raw, _ := json.Marshal(map[string]int{"queued": queued})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw))}, nil
	})}
	bar := &barkeepProgress{address: "http://127.0.0.1:1", kind: "render", signal: "film", client: client}
	reading := progressReading{Updated: time.Unix(1000, 0), Total: 10, Written: 1, Percent: 10, Status: "observing"}
	for range 3 {
		if err := bar.update(context.Background(), reading); err != nil {
			t.Fatal(err)
		}
		reading.Updated = reading.Updated.Add(2 * time.Minute)
	}
	reading.Status = "completed movie"
	if err := bar.update(context.Background(), reading); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actions, []string{"running", "say", "pass"}) {
		t.Fatalf("stale queue: %v", actions)
	}
}

// TestBarkeepLoopbackOnly prevents progress data from leaving the host via
// an address, credentials, path or a DNS name supplied as the bar target.
func TestBarkeepLoopbackOnly(t *testing.T) {
	for _, address := range []string{"https://127.0.0.1:80", "http://example.com:80", "http://192.168.1.1:80", "http://localhost:80", "http://a:b@127.0.0.1:80", "http://127.0.0.1:80/path", "http://127.0.0.1:80?q=secret"} {
		if _, err := loopbackBar(address); err == nil {
			t.Errorf("accepted %s", address)
		}
	}
	if got, err := loopbackBar("http://127.0.0.1:8765/"); err != nil || got != "http://127.0.0.1:8765" {
		t.Fatal(got, err)
	}
	if _, err := loopbackBar("http://[::1]:8765"); err != nil {
		t.Fatal(err)
	}
}
