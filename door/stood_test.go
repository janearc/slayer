package door

// The places flag's refusals and dry run, which need no shelf: a world
// with no ground, frames already rendered (a mock source that cannot say
// its places) and the ways of assessing that stage nothing.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

// --places is for the ways of assessing that stage a scene: with
// --judge it is refused by both names.
func TestAssessPlacesIsRefusedWithJudge(t *testing.T) {
	directory := t.TempDir()
	got := assessed(t, directory, widow(t), "--judge", "--places", "p.csv")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"--places does nothing with judge") {
		t.Errorf("--judge --places gave %d: %q", got.code, got.wrongSaid)
	}
}

// a set of a world and no ground stands nothing on a ground: its places
// are refused by name, and no file or folder is made.
func TestAssessPlacesOfAWorldWithNoGroundIsRefused(t *testing.T) {
	directory := t.TempDir()
	got := assessed(t, directory, widow(t), "--places", "out/p.csv")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"stands nothing on a ground") {
		t.Errorf("a world's places gave %d: %q", got.code, got.wrongSaid)
	}
	if listed := files(t, directory); len(listed) != nothing {
		t.Errorf("a refused places table wrote %v", listed)
	}
}

// a scene of frames already rendered has no staged things: its places
// are refused by name, writing nothing, and the source is let go.
func TestAssessPlacesOfPicturesIsRefused(t *testing.T) {
	control := gomock.NewController(t)
	shop := NewMockworkshop(control)
	opened := NewMocksource(control)
	shop.EXPECT().Source(gomock.Any(), gomock.Any()).Return(opened, nil)
	opened.EXPECT().Close()
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	got := running(tool(env, shop, machine{}), "assess", widow(t),
		"--places", "p.csv")
	if got.code != one || !strings.Contains(got.wrongSaid,
		"frames already rendered") {
		t.Errorf("an unplaced source gave %d: %q", got.code,
			got.wrongSaid)
	}
	if listed := files(t, directory); len(listed) != nothing {
		t.Errorf("a refused places table wrote %v", listed)
	}
}

// a dry run with --places says the table it would write, where it
// would, and writes nothing; draw without the flag says no table.
func TestADryRunSaysThePlacesTableItWouldWrite(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "draw", path, "--places", "p.csv",
		"--dry-run")
	want := "would write places table " + filepath.Join(directory, "p.csv")
	if got.code != nothing || !strings.Contains(got.said, want) {
		t.Errorf("a dry run gave %d: %q %q", got.code, got.said,
			got.wrongSaid)
	}
	plain := running(Tool(env), "draw", path, "--dry-run")
	if strings.Contains(plain.said, "places table") {
		t.Errorf("a dry run with no flag said %q", plain.said)
	}
	if listed := files(t, directory); len(listed) != one {
		t.Errorf("a dry run wrote %v", listed)
	}
}
