package smokecache

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRecordNamesTheActualRuntimeAndSourceTree(t *testing.T) {
	dir, err := slayerDir()
	if err != nil || !isSlayer(dir) {
		t.Fatalf("%q %v", dir, err)
	}
	record := RecordText()
	for _, want := range []string{"drawn ", "go " + runtime.Version(), "slayer ", "libtheme "} {
		if !strings.Contains(record, want) {
			t.Fatalf("missing %q in %q", want, record)
		}
	}
	if strings.Contains(record, "trees unknown:") {
		t.Fatalf("failed to find this source tree: %q", record)
	}
	_, source, found := strings.Cut(record, "\nslayer ")
	if !found {
		t.Fatalf("missing Slayer provenance: %q", record)
	}
	source, _, _ = strings.Cut(source, "\n")
	// Mutation copies keep go.mod, but do not carry checkout history.
	// Ask Git independently; an available revision must still be exact.
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = dir
	head, lookup := command.Output()
	if lookup != nil {
		if !strings.HasPrefix(source, "unknown: ") {
			t.Fatalf("unavailable history reported as %q", source)
		}
	} else {
		want := strings.TrimSpace(string(head))
		if source != want && !strings.HasPrefix(source, want+", ") {
			t.Fatalf("source %q, want revision %s", source, want)
		}
	}
	if isSlayer(t.TempDir()) {
		t.Fatal("empty folder is a Slayer module")
	}
}

func TestCacheErrorsNeverServeAnIncompleteEntry(t *testing.T) {
	k := key()
	held := filepath.Join(t.TempDir(), "held")
	if err := os.WriteFile(held, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Store(held, k, "", false, fill("x")); err == nil {
		t.Fatal("blocked root")
	}
	sentinel := errors.New("render failed")
	root := t.TempDir()
	if _, err := Store(root, k, "", false, func(string) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if _, ok := Open(root, k); ok {
		t.Fatal("failed render was cached")
	}
	for _, block := range []string{keyFile, Record} {
		_, err := Store(root, k, "", false, func(dir string) error {
			if err := fill("x")(dir); err != nil {
				return err
			}
			return os.Mkdir(filepath.Join(dir, block), 0700)
		})
		if err == nil {
			t.Fatal("blocked metadata write")
		}
	}
	if _, err := Store(root, k, "", false, func(string) error { return nil }); err == nil {
		t.Fatal("missing pictures accepted")
	}
	if _, err := Hold(filepath.Join(root, "absent"), Text); err == nil {
		t.Fatal("missing entry")
	}
	e, err := Store(root, k, "", false, fill("x"))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if _, err := e.Bytes("unknown"); err == nil {
		t.Fatal("unknown entry file")
	}
}

func TestInputErrorsRetainTheirSourceAndEscapeIsRefused(t *testing.T) {
	sentinel := errors.New("survey unavailable")
	if _, err := Inputs([]string{"survey"}, func(string) ([]string, error) { return nil, sentinel }); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("SLAYER_TEST_INPUT", dir)
	outside := filepath.Join(t.TempDir(), "outside")
	if _, err := Inputs([]string{"$SLAYER_TEST_INPUT"}, func(string) ([]string, error) { return []string{outside}, nil }); err == nil {
		t.Fatal("input escaped root")
	}
	if _, err := Inputs([]string{"missing"}, func(string) ([]string, error) { return []string{outside}, nil }); err == nil {
		t.Fatal("missing input hashed")
	}
	if _, _, err := measure(dir); err == nil {
		t.Fatal("directory was hashed as a regular file")
	}
	if got, err := Inputs([]string{"empty"}, func(string) ([]string, error) { return nil, nil }); err != nil || got != "empty nothing" {
		t.Fatalf("%q %v", got, err)
	}
}
