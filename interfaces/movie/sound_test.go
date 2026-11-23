package movie

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// audible is whether a sample is anything but silence.
func audible(sample float32) bool { return sample != zero }

// a sound is refused by name when its rate is not offered, it is
// empty, it ends on half a pair, or a sample is not in -1..1.
func TestSoundsAreRefusedByName(t *testing.T) {
	for _, row := range casesOf(t, soundsRefusedFile) {
		rate, samples := wholeIn(t, row, "rate"), samplesIn(t, row, "samples")
		_, err := NewSound(rate, samples)
		says := textIn(t, row, "says")
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: %v, want %q", row["case"], err, says)
		}
	}
}

// the near miss: full scale either way, at either rate, is taken, and
// the sound keeps its own copy: neither the slice handed in nor the one
// handed back reaches the sound's samples.
func TestFullScaleIsTakenAndCopied(t *testing.T) {
	for _, row := range casesOf(t, soundsTakenFile) {
		for _, rate := range soundRates {
			given := samplesIn(t, row, "samples")
			want := slices.Clone(given)
			sound, err := NewSound(rate, given)
			if err != nil {
				t.Fatalf("%d: %v", rate, err)
			}
			clear(given)
			held := sound.Samples()
			clear(held)
			if !slices.Equal(sound.Samples(), want) ||
				sound.Length() != len(want)/channels ||
				sound.Rate() != rate {
				t.Errorf("%d: held %v, want %v", rate, sound.samples, want)
			}
		}
	}
}

// a sound knows how long it plays; no sound plays for no time.
func TestASoundKnowsItsDuration(t *testing.T) {
	if got := tone(t, readRate, one).Duration(); got != time.Second {
		t.Errorf("a second of tone plays %v", got)
	}
	if (Sound{}).Duration() != zero || (Sound{}).present() {
		t.Errorf("no sound plays, or is present")
	}
}

// the spans of every frame tile the picture: together they hold
// exactly the picture's length of sound, cut where the sound is longer.
func TestSpansTileThePicture(t *testing.T) {
	sound := tone(t, readRate, pickedValue("span-tone-seconds"))
	frameRate, frames := picked("small-rate"), picked("small-frames")
	var joined []float32
	for i := range frames {
		joined = append(joined, sound.span(i, i+one, frameRate)...)
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
	sound := tone(t, readRate, pickedValue("short-tone-seconds"))
	frameRate, frames := picked("small-rate"), picked("small-frames")
	before := sound.span(zero, one, frameRate)
	after := sound.span(frames-one, frames, frameRate)
	if !slices.Equal(before, sound.samples[:len(before)]) {
		t.Errorf("the first span is not the sound's start")
	}
	if slices.ContainsFunc(after, audible) {
		t.Errorf("the last span, after the sound ended, is not silent")
	}
}

// decoded sound is made stereo as the table of taken ones says: mono put
// on both channels, stereo kept, past full scale clipped, and a half
// instant at the end dropped.
func TestDecodedSoundIsMadeStereo(t *testing.T) {
	for _, row := range casesOf(t, decodedTakenFile) {
		got, err := stereoOf(wholeIn(t, row, "channels"),
			samplesIn(t, row, "decoded"))
		want := samplesIn(t, row, "want")
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: read as %v, %v, want %v", row["case"], got, err,
				want)
		}
	}
}

// the anti-test: more channels, none, no samples, a sample that is not a
// number, or less than one instant is refused, by name.
func TestDecodedSoundIsRefused(t *testing.T) {
	for _, row := range casesOf(t, decodedRefusedFile) {
		got, err := stereoOf(wholeIn(t, row, "channels"),
			samplesIn(t, row, "decoded"))
		says := textIn(t, row, "says")
		if err == nil || got != nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: %v, %v, want a refusal saying %q", row["case"],
				got, err, says)
		}
	}
}
