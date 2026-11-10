package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// weatherOutput bounds diagnostics from a broken progress loop.
type weatherOutput struct{ bytes []byte }

func (w *weatherOutput) Write(p []byte) (int, error) {
	n := len(p)
	if left := 2048 - len(w.bytes); left > 0 {
		w.bytes = append(w.bytes, p[:min(left, n)]...)
	}
	return n, nil
}

func weatherCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, coverageCommand, args...)
	output := new(weatherOutput)
	cmd.Stdout, cmd.Stderr = output, output
	err := cmd.Run()
	if ctx.Err() != nil {
		return string(output.bytes), ctx.Err()
	}
	return string(output.bytes), err
}

func tinyWeather(out string) []string {
	return []string{"-across", "8", "-rows", "4", "-seconds", "3600", "-every", "3600", "-out", out}
}

func TestSmallLegalGridFinishesAndWritesThreeMaps(t *testing.T) {
	dir := t.TempDir()
	out, err := weatherCommand(t, tinyWeather(dir)...)
	if err != nil || !strings.Contains(out, "mean east wind") {
		t.Fatalf("weather failed: %v: %q", err, out)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 3 {
		t.Fatalf("three maps: %v %v", entries, err)
	}
	for _, name := range []string{"spun-00003600.png", "bands-00003600.png", "aged-00003600.png"} {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil || img.Bounds() != image.Rect(0, 0, 8, 4) {
			t.Fatalf("%s: %v %v", name, img, err)
		}
	}
}

func TestInvalidTimesFailBeforeCreatingOutputs(t *testing.T) {
	for _, test := range []struct{ flag, value string }{{"-every", "0"}, {"-every", "-1"}, {"-every", "NaN"}, {"-every", "+Inf"}, {"-seconds", "-1"}, {"-seconds", "NaN"}, {"-seconds", "+Inf"}} {
		t.Run(test.flag+test.value, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "absent")
			args := append(tinyWeather(dir), test.flag, test.value)
			out, err := weatherCommand(t, args...)
			if err == nil || err == context.DeadlineExceeded || !strings.Contains(out, "weather: invalid time") {
				t.Fatalf("bad time: %v: %q", err, out)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("invalid time created output: %v", err)
			}
		})
	}
}

// Public filesystem errors must reach the caller, including a blocked frame.
func TestWeatherReportsInputsAndOutputFailures(t *testing.T) {
	for _, test := range []struct{ name, flag, value, want string }{
		{"grid", "-rows", "3", "grid"}, {"storms", "-rate", "0", "storms"},
		{"day", "-day", "0", "day"},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := append(tinyWeather(t.TempDir()), test.flag, test.value)
			out, err := weatherCommand(t, args...)
			if err == nil || !strings.Contains(out, test.want) {
				t.Fatalf("%v %q", err, out)
			}
		})
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := weatherCommand(t, tinyWeather(filepath.Join(file, "maps"))...); err == nil || !strings.Contains(out, "not a directory") {
		t.Fatalf("%v %q", err, out)
	}
	dir := t.TempDir()
	for _, prefix := range []string{"spun-", "bands-", "aged-"} {
		if err := os.Mkdir(filepath.Join(dir, prefix+"00003600.png"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := weatherCommand(t, tinyWeather(dir)...); err == nil || !strings.Contains(out, "is a directory") {
		t.Fatalf("%v %q", err, out)
	}
}

func TestWeatherCacheAndGPUReachTheSameOutputClock(t *testing.T) {
	for _, gpu := range []bool{false, true} {
		for _, cached := range []bool{false, true} {
			t.Run(fmt.Sprintf("gpu%t-cache%t", gpu, cached), func(t *testing.T) {
				dir := t.TempDir()
				args := tinyWeather(dir)
				if gpu {
					args = append(args, "-gpu")
				}
				if cached {
					args = append(args, "-cache", t.TempDir())
				}
				out, err := weatherCommand(t, args...)
				if err != nil || !strings.Contains(out, "3600 s:") {
					t.Fatalf("%v %q", err, out)
				}
				if _, err := os.Stat(filepath.Join(dir, "spun-00003600.png")); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
