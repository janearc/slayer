//go:build draw

package door

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// a still drawn under a renders home lands in its dated folder, the job
// beside it, and its note is one line with its tab; drawing it again
// appends a line for the folder, never rewriting another's.
func TestDrawUnderTheRendersHome(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	path := written(t, directory, "lake.yaml", stillJob(t,
		"outputs: {picture: lake.png, note: the first try}\n"))
	drew(t, env, path)
	folder := filepath.Join("frames", today+"-lake")
	if got := files(t, filepath.Join(home, folder)); !slices.Equal(got,
		[]string{"lake.png", "lake.yaml"}) {
		t.Errorf("the dated folder holds %v", got)
	}
	notes := filepath.Join(home, "team", notesName)
	written(t, filepath.Dir(notes), notesName, "frames/other\tkept\n")
	drew(t, env, path)
	path = written(t, directory, "lake.yaml", stillJob(t,
		"outputs: {picture: lake.png, note: the second try}\n"))
	drew(t, env, path, "--replace")
	raw, err := os.ReadFile(notes)
	if err != nil {
		t.Fatal(err)
	}
	want := "frames/other\tkept\n" + folder + "\tthe first try\n" +
		folder + "\tthe second try\n"
	if string(raw) != want {
		t.Errorf("the notes file holds %q, not %q", raw, want)
	}
	if got := files(t, directory); !slices.Equal(got,
		[]string{"lake.yaml"}) {
		t.Errorf("drawing under a home wrote beside the job: %v", got)
	}
}

// another job of the same name on the same day does not
// draw over the still or its job file unless told to replace them.
// anti-test of TestDrawUnderTheRendersHome
func TestDrawRefusesAnotherJobsStill(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	env := envIn(t, context.Background(), directory,
		rendersVariable, home, teamVariable, "team")
	first := written(t, directory, "lake.yaml", stillJob(t, ""))
	drew(t, env, first)
	inside := filepath.Join(home, "frames", today+"-lake")
	picture := filepath.Join(inside, "lake.png")
	before, err := os.ReadFile(filepath.Join(inside, "lake.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	second := written(t, other, "lake.yaml", stillJob(t,
		"stock: {name: lantern}\n"))
	got := running(Tool(envIn(t, context.Background(), other,
		rendersVariable, home, teamVariable, "team")), "draw", second)
	if got.code == nothing || !strings.Contains(got.wrongSaid,
		picture+" was drawn from another job") {
		t.Errorf("another job's still gave %d: %q", got.code, got.wrongSaid)
	}
	after, _ := os.ReadFile(filepath.Join(inside, "lake.yaml"))
	if string(after) != string(before) {
		t.Error("a refused still replaced the job file beside the picture")
	}
	drew(t, env, first)
}
