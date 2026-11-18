//go:build draw && darwin && cgo

package door

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// A film of the weather with a movie writes an mp4 of the map it shows,
// one frame a frame, at the grid's size, and says which map it is.
func TestDrawWeatherMovieIsTheMapItShows(t *testing.T) {
	cached(t)
	directory := t.TempDir()
	path := written(t, directory, "sky.yaml", weatherFilm(t, "",
		"outputs: {movie: sky.mp4}\n"))
	said := drew(t, envIn(t, context.Background(), directory,
		teamVariable, "team"), path)
	facts, err := movie.Probe(filepath.Join(directory, "sky.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if facts.Width() != number(t, "weather-across") ||
		facts.Height() != number(t, "weather-rows") ||
		facts.Frames() != pair {
		t.Errorf("the movie is %dx%d, %d frames", facts.Width(),
			facts.Height(), facts.Frames())
	}
	if !strings.Contains(said, "of the bands map") ||
		!strings.Contains(said, "removed the working maps") {
		t.Errorf("which map the movie is, or the maps' going, was not "+
			"said: %q", said)
	}
	if got := files(t, directory); !slices.Equal(got,
		[]string{"sky.mp4", "sky.yaml"}) {
		t.Errorf("the folder holds %v; its one picture is the film", got)
	}
}

// asked to keep its working frames, a film of the weather
// keeps its maps, and says where.
// anti-test of TestDrawWeatherMovieIsTheMapItShows
func TestDrawWeatherKeepsItsMapsWhenAsked(t *testing.T) {
	cached(t)
	directory := t.TempDir()
	path := written(t, directory, "sky.yaml", weatherFilm(t, "",
		"outputs: {movie: sky.mp4}\n"))
	said := drew(t, envIn(t, context.Background(), directory,
		teamVariable, "team"), "--keep-frames", path)
	if !strings.Contains(said, "kept the working maps in") {
		t.Errorf("the kept maps were not said: %q", said)
	}
}
