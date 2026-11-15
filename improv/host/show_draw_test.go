//go:build draw

package host

import (
	"errors"
	"image"
	"image/color"
	"testing"
)

const (
	frameWide = 270
	frameTall = 480
	shortLong = 48
	introLong = 0.5
	grey      = 100
)

func grey100(int) (*image.RGBA, error) {
	frame := image.NewRGBA(image.Rect(none, none, frameWide, frameTall))
	for i := range frame.Pix {
		frame.Pix[i] = grey
	}
	return frame, nil
}

func testShow(t *testing.T) Show {
	t.Helper()
	piece, sheet, sound := libraryTune(t)
	var kicking []string
	for _, part := range sheet.Parts {
		kicking = append(kicking, part.Name)
	}
	tune, err := NewTune(piece.Score, sheet, sound, kicking)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewBox(image.Rect(none, frameTall/two, frameWide,
		frameTall), image.Pt(frameWide/4, frameWide/4))
	if err != nil {
		t.Fatal(err)
	}
	lifts := map[string]bool{}
	for _, name := range kicking {
		lifts[name] = true
	}
	return Show{Frames: grey100, Count: shortLong, Tune: tune, Box: box,
		Speaks: []Span{{From: none, Until: one}}, Lifts: lifts,
		Picture: image.Rect(none, none, frameWide, frameTall/two),
		Intro:   Intro{Words: "hi! i am zappy.", Seconds: introLong},
		Seed:    seed, Size: image.Pt(frameWide, frameTall)}
}

func TestTheIntroIsWordsOnBlack(t *testing.T) {
	show := testShow(t)
	if show.Total() != shortLong+int(introLong*filmRate) {
		t.Fatalf("the film is %d frames", show.Total())
	}
	frame, err := show.Frame(none)
	if err != nil {
		t.Fatal(err)
	}
	lit := none
	for y := range frameTall {
		for x := range frameWide {
			if frame.RGBAAt(x, y).R > grey {
				lit++
			}
		}
	}
	if lit == none || frame.RGBAAt(none, none) != (color.RGBA{A: full}) {
		t.Fatalf("the intro has %d lit pixels on %v", lit,
			frame.RGBAAt(none, none))
	}
}

func TestAShortFrameKeepsItsPictureAndOutlinesTheHost(t *testing.T) {
	show := testShow(t)
	for i := show.Total() - shortLong; i < show.Total(); i++ {
		frame, err := show.Frame(i)
		if err != nil {
			t.Fatal(err)
		}
		if got := frame.RGBAAt(frameWide-one, none).R; got < grey {
			t.Fatalf("frame %d darkened the picture to %d", i, got)
		}
		seconds := float64(i) / filmRate
		at := show.Tune.At(seconds, show.speaks(
			float64(i-show.introFrames())/filmRate))
		place := show.Box.Place(at, seconds, show.Seed)
		if got := frame.RGBAAt(place.X, place.Y).R; got != placeholderInk {
			t.Fatalf("frame %d has no outline at %v: %d", i, place, got)
		}
	}
}

func TestAFrameThatCannotBeMadeSaysSo(t *testing.T) {
	show := testShow(t)
	show.Frames = func(int) (*image.RGBA, error) {
		return nil, errors.New("no picture")
	}
	if _, err := show.Frame(show.Total() - one); err == nil {
		t.Fatal("a missing picture made a frame")
	}
}

func TestHostClipsAreLaidAsLight(t *testing.T) {
	show := testShow(t)
	white := image.NewRGBA(image.Rectangle{Max: show.Box.size})
	for i := range white.Pix {
		white.Pix[i] = full
	}
	loop := Loop{frames: []*image.RGBA{white}, rate: filmRate}
	show.Clips = Clips{Resting: loop, Speaking: loop, Angry: loop,
		Flash: loop, Quiet: loop}
	frame, err := show.Frame(show.Total() - one)
	if err != nil {
		t.Fatal(err)
	}
	seconds := float64(show.Total()-one) / filmRate
	at := show.Tune.At(seconds, false)
	place := show.Box.Place(at, seconds, show.Seed)
	if got := frame.RGBAAt(place.X+one, place.Y+one).R; got <= grey {
		t.Fatalf("the host's light did not show: %d", got)
	}
	if loop.At(-one) != white {
		t.Fatal("a loop before its start is not its last turn")
	}
}

func TestReadClipsWithNoFolderIsAPlaceholder(t *testing.T) {
	clips, err := ReadClips(t.TempDir()+"/none", image.Pt(one, one))
	if err != nil || len(clips) != none {
		t.Fatalf("no folder gave %v, %v", clips, err)
	}
	if _, err := ReadClips(t.TempDir(), image.Pt(one, one)); err == nil {
		t.Fatal("a folder with no clips was taken")
	}
}
