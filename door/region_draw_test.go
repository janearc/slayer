//go:build draw

package door

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestRegionPNGMatchesTheWholePicture(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	env := envIn(t, context.Background(), directory)
	wholePath := filepath.Join(directory, "whole.png")
	partPath := filepath.Join(directory, "part.png")
	drew(t, env, path, "--out", wholePath)
	whole := pictureOf(t, wholePath)
	left, top := -1, -1
	for y := 0; y < whole.Bounds().Dy() && left < 0; y++ {
		for x := 0; x+1 < whole.Bounds().Dx(); x++ {
			if whole.RGBAAt(x, y) != whole.RGBAAt(x+1, y) {
				left, top = x, y
				break
			}
		}
	}
	if left < 0 {
		t.Fatal("whole picture has no neighbouring pixel difference")
	}
	drew(t, env, path, "--out", partPath, "--region", fmt.Sprintf("%d,%d,1,1", left, top))
	part := pictureOf(t, partPath)
	if part.Bounds().Dx() != 1 || part.Bounds().Dy() != 1 {
		t.Fatalf("part size %v", part.Bounds())
	}
	if got, want := part.RGBAAt(0, 0), whole.RGBAAt(left, top); got != want {
		t.Fatalf("region pixel %v != %v", got, want)
	}
	if part.RGBAAt(0, 0) == whole.RGBAAt(left+1, top) {
		t.Fatal("region also matches the shifted pixel")
	}
}

// Anti: an unsafe print is refused before its output appears.
func TestRegionPNGRefusesBloom(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, "print: {knee: 0.8, top: 4, bloom: 2}\n"))
	env := envIn(t, context.Background(), directory)
	got := running(Tool(env), "draw", path, "--region", "1,1,2,2")
	if got.code == nothing {
		t.Fatal("a bloom region was drawn")
	}
}
