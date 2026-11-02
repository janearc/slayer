package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

// grey is a frame of ten by ten, every pixel the same grey.
func grey(level float32) *frame.Frame {
	made := frame.New(10, 10)
	for y := range 10 {
		for x := range 10 {
			made.Set(x, y, frame.Pixel{R: level, G: level, B: level,
				Cover: 1, Depth: 1})
		}
	}
	return made
}

// kept is a frame written as a raw in a test's own directory.
func kept(t *testing.T, name string, picture *frame.Frame) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := raw.Write(path, picture); err != nil {
		t.Fatal(err)
	}
	return path
}

// two raws the same are zero apart, and within the budget; one pixel in a
// hundred made white in a grey picture is one in a hundred past one jnd, and
// the worst is far past it, and still within.
func TestApartSaysHowFarApart(t *testing.T) {
	same := kept(t, "same.pfm", grey(0.2))
	again := kept(t, "again.pfm", grey(0.2))
	spotted := grey(0.2)
	spotted.Set(3, 4, frame.Pixel{R: 1, G: 1, B: 1, Cover: 1, Depth: 1})
	spot := kept(t, "spot.pfm", spotted)
	for _, each := range []struct {
		other, want string
	}{
		{again, "0.0000% of 100 pixels past one jnd, worst 0.00 jnd " +
			"at 0, 0; within"},
		{spot, "1.0000% of 100 pixels past one jnd"},
		{spot, "at 3, 4;"},
	} {
		var said, wrong bytes.Buffer
		if code := run([]string{same, each.other}, &said, &wrong); code != 0 {
			t.Fatalf("it gave back %d: %s", code, wrong.String())
		}
		if !strings.Contains(said.String(), each.want) {
			t.Errorf("it said %q, not %q", said.String(), each.want)
		}
	}
}

// what cannot be compared is said so: one raw is a usage, two for its
// number; a raw that is not there is one.
func TestApartRefusesWhatItCannotCompare(t *testing.T) {
	same := kept(t, "same.pfm", grey(0.2))
	var said, wrong bytes.Buffer
	if code := run([]string{same}, &said, &wrong); code != 2 {
		t.Errorf("one raw gave back %d", code)
	}
	nowhere := filepath.Join(t.TempDir(), "nowhere.pfm")
	if code := run([]string{same, nowhere}, &said, &wrong); code != 1 ||
		!strings.Contains(wrong.String(), "apart:") {
		t.Errorf("a raw not there gave back %d: %s", code, wrong.String())
	}
}
