//go:build draw

package door

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// a note the job says with no renders home is not written, and the dry
// run and the draw each say why.
func TestDrawSaysWhyANoteIsNotWritten(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "lake.yaml", stillJob(t,
		"outputs: {picture: lake.png, note: the lake}\n"))
	env := envIn(t, context.Background(), directory)
	dry := drew(t, env, path, "--dry-run")
	if !strings.Contains(dry, "would not write the note: no renders") {
		t.Errorf("the dry run did not say why: %q", dry)
	}
	said := drew(t, env, path)
	if !strings.Contains(said, "the note is not written: no renders") {
		t.Errorf("the draw did not say why: %q", said)
	}
}

// the frames already rendered are refused where they cannot be fitted:
// a film that names half a size.
// anti-test of TestDrawFramesAlreadyRendered
func TestDrawFramesRefusesHalfASize(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	drew(t, env, written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n")))
	half := written(t, directory, "half.yaml", jobLine+
		"scene: {frames: film}\n"+fmt.Sprintf("film: {width: %d}\n",
		number(t, "width"))+fmt.Sprintf("fps: %d\n", number(t, "fps"))+
		"duration: {value: 1, unit: s}\noutputs: {frames: half}\n")
	got := running(Tool(env), "draw", half)
	if got.code == nothing || !strings.Contains(got.wrongSaid, "not both") {
		t.Errorf("a film of half a size gave %q", got.wrongSaid)
	}
}
