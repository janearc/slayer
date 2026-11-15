//go:build draw

package door

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// widowCopy is the widow set copied into a folder, so a test can change
// it, and a still job of it there.
func widowCopy(t *testing.T, directory, picture string) (string, string) {
	t.Helper()
	raw, err := os.ReadFile(widow(t))
	if err != nil {
		t.Fatal(err)
	}
	set := written(t, directory, "widow.yaml", string(raw))
	job := written(t, directory, picture+".yaml", jobLine+
		"scene: {set: widow.yaml}\n"+sized(t)+"renderer: cpu\n"+
		"outputs: {picture: "+picture+".png}\n")
	return set, job
}

func TestAStillDrawnAgainIsTheStillFromTheCache(t *testing.T) {
	directory := t.TempDir()
	_, path := widowCopy(t, directory, "still")
	env := envIn(t, context.Background(), directory,
		keptStillsVariable, keptStillsOn)
	cold := drew(t, env, path)
	picture := filepath.Join(directory, "still.png")
	drawn, _ := os.ReadFile(picture)
	if !strings.Contains(cold, "kept the picture") {
		t.Fatalf("the cold draw kept nothing: %s", cold)
	}
	warm := drew(t, env, path)
	served, _ := os.ReadFile(picture)
	if !strings.Contains(warm, "from the cache, drawn") ||
		!bytes.Equal(served, drawn) {
		t.Fatalf("the warm draw: %s", warm)
	}
	fresh := drew(t, env, path, "--fresh")
	again, _ := os.ReadFile(picture)
	if strings.Contains(fresh, "from the cache") ||
		!bytes.Equal(again, drawn) {
		t.Fatalf("--fresh: %s, or a fresh draw is not the kept one", fresh)
	}
}

// unasked, a kept still is kept and never served, since the key
// does not yet hold every file a set reads from beside itself.
// anti-test of TestAStillDrawnAgainIsTheStillFromTheCache
func TestAKeptStillIsNotServedUnasked(t *testing.T) {
	directory := t.TempDir()
	_, path := widowCopy(t, directory, "still")
	env := envIn(t, context.Background(), directory)
	cold := drew(t, env, path)
	if !strings.Contains(cold, "kept the picture") {
		t.Fatalf("the cold draw kept nothing: %s", cold)
	}
	if warm := drew(t, env, path); strings.Contains(warm, "from the cache") {
		t.Fatalf("a kept still was served unasked: %s", warm)
	}
}

// anti-test of TestAStillDrawnAgainIsTheStillFromTheCache
func TestAStillOfASetOneByteApartIsDrawn(t *testing.T) {
	directory := t.TempDir()
	set, path := widowCopy(t, directory, "near")
	env := envIn(t, context.Background(), directory,
		keptStillsVariable, keptStillsOn)
	drew(t, env, path)
	raw, _ := os.ReadFile(set)
	if err := os.WriteFile(set, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if said := drew(t, env, path); strings.Contains(said, "from the cache") {
		t.Fatalf("a changed set was served from the cache: %s", said)
	}
}

// anti-test of TestAStillDrawnAgainIsTheStillFromTheCache
func TestAStillWithRawIsNotKept(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "rawed.yaml", stillJob(t,
		"outputs: {picture: rawed.png, raw: rawed.pfm}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	if said := drew(t, env, path); strings.Contains(said, "cache") {
		t.Fatalf("a still with raw went through the cache: %s", said)
	}
}
