package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runAtelier(args ...string) (int, string) {
	var out, errs bytes.Buffer
	code := command().RunWith(args, strings.NewReader(""), &out, &errs)
	return code, out.String() + errs.String()
}

func TestOpenDryRunAndArgumentRefusal(t *testing.T) {
	code, output := runAtelier("open", "--dry-run")
	if code != 0 || !strings.Contains(output, "no render submission") {
		t.Fatalf("open dry-run returned %d: %s", code, output)
	}
	code, output = runAtelier("open", "one.job.yaml", "two.job.yaml", "--dry-run")
	if code == 0 || !strings.Contains(output, "at most one job file") {
		t.Fatalf("open with extra arguments returned %d: %s", code, output)
	}
}

func TestProofDryRunRequiresOutputAndCreatesNoFiles(t *testing.T) {
	code, output := runAtelier("proof", "--dry-run")
	if code == 0 || !strings.Contains(output, "needs --out") {
		t.Fatalf("proof without output returned %d: %s", code, output)
	}
	path := filepath.Join(t.TempDir(), "proof.png")
	code, output = runAtelier("proof", "--out", path, "--dry-run")
	if code != 0 || !strings.Contains(output, "capture its canvas") {
		t.Fatalf("proof dry-run returned %d: %s", code, output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("proof dry-run output path stat error = %v; wanted no file", err)
	}
}

func TestDeclaredExamples(t *testing.T) {
	if err := command().Examples(os.DirFS(".")); err != nil {
		t.Fatal(err)
	}
}

func Example_open() {
	code, output := runAtelier("open", "--dry-run")
	fmt.Println(code, strings.TrimSpace(output))
	// Output: 0 open the Atelier interface; no render submission
}

func Example_proof() {
	directory, err := os.MkdirTemp("", "atelier-proof-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "proof.png")
	code, output := runAtelier("proof", "--out", path, "--dry-run")
	_, statErr := os.Stat(path)
	fmt.Println(code, strings.Contains(output, "capture its canvas"), os.IsNotExist(statErr))
	// Output: 0 true true
}
