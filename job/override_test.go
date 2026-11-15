package job

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTheCommandLineStandsOverTheJob holds every override over what the
// job and its preset say, and the job itself unchanged.
func TestTheCommandLineStandsOverTheJob(t *testing.T) {
	job := read(t, still+"film: {preset: portrait, rays: 2}\n"+
		"camera:\n  name: rim\n"+zoom+"outputs: {raw: light.pfm}\n")
	over, err := job.With(Width(640), Height(480), Rays(4),
		At(time.Minute), CameraName("low"), Renderer("cpu"),
		Quality("preview"), Output("out.png"))
	if err != nil {
		t.Fatal(err)
	}
	width, _ := over.Film().Width()
	height, _ := over.Film().Height()
	rays, _ := over.Film().Rays()
	at, _ := over.Clock().At()
	lens, _ := over.Camera()
	name, _ := lens.Name()
	_, zoomed := lens.Zoom()
	renderer, _ := over.Renderer()
	quality, _ := over.Quality()
	picture, _ := over.Outputs().Picture()
	_, raw := over.Outputs().Raw()
	got := spoken(width, height, rays, at, name, zoomed, renderer,
		quality, filepath.Base(picture), raw)
	if want := "640 480 4 1m0s low true cpu preview out.png true"; got != want {
		t.Errorf("the command line gave %s, want %s", got, want)
	}
	if width, _ := job.Film().Width(); fmt.Sprint(width) != "1080" {
		t.Errorf("With changed the job under it: width %d", width)
	}
	if over.Key() == job.Key() || over.File() != job.File() {
		t.Error("an override left the key, or lost the file")
	}
}

// TestAnOverridesPathIsTheWorkingDirectorys holds a command line's path
// to the working directory, not the job file's.
func TestAnOverridesPathIsTheWorkingDirectorys(t *testing.T) {
	job := read(t, film)
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	over, err := job.With(Output("out/frames"))
	if err != nil {
		t.Fatal(err)
	}
	frames, _ := over.Outputs().Frames()
	if frames != filepath.Join(working, "out", "frames") {
		t.Errorf("a film's output went to %s", frames)
	}
	over, err = job.With(Output("film.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if movie, _ := over.Outputs().Movie(); filepath.Base(movie) != "film.mp4" {
		t.Errorf("a film's .mp4 went to %s", movie)
	}
}

// TestAnOverrideTheJobCannotTakeIsRefused is the anti-test: an override
// out of bounds, empty, or at on a film is refused by name.
// anti-test of TestTheCommandLineStandsOverTheJob
// anti-test of TestAnOverridesPathIsTheWorkingDirectorys
func TestAnOverrideTheJobCannotTakeIsRefused(t *testing.T) {
	job := read(t, film)
	for words, override := range map[string]Override{
		"/film/width":            Width(9000),
		"/film/rays":             Rays(-one),
		"at is a still's time":   At(time.Second),
		"/renderer":              Renderer("vulkan"),
		"height is empty":        Height(nothing),
		"output is empty":        Output(""),
		"camera is empty":        CameraName(""),
		"quality is empty":       Quality(""),
		"the command line over ": Quality("draft"),
	} {
		_, err := job.With(override)
		if err == nil {
			t.Errorf("an override that should say %q was taken", words)
			continue
		}
		saying(t, err, words)
	}
}

// TestWithKeepsTheTitlesAsWritten is the anti-test of re-reading: the
// titles come through an override in their order, ramps and all, so the
// job's key changes only by the override.
// anti-test of TestTheCommandLineStandsOverTheJob
func TestWithKeepsTheTitlesAsWritten(t *testing.T) {
	job := read(t, titled)
	over, err := job.With(Quality("exact"))
	if err != nil {
		t.Fatal(err)
	}
	if over.Key() != job.Key() {
		t.Error("saying the default quality changed the titles' key")
	}
	words, _ := over.Titles()
	var names []string
	for _, each := range words.Elements() {
		names = append(names, each.Name())
	}
	if got := spoken(names); got != "[name height border progress]" {
		t.Errorf("the titles came through as %s", got)
	}
}
