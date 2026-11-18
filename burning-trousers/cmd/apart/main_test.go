package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/maths"
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

// one pixel set to a level in a ten by ten grey of 0.2.
func marked(x, y int, level float32) *frame.Frame {
	made := grey(0.2)
	made.Set(x, y, frame.Pixel{R: level, G: level, B: level, Cover: 1,
		Depth: 1})
	return made
}

// what run says and gives back for two raws.
func ran(t *testing.T, one, other string) (said, wrong string, code int) {
	t.Helper()
	var out, err bytes.Buffer
	code = run([]string{one, other}, &out, &err)
	return out.String(), err.String(), code
}

// A pixel past one jnd and not past two counts in the share past one: a
// grey of 0.23 is about one and a third jnd from a grey of 0.2.
func TestApartCountsAPixelBetweenOneAndTwoJNDsAsPastOne(t *testing.T) {
	const level = 0.23
	between := divergence.Apart(maths.RGB(0.2, 0.2, 0.2),
		maths.RGB(level, level, level))
	if !(between > 1 && between < 2) {
		t.Fatalf("a grey of %v is %v jnd from 0.2, not between one and two",
			level, float64(between))
	}
	said, wrong, code := ran(t, kept(t, "a.pfm", grey(0.2)),
		kept(t, "b.pfm", marked(7, 2, level)))
	if code != 0 {
		t.Fatalf("it gave back %d: %s", code, wrong)
	}
	if !strings.Contains(said, "1.0000% of 100 pixels past one jnd") ||
		!strings.Contains(said, "at 7, 2; within") {
		t.Errorf("it said %q", said)
	}
}

// More than the allowed share of pixels past one jnd is outside the
// budget, and it says so; the budget's own share is within.
func TestApartSaysOutsideWhenTooManyPixelsAreFarApart(t *testing.T) {
	over := grey(0.2)
	for x := range 3 {
		over.Set(x, 0, frame.Pixel{R: 1, G: 1, B: 1, Cover: 1, Depth: 1})
	}
	said, _, code := ran(t, kept(t, "a.pfm", grey(0.2)),
		kept(t, "b.pfm", over))
	if code != 0 || !strings.Contains(said, "3.0000% of 100 pixels past "+
		"one jnd") || !strings.Contains(said, "; outside the still budget") {
		t.Errorf("three in a hundred past one gave %d: %q", code, said)
	}
	edge := grey(0.2)
	for x := range 2 {
		edge.Set(x, 0, frame.Pixel{R: 1, G: 1, B: 1, Cover: 1, Depth: 1})
	}
	said, _, code = ran(t, kept(t, "c.pfm", grey(0.2)), kept(t, "d.pfm", edge))
	if code != 0 || !strings.Contains(said, "; within the still budget") {
		t.Errorf("two in a hundred past one gave %d: %q", code, said)
	}
}

// It says what is at the worst pixel in each picture, on a line of its
// own.
func TestApartSaysWhatIsAtTheWorstPixel(t *testing.T) {
	said, _, code := ran(t, kept(t, "a.pfm", grey(0.2)),
		kept(t, "b.pfm", marked(3, 4, 1)))
	if code != 0 || strings.Count(said, "\n") != 2 ||
		!strings.Contains(said, "\nthere, the first is ") ||
		!strings.Contains(said, " and the second ") {
		t.Errorf("it said %q", said)
	}
}

// A raw that cannot be read, first or second, and two that are not one
// size, are one apiece: said on the error writer, with nothing on the
// other.
func TestApartGivesBackOneAndSaysWhyForEachRawItCannotUse(t *testing.T) {
	good := kept(t, "good.pfm", grey(0.2))
	nowhere := filepath.Join(t.TempDir(), "nowhere.pfm")
	small := frame.New(4, 4)
	smaller := kept(t, "small.pfm", small)
	for name, each := range map[string][2]string{
		"the first not there":  {nowhere, good},
		"the second not there": {good, nowhere},
		"not one size":         {good, smaller},
	} {
		said, wrong, code := ran(t, each[0], each[1])
		if code != 1 {
			t.Errorf("%s gave back %d, not one", name, code)
		}
		if !strings.HasPrefix(wrong, "apart: ") {
			t.Errorf("%s said %q on the error writer", name, wrong)
		}
		if said != "" {
			t.Errorf("%s said %q as if it had compared", name, said)
		}
	}
}

// Three raws, or none, are a usage and two; the usage is said on the
// error writer.
func TestApartSaysUsageForTheWrongNumberOfRaws(t *testing.T) {
	for _, arguments := range [][]string{nil, {"a", "b", "c"}} {
		var said, wrong bytes.Buffer
		if code := run(arguments, &said, &wrong); code != 2 {
			t.Errorf("%d arguments gave back %d", len(arguments), code)
		}
		if wrong.String() != "usage: apart one.pfm other.pfm\n" ||
			said.Len() != 0 {
			t.Errorf("%d arguments said %q and %q", len(arguments),
				said.String(), wrong.String())
		}
	}
}

// The worst pixel is where the two are furthest apart, the first such
// row by row: at index nought when that is the worst, at the first of
// two as bad, wherever the only difference is however small, and at
// nought when all are the same or there are no pixels.
func TestWorstAtIsTheFirstPixelFurthestApart(t *testing.T) {
	for name, each := range map[string]struct {
		one, other *frame.Frame
		x, y       int
	}{
		"worst at index nought": {grey(0.2), marked(0, 0, 1), 0, 0},
		"all the same":          {grey(0.2), grey(0.2), 0, 0},
		"the first of two as bad": {grey(0.2), func() *frame.Frame {
			made := marked(6, 1, 1)
			made.Set(2, 5, frame.Pixel{R: 1, G: 1, B: 1, Cover: 1, Depth: 1})
			return made
		}(), 6, 1},
		"a small difference alone": {grey(0.2), marked(5, 7, 0.2005), 5, 7},
		"no pixels":                {frame.New(5, 0), frame.New(5, 0), 0, 0},
	} {
		x, y := worstAt(each.one, each.other)
		if x != each.x || y != each.y {
			t.Errorf("%s: at %d, %d, not %d, %d", name, x, y, each.x, each.y)
		}
	}
}

// main takes its arguments after the command's own name: run as a
// command, two raws that are the same are compared and give back zero.
func TestMainComparesTheTwoRawsItIsGiven(t *testing.T) {
	if os.Getenv("APART_RUNS_MAIN") == "1" {
		os.Args = append([]string{"apart"}, strings.Fields(
			os.Getenv("APART_ARGUMENTS"))...)
		main()
		return
	}
	same := kept(t, "same.pfm", grey(0.2))
	other := kept(t, "other.pfm", marked(1, 1, 1))
	command := exec.Command(os.Args[0], "-test.run=^TestMainComparesTheTwoRawsItIsGiven$")
	command.Env = append(os.Environ(), "APART_RUNS_MAIN=1",
		"APART_ARGUMENTS="+same+" "+other)
	var said, wrong bytes.Buffer
	command.Stdout, command.Stderr = &said, &wrong
	if err := command.Run(); err != nil {
		t.Fatalf("main gave back %v: %s", err, wrong.String())
	}
	if !strings.Contains(said.String(), "at 1, 1;") {
		t.Errorf("main said %q", said.String())
	}
}
