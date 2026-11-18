package workload

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// resolveTestSource declares dependencies from actual fixture file contents.
func resolveTestSource(_ context.Context, _ string, source io.ReadSeeker) (Dependencies, error) {
	var description struct {
		Uses  []string
		Known bool
	}
	if err := json.NewDecoder(source).Decode(&description); err != nil {
		return Dependencies{}, err
	}
	return Dependencies{Files: description.Uses, Complete: description.Known}, nil
}

func TestWorkloadSourceRecursiveClosureChangesWithNestedContent(t *testing.T) {
	home := t.TempDir()
	write := func(name, content string) string {
		t.Helper()
		path := filepath.Join(home, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	root := write("job.json", `{"uses":["plan.json","plan.json"],"known":true}`)
	write("plan.json", `{"uses":["nested.json"],"known":true}`)
	write("nested.json", `{"known":true,"value":1}`)
	manifest, key, err := SourceManifest(context.Background(), []string{root, root}, resolveTestSource)
	if err != nil || len(manifest.Files) != 3 || len(manifest.Roots) != 1 {
		t.Fatalf("incomplete closure: %+v %v", manifest, err)
	}
	_, repeated, err := SourceManifest(context.Background(), []string{root}, resolveTestSource)
	if err != nil || repeated != key {
		t.Fatalf("unstable graph key: %s %s %v", key, repeated, err)
	}
	write("nested.json", `{"known":true,"value":2}`)
	_, changed, err := SourceManifest(context.Background(), []string{root}, resolveTestSource)
	if err != nil || changed == key {
		t.Fatalf("reused key after transitive mutation: %v", err)
	}
	// A parser that cannot enumerate an input must not turn job.Inputs into
	// a declaration of recursive completeness.
	write("plan.json", `{"uses":["nested.json"],"known":false}`)
	partial, partialKey, err := SourceManifest(context.Background(), []string{root}, resolveTestSource)
	if !errors.Is(err, ErrIncomplete) || partialKey != "" || len(partial.Files) != 0 {
		t.Fatalf("served partial closure: %+v %s %v", partial, partialKey, err)
	}
}

func TestWorkloadSourceRefusesCyclesMissingFilesAndUnknownRoots(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "root.json")
	if err := os.WriteFile(root, []byte(`{"uses":["alias.json"],"known":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(home, "alias.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SourceManifest(context.Background(), []string{root}, resolveTestSource); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("followed alias cycle: %v", err)
	}
	if _, _, err := SourceManifest(context.Background(), []string{filepath.Join(home, "missing.json")}, resolveTestSource); err == nil {
		t.Fatal("missing input was considered known")
	}
	if _, _, err := SourceManifest(context.Background(), []string{"relative.json"}, resolveTestSource); err == nil {
		t.Fatal("guessed root path")
	}
	if _, _, err := SourceManifest(context.Background(), []string{root}, nil); !errors.Is(err, ErrIncomplete) {
		t.Fatal("missing resolver accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := SourceManifest(ctx, []string{root}, resolveTestSource); !errors.Is(err, context.Canceled) {
		t.Fatalf("ignored cancellation: %v", err)
	}
}

func TestWorkloadSourceRefusesMutationDuringInspection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leaf")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	mutate := func(_ context.Context, path string, _ io.ReadSeeker) (Dependencies, error) {
		return Dependencies{Complete: true}, os.WriteFile(path, []byte("changed size"), 0600)
	}
	if _, key, err := SourceManifest(context.Background(), []string{path}, mutate); !errors.Is(err, ErrIncomplete) || key != "" {
		t.Fatalf("accepted changed source: %s %v", key, err)
	}
}

func TestWorkloadSourcePreservesLogicalSymlinkDependencyContext(t *testing.T) {
	home := t.TempDir()
	physical := filepath.Join(home, "physical")
	logical := filepath.Join(home, "logical")
	for _, dir := range []string{physical, logical} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	realRoot := filepath.Join(physical, "root.json")
	if err := os.WriteFile(realRoot, []byte(`{"uses":["leaf.json"],"known":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logical, "leaf.json"), []byte(`{"known":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(logical, "root.json")
	if err := os.Symlink(realRoot, alias); err != nil {
		t.Fatal(err)
	}
	manifest, _, err := SourceManifest(context.Background(), []string{alias}, resolveTestSource)
	if err != nil || len(manifest.Files) != 2 {
		t.Fatalf("lost logical path context: %+v %v", manifest, err)
	}
}

func TestWorkloadProbeKeysPreserveExactClocksAndRoles(t *testing.T) {
	probe := Probe{Shot: 0, Scene: "heard", Frames: []ProbeFrame{{Index: 17, At: 123456789, Roles: []string{"entry", "late"}}}}
	key, err := probe.Key()
	if err != nil {
		t.Fatal(err)
	}
	changed := probe
	changed.Frames = append([]ProbeFrame(nil), probe.Frames...)
	changed.Frames[0].At++
	other, err := changed.Key()
	if err != nil || other == key {
		t.Fatal("clock change did not invalidate probe")
	}
	changed.Frames[0] = probe.Frames[0]
	changed.Frames[0].Roles = []string{"entry"}
	other, err = changed.Key()
	if err != nil || other == key {
		t.Fatal("role change did not invalidate probe")
	}
	probe.Frames = append(probe.Frames, probe.Frames[0])
	if _, err := probe.Key(); err == nil {
		t.Fatal("duplicate frame index accepted")
	}
}
