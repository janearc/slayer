package hillside

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
)

// The numbers the session's tests pick, each picked by the session
// slice's author, 2026-11-12, small so a frame on the cpu is quick.
const (
	// sessionWidth, sessionHeight and sessionRays are the engine tests'
	// own small still: 48 by 27, one ray a pixel.
	sessionWidth  = 48
	sessionHeight = 27
	sessionRays   = 1
	// sessionSmallWidth and sessionSmallHeight are a smaller still, as a
	// live view draws first.
	sessionSmallWidth  = 24
	sessionSmallHeight = 14
	// sessionAngle is a lens narrower than widow's own 40 degrees.
	sessionAngle = "25"
	// sessionTall is an angle down the picture, in degrees.
	sessionTall = "20"
	// sessionExposure is the job's exposure, and sessionSetExposure the
	// set's, which the job's stands over.
	sessionExposure    = "0.5"
	sessionSetExposure = "3"
	// sessionSquare is a pixel of the picture as one pixel of the png,
	// as save writes it.
	sessionSquare = 1
)

// The smokes the tests open, each defined by scene/smoke's table.
const (
	sessionPlainSmoke  = 1  // defined: a ball on a floor
	sessionBandedSmoke = 28 // defined: a magnetar flares, in gamma
	sessionGaugedSmoke = 22 // defined: the ringed giant on super 8
)

// jobFrom is the job a document says, after its schema line, written to
// a file of its own and read.
func jobFrom(t *testing.T, document string) job.Job {
	t.Helper()
	path := filepath.Join(t.TempDir(), "job.yaml")
	text := "$schema: https://github.com/janearc/slayer/schemas/" +
		"job-v1.json\n" + document
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return wanted
}

// sized is the command line's size for a still w by h with the tests'
// rays.
func sized(width, height int) []string {
	return []string{"-w", strconv.Itoa(width), "-h", strconv.Itoa(height),
		"-n", strconv.Itoa(sessionRays)}
}

// shortSmokePlan is the camera smoke's plan, shortened to two frames of a
// small picture as the smoke sequence test shortens it, written to a
// file, with the smoke of a number.
func shortSmokePlan(t *testing.T, number int) string {
	t.Helper()
	raw, err := os.ReadFile("../doc/camera-smoke-sequence.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	changes := [][2]string{
		{"width: 640, height: 360", "width: 24, height: 18"},
		{"fps: 24", "fps: 2"},
		{"duration: {value: 4", "duration: {value: 1"},
	}
	if number != sessionPlainSmoke {
		changes = append(changes, [2]string{"smoke: 1",
			"smoke: " + strconv.Itoa(number)})
	}
	for _, change := range changes {
		changed := strings.Replace(text, change[0], change[1], 1)
		if changed == text {
			t.Fatalf("the plan has no %q", change[0])
		}
		text = changed
	}
	path := filepath.Join(t.TempDir(), "move.yaml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
