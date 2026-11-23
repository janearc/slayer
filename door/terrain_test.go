package door

// The terrain flags' refusals and dry run, which need no shelf and no
// network: every run here has no shelf named, or a shelf of its own with
// nothing in it, and a mock workshop that is told nothing, so a draw, a
// fetch or a staging would fail the test.

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

// the two terrain flags of draw are one at a time: both together are
// refused by their names before anything is read or tended, and the
// workshop is not asked for a piece.
func TestDrawRefusesTwoTerrainFlagsTogether(t *testing.T) {
	t.Setenv("SLAYER_SHELF", "")
	shop := NewMockworkshop(gomock.NewController(t))
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "draw", path,
		"--fetch-terrain", "--refresh-terrain")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"only one of --fetch-terrain, --refresh-terrain and "+
			"--check-terrain at a time") {
		t.Errorf("two terrain flags gave %d: %q", got.code, got.wrongSaid)
	}
	if listed := files(t, directory); len(listed) != one {
		t.Errorf("a refused draw left %v", listed)
	}
}

// draw has no --check-terrain, which is assess's: the flag is refused
// as one the verb does not have.
func TestDrawHasNoCheckTerrainFlag(t *testing.T) {
	t.Setenv("SLAYER_SHELF", "")
	shop := NewMockworkshop(gomock.NewController(t))
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "draw", path,
		"--check-terrain")
	if got.code == nothing || !strings.Contains(got.wrongSaid,
		"check-terrain") {
		t.Errorf("draw --check-terrain gave %d: %q", got.code,
			got.wrongSaid)
	}
}

// a terrain flag of a job whose scene is no set is refused by name,
// since there are no datasets to tend, and nothing is drawn.
func TestATerrainFlagOfAJobOfNoSetIsRefused(t *testing.T) {
	t.Setenv("SLAYER_SHELF", "")
	shop := NewMockworkshop(gomock.NewController(t))
	directory := t.TempDir()
	path := written(t, directory, "film.yaml", filmJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "draw", path,
		"--fetch-terrain")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"--fetch-terrain tends the datasets a set names, and this "+
			"job's scene is not a set") {
		t.Errorf("a smoke's terrain flag gave %d: %q", got.code,
			got.wrongSaid)
	}
}

// a dry run with a terrain flag says what it would tend and tends
// nothing: with no shelf named, a tending would fail for want of one.
// Without the flag a dry run says nothing of tending.
func TestADryRunSaysTheTendingItWouldDo(t *testing.T) {
	t.Setenv("SLAYER_SHELF", "")
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "draw", path, "--fetch-terrain",
		"--dry-run")
	if got.code != nothing || !strings.Contains(got.said, "would fetch "+
		"what the shelf lacks of the datasets ") ||
		!strings.Contains(got.said, "widow.yaml names") {
		t.Errorf("a dry run gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	plain := running(Tool(env), "draw", path, "--dry-run")
	if plain.code != nothing || strings.Contains(plain.said, "would "+
		"fetch") || strings.Contains(plain.said, "would compare") {
		t.Errorf("a dry run with no flag gave %d: %q", plain.code,
			plain.said)
	}
}

// --check-terrain takes no other flag of assess: with --census it is
// refused by both names.
func TestAssessCheckTerrainTakesNoOtherFlag(t *testing.T) {
	t.Setenv("SLAYER_SHELF", t.TempDir())
	directory := t.TempDir()
	got := assessed(t, directory, widow(t), "--check-terrain", "--census",
		"c.json")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"--census does nothing with check-terrain") {
		t.Errorf("--check-terrain --census gave %d: %q", got.code,
			got.wrongSaid)
	}
	if listed := files(t, directory); len(listed) != nothing {
		t.Errorf("a refused assess wrote %v", listed)
	}
}

// a set that reads no terrain and no moon face has nothing to compare:
// the check says so, exits 0 and downloads nothing, the shelf being an
// empty folder of the test's own.
func TestAssessCheckTerrainOfASetWithNoTerrainHasNothingToTend(t *testing.T) {
	t.Setenv("SLAYER_SHELF", t.TempDir())
	directory := t.TempDir()
	got := assessed(t, directory, widow(t), "--check-terrain")
	if got.code != nothing || !strings.Contains(got.said,
		"names no terrain and no moon face; nothing to tend") {
		t.Errorf("the check gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
}
