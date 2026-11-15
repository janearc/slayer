//go:build draw

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// sized is the flags that ask a command for the small picture: slayer's
// render takes its size by puzzlemouth's long names and refuses the
// short ones, and azazel's still takes hillside's short ones.
func sized(command string) []string {
	if command == slayer {
		return []string{"--width", "48", "--height", "27", "--rays", "1"}
	}
	return []string{"-w", "48", "-h", "27", "-n", "1"}
}

// drawn is the picture a command writes for a render of the widow set at
// a small size, and the number it exited with.
func drawn(t *testing.T, command string) ([]byte, int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "picture.png")
	args := append([]string{"render", widow}, sized(command)...)
	_, code := run(t, command, append(args, "-out", out)...)
	picture, _ := os.ReadFile(out)
	return picture, code
}

// azazel's render and slayer's render write the same picture, byte for
// byte, for the same set and settings.
func TestAzazelRenderDrawsAsSlayerRenderDoes(t *testing.T) {
	got, code := drawn(t, azazel)
	want, wanted := drawn(t, slayer)
	if code != nothing || wanted != nothing || len(got) == nothing {
		t.Fatalf("azazel exited %d and slayer %d", code, wanted)
	}
	if !bytes.Equal(got, want) {
		t.Error("azazel and slayer wrote different pictures")
	}
}
