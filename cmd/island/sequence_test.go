package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sequenceText builds a track of three keyframes with a fixed lens. Longitude
// of the camera runs 0, 10, 10 degrees at 0, 10, 20 seconds.
func sequenceText(interpolation, ease string, scene string) string {
	key := func(seconds, longitude float64, wide string) string {
		return "  - {at-seconds: " + formatFloat(seconds) + ", wide-deg: " + wide + ", ease: " + ease +
			", at: {longitude-deg: " + formatFloat(longitude) + ", latitude-deg: 0, height-metres: 100}" +
			", look-at: {longitude-deg: 5, latitude-deg: 0, height-metres: 0}}\n"
	}
	text := ""
	if interpolation != "" {
		text += "interpolation: " + interpolation + "\n"
	}
	text += "scene: " + scene + "\nframes: 5\nfps: 10\nscene-duration-seconds: 20\nkeyframes:\n"
	return text + key(0, 0, "40") + key(10, 10, "60") + key(20, 10, "60")
}

// formatFloat renders a whole or fractional number for a YAML fixture.
func formatFloat(value float64) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

// loadSequence writes the track and its scene and reads the track back.
func loadSequence(t *testing.T, text string) (*sequencePlan, string) {
	t.Helper()
	dir := t.TempDir()
	path := writeFile(t, dir, "track.yaml", text)
	got, err := readSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	return got, dir
}

// TestReadSequenceResolvesScene checks the scene path, the hash and the zoom
// prepared for each leg.
func TestReadSequenceResolvesScene(t *testing.T) {
	text := sequenceText("", "linear", "scene.yaml")
	got, dir := loadSequence(t, text)
	if want := filepath.Join(dir, "scene.yaml"); got.Scene != want {
		t.Errorf("scene = %q, want %q", got.Scene, want)
	}
	if len(got.zooms) != 2 || len(got.Keys) != 3 || got.Frames != 5 || got.FPS != 10 {
		t.Errorf("zooms %d keys %d frames %d fps %d", len(got.zooms), len(got.Keys), got.Frames, got.FPS)
	}
	if got.hash != configHash([]byte(text)) || string(got.raw) != text {
		t.Error("raw bytes or hash not retained")
	}
	if w, err := got.zooms[0].At(5e9); err != nil || w != 50 {
		t.Errorf("midpoint of first zoom = %v, %v; want 50", w, err)
	}
	absolute := filepath.Join(dir, "other", "scene.yaml")
	again, _ := loadSequence(t, sequenceText("", "linear", absolute))
	if again.Scene != absolute {
		t.Errorf("absolute scene was rewritten to %q", again.Scene)
	}
}

