package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/oceantex"
)

// near reports whether a and b differ by less than 1e-12.
func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

// TestTowardPointsAlongTheCompass checks toward on the four directions whose answer is known.
func TestTowardPointsAlongTheCompass(t *testing.T) {
	cases := []struct {
		azimuth, altitude float64
		want              oceantex.Vec
	}{
		{0, 0, oceantex.Vec{X: 0, Y: 1, Z: 0}},
		{90, 0, oceantex.Vec{X: 1, Y: 0, Z: 0}},
		{180, 0, oceantex.Vec{X: 0, Y: -1, Z: 0}},
		{0, 90, oceantex.Vec{X: 0, Y: 0, Z: 1}},
	}
	for _, tc := range cases {
		got := toward(tc.azimuth, tc.altitude)
		if !near(got.X, tc.want.X) || !near(got.Y, tc.want.Y) || !near(got.Z, tc.want.Z) {
			t.Errorf("toward(%v, %v) = %v, want %v", tc.azimuth, tc.altitude, got, tc.want)
		}
	}
}

// TestVectorHelpers checks dot, cross and unit on hand-worked vectors.
func TestVectorHelpers(t *testing.T) {
	if got := dot(oceantex.Vec{X: 1, Y: 2, Z: 3}, oceantex.Vec{X: 4, Y: 5, Z: 6}); got != 32 {
		t.Errorf("dot=%v, want 32", got)
	}
	if got := cross(oceantex.Vec{X: 1}, oceantex.Vec{Y: 1}); got != (oceantex.Vec{Z: 1}) {
		t.Errorf("cross=%v, want the z axis", got)
	}
	if got := unit(oceantex.Vec{X: 3, Z: 4}); !near(got.X, 0.6) || got.Y != 0 || !near(got.Z, 0.8) {
		t.Errorf("unit=%v, want (0.6, 0, 0.8)", got)
	}
}

// TestNumberFromReadsTheEnvironment checks the default is used when the variable is unset and the value when it is a number.
func TestNumberFromReadsTheEnvironment(t *testing.T) {
	t.Setenv("SEAWATER_TEST_NUMBER", "")
	if got := numberFrom("SEAWATER_TEST_NUMBER", 4.5); got != 4.5 {
		t.Errorf("unset: %v", got)
	}
	t.Setenv("SEAWATER_TEST_NUMBER", "-0.25")
	if got := numberFrom("SEAWATER_TEST_NUMBER", 4.5); got != -0.25 {
		t.Errorf("set: %v", got)
	}
}

// pfmFile writes a PFM with the given scale text and byte order, and returns its path.
func pfmFile(t *testing.T, header string, order binary.ByteOrder, values []float32) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "in.pfm")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(header); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(file, order, values); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestWriteThenReadKeepsEveryValue checks a picture written to a file reads back with its size and light, in the same bottom-up order.
func TestWriteThenReadKeepsEveryValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.pfm")
	picture := &oceantex.Texture{Width: 2, Height: 1, Light: [][3]float64{{1, 2, 3}, {4, 5, 6}}}
	if err := write(path, picture); err != nil {
		t.Fatal(err)
	}
	w, h, light, err := readPFM(path)
	if err != nil {
		t.Fatal(err)
	}
	if w != 2 || h != 1 || !reflect.DeepEqual(light, []float32{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("read %dx%d %v", w, h, light)
	}
}

// TestWriteRefusesAMissingDirectory checks write reports a path it cannot create.
func TestWriteRefusesAMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "none", "out.pfm")
	if err := write(path, &oceantex.Texture{Width: 1, Height: 1, Light: [][3]float64{{1, 1, 1}}}); err == nil {
		t.Fatal("write into a missing directory succeeded")
	}
}

// TestReadPFMByteOrderAndRefusals checks a positive scale reads big-endian, and bad files are refused.
func TestReadPFMByteOrderAndRefusals(t *testing.T) {
	big := pfmFile(t, "PF\n1 1\n1\n", binary.BigEndian, []float32{0.5, 0.25, 2})
	w, h, light, err := readPFM(big)
	if err != nil || w != 1 || h != 1 || !reflect.DeepEqual(light, []float32{0.5, 0.25, 2}) {
		t.Fatalf("big-endian read: %d %d %v %v", w, h, light, err)
	}
	if _, _, _, err := readPFM(filepath.Join(t.TempDir(), "absent.pfm")); err == nil {
		t.Error("missing file accepted")
	}
	empty := pfmFile(t, "", binary.LittleEndian, nil)
	if _, _, _, err := readPFM(empty); err == nil {
		t.Error("empty file accepted")
	}
	grey := pfmFile(t, "Pf\n1 1\n-1\n", binary.LittleEndian, []float32{1})
	if _, _, _, err := readPFM(grey); err == nil || !strings.Contains(err.Error(), "is not a colour pfm") {
		t.Errorf("grey pfm: %v", err)
	}
	short := pfmFile(t, "PF\n2 2\n-1\n", binary.LittleEndian, []float32{1, 2, 3})
	if _, _, _, err := readPFM(short); err == nil {
		t.Error("truncated pixels accepted")
	}
}

