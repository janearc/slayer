package door

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// Only names written without a directory are moved into the renders home.
// Explicit sibling paths must not be mistaken for bare names after resolution.
func TestExplicitSiblingOutputsStayWhereNamed(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory, rendersVariable, home)
	frames := filepath.Join(directory, "frames")
	movie := filepath.Join(directory, "film.mp4")
	picture := filepath.Join(directory, "still.png")
	raw := filepath.Join(directory, "light.pfm")
	for _, each := range []struct {
		name string
		job  string
		args []string
		want []string
	}{
		{"bare-film", filmJob(t, "outputs: {frames: frames, movie: film.mp4}\n"), nil, []string{
			"would write frames in " + filepath.Join(home, "animations", today+"-bare-film"),
			"would write movie " + filepath.Join(home, "animations", today+"-bare-film", "film.mp4"),
		}},
		{"bare-film-width", filmJob(t, "outputs: {frames: frames, movie: film.mp4}\n"), []string{"--width", "40"}, []string{
			"would write frames in " + filepath.Join(home, "animations", today+"-bare-film-width"),
			"would write movie " + filepath.Join(home, "animations", today+"-bare-film-width", "film.mp4"),
		}},
		{"absolute-film", filmJob(t, fmt.Sprintf("outputs: {frames: %q, movie: %q}\n", frames, movie)), nil, []string{
			"would write frames in " + frames, "would write movie " + movie,
		}},
		{"absolute-film-width", filmJob(t, fmt.Sprintf("outputs: {frames: %q, movie: %q}\n", frames, movie)), []string{"--width", "40"}, []string{
			"would write frames in " + frames, "would write movie " + movie,
		}},
		{"dot-film", filmJob(t, "outputs: {frames: ./frames, movie: ./film.mp4}\n"), nil, []string{
			"would write frames in " + frames, "would write movie " + movie,
		}},
		{"absolute-still", stillJob(t, fmt.Sprintf("outputs: {picture: %q, raw: %q}\n", picture, raw)), nil, []string{
			"would write picture " + picture, "would write raw light " + raw,
		}},
		{"override-film", filmJob(t, "outputs: {frames: frames}\n"), []string{"--out", frames}, []string{
			"would write frames in " + frames,
		}},
		{"override-keeps-movie-form", filmJob(t, "outputs: {frames: frames, movie: film.mp4}\n"), []string{"--out", frames}, []string{
			"would write frames in " + frames,
			"would write movie " + filepath.Join(home, "animations", today+"-override-keeps-movie-form", "film.mp4"),
		}},
		{"override-keeps-frames-form", filmJob(t, "outputs: {frames: frames, movie: film.mp4}\n"), []string{"--out", movie}, []string{
			"would write frames in " + filepath.Join(home, "animations", today+"-override-keeps-frames-form"),
			"would write movie " + movie,
		}},
	} {
		t.Run(each.name, func(t *testing.T) {
			path := written(t, directory, each.name+".yaml", each.job)
			args := append([]string{path, "--dry-run"}, each.args...)
			said := drew(t, env, args...)
			for _, want := range each.want {
				if !strings.Contains(said, want) {
					t.Errorf("missing %q in %s", want, said)
				}
			}
		})
	}
}
