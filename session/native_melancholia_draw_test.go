//go:build draw && darwin

package session

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Melancholia, whose set says renderer: gpu-native, drawn at 1024 wide
// and four to a side draws to the end. A shadow ray from the far planet,
// 2.4e8 m off and as long, once never came back from the hardware query,
// at the band of rows 896 to 959; the wait now gives up after a minute
// with an error, so a hang shows here as a failure. It needs the shelf
// and takes about a minute and five gigabytes, so it skips with the other
// heavy set tests, without SLAYER_SHELF or in short mode.
func TestMelancholiaDrawnNativeFinishes(t *testing.T) {
	if os.Getenv("SLAYER_SHELF") == "" || testing.Short() {
		t.Skip("no shelf named by SLAYER_SHELF, or short")
	}
	requireCommandDevice(t)
	authored, err := os.ReadFile("../sets/melancholia.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authored), "\n  renderer: gpu-native\n") {
		t.Fatal("melancholia no longer says renderer: gpu-native, which " +
			"is the path this test draws")
	}
	path := filepath.Join("..", "sets", "melancholia.yaml")
	var said, wrong bytes.Buffer
	output := filepath.Join(t.TempDir(), "melancholia.png")
	args := []string{"-set", path, "-gpu", "-w", "1024", "-n", "4", "-out", output}
	if code := run(args, &said, &wrong); code != 0 {
		t.Fatalf("melancholia drawn native: exit %d: %s", code, wrong.String())
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("melancholia drawn native left no picture: %v", err)
	}
}
