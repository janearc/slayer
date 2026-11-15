package main_test

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/internal/commandtest"
)

// success is defined as the zero process exit status.
const success = 0

// failure is defined as the unsuccessful process exit status.
const failure = 1

// usage is defined by Pzm as the command-usage error status.
const usage = 2

//go:embed testdata/numbers.yaml
var chosenRaw []byte
var choices numbers.Numbers
var callLimit time.Duration
var executable, root string

func TestMain(tests *testing.M) {
	var err error
	choices, err = numbers.Read("testdata/numbers.yaml", chosenRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
	limit, err := choices.Get("command-seconds")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
	callLimit = time.Duration(limit.Value * float64(time.Second))
	root, err = filepath.Abs("../..")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
	directory, err := os.MkdirTemp("", "slayer-demo-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
	executable = filepath.Join(directory, "demo")
	if err := commandtest.Build(root, "./cmd/demo", executable, testing.CoverMode()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.RemoveAll(directory)
		os.Exit(failure)
	}
	code := tests.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

func call(arguments ...string) (string, int) {
	context, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	command := exec.CommandContext(context, executable, arguments...)
	command.Dir = root
	command.Env = append(os.Environ(), "SLAYER_MARBLE=")
	output, err := command.CombinedOutput()
	if err == nil {
		return string(output), success
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return string(output), exit.ExitCode()
	}
	return fmt.Sprint(err), -failure
}

func TestCatalogueNamesTheSmokesAndEveryShippedDemo(test *testing.T) {
	output, code := call("list")
	if code != success {
		test.Fatal(output)
	}
	for _, name := range []string{"orbital:", "swarm:", "ring:", "hillside:", "moonrise:", "house:", "camera:", "1:", "38:", "SLAYER_MARBLE"} {
		if !strings.Contains(output, name) {
			test.Fatalf("catalogue omitted %s", name)
		}
	}
}

func TestDryRunsAndRefusalsCreateNoOutput(test *testing.T) {
	for _, args := range [][]string{
		{"show", "1", "--dry-run"}, {"show", "orbital", "--dry-run"},
		{"show", "camera", "--dry-run"},
		{"show", "1", "--width", "3840", "--height", "2160", "--dry-run"},
		{"show"}, {"show", "1", "extra"},
		{"show", "1", "--width", "3841"}, {"show", "1", "--height", "0"},
		{"show", "1", "--height", "2161"}, {"show", "1", "--rays", "0"},
		{"show", "1", "--rays", "5"}, {"show", "1", "--width", "thing"},
		{"show", "no-such-demo"}, {"show", "99999"}, {"show", "1", "--width", "-1"},
		{"show", "1", "--cells", "0x0"},
		{"show", "house", "--width", "4"}, {"show", "38"},
	} {
		directory := filepath.Join(test.TempDir(), "output")
		args = append(args, "--out", directory)
		output, code := call(args...)
		dry := strings.Contains(strings.Join(args, " "), "--dry-run")
		if dry && code != success || !dry && code == success {
			test.Fatalf("%v: %d %s", args, code, output)
		}
		if _, err := os.Stat(directory); !os.IsNotExist(err) {
			test.Fatalf("%v created output", args)
		}
	}
	output, code := call("show", "1", "--out", "")
	if code == success || !strings.Contains(output, "out needs a directory") {
		test.Fatal("empty output directory was not refused", code, output)
	}
	output, code = call("show", "1", "--unknown")

	if code != usage || !strings.Contains(output, "unknown flag") {
		test.Fatal(code, output)
	}
}

func Example_list() {
	output, code := call("list")
	fmt.Println(code, strings.Contains(output, "orbital:"))
	// Output: 0 true
}

func Example_show() {
	output, code := call("show", "1", "--dry-run")
	fmt.Println(code, strings.Contains(output, "would draw smoke 1"))
	// Output: 0 true
}
