//go:build draw

package session

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/finish"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/take"
)

const (
	// shutterGlimpses is four glimpses a frame.
	shutterGlimpses = 4
	// shutterStretch is how many times slower than the film scene time
	// runs in a stretched film: ten seconds of scene in one of film.
	shutterStretch = 10
)

// The shutter's angles the tests open, in degrees, each defined: a
// quarter, a half and a whole turn.
const (
	shutterQuarter = 90
	shutterHalf    = 180
	shutterWhole   = 360
)

// shutterSides is defined as the two halves a share is cut into at its
// middle.
const shutterSides = 2

// shutterFilm is a film of a second at the shutter's rate, its shutter
// open for an angle with so many glimpses.
func shutterFilm(angle, glimpses int) string {
	return fmt.Sprintf("fps: %d\nduration: {value: 1, unit: s}\n"+
		"shutter: {angle: {value: %d, unit: deg}, glimpses: %d}\n",
		shutterRate, angle, glimpses)
}

// shutterSmoke is a job of the first smoke at the tests' size, with more.
func shutterSmoke(t *testing.T, more string) *Session {
	t.Helper()
	return opening(t, jobFrom(t, namedJob(fmt.Sprintf("smoke: %d",
		sessionPlainSmoke), more)))
}

// sameLight is whether two frames hold the same light, bit for bit.
func sameLight(one, other *frame.Frame) bool {
	if one.W != other.W || one.H != other.H {
		return false
	}
	for i := range one.Pix {
		if one.Pix[i] != other.Pix[i] {
			return false
		}
	}
	return true
}

// takes are the times take places a shutter's glimpses at, from a
// frame's opening, for a rate, an angle in degrees and glimpses.
func takes(t *testing.T, rate float64, angle float64,
	glimpses int) []time.Duration {
	t.Helper()
	pace, err := take.Cranked(rate, angle*maths.Degree, glimpses)
	if err != nil {
		t.Fatal(err)
	}
	shot, err := take.Take(thing.Still(maths.Subspace{}), pace,
		time.Duration(float64(time.Second)/rate))
	if err != nil {
		t.Fatal(err)
	}
	var times []time.Duration
	for _, caught := range shot.Glimpses(nothing) {
		times = append(times, caught.At())
	}
	return times
}

// the glimpses' times are take's: the midpoints of equal shares of the
// open part of the frame's interval, from the frame's own time.
func TestShutterGlimpsesAreAtTakesTimes(t *testing.T) {
	half := shutterSmoke(t, shutterFilm(shutterHalf, shutterGlimpses))
	want := takes(t, shutterRate, shutterHalf, shutterGlimpses)
	if !slices.Equal(half.offsets, want) {
		t.Errorf("the glimpses are at %v, take's at %v", half.offsets, want)
	}
	// a half second's interval, half of it open: a quarter second in
	// four shares, each glimpse in the middle of its share
	open := time.Second / shutterRate * shutterHalf / shutterWhole
	for place, offset := range half.offsets {
		middle := open * time.Duration(shutterSides*place+singleRay) /
			time.Duration(shutterSides*shutterGlimpses)
		if offset != middle {
			t.Errorf("glimpse %d is at %v, not the middle of its share, %v",
				place, offset, middle)
		}
	}
	whole := shutterSmoke(t, shutterFilm(shutterWhole, shutterGlimpses))
	if !slices.Equal(whole.offsets, takes(t, shutterRate, shutterWhole,
		shutterGlimpses)) || whole.offsets[shutterGlimpses-singleRay] <=
		half.offsets[shutterGlimpses-singleRay] {
		t.Errorf("a shutter open the whole turn has glimpses at %v",
			whole.offsets)
	}
	plain := shutterSmoke(t, fmt.Sprintf("fps: %d\nduration: {value: 1, "+
		"unit: s}\n", shutterRate))
	if !slices.Equal(plain.offsets, unshuttered) {
		t.Errorf("a film with no shutter has glimpses at %v", plain.offsets)
	}
}

