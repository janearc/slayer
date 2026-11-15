package door

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/pzm"
)

//go:embed testdata/numbers.yaml
var numbersRaw []byte

// number is a value the tests chose, by name, from testdata/numbers.yaml.
func number(t *testing.T, name string) int {
	t.Helper()
	chosen, err := numbers.Read("testdata/numbers.yaml", numbersRaw)
	if err != nil {
		t.Fatal(err)
	}
	value, err := chosen.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return int(value.Value)
}

// filmFrames is how many frames the tests' one-second film holds.
func filmFrames(t *testing.T) int {
	t.Helper()
	return number(t, "fps")
}

// The words of the tests' jobs.
const (
	// jobLine is every job's first line.
	jobLine = "$schema: " + jobSchema + "\n"
	// today is the tests' clock's date, as the dated folders write it.
	today = "2026-11-12"
	// private is a file only its owner reads and writes.
	private = 0o600
)

// noon is the tests' clock: noon on the day the door was built.
func noon() time.Time {
	// the words are a constant the layout reads, so they parse
	at, _ := time.Parse(time.RFC3339, today+"T12:00:00Z")
	return at
}

// widow is sets/widow.yaml, absolute: a set of a world and no ground,
// which stages with no shelf.
func widow(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../sets/widow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// sized is the tests' film line: their size and rays.
func sized(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("film: {width: %d, height: %d, rays: %d}\n",
		number(t, "width"), number(t, "height"), number(t, "rays"))
}

// stillJob is a still of widow on the cpu at the tests' size, with more
// lines after it.
func stillJob(t *testing.T, after string) string {
	t.Helper()
	return jobLine + "scene: {set: " + widow(t) + "}\n" + sized(t) +
		"renderer: cpu\n" + after
}

// filmJob is a one-second film of smoke 1 on the cpu at the tests' size
// and rate, with more lines after it.
func filmJob(t *testing.T, after string) string {
	t.Helper()
	return jobLine + "scene: {smoke: 1}\n" + sized(t) + "renderer: cpu\n" +
		fmt.Sprintf("fps: %d\n", number(t, "fps")) +
		"duration: {value: 1, unit: s}\n" + after
}

// written is text written to a file by name in directory.
func written(t *testing.T, directory, name, text string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(text), private); err != nil {
		t.Fatal(err)
	}
	return path
}

// envIn is the door's surroundings for a test: its working directory,
// the tests' clock, and an environment of the variables given, as
// name, value, name, value.
func envIn(t *testing.T, ctx context.Context, directory string,
	variables ...string) Env {
	t.Helper()
	return envAt(t, Given{Context: ctx, Now: noon, Directory: directory},
		variables...)
}

// envAt is the door's surroundings given, with an environment of the
// variables given, as name, value, name, value.
func envAt(t *testing.T, given Given, variables ...string) Env {
	t.Helper()
	set := map[string]string{tempVariable: t.TempDir(),
		"SLAYER_CACHE": t.TempDir()}
	for i := nothing; i+one < len(variables); i += pair {
		set[variables[i]] = variables[i+1]
	}
	given.Environment = func(name string) (string, bool) {
		value, has := set[name]
		return value, has
	}
	env, err := NewEnv(given)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

// ran is what a call of a tool said and its exit number.
type ran struct {
	code      int
	said      string
	wrongSaid string
}

// running runs a tool on args with buffers.
func running(made *pzm.Tool, args ...string) ran {
	var said, wrong bytes.Buffer
	code := made.RunWith(args, strings.NewReader(""), &said, &wrong)
	return ran{code: code, said: said.String(), wrongSaid: wrong.String()}
}

// drew runs the real tool's draw on a job, and fails the test unless it
// ran.
func drew(t *testing.T, env Env, args ...string) string {
	t.Helper()
	got := running(Tool(env), append([]string{"draw"}, args...)...)
	if got.code != nothing {
		t.Fatalf("draw %v gave %d: %s%s", args, got.code, got.said,
			got.wrongSaid)
	}
	return got.said
}

// pictureOf is the png at path, decoded to RGBA.
func pictureOf(t *testing.T, path string) *image.RGBA {
	t.Helper()
	picture, err := decoded(path)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// files is every file under a directory, by its path relative to it.
func files(t *testing.T, directory string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry,
		err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(directory, path)
		found = append(found, relative)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}
