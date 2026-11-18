package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCapture(args ...string) (int, string) {
	var out, errs bytes.Buffer
	code := command().RunWith(args, strings.NewReader(""), &out, &errs)
	return code, out.String() + errs.String()
}

func TestSnapshotDryRunAndRefusalsCreateNoFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.png")
	base := []string{"snapshot", "--out", path, "--dry-run"}
	code, output := runCapture(base...)
	if code != 0 || !strings.Contains(output, `"native_window":false`) {
		t.Fatalf("snapshot dry-run returned %d: %s", code, output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("snapshot dry-run output path stat error = %v; wanted no file", err)
	}

	for _, invalid := range []struct {
		name  string
		flag  string
		value string
	}{
		{"study", "--study", "mars"},
		{"area", "--area", "orbit"},
		{"size", "--size", "huge"},
		{"pane", "--pane", "audio"},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			args := append(append([]string(nil), base...), invalid.flag, invalid.value)
			code, output := runCapture(args...)
			if code == 0 {
				t.Fatalf("invalid %s accepted: %s", invalid.name, output)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("invalid %s created output: %v", invalid.name, err)
			}
		})
	}

	code, output = runCapture("snapshot", "--dry-run")
	if code == 0 || !strings.Contains(output, "needs --out") {
		t.Fatalf("snapshot without output returned %d: %s", code, output)
	}
}

func TestDeclaredExamples(t *testing.T) {
	if err := command().Examples(os.DirFS(".")); err != nil {
		t.Fatal(err)
	}
}

func Example_snapshot() {
	directory, err := os.MkdirTemp("", "atelier-snapshot-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "snapshot.png")
	code, output := runCapture("snapshot", "--out", path, "--dry-run")
	_, statErr := os.Stat(path)
	fmt.Println(code, strings.Contains(output, `"native_window":false`), os.IsNotExist(statErr))
	// Output: 0 true true
}
