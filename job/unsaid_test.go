package job

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAJobThatSaysNothingLeavesEveryKeyUnsaid is the anti-test of every
// read: a key the job leaves out is reported unsaid, for the levels below
// to decide.
// anti-test of TestEveryKeyReads
// anti-test of TestSaidIsTheJobsOwnWordAndNothingElse
func TestAJobThatSaysNothingLeavesEveryKeyUnsaid(t *testing.T) {
	job := read(t, still)
	lens, hasCamera := job.Camera()
	_, hasName := lens.Name()
	_, hasPath := lens.Path()
	_, hasZoom := lens.Zoom()
	quality, hasQuality := job.Quality()
	_, hasRenderer := job.Renderer()
	_, hasShutter := job.Shutter()
	_, hasPrint := job.Print()
	_, hasStock := job.Stock()
	_, hasTitles := job.Titles()
	_, hasSound := job.Sound()
	_, hasAt := job.Clock().At()
	_, hasRate := job.Clock().Rate()
	_, hasStart := job.Clock().Start()
	_, hasStretch := job.Clock().SceneDuration()
	_, hasMemory := job.Limits().Memory()
	_, hasTime := job.Limits().Time()
	for _, each := range Switches() {
		if on, said := job.Said(each.Name()); on || said ||
			each.Shortcut() && job.Shortcut(each.Name()) {
			t.Errorf("%s is said by a job that does not say it",
				each.Name())
		}
	}
	for key, has := range map[string]bool{"camera": hasCamera,
		"camera name": hasName, "camera path": hasPath,
		"camera zoom": hasZoom, "quality": hasQuality,
		"renderer": hasRenderer, "shutter": hasShutter,
		"print": hasPrint, "stock": hasStock, "titles": hasTitles,
		"sound": hasSound, "at": hasAt, "fps": hasRate,
		"start": hasStart, "scene-duration": hasStretch,
		"memory": hasMemory, "time": hasTime,
		"film": filmSaid(job.Film()), "outputs": outputsSaid(job),
		"scene": sceneOthers(job.Scene())} {
		if has {
			t.Errorf("%s is said by a job that does not say it", key)
		}
	}
	if quality != defaultQuality {
		t.Errorf("an unsaid quality is %s, want %s", quality,
			defaultQuality)
	}
}

// filmSaid is whether a film says any of its keys.
func filmSaid(made Film) bool {
	_, preset := made.Preset()
	_, width := made.Width()
	_, height := made.Height()
	_, rays := made.Rays()
	_, wide := made.Wide()
	_, tall := made.Tall()
	_, exposure := made.Exposure()
	_, across := made.WideFor(one, one)
	_, aperture := made.Aperture()
	_, focus := made.Focus()
	return preset || width || height || rays || wide || tall ||
		exposure || across || aperture || focus
}

// outputsSaid is whether a job says any output.
func outputsSaid(job Job) bool {
	out := job.Outputs()
	_, picture := out.Picture()
	_, raw := out.Raw()
	_, frames := out.Frames()
	_, movie := out.Movie()
	_, note := out.Note()
	return picture || raw || frames || movie || note
}

// sceneOthers is whether a set's scene says any other form.
func sceneOthers(scene Scene) bool {
	_, smoke := scene.Smoke()
	_, named := scene.Named()
	_, frames := scene.Frames()
	_, play := scene.Play()
	return smoke || named || frames || play || scene.Parameters() != nil
}

// TestAnUnsaidPartIsEmpty holds that the parts of a job that says none
// of them are their zeros, not another job's.
func TestAnUnsaidPartIsEmpty(t *testing.T) {
	job := read(t, film)
	element := Title{}
	_, _, counted, _ := element.Words()
	_, hasValue := element.Value()
	_, hasFrom := element.From()
	_, hasUntil := element.Until()
	track := Sound{}
	_, _, tune := track.Tune()
	_, file := track.File()
	_, from := track.From()
	_, beats := track.Beats()
	if counted || hasValue || hasFrom || hasUntil || tune || file ||
		from || beats {
		t.Error("an empty title or sound says something")
	}
	if job.Clock().Still() || len(job.File()) == nothing {
		t.Error("a film read from a file is a still, or has no file")
	}
}

// TestPathsResolveAgainstTheJobFile holds that a relative path is the
// job file's directory's, and an absolute one and a built-in are kept.
func TestPathsResolveAgainstTheJobFile(t *testing.T) {
	path := written(t, film+"sound: {tune: '@moon', voicing: v/x.yaml}\n"+
		"outputs: {frames: /absolute/frames, movie: ../up.mp4}\n")
	job, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Dir(path)
	set, _ := job.Scene().Set()
	track, _ := job.Sound()
	score, voicing, _ := track.Tune()
	frames, _ := job.Outputs().Frames()
	movie, _ := job.Outputs().Movie()
	for _, each := range []struct{ got, want string }{
		{set, filepath.Join(directory, "moonlake.yaml")},
		{score, "@moon"},
		{voicing, filepath.Join(directory, "v", "x.yaml")},
		{frames, "/absolute/frames"},
		{movie, filepath.Join(filepath.Dir(directory), "up.mp4")},
	} {
		if each.got != each.want {
			t.Errorf("resolved %s, want %s", each.got, each.want)
		}
	}
}

// TestAnAtNameIsLeftOnlyForTunesAndVoicings is the anti-test: a file or
// a score that starts with @ is a path like any other.
// anti-test of TestPathsResolveAgainstTheJobFile
func TestAnAtNameIsLeftOnlyForTunesAndVoicings(t *testing.T) {
	job := read(t, film+"sound: {file: '@take.wav', score: '@s.yaml'}\n")
	track, _ := job.Sound()
	file, _ := track.File()
	score, _ := track.Beats()
	for _, path := range []string{file, score} {
		if strings.HasPrefix(path, builtIn) || !filepath.IsAbs(path) {
			t.Errorf("%s was left as written", path)
		}
	}
}

// TestReadSaysWhenTheFileIsNotThere holds that a missing file is the
// file's error and not a refusal of a job.
// anti-test of TestEveryKeyReads
func TestReadSaysWhenTheFileIsNotThere(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "absent.yaml")
	_, err := Read(absent)
	if err == nil || strings.Contains(err.Error(), ErrRefused.Error()) {
		t.Errorf("a missing file read as %v", err)
	}
}