// a film whose scene time is stretched opens its shutter over a share of
// the scene time between its frames, not of the film's.
func TestShutterIntervalIsInSceneTime(t *testing.T) {
	stretched := shutterSmoke(t, shutterFilm(shutterHalf, shutterGlimpses)+
		fmt.Sprintf("scene-duration: {value: %d, unit: s}\n",
			shutterStretch))
	want := takes(t, float64(shutterRate)/shutterStretch, shutterHalf,
		shutterGlimpses)
	if !slices.Equal(stretched.offsets, want) {
		t.Errorf("a stretched film's glimpses are at %v, not %v",
			stretched.offsets, want)
	}
}

// a shutter of one glimpse draws the frame at the middle of the open
// interval: the same light as a film without one drawn at that time.
// The orbital from far keeps its beams' own time, so a wrong time shows.
func TestShutterOfOneGlimpseIsTheFrameAtItsMiddle(t *testing.T) {
	scene := "named: orbital, with: {far: true}"
	shuttered := opening(t, jobFrom(t, namedJob(scene,
		shutterFilm(shutterQuarter, singleRay))))
	middle := shuttered.Times()[nothing] + shuttered.offsets[nothing]
	plain := opening(t, jobFrom(t, namedJob(scene, fmt.Sprintf(
		"fps: %d\nduration: {value: 1, unit: s}\nstart: {value: %s, "+
			"unit: s}\n", shutterRate, secondsWritten(middle)))))
	got, err := shuttered.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := plain.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	if !sameLight(got.Light(), want.Light()) || want.At() != middle ||
		got.At() != shuttered.Times()[nothing] {
		t.Errorf("one glimpse at %v is not the frame drawn there", middle)
	}
	if len(got.glimpses) != singleRay ||
		!sameLight(got.glimpses[nothing], got.Light()) {
		t.Error("one glimpse is not the frame's one light")
	}
}

// secondsWritten is a duration written in seconds, as a job writes one.
func secondsWritten(span time.Duration) string {
	return strings.TrimRight(fmt.Sprintf("%.9f", span.Seconds()), "0")
}

// a still scene seen from a still camera gives glimpses equal to each
// other to the bit, and their mean is each.
func TestShutterOfAStillSceneGivesEqualGlimpses(t *testing.T) {
	session := shutterSmoke(t, shutterFilm(shutterHalf, shutterGlimpses))
	drawn, err := session.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	glimpses := drawn.glimpses
	if len(glimpses) != shutterGlimpses {
		t.Fatalf("%d glimpses, not %d", len(glimpses), shutterGlimpses)
	}
	for place, glimpse := range glimpses {
		if !sameLight(glimpse, glimpses[nothing]) {
			t.Errorf("glimpse %d of a still scene differs", place)
		}
	}
	if !sameLight(drawn.Light(), glimpses[nothing]) {
		t.Error("the mean of equal glimpses is not each of them")
	}
}

// a moving camera gives glimpses that differ, and the frame's light is
// their mean.
func TestShutterOfAMovingCameraGivesDifferentGlimpses(t *testing.T) {
	raw, err := os.ReadFile("../doc/camera-smoke-sequence.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	camera := text[strings.Index(text, "camera:"):]
	session := shutterSmoke(t, shutterFilm(shutterHalf, shutterGlimpses)+
		camera)
	drawn, err := session.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	glimpses := drawn.glimpses
	for place := singleRay; place < len(glimpses); place++ {
		if sameLight(glimpses[place], glimpses[place-singleRay]) {
			t.Errorf("glimpses %d and %d of a moving camera are equal",
				place-singleRay, place)
		}
	}
	mean, err := finish.Mean(glimpses)
	if err != nil || !sameLight(drawn.Light(), mean) {
		t.Errorf("the frame's light is not its glimpses' mean: %v", err)
	}
}
