//go:build draw

package hillside

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/raw"
)

// The numbers the named scenes' tests pick, each by the session slice's
// author, 2026-11-12: small, so each scene draws on the cpu quickly.
const (
	namedWidth  = 32
	namedHeight = 18
	namedRays   = 1
	// namedFrames and namedRate are a film of two frames, half a second
	// apart: the orbital's frames at a second and a half second in.
	namedFrames = 2
	namedRate   = 2
)

// commandBuilt is one of slayer's commands, built from this tree into the
// test's own directory.
func commandBuilt(t *testing.T, name string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary,
		"github.com/janearc/slayer/cmd/"+name)
	if said, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building %s: %v: %s", name, err, said)
	}
	return binary
}

// commandRan runs a command built from this tree with arguments, and
// fails the test if it fails.
func commandRan(t *testing.T, name string, arguments ...string) {
	t.Helper()
	run := exec.Command(commandBuilt(t, name), arguments...)
	if said, err := run.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v: %s", name, arguments, err, said)
	}
}

// namedSize is the command line's size for the tests' small picture.
func namedSize() []string {
	return []string{"-w", strconv.Itoa(namedWidth), "-h",
		strconv.Itoa(namedHeight), "-n", strconv.Itoa(namedRays)}
}

// The orbital's own film and first moment, defined by scene/named.
const (
	orbitalOwnWidth  = 1600
	orbitalOwnHeight = 900
	orbitalFirst     = 500 * time.Millisecond
)

// namedJob is a job of a scene, written as its scene key holds it, at
// the tests' size, with more lines.
func namedJob(scene, more string) string {
	return fmt.Sprintf("scene: {%s}\nfilm: {width: %d, height: %d, "+
		"rays: %d}\n%s", scene, namedWidth, namedHeight, namedRays, more)
}

// a job still of each named scene at its defaults is its command's
// picture, byte for byte: the png each writes, or moonrise's raw.
func TestSessionNamedStillIsItsCommandsPicture(t *testing.T) {
	for _, name := range []string{"swarm", "orbital", "ring"} {
		out := filepath.Join(t.TempDir(), name+".png")
		commandRan(t, name, append(namedSize(), "-out", out)...)
		want, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		got := sessionPicture(t, jobFrom(t, namedJob("named: "+name, "")), Ask{})
		if !bytes.Equal(got, want) {
			t.Errorf("the session and %s drew different pictures", name)
		}
	}
	out := filepath.Join(t.TempDir(), "moonrise.pfm")
	commandRan(t, "moonrise", append(namedSize(), "-out", out)...)
	want, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	session := opening(t, jobFrom(t, namedJob("named: moonrise", "")))
	drawn, err := session.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(t.TempDir(), "session.pfm")
	if err := raw.Write(kept, drawn.Light()); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(kept); err != nil || !bytes.Equal(got,
		want) {
		t.Errorf("the session and moonrise kept different raws: %v", err)
	}
}

// a two-frame film of the orbital, from far with its lantern's beams
// keeping their own time, is the command's frames for the same times:
// the film starts at the scene's own time, half a second in.
func TestSessionNamedFilmIsTheOrbitalsFrames(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "orbital.png")
	commandRan(t, "orbital", append(namedSize(), "-far", "-frames",
		strconv.Itoa(namedFrames), "-fps", strconv.Itoa(namedRate),
		"-out", out)...)
	wanted := jobFrom(t, namedJob("named: orbital, with: {far: true}",
		fmt.Sprintf("fps: %d\nduration: {value: 1, unit: s}\n",
			namedRate)))
	session := opening(t, wanted)
	for index := range namedFrames {
		drawn, err := session.Frame(context.Background(), index, Ask{})
		if err != nil {
			t.Fatal(err)
		}
		picture := session.Tone().Develop(drawn.Light())
		written := filepath.Join(t.TempDir(), "frame.png")
		if err := file.PNG(written, picture, sessionSquare); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(written)
		name := fmt.Sprintf("orbital-%04d.png", index)
		want, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("frame %d at %v is not the command's: %v", index,
				drawn.At(), err)
		}
	}
}

// what a named scene cannot be drawn with is refused at Open by name,
// with the job file: a duration written as a quantity, a parameter the
// scene has not, a renderer it does not draw, a camera by name, and a
// camera path in another unit than the scene's.
func TestOpenRefusesWhatANamedSceneCannotDraw(t *testing.T) {
	path := "camera:\n  unit: %s\n  look-at: {x: 0, y: 0, z: 0}\n" +
		"  moves:\n    - from: {x: 0, y: 0, z: 6}\n" +
		"      to: {x: 0, y: 0, z: 6}\n      duration: {value: 1, unit: s}\n"
	for _, each := range []struct{ document, says string }{
		{"scene: {named: orbital, with: {day: {value: 12, unit: h}}}\n",
			"day: 12h"},
		{"scene: {named: orbital, with: {nonsense: 1}}\n", "nonsense"},
		{"scene: {named: nowhere}\n", "orbital"},
		{"scene: {named: ring}\nrenderer: gpu\n", "drawn by cpu"},
		{"scene: {named: orbital}\ncamera: {name: eye}\n", "one camera"},
		{"scene: {named: orbital}\n" + fmt.Sprintf(path, "m"),
			`"a thousand kilometres"`},
	} {
		wanted := jobFrom(t, each.document)
		_, err := Open(context.Background(), wanted, Stamp{})
		if err == nil || !strings.Contains(err.Error(), each.says) ||
			!strings.Contains(err.Error(), wanted.File()) {
			t.Errorf("%q gave %v, not a refusal saying %q", each.document,
				err, each.says)
		}
	}
}

// a named scene takes the job's with block, film and camera path in its
// own unit, and its defaults where the job says nothing.
func TestSessionNamedTakesWhatTheJobSays(t *testing.T) {
	plain := opening(t, jobFrom(t, "scene: {named: orbital}\n"))
	if plain.given.width != orbitalOwnWidth ||
		plain.given.height != orbitalOwnHeight ||
		plain.Times()[nothing] != orbitalFirst {
		t.Errorf("an unsaid orbital is %dx%d at %v", plain.given.width,
			plain.given.height, plain.Times())
	}
	at := opening(t, jobFrom(t, "scene: {named: orbital}\n"+
		"at: {value: 2, unit: s}\n"))
	if at.Times()[nothing] != 2*time.Second {
		t.Errorf("a said time became %v", at.Times())
	}
	day := sessionPicture(t, jobFrom(t, namedJob("named: orbital, "+
		"with: {day: 2h}", "")), Ask{})
	own := sessionPicture(t, jobFrom(t, namedJob("named: orbital", "")),
		Ask{})
	if bytes.Equal(day, own) {
		t.Error("the with block's day did not reach the picture")
	}
	moved := sessionPicture(t, jobFrom(t, namedJob("named: orbital",
		"camera:\n  unit: a thousand kilometres\n"+
			"  look-at: {x: 0, y: 0, z: 0}\n  moves:\n"+
			"    - from: {x: 0, y: 2000, z: 3000}\n"+
			"      to: {x: 0, y: 2000, z: 3000}\n"+
			"      duration: {value: 1, unit: s}\n")), Ask{})
	if bytes.Equal(moved, own) {
		t.Error("the job's camera path did not stand over the scene's eye")
	}
}
