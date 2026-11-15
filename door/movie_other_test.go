//go:build !(darwin && cgo)

package door

import (
	"context"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// away from macOS a film with a movie is refused with the writer's own
// refusal.
// anti-test of TestDrawFilmWritesAMovieWithSound
func TestDrawFilmMovieNeedsMacOS(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {movie: film.mp4}\n"))
	got := running(Tool(envIn(t, context.Background(), directory)),
		"draw", path)
	if got.code == nothing || !strings.Contains(got.wrongSaid,
		movie.ErrNeedsMacOS.Error()) {
		t.Errorf("a movie away from macOS gave %d: %q", got.code,
			got.wrongSaid)
	}
}
