package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(args ...string) (int, string) {
	var out bytes.Buffer
	code := tool().Run(args, &out, &out)
	return code, out.String()
}

func Example_compare() {
	tool().Run([]string{"compare", "oracle.png", "candidate.png", "--dry-run"}, &bytes.Buffer{}, &bytes.Buffer{})
}

func pngFixture(t *testing.T, path string, value uint8) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: value, G: value, B: value, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPNGComparison(t *testing.T) {
	dir := t.TempDir()
	oracle, candidate, diff := filepath.Join(dir, "oracle.png"), filepath.Join(dir, "candidate.png"), filepath.Join(dir, "difference.png")
	pngFixture(t, oracle, 20)
	pngFixture(t, candidate, 21)
	code, out := run("compare", oracle, candidate, "--check", "--difference", diff)
	if code != 0 || !strings.Contains(out, "max difference RGB: [1 1 1]") || !strings.Contains(out, "1: 3\n") || !strings.Contains(out, "255: 0\n") {
		t.Fatalf("code %d: %s", code, out)
	}
	file, err := os.Open(diff)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	red, _, _, _ := img.At(0, 0).RGBA()
	if red != 65535 {
		t.Fatalf("difference not contrast stretched: %d", red)
	}
	before, err := os.ReadFile(diff)
	if err != nil {
		t.Fatal(err)
	}
	if code, out := run("compare", oracle, candidate, "--difference", diff); code == 0 {
		t.Fatalf("overwrote destination: %s", out)
	}
	after, err := os.ReadFile(diff)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing destination changed", err)
	}
	pngFixture(t, candidate, 255)
	if code, out := run("compare", oracle, candidate, "--check"); code == 0 || !strings.Contains(out, "check failed") || !strings.Contains(out, "PSNR:") {
		t.Fatalf("bad check: %d %s", code, out)
	}
	if code, out := run("compare", oracle, candidate); code != 0 {
		t.Fatalf("unchecked comparison failed: %d %s", code, out)
	}
}

func TestDryRunAndValidation(t *testing.T) {
	diff := filepath.Join(t.TempDir(), "difference.png")
	if code, out := run("compare", "missing.png", "also-missing.png", "--difference", diff, "--dry-run"); code != 0 || !strings.Contains(out, `would compare "missing.png" against "also-missing.png"`) {
		t.Fatalf("dry run: %d %s", code, out)
	}
	if _, err := os.Stat(diff); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote output: %v", err)
	}
	for _, args := range [][]string{
		{"compare"},
		{"compare", "a.png"},
		{"compare", "a.png", "b.pfm"},
		{"compare", "a.jpg", "b.jpg"},
		{"compare", "a.pfm", "b.pfm", "--check"},
		{"compare", "a.png", "b.png", "--diff", "x"},
		{"compare", "a.png", "b.png"},
	} {
		if code, out := run(args...); code == 0 {
			t.Errorf("accepted %v: %s", args, out)
		}
	}
}

func TestPFMComparison(t *testing.T) {
	dir := t.TempDir()
	oracle, candidate := filepath.Join(dir, "oracle.pfm"), filepath.Join(dir, "candidate.pfm")
	for _, path := range []string{oracle, candidate} {
		var data bytes.Buffer
		data.WriteString("PF\n1 1\n-1\n")
		if err := binary.Write(&data, binary.LittleEndian, [3]float32{1, 2, 3}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	diff := filepath.Join(dir, "diff.png")
	code, out := run("compare", oracle, candidate, "--difference", diff)
	if code != 0 || !strings.Contains(out, "max difference RGB: [0 0 0]") || !strings.Contains(out, "max at RGB:") || !strings.Contains(out, "0: 3\n") || !strings.Contains(out, "PSNR: +Inf dB") {
		t.Fatalf("PFM: %d %s", code, out)
	}
	if _, err := os.Stat(diff); err != nil {
		t.Fatal(err)
	}
}
