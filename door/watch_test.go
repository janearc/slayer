package door

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/job"
)

// exampleEnv is the door's surroundings for an example: here, with no
// renders home and a cache under the system's temporary folder.
func exampleEnv() (Env, string) {
	here, _ := os.Getwd()
	cache, _ := os.MkdirTemp("", "slayer-example")
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(name string) (string, bool) {
			if name == "SLAYER_CACHE" {
				return cache, true
			}
			return "", false
		},
		Now: time.Now, Directory: here})
	return env, filepath.Dir(here)
}

func Example_watch() {
	env, top := exampleEnv()
	var said bytes.Buffer
	Tool(env).Run([]string{"watch", "../doc/examples/jobs/smoke-still.yaml",
		"--dry-run"}, &said, &said)
	fmt.Print(strings.ReplaceAll(said.String(), top, ".."))
	// Output:
	// would watch ../doc/examples/jobs/smoke-still.yaml at preview quality
	// would write ../doc/examples/jobs/.smoke-still.watch
}

func Example_edit() {
	env, top := exampleEnv()
	var said bytes.Buffer
	Tool(env).Run([]string{"edit", "../doc/examples/jobs/smoke-still.yaml",
		"--dry-run"}, &said, &said)
	for _, line := range strings.Split(said.String(), "\n") {
		if strings.HasPrefix(line, "would lay out") {
			_, after, _ := strings.Cut(line, ", and ")
			fmt.Println(strings.ReplaceAll(after, top, ".."))
		}
	}
	// Output: watch ../doc/examples/jobs/smoke-still.yaml
}

func Example_schema() {
	env, _ := exampleEnv()
	var said bytes.Buffer
	Tool(env).Run([]string{"schema"}, &said, &said)
	fmt.Println(strings.Contains(said.String(), jobSchema))
	said.Reset()
	Tool(env).Run([]string{"schema", "--where", "--dry-run"}, &said,
		&said)
	fmt.Println(strings.HasSuffix(strings.TrimSpace(said.String()),
		filepath.Join(schemaFolder, schemaName)))
	// Output:
	// true
	// true
}

// watch reads a job at preview quality where neither the job nor
// --quality says one, and lays the flags over it as draw does.
func TestWatchReadsAtPreviewQuality(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "a.yaml", filmJob(t, ""))
	wanted, err := reading([]job.Override{job.Rays(pair)}, false)(path)
	if err != nil {
		t.Fatal(err)
	}
	if quality, _ := wanted.Quality(); quality != previewQuality {
		t.Errorf("an unsaid quality is watched at %s", quality)
	}
	if rays, _ := wanted.Film().Rays(); rays != pair {
		t.Errorf("--rays 2 gave %d", rays)
	}
}

// a job or a flag that says exact is watched at exact; a file
// that does not read is refused with its name.
// anti-test of TestWatchReadsAtPreviewQuality
func TestWatchKeepsAnExactQuality(t *testing.T) {
	directory := t.TempDir()
	said := written(t, directory, "a.yaml", filmJob(t, "quality: exact\n"))
	if wanted, err := reading(nil, false)(said); err != nil {
		t.Fatal(err)
	} else if quality, _ := wanted.Quality(); quality != "exact" {
		t.Errorf("a job that says exact is watched at %s", quality)
	}
	plain := written(t, directory, "b.yaml", filmJob(t, ""))
	flagged, err := reading([]job.Override{job.Quality("exact")},
		true)(plain)
	if quality, _ := flagged.Quality(); err != nil || quality != "exact" {
		t.Errorf("--quality exact is watched at %s, %v", quality, err)
	}
	broken := written(t, directory, "c.yaml", jobLine+"scene: [\n")
	if _, err := reading(nil, false)(broken); err == nil ||
		!strings.Contains(err.Error(), "c.yaml") {
		t.Errorf("a broken job gave %v", err)
	}
}

// watch with output that is not a terminal refuses, saying to use draw;
// and a file that is not there is refused, saying edit starts one.
// anti-test of TestWatchUnderAPty
func TestWatchRefusesWithoutATerminal(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "a.yaml", filmJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "watch", path)
	if got.code == nothing || !strings.Contains(got.wrongSaid,
		"slayer draw") {
		t.Errorf("watch to a buffer gave %d: %s", got.code, got.wrongSaid)
	}
	missing := running(Tool(env), "watch", filepath.Join(directory,
		"none.yaml"))
	if missing.code == nothing || !strings.Contains(missing.wrongSaid,
		"slayer edit") {
		t.Errorf("watch of nothing gave %d: %s", missing.code,
			missing.wrongSaid)
	}
}

// watch takes one file, not none and not two.
// anti-test of TestWatchReadsAtPreviewQuality
func TestWatchTakesOneFile(t *testing.T) {
	env := envIn(t, context.Background(), t.TempDir())
	for _, args := range [][]string{{"watch"}, {"watch", "a", "b"}} {
		if got := running(Tool(env), args...); got.code == nothing {
			t.Errorf("%v ran", args)
		}
	}
}

// the status folder is beside the job, named for it.
func TestStatusFolderIsBesideTheJob(t *testing.T) {
	if got := statusFolder("/jobs/fire.night.yaml"); got !=
		"/jobs/.fire.night.watch" {
		t.Errorf("the status folder is %s", got)
	}
}

// frames already rendered are refused by name, a size below
// nought is refused, and a scene that is not a set has no cameras.
// anti-test of TestSessionsStageAJob
func TestSessionsRefuseWhatTheyCannotWatch(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "a.yaml", jobLine+"scene: {frames: "+
		directory+"}\nfps: 3\nduration: {value: 1, unit: s}\n")
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (sessions{}).Stage(context.Background(),
		wanted); err == nil || !strings.Contains(err.Error(), "frames") {
		t.Errorf("frames were staged: %v", err)
	}
	smoke, err := job.Read(written(t, directory, "b.yaml",
		filmJob(t, "")))
	if err != nil {
		t.Fatal(err)
	}
	if got := camerasOf(smoke); got != nil {
		t.Errorf("a smoke has cameras %v", got)
	}
}
