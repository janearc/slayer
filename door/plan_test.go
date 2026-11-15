package door

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
)

// --dry-run prints the job as YAML and each file it would write, and
// writes nothing.
func TestDrawDryRunWritesNothing(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml", filmJob(t,
		"outputs: {frames: film, movie: film.mp4}\n"))
	said := drew(t, envIn(t, context.Background(), directory), path,
		"--dry-run")
	for _, line := range []string{"job ", "fps: 3", "smoke: 1",
		"would write frames in " + filepath.Join(directory, "film"),
		"would write movie " + filepath.Join(directory, "film.mp4")} {
		if !strings.Contains(said, line) {
			t.Errorf("the dry run did not say %q: %q", line, said)
		}
	}
	if got := files(t, directory); !slices.Equal(got,
		[]string{"film.yaml"}) {
		t.Errorf("the dry run wrote %v", got)
	}
}

// each override stands over the job, as the dry run shows it.
func TestDrawOverridesStandOverTheJob(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml",
		stillJob(t, "outputs: {picture: still.png}\n"))
	env := envIn(t, context.Background(), directory)
	for _, each := range []struct{ flag, value, says string }{
		{"--width", "40", "width: 40"},
		{"--height", "20", "height: 20"},
		{"--rays", "2", "rays: 2"},
		{"--at", "2s", "value: 2"},
		{"--camera", "eye", "name: eye"},
		{"--renderer", "gpu", "renderer: gpu"},
		{"--quality", "preview", "quality: preview"},
		{"--vibes", "on", "vibes: \"on\""},
		{"--fastlight", "off", "fastlight: \"off\""},
		{"--lamp-store", "off", "lamp-store: \"off\""},
		{"--sky-looks", "on", "sky-looks: \"on\""},
		{"--out", "elsewhere.png", "would write picture " +
			filepath.Join(directory, "elsewhere.png")},
	} {
		said := drew(t, env, path, "--dry-run", each.flag, each.value)
		if !strings.Contains(said, each.says) {
			t.Errorf("%s %s did not stand over the job: %q", each.flag,
				each.value, said)
		}
	}
}

// a value a flag cannot take is refused in one wording,
// by the flag's name, and an override the job cannot take is refused by
// the job; a flag not given leaves the job's value alone.
// anti-test of TestDrawOverridesStandOverTheJob
// anti-test of TestDrawDryRunWritesNothing
func TestDrawOverridesRefuseBadValues(t *testing.T) {
	directory := t.TempDir()
	still := written(t, directory, "still.yaml", stillJob(t, ""))
	film := written(t, directory, "film.yaml", filmJob(t, ""))
	env := envIn(t, context.Background(), directory)
	for _, each := range []struct {
		args []string
		says string
	}{
		{[]string{still, "--width", "wide"},
			`--width takes a whole number, not "wide"`},
		{[]string{still, "--rays", "1.5"},
			`--rays takes a whole number, not "1.5"`},
		{[]string{still, "--at", "noon"},
			`--at takes a time such as 20s or 1m30s, not "noon"`},
		{[]string{still, "--camera="}, `--camera takes a camera's name`},
		{[]string{still, "--out="}, `--out takes a path`},
		{[]string{film, "--at", "2s"}, "one or the other"},
	} {
		got := running(Tool(env), append([]string{"draw", "--dry-run"},
			each.args...)...)
		if got.code == nothing || !strings.Contains(got.wrongSaid,
			each.says) {
			t.Errorf("%v gave %d: %q", each.args, got.code, got.wrongSaid)
		}
	}
	said := drew(t, env, still, "--dry-run")
	if strings.Contains(said, "quality: preview") {
		t.Error("a quality not given stood over the job's")
	}
}

