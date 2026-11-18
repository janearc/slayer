package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandPrintsMissingReferenceAsData(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"version":1,"identity":{"executable_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"options":{"scene_selection":"all","mode_selection":"all","width":1,"height":1,"times_ns":[0]},"cases":[{"scene":1,"samples_per_side":1,"mode":{"name":"cpu","fidelity":"reference","definition":"CPU visible-light reference"},"status":"not-run"}]}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var errs bytes.Buffer
	if code := tool().Run([]string{"read", "--root", dir, "--reference", "cpu"}, &output, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	if !strings.Contains(output.String(), `"status": "missing"`) {
		t.Fatalf("missing case disappeared: %s", output.String())
	}
}

func TestCommandRefusesAnApproximateReference(t *testing.T) {
	if code := tool().Run([]string{"read", "--root", t.TempDir(), "--reference", "gpu-fast-lighting"}, &bytes.Buffer{}, &bytes.Buffer{}); code == 0 {
		t.Fatal("accepted approximate lighting as a reference")
	}
}

func TestCommandRefusesMissingRootAndUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"read"}, {"read", "--root", t.TempDir(), "extra"}, {"read", "--unknown"}} {
		if code := tool().Run(args, &bytes.Buffer{}, &bytes.Buffer{}); code == 0 {
			t.Fatalf("accepted invalid arguments %q", args)
		}
	}
}