// TestReadSequenceRefusals checks each way a track can be rejected.
func TestReadSequenceRefusals(t *testing.T) {
	good := sequenceText("", "linear", "scene.yaml")
	cases := []struct{ name, text, want string }{
		{"first key not at zero", strings.Replace(good, "at-seconds: 0,", "at-seconds: 1,", 1), "keyframes must run from zero"},
		{"last key short of duration", strings.Replace(good, "scene-duration-seconds: 20", "scene-duration-seconds: 30", 1), "keyframes must run from zero"},
		{"continuous with smooth easing", sequenceText("continuous", "smooth", "s.yaml"), "continuous interpolation requires linear"},
		{"times not increasing", strings.Replace(good, "at-seconds: 10,", "at-seconds: 20,", 1), "increasing times"},
		{"longitude wraps", strings.Replace(good, "longitude-deg: 0, latitude-deg: 0, height-metres: 100}, look-at", "longitude-deg: -180, latitude-deg: 0, height-metres: 100}, look-at", 1), "non-wrapping"},
		{"schema refusal", strings.Replace(good, "frames: 5", "frames: 1", 1), "frames"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readSequence(writeFile(t, t.TempDir(), "track.yaml", tc.text))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
	if _, err := readSequence(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Error("a missing track was accepted")
	}
	if _, err := readSequence(writeFile(t, t.TempDir(), "track.yaml", "{")); err == nil {
		t.Error("unparsable YAML was accepted")
	}
}

// TestSampleSegments checks hand-worked positions, lens angle, clock and
// retained bytes for segment interpolation.
func TestSampleSegments(t *testing.T) {
	track, _ := loadSequence(t, sequenceText("segments", "linear", "scene.yaml"))
	base := &plan{Time: 100}
	cases := []struct {
		index          int
		seconds, lon   float64
		wide           float64
		heightOfCamera float64
	}{
		{0, 0, 0, 40, 100},
		{1, 5, 5, 50, 100},
		{2, 10, 10, 60, 100},
		{3, 15, 10, 60, 100},
		{4, 20, 10, 60, 100},
	}
	for _, tc := range cases {
		got, err := track.sample(base, tc.index)
		if err != nil {
			t.Fatal(err)
		}
		if got.Camera.At.Longitude != tc.lon || got.Film.Wide != tc.wide || got.Camera.At.Height != tc.heightOfCamera {
			t.Errorf("frame %d: longitude %v wide %v height %v", tc.index, got.Camera.At.Longitude, got.Film.Wide, got.Camera.At.Height)
		}
		if got.Time != 100+tc.seconds {
			t.Errorf("frame %d: time %v, want %v", tc.index, got.Time, 100+tc.seconds)
		}
		if got.hash != configHash(got.raw) || !strings.HasSuffix(string(got.raw), "}\n") {
			t.Errorf("frame %d: retained bytes do not match their hash", tc.index)
		}
	}
	if base.Time != 100 || base.Film.Wide != 0 {
		t.Error("sample changed the base plan")
	}
}

// TestSampleSmoothEase checks the smooth share of a quarter of the way along
// a leg: three times a sixteenth minus two times a sixty-fourth.
func TestSampleSmoothEase(t *testing.T) {
	track, _ := loadSequence(t, sequenceText("", "smooth", "scene.yaml"))
	track.Frames = 9
	got, err := track.sample(&plan{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if want := 10 * (3.0/16 - 2.0/64); math.Abs(got.Camera.At.Longitude-want) > 1e-12 {
		t.Errorf("longitude = %v, want %v", got.Camera.At.Longitude, want)
	}
}

// TestSampleContinuous checks the shared-velocity track against a value
// worked by hand: halfway along a leg that starts at one degree per second
// and ends at rest, a ten degree leg is at 6.25 degrees.
func TestSampleContinuous(t *testing.T) {
	track, _ := loadSequence(t, sequenceText("continuous", "linear", "scene.yaml"))
	track.Frames = 5
	got, err := track.sample(&plan{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Camera.At.Longitude-6.25) > 1e-12 {
		t.Errorf("longitude = %v, want 6.25", got.Camera.At.Longitude)
	}
	if got.Camera.Look.Longitude != 5 || got.Camera.At.Latitude != 0 || got.Camera.At.Height != 100 {
		t.Errorf("a fixed look target or fixed axis moved: %+v", got.Camera)
	}
	if math.Abs(got.Film.Wide-52.5) > 1e-12 {
		t.Errorf("wide = %v, want 52.5", got.Film.Wide)
	}
}

// TestSampleRefusesBrokenZoom checks that a missing zoom is an error.
func TestSampleRefusesBrokenZoom(t *testing.T) {
	track, _ := loadSequence(t, sequenceText("", "linear", "scene.yaml"))
	track.zooms[0] = nil
	if _, err := track.sample(&plan{}, 0); err == nil {
		t.Error("a missing zoom was accepted")
	}
}

// TestGeographicBetween checks the endpoints are exact and the middle is the
// mean of each coordinate.
func TestGeographicBetween(t *testing.T) {
	from, to := location{Longitude: -10, Latitude: 20, Height: 100}, location{Longitude: 10, Latitude: 40, Height: 300}
	if geographicBetween(from, to, -1) != from || geographicBetween(from, to, 0) != from {
		t.Error("share at or below zero is not the start")
	}
	if geographicBetween(from, to, 1) != to || geographicBetween(from, to, 2) != to {
		t.Error("share at or above one is not the end")
	}
	if got, want := geographicBetween(from, to, 0.5), (location{Longitude: 0, Latitude: 30, Height: 200}); got != want {
		t.Errorf("midpoint = %+v, want %+v", got, want)
	}
}

// TestConfigHash checks the hash against the published digest of "abc".
func TestConfigHash(t *testing.T) {
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := configHash([]byte("abc")); got != want {
		t.Errorf("hash = %q, want %q", got, want)
	}
}

// TestCoordinateSlope checks the slope choices: end secants, the slower
// neighbour, and rest at a stop or a reversal.
func TestCoordinateSlope(t *testing.T) {
	keys := func(values ...float64) *sequencePlan {
		plan := &sequencePlan{}
		for index, value := range values {
			plan.Keys = append(plan.Keys, sequenceKey{Seconds: float64(index) * 2, At: location{Longitude: value}})
		}
		return plan
	}
	value := func(key sequenceKey) float64 { return key.At.Longitude }
	cases := []struct {
		name   string
		values []float64
		index  int
		want   float64
	}{
		{"first key uses its secant", []float64{0, 4, 12}, 0, 2},
		{"last key uses its secant", []float64{0, 4, 12}, 2, 4},
		{"rising picks the slower side", []float64{0, 4, 12}, 1, 2},
		{"falling picks the slower side", []float64{12, 4, 0}, 1, -2},
		{"reversal rests", []float64{0, 4, 0}, 1, 0},
		{"stationary before rests", []float64{4, 4, 8}, 1, 0},
		{"stationary after rests", []float64{0, 4, 4}, 1, 0},
	}
	for _, tc := range cases {
		if got := keys(tc.values...).coordinateSlope(tc.index, value); got != tc.want {
			t.Errorf("%s: slope = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestCoordinateEndpointsAndMiddle checks that the Hermite curve holds both
// ends exactly and has the straight-line value where both slopes match.
func TestCoordinateEndpointsAndMiddle(t *testing.T) {
	plan := &sequencePlan{Keys: []sequenceKey{
		{Seconds: 0, At: location{Height: 0}}, {Seconds: 2, At: location{Height: 8}}}}
	height := func(key sequenceKey) float64 { return key.At.Height }
	if plan.coordinate(0, 0, height) != 0 || plan.coordinate(0, -1, height) != 0 {
		t.Error("start not exact")
	}
	if plan.coordinate(0, 1, height) != 8 || plan.coordinate(0, 3, height) != 8 {
		t.Error("end not exact")
	}
	if got := plan.coordinate(0, 0.5, height); math.Abs(got-4) > 1e-12 {
		t.Errorf("middle = %v, want 4", got)
	}
}

// TestCurveLocationFollowsTarget checks that the position and look-at tracks
// are separate and each carries all three coordinates.
func TestCurveLocationFollowsTarget(t *testing.T) {
	plan := &sequencePlan{Keys: []sequenceKey{
		{Seconds: 0, At: location{1, 2, 3}, Look: location{10, 20, 30}},
		{Seconds: 1, At: location{3, 6, 9}, Look: location{30, 60, 90}}}}
	if got := plan.curveLocation(0, 1, false); got != (location{3, 6, 9}) {
		t.Errorf("position = %+v", got)
	}
	if got := plan.curveLocation(0, 1, true); got != (location{30, 60, 90}) {
		t.Errorf("target = %+v", got)
	}
	if got := plan.curveLocation(0, 0.5, false); math.Abs(got.Longitude-2) > 1e-12 || math.Abs(got.Latitude-4) > 1e-12 || math.Abs(got.Height-6) > 1e-12 {
		t.Errorf("middle position = %+v, want 2 4 6", got)
	}
}

// TestWriteSequenceStatus checks the record is written whole and the
// temporary name is gone.
func TestWriteSequenceStatus(t *testing.T) {
	dir := t.TempDir()
	if err := writeSequenceStatus(dir, sequenceRecord{Complete: true, FPS: 24, ExpectedFrames: 7}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "sequence.json"))
	if err != nil {
		t.Fatal(err)
	}
	var back sequenceRecord
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Complete || back.FPS != 24 || back.ExpectedFrames != 7 || raw[len(raw)-1] != '\n' {
		t.Errorf("record = %+v", back)
	}
	if _, err := os.Stat(filepath.Join(dir, "sequence.json.tmp")); !os.IsNotExist(err) {
		t.Error("temporary file was left behind")
	}
	if err := writeSequenceStatus(filepath.Join(dir, "absent"), sequenceRecord{}); err == nil {
		t.Error("writing into a missing directory succeeded")
	}
}
