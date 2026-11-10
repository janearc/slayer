package main_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image/png"
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

// channels is defined by image.Color.RGBA as four returned components.
const channels = 4

// distinct is defined as the smallest count proving colour variation.
const distinct = 2

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

func TestSmokeAndNamedSceneProduceActualPNGs(test *testing.T) {
	for _, selector := range []string{"1", "swarm"} {
		directory := test.TempDir()
		output, code := call("show", selector, "--width", "16", "--height", "12",
			"--rays", "1", "--out", directory)
		if code != success {
			test.Fatal(output)
		}
		name := selector
		if selector == "1" {
			name = "smoke-1"
		}
		readPNG(test, filepath.Join(directory, name+".png"), picked(test, "width"), picked(test, "height"))
	}
}

func readPNG(test *testing.T, path string, width, height int) {
	test.Helper()
	file, err := os.Open(path)
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil || picture.Bounds().Dx() != width || picture.Bounds().Dy() != height {
		test.Fatalf("bad PNG %s: %v", path, err)
	}
	colours := map[[channels]uint32]bool{}
	for y := range height {
		for x := range width {
			red, green, blue, alpha := picture.At(x, y).RGBA()
			colours[[channels]uint32{red, green, blue, alpha}] = true
		}
	}
	if len(colours) < distinct {
		test.Fatal("render contains no visible scene variation")
	}
}

func TestTerminalPreviewAlsoKeepsThePNG(test *testing.T) {
	directory := test.TempDir()
	output, code := call("show", "1", "--width", "16", "--height", "12",
		"--terminal", "--cells", "8x4", "--out", directory)
	if code != success || !bytes.Contains([]byte(output), []byte("\x1b[38;2;")) {
		test.Fatal(code, output)
	}
	readPNG(test, filepath.Join(directory, "smoke-1.png"), picked(test, "width"), picked(test, "height"))
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

// picked reads one dimension of the deliberately tiny command fixture.
func picked(test *testing.T, name string) int {
	test.Helper()
	value, err := choices.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return int(value.Value)
}
