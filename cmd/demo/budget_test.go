package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewBudgetRefusesBeforeEvenADryRun(test *testing.T) {
	directory := filepath.Join(test.TempDir(), "output")
	output, code := call("show", "1", "--width", "3840", "--height", "2160",
		"--rays", "2", "--dry-run", "--out", directory)
	if code == success || !strings.Contains(output, "sample budget") {
		test.Fatal("oversized preview was not explained", code, output)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		test.Fatal("budget refusal created output", err)
	}
}

func TestUnknownSelectorsExplainTheRefusal(test *testing.T) {
	for _, sample := range []struct{ selector, reason string }{
		{"99999", "no smoke 99999"}, {"no-such-demo", "no demo"},
	} {
		output, code := call("show", sample.selector, "--dry-run")
		if code == success || !strings.Contains(output, sample.reason) {
			test.Fatal("selector refusal was not explained", code, output)
		}
	}
}