// with no outputs said, a still goes to out/NAME.png and a film to
// out/NAME/ and out/NAME.mp4, under the working directory; a renders
// home not named touches nothing else.
func TestPlacesWithNoOutputsAreUnderOut(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	still := placedFor(t, env, written(t, directory, "lake.yaml",
		stillJob(t, "")))
	if still.picture != filepath.Join(directory, "out", "lake.png") ||
		still.folder != "" || still.copied != "" {
		t.Errorf("a still with no outputs is placed %+v", still)
	}
	film := placedFor(t, env, written(t, directory, "move.yaml",
		filmJob(t, "")))
	temporary, _ := env.lookup(tempVariable)
	if !film.work || !strings.HasPrefix(film.frames,
		filepath.Join(temporary, workFolder, "move-")) ||
		film.movie != filepath.Join(directory, "out", "move.mp4") {
		t.Errorf("a film with no outputs is placed %+v", film)
	}
}

// a film that writes a movie and no frames draws its frames in a working
// folder under the renders home's cache, by team, where the page makes
// no card; the anti-test: one that names its frames keeps them there,
// and they are not working files.
func TestPlacesWorkingFramesUnderTheCache(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	movie := placedFor(t, env, written(t, directory, "move.yaml",
		filmJob(t, "outputs: {movie: move.mp4}\n")))
	if !movie.work || !strings.HasPrefix(movie.frames,
		filepath.Join(home, cacheFolder, "team", "move-")) {
		t.Errorf("a movie's working frames are placed %+v", movie)
	}
	// anti-test of TestPlacesWorkingFramesUnderTheCache
	named := placedFor(t, env, written(t, directory, "named.yaml",
		filmJob(t, "outputs: {frames: frames, movie: named.mp4}\n")))
	if named.work || strings.Contains(named.frames, cacheFolder) {
		t.Errorf("frames the job names are placed %+v", named)
	}
}

// with a renders home, outputs said with no directory or not said go in
// the dated folder, a film's frames in the folder itself, and the job is
// copied there; an output said with a directory stays where it is said.
// anti-test of TestPlacesWithNoOutputsAreUnderOut
func TestPlacesUnderTheRendersHome(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	folder := filepath.Join("animations", today+"-move")
	film := placedFor(t, env, written(t, directory, "move.yaml",
		filmJob(t, "outputs: {frames: frames, movie: move.mp4}\n")))
	inside := filepath.Join(home, folder)
	if film.frames != inside || film.movie != filepath.Join(inside,
		"move.mp4") || film.folder != folder ||
		film.copied != filepath.Join(inside, "move.yaml") {
		t.Errorf("a film under the home is placed %+v", film)
	}
	kept := filepath.Join(directory, "kept", "move.mp4")
	said := placedFor(t, env, written(t, directory, "said.yaml",
		filmJob(t, "outputs: {movie: "+kept+"}\n")))
	if said.movie != kept || said.copied != "" || said.folder != "" {
		t.Errorf("a movie said with a directory was moved: %+v", said)
	}
	still := placedFor(t, env, written(t, directory, "lake.yaml",
		stillJob(t, "")))
	if still.picture != filepath.Join(home, "frames", today+"-lake",
		"lake.png") {
		t.Errorf("a still under the home is placed %+v", still)
	}
}

// a job file already in a folder under the renders home is drawn into
// that folder as it says, never moved to a dated one, and its note is
// that folder's; the anti-test of the move above.
// anti-test of TestPlacesUnderTheRendersHome
func TestPlacesOfAJobInsideTheRendersHomeStay(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	folder := filepath.Join(home, "animations", "short")
	if err := os.MkdirAll(folder, folderMode); err != nil {
		t.Fatal(err)
	}
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	got := placedFor(t, env, written(t, folder, "short.yaml", filmJob(t,
		"outputs: {frames: "+folder+", movie: short.mp4, note: a}\n")))
	if got.frames != folder || got.movie != filepath.Join(folder,
		"short.mp4") || got.copied != "" ||
		got.folder != filepath.Join("animations", "short") ||
		got.notes == "" {
		t.Errorf("a job inside the home is placed %+v", got)
	}
}

