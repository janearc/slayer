package develop

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

func fixture() *frame.Frame {
	f := frame.New(5, 2)
	for i := range f.Pix {
		f.Pix[i] = frame.Pixel{R: float32(i) * .19, G: float32(i) * .31, B: float32(i) * .07, Cover: 1, Depth: 2}
	}
	f.Pix[3] = frame.Pixel{R: 20, G: 12, B: 8}
	return f
}

func TestDefaultsAndExplicitTone(t *testing.T) {
	defaults := DefaultOptions()
	if defaults != (Options{Knee: .7, Top: 8}) {
		t.Fatalf("defaults %+v", defaults)
	}
	for _, options := range []Options{defaults, {Exposure: -1.25, Knee: .4, Top: 12, Bloom: 1.2}, {Knee: 0, Top: 0}} {
		input := fixture()
		before := append([]frame.Pixel(nil), input.Pix...)
		frames := []*frame.Frame{input, input}
		got, err := Develop(frames, options)
		if err != nil {
			t.Fatal(err)
		}
		total, err := raw.Sum(frames)
		if err != nil {
			t.Fatal(err)
		}
		want := (colour.Tone{Exposure: math.Exp2(options.Exposure), Knee: options.Knee, Top: options.Top, Bloom: options.Bloom}).Develop(total)
		if got.Rect != want.Rect || !bytes.Equal(got.Pix, want.Pix) {
			t.Fatalf("tone mismatch for %+v", options)
		}
		if !reflect.DeepEqual(input.Pix, before) {
			t.Fatal("changed input")
		}
	}
}

func TestSumEqualsOneStop(t *testing.T) {
	input := fixture()
	options := DefaultOptions()
	two, err := Develop([]*frame.Frame{input, input}, options)
	if err != nil {
		t.Fatal(err)
	}
	options.Exposure = 1
	one, err := Develop([]*frame.Frame{input}, options)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(one.Pix, two.Pix) {
		t.Fatal("additive exposure differs")
	}
}

func TestWhiteBalanceAndInputOwnership(t *testing.T) {
	input := fixture()
	before := append([]frame.Pixel(nil), input.Pix...)
	options := DefaultOptions()
	plain, err := Develop([]*frame.Frame{input}, options)
	if err != nil {
		t.Fatal(err)
	}
	options.White = 3200
	balanced, err := Develop([]*frame.Frame{input}, options)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(plain.Pix, balanced.Pix) {
		t.Fatal("ignored white")
	}
	again, err := Develop([]*frame.Frame{input}, options)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Pix, balanced.Pix) || !reflect.DeepEqual(input.Pix, before) {
		t.Fatal("development changed input or retained state")
	}
}

func TestNilFramesAreRefused(t *testing.T) {
	for _, frames := range [][]*frame.Frame{{nil}, {frame.New(1, 1), nil}} {
		if picture, err := Develop(frames, DefaultOptions()); err == nil || picture != nil {
			t.Fatalf("accepted nil frame: %v, %v", picture, err)
		}
	}
	if err := Balance(nil, 3200); err == nil {
		t.Fatal("balanced a nil frame")
	}
}

func TestErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		frames []*frame.Frame
		white  float64
		want   string
	}{
		{"empty", nil, 0, "raw: nothing to add up"},
		{"mismatch", []*frame.Frame{frame.New(1, 1), frame.New(2, 1)}, -1, "raw: a 2x1 frame among 1x1"},
		{"negative", []*frame.Frame{fixture()}, -1, "a white balance of -1 kelvin"},
		{"nan", []*frame.Frame{fixture()}, math.NaN(), "a white balance of NaN kelvin"},
		{"infinity", []*frame.Frame{fixture()}, math.Inf(1), "a white balance of +Inf kelvin"},
		{"no-white", []*frame.Frame{fixture()}, 1e-300, "no white at 1e-300 kelvin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := DefaultOptions()
			options.White = test.white
			got, err := Develop(test.frames, options)
			if err == nil || err.Error() != test.want || got != nil {
				t.Fatalf("got %v, %v; want %s", got, err, test.want)
			}
		})
	}
}

func TestBalanceRetainsGreenAndMetadata(t *testing.T) {
	picture := frame.New(1, 1)
	picture.Pix[0] = frame.Pixel{R: .2, G: .3, B: .4, Cover: 1, Depth: 2}
	if err := Balance(picture, 3200); err != nil {
		t.Fatal(err)
	}
	pixel := picture.Pix[0]
	if pixel.G != .3 || pixel.R >= .2 || pixel.B <= .4 || pixel.Cover != 1 || pixel.Depth != 2 {
		t.Fatalf("gains or metadata %+v", pixel)
	}
	for _, kelvin := range []float64{0, -1, math.NaN(), math.Inf(1), 1e-300} {
		before := picture.Pix[0]
		if err := Balance(picture, kelvin); err == nil {
			t.Fatalf("accepted %g K", kelvin)
		}
		if picture.Pix[0] != before {
			t.Fatal("failed balance changed input")
		}
	}
}
