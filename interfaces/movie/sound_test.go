package movie

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

// a sound is refused by name when its rate is not offered, it is
// empty, it ends on half a pair, or a sample is not in -1..1.
func TestSoundsAreRefusedByName(t *testing.T) {
	nan := float32(math.NaN())
	cases := []struct {
		rate    int
		samples []float32
		says    string
	}{
		{rate: 22050, samples: []float32{0, 0}, says: "not offered"},
		{rate: 0, samples: []float32{0, 0}, says: "not offered"},
		{rate: readRate, samples: nil, says: "no samples"},
		{rate: readRate, samples: []float32{0, 0, 0}, says: "pairs"},
		{rate: readRate, samples: []float32{0, 1.5}, says: "sample 1"},
		{rate: readRate, samples: []float32{-1.01, 0}, says: "sample 0"},
		{rate: readRate, samples: []float32{nan, 0}, says: "outside"},
	}
	for _, tried := range cases {
		_, err := NewSound(tried.rate, tried.samples)
		if err == nil || !strings.Contains(err.Error(), tried.says) {
			t.Errorf("%d, %v: %v, want %q", tried.rate, tried.samples, err,
				tried.says)
		}
	}
}

// the near miss: full scale either way, at either rate, is taken, and
// the sound keeps its own copy.
func TestFullScaleIsTakenAndCopied(t *testing.T) {
	for _, rate := range soundRates {
		given := []float32{1, -1, 0, 0.5}
		sound, err := NewSound(rate, given)
		if err != nil {
			t.Fatalf("%d: %v", rate, err)
		}
		given[0] = 0
		held := sound.Samples()
		if held[0] != 1 || sound.Length() != 2 || sound.Rate() != rate {
			t.Errorf("%d: held %v, length %d", rate, held, sound.Length())
		}
		held[1] = 0
		if sound.Samples()[1] != -1 {
			t.Errorf("Samples gave the sound's own slice away")
		}
	}
}

// a sound knows how long it plays; no sound plays for no time.
func TestASoundKnowsItsDuration(t *testing.T) {
	if got := tone(t, readRate, 1).Duration(); got != time.Second {
		t.Errorf("a second of tone plays %v", got)
	}
	if (Sound{}).Duration() != 0 || (Sound{}).present() {
		t.Errorf("no sound plays, or is present")
	}
}

// the spans of every frame tile the picture: together they hold
// exactly the picture's length of sound, cut where the sound is longer.
func TestSpansTileThePicture(t *testing.T) {
	sound := tone(t, readRate, 2)
	frameRate, frames := picked("small-rate"), picked("small-frames")
	var joined []float32
	for i := range frames {
		joined = append(joined, sound.span(i, i+1, frameRate)...)
	}
	want := frames * readRate / frameRate * channels
	if len(joined) != want {
		t.Fatalf("spans hold %d samples, want %d", len(joined), want)
	}
	if !slices.Equal(joined, sound.samples[:want]) {
		t.Errorf("the spans are not the sound's first second")
	}
}

// the anti-test: where the sound is shorter than the picture, the
// spans after its end are silence, and the ones before are the sound.
func TestSpansAfterTheSoundAreSilent(t *testing.T) {
	sound := tone(t, readRate, 0.5)
	frameRate, frames := picked("small-rate"), picked("small-frames")
	before := sound.span(0, 1, frameRate)
	after := sound.span(frames-1, frames, frameRate)
	if !slices.Equal(before, sound.samples[:len(before)]) {
		t.Errorf("the first span is not the sound's start")
	}
	if slices.ContainsFunc(after, func(sample float32) bool {
		return sample != 0
	}) {
		t.Errorf("the last span, after the sound ended, is not silent")
	}
}

// mono decoded sound is put on both channels; stereo is kept as it is.
func TestMonoIsPutOnBothChannels(t *testing.T) {
	sound, err := fromChannels(readRate, mono, []float32{0.1, 0.2})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sound.samples, []float32{0.1, 0.1, 0.2, 0.2}) {
		t.Errorf("mono read as %v", sound.samples)
	}
	stereo, err := fromChannels(readRate, channels, []float32{0.1, 0.2})
	if err != nil || !slices.Equal(stereo.samples, []float32{0.1, 0.2}) {
		t.Errorf("stereo read as %v, %v", stereo.samples, err)
	}
}

// the anti-test: more channels, no samples, or a sample that is not a
// number is refused; past full scale is clipped, not refused.
func TestDecodedSoundIsRefusedOrClipped(t *testing.T) {
	refusals := map[int][]float32{3: {0, 0, 0}, mono: nil,
		channels: {0, float32(math.NaN())}}
	for count, decoded := range refusals {
		if _, err := fromChannels(readRate, count, decoded); err == nil {
			t.Errorf("%d channels of %v taken", count, decoded)
		}
	}
	loud, err := fromChannels(readRate, channels, []float32{1.25, -1.25})
	if err != nil || !slices.Equal(loud.samples, []float32{1, -1}) {
		t.Errorf("past full scale read as %v, %v", loud.samples, err)
	}
}