// a note is written for a folder under the renders home with a team,
// and why not is said for each other case.
func TestPlacesNoteOnlyUnderAHomeWithATeam(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	path := written(t, directory, "lake.yaml",
		stillJob(t, "outputs: {note: the lake at night}\n"))
	noted := placedFor(t, envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team"), path)
	if noted.notes != filepath.Join(home, "team", notesName) ||
		noted.note != "the lake at night" || noted.why != "" {
		t.Errorf("a note under a home is placed %+v", noted)
	}
	// anti-test of TestPlacesNoteOnlyUnderAHomeWithATeam
	for _, each := range []struct {
		env  Env
		says string
	}{
		{envIn(t, context.Background(), directory), rendersVariable},
		{envIn(t, context.Background(), directory, rendersVariable,
			home), teamVariable},
	} {
		got := placedFor(t, each.env, path)
		if got.notes != "" || !strings.Contains(got.why, each.says) {
			t.Errorf("a note with no %s is placed %+v", each.says, got)
		}
	}
	away := written(t, directory, "away.yaml", stillJob(t,
		"outputs: {picture: "+filepath.Join(directory, "x", "a.png")+
			", note: away}\n"))
	got := placedFor(t, envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team"), away)
	if got.notes != "" || !strings.Contains(got.why, "not under") {
		t.Errorf("a note outside the home is placed %+v", got)
	}
}

// placedFor is the places of the job at path, read as draw reads it.
func placedFor(t *testing.T, env Env, path string) places {
	t.Helper()
	wanted, err := jobOf(path)
	if err != nil {
		t.Fatal(err)
	}
	return placesOf(placing{wanted: wanted, given: path, env: env})
}

// sound with no movie, and a movie with no rate, are refused by name;
// a film with a movie and sound is not.
func TestRefusedSoundAndRate(t *testing.T) {
	directory := t.TempDir()
	silent, err := job.Read(written(t, directory, "silent.yaml",
		filmJob(t, "sound: {tune: \"@crypt\", voicing: \"@crypt-dmg\"}\n")))
	if err != nil {
		t.Fatal(err)
	}
	err = refused(silent, places{frames: directory})
	if err == nil || !strings.Contains(err.Error(), "no movie") {
		t.Errorf("sound with no movie gave %v", err)
	}
	// anti-test of TestRefusedSoundAndRate
	if err := refused(silent, places{movie: "a.mp4"}); err != nil {
		t.Errorf("sound with a movie was refused: %v", err)
	}
	still, err := job.Read(written(t, directory, "still.yaml",
		stillJob(t, "")))
	if err != nil {
		t.Fatal(err)
	}
	err = refused(still, places{movie: "a.mp4"})
	if err == nil || !strings.Contains(err.Error(), "no rate") {
		t.Errorf("a movie with no rate gave %v", err)
	}
}

// a sequence plan is lifted, its output its frames; a job file is read.
func TestJobOfReadsEachKind(t *testing.T) {
	directory := t.TempDir()
	plan, err := os.ReadFile("../doc/camera-smoke-sequence.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := written(t, directory, "plan.yaml", string(plan))
	lifted, err := jobOf(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := lifted.Outputs().Frames(); !has || lifted.Clock().Still() {
		t.Error("the plan was not lifted to a film of frames")
	}
	read, err := jobOf(written(t, directory, "job.yaml", filmJob(t, "")))
	if err != nil || read.File() == "" {
		t.Errorf("a job file was not read as a job: %v", err)
	}
}

// NewEnv takes what it is given, and refuses each missing piece by name.
func TestNewEnvRefusesWhatIsMissing(t *testing.T) {
	_, err := NewEnv(Given{Directory: "relative"})
	for _, says := range []string{"context", "environment", "clock",
		"not absolute"} {
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("a missing %s gave %v", says, err)
		}
	}
	// anti-test of TestNewEnvRefusesWhatIsMissing
	env := envIn(t, context.Background(), t.TempDir(), "EMPTY", "")
	if _, has := env.lookup("EMPTY"); has {
		t.Error("a variable set to nothing was taken as said")
	}
}
