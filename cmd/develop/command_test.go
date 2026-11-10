package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

var developCommand string

// TestMain builds one real command, with matching instrumentation under cover.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-develop-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	developCommand = filepath.Join(directory, "develop")
	if err = commandtest.Build(".", ".", developCommand, testing.CoverMode()); err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

// developRun bounds the actual command, retaining its diagnostic output.
func developRun(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	return exec.CommandContext(ctx, developCommand, args...).CombinedOutput()
}

// rawFixture records finite linear light without a renderer or a model oracle.
func rawFixture(t *testing.T, width int) string {
	t.Helper()
	picture := frame.New(width, 1)
	for i := range picture.Pix {
		picture.Pix[i] = frame.Pixel{R: .08, G: .12, B: .2, Cover: 1, Depth: 2}
	}
	path := filepath.Join(t.TempDir(), "input.pfm")
	if err := raw.Write(path, picture); err != nil {
		t.Fatal(err)
	}
	return path
}

// readPrint decodes the actual output, independent of the developer's encoder.
func readPrint(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

func TestTwoRawsEqualOneRawOneStopHigher(t *testing.T) {
	input := rawFixture(t, 2)
	one, two := filepath.Join(t.TempDir(), "one.png"), filepath.Join(t.TempDir(), "two.png")
	for _, test := range []struct {
		args    []string
		message string
	}{{[]string{"-out", one, "-exposure", "1", input}, "1 raw at +1.0 stops"}, {[]string{"-out", two, input, input}, "2 raws at +0.0 stops"}} {
		out, err := developRun(t, test.args...)
		if err != nil || !strings.Contains(string(out), test.message) {
			t.Fatalf("%v: %s", err, out)
		}
	}
	a, b := readPrint(t, one), readPrint(t, two)
	if a.Bounds() != image.Rect(0, 0, 2, 1) || b.Bounds() != a.Bounds() {
		t.Fatal("print size")
	}
	for x := range 2 {
		if a.At(x, 0) != b.At(x, 0) {
			t.Fatalf("additive exposure disagrees at %d", x)
		}
	}
	balanced := filepath.Join(t.TempDir(), "white.png")
	if out, err := developRun(t, "-out", balanced, "-white", "3200", input); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	plain := filepath.Join(t.TempDir(), "plain.png")
	if out, err := developRun(t, "-out", plain, input); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if readPrint(t, balanced).At(0, 0) == readPrint(t, plain).At(0, 0) {
		t.Fatal("white balance was ignored")
	}
}

func TestDeveloperReportsMissingMalformedAndMismatchedRaws(t *testing.T) {
	input, other := rawFixture(t, 1), rawFixture(t, 2)
	bad := filepath.Join(t.TempDir(), "bad.pfm")
	if err := os.WriteFile(bad, []byte("not a PFM\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"arguments", nil, "wants -out"},
		{"missing", []string{"-out", filepath.Join(t.TempDir(), "x.png"), filepath.Join(t.TempDir(), "missing.pfm")}, "missing.pfm"},
		{"malformed", []string{"-out", filepath.Join(t.TempDir(), "x.png"), bad}, "raw:"},
		{"mismatch", []string{"-out", filepath.Join(t.TempDir(), "x.png"), input, other}, "frame among"},
		{"white", []string{"-out", filepath.Join(t.TempDir(), "x.png"), "-white", "-1", input}, "white balance"},
		{"output", []string{"-out", t.TempDir(), input}, "develop: rename"},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, err := developRun(t, test.args...)
			if err == nil || !strings.Contains(string(out), test.want) {
				t.Fatalf("%v: %s", err, out)
			}
		})
	}
}

func TestWhiteBalanceRetainsGreenAndRejectsNonfiniteTemperature(t *testing.T) {
	picture := frame.New(1, 1)
	picture.Pix[0] = frame.Pixel{R: .2, G: .3, B: .4, Cover: 1, Depth: 2}
	if err := balance(picture, 3200); err != nil {
		t.Fatal(err)
	}
	pixel := picture.Pix[0]
	if pixel.G != .3 || pixel.R >= .2 || pixel.B <= .4 || pixel.Cover != 1 || pixel.Depth != 2 {
		t.Fatalf("white gains or metadata %+v", pixel)
	}
	for _, kelvin := range []float64{0, -1, math.NaN(), math.Inf(1), 1e-300} {
		before := picture.Pix[0]
		if err := balance(picture, kelvin); err == nil {
			t.Fatalf("accepted %g K", kelvin)
		}
		if picture.Pix[0] != before {
			t.Fatal("failed balance changed light")
		}
	}
	if plural(1) != "" || plural(2) != "s" {
		t.Fatal("count grammar")
	}
}