// TestBrightestAveragesTheMoonDisc checks the mean of the pixels at least half as bright as the brightest.
func TestBrightestAveragesTheMoonDisc(t *testing.T) {
	sky := &rendered{light: []float32{
		1, 1, 1,
		4, 4, 4,
		6, 6, 6,
		0, 0, 0,
	}}
	// The brightest sums to 18; pixels of 9 or more are the 4s and the 6s.
	if got := sky.brightest(); got != [3]float64{5, 5, 5} {
		t.Fatalf("brightest=%v, want (5, 5, 5)", got)
	}
}

// TestAboveOfIsTheMeanOfTheTopRow checks the top row, the last in a PFM, is the one averaged.
func TestAboveOfIsTheMeanOfTheTopRow(t *testing.T) {
	sky := &rendered{width: 2, height: 2, light: []float32{
		1, 1, 1, 1, 1, 1,
		2, 4, 6, 4, 8, 10,
	}}
	if got := sky.aboveOf(); got != [3]float64{3, 6, 8} {
		t.Fatalf("above=%v, want (3, 6, 8)", got)
	}
}

// TestSkyFromTakesTheMoonFromTheSkyOrFromMOONREF checks where the moon's light comes from.
func TestSkyFromTakesTheMoonFromTheSkyOrFromMOONREF(t *testing.T) {
	lit := pfmFile(t, "PF\n2 1\n-1\n", binary.LittleEndian, []float32{8, 8, 8, 1, 1, 1})
	plain := pfmFile(t, "PF\n2 1\n-1\n", binary.LittleEndian, []float32{2, 4, 6, 0, 0, 0})
	t.Setenv("MOONREF", "")
	sky, err := skyFrom(lit)
	if err != nil {
		t.Fatal(err)
	}
	if sky.moon != [3]float64{8, 8, 8} || sky.width != 2 || sky.height != 1 {
		t.Fatalf("moon=%v size=%dx%d", sky.moon, sky.width, sky.height)
	}
	t.Setenv("MOONREF", plain)
	sky, err = skyFrom(lit)
	if err != nil {
		t.Fatal(err)
	}
	if sky.moon != [3]float64{2, 4, 6} {
		t.Fatalf("moon=%v, want the plain plate's (2, 4, 6)", sky.moon)
	}
	t.Setenv("MOONREF", filepath.Join(t.TempDir(), "absent.pfm"))
	if _, err := skyFrom(lit); err == nil {
		t.Error("missing MOONREF accepted")
	}
	t.Setenv("MOONREF", "")
	if _, err := skyFrom(filepath.Join(t.TempDir(), "absent.pfm")); err == nil {
		t.Error("missing sky accepted")
	}
}

// TestAtLooksUpTheRawAndFallsBackPastTheFrame checks a direction in frame reads its pixel, and one outside gets the sky above or the dark ground.
func TestAtLooksUpTheRawAndFallsBackPastTheFrame(t *testing.T) {
	light := make([]float32, 4*4*3)
	// Direction (0, 1, 0) lands on pixel 2, 2; rows run bottom up, so that is row 1 of the data.
	copy(light[(1*4+2)*3:], []float32{0.25, 0.5, 0.75})
	// Direction (0, 1, 1) lands on pixel 2, 0, the top row, which is row 3 of the data.
	copy(light[(3*4+2)*3:], []float32{1, 0.5, 0.25})
	sky := &rendered{width: 4, height: 4, light: light, moon: [3]float64{1, 1, 1},
		forward: oceantex.Vec{Y: 1}, right: oceantex.Vec{X: 1}, up: oceantex.Vec{Z: 1},
		focal: 2, disc: 0.5, above: [3]float64{0.5, 0.25, 0.125}}
	want := func(r, g, b float64) interface{} { return srgb.FromLight(r/0.5, g/0.5, b/0.5) }
	cases := []struct {
		name string
		from oceantex.Vec
		want interface{}
	}{
		{"in frame", oceantex.Vec{Y: 1}, want(0.25, 0.5, 0.75)},
		{"top row", oceantex.Vec{Y: 1, Z: 1}, want(1, 0.5, 0.25)},
		{"above the frame", oceantex.Vec{Y: 1, Z: 2}, want(0.5, 0.25, 0.125)},
		{"behind and up", oceantex.Vec{Y: -1, Z: 1}, want(0.5, 0.25, 0.125)},
		{"behind the camera, level", oceantex.Vec{Y: -1}, srgb.FromLight(0, 0, 0)},
		{"beside the frame, level", oceantex.Vec{X: 10, Y: 1}, srgb.FromLight(0, 0, 0)},
		{"below the frame", oceantex.Vec{Y: 1, Z: -2}, srgb.FromLight(0, 0, 0)},
	}
	for _, tc := range cases {
		if got := sky.at(tc.from); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
