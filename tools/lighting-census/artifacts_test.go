package census

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

func TestReadFramePreservesRawLightAndCoverageDepth(t *testing.T) {
	dir := t.TempDir()
	image, detail := filepath.Join(dir, "image.pfm"), filepath.Join(dir, "image.cd32")
	want := frame.New(2, 2)
	want.Pix[0] = frame.Pixel{R: 24.5, G: 0.125, B: 1.5, Cover: .25, Depth: 3}
	want.Pix[3] = frame.Pixel{R: 2, G: 3, B: 4, Cover: .75, Depth: 9}
	if err := raw.Write(image, want); err != nil {
		t.Fatal(err)
	}
	writeDetails(t, detail, want)
	got, err := ReadFrame(image, detail)
	if err != nil {
		t.Fatal(err)
	}
	if got.W != want.W || got.H != want.H {
		t.Fatalf("dimensions: got %dx%d", got.W, got.H)
	}
	for i := range want.Pix {
		if got.Pix[i] != want.Pix[i] {
			t.Fatalf("pixel %d: got %+v, want %+v", i, got.Pix[i], want.Pix[i])
		}
	}
}

func TestReadFrameRefusesMissingTruncatedAndNonfiniteArtifacts(t *testing.T) {
	dir := t.TempDir()
	image, detail := filepath.Join(dir, "image.pfm"), filepath.Join(dir, "image.cd32")
	imageBytes := append([]byte("PF\n1 1\n-1\n"), make([]byte, 12)...)
	if err := os.WriteFile(image, imageBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFrame(image, detail); err == nil {
		t.Fatal("missing coverage/depth was accepted")
	}
	if err := os.WriteFile(detail, []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFrame(image, detail); err == nil {
		t.Fatal("truncated coverage/depth was accepted")
	}
	data := make([]byte, 8)
	notANumber := float32(math.NaN())
	binary.LittleEndian.PutUint32(data, math.Float32bits(notANumber))
	if err := os.WriteFile(detail, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFrame(image, detail); err == nil {
		t.Fatal("nonfinite coverage was accepted")
	}
	binary.LittleEndian.PutUint32(data, math.Float32bits(0))
	if err := os.WriteFile(detail, data, 0o600); err != nil {
		t.Fatal(err)
	}
	infinite := float32(math.Inf(1))
	binary.LittleEndian.PutUint32(imageBytes[len(imageBytes)-4:], math.Float32bits(infinite))
	if err := os.WriteFile(image, imageBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFrame(image, detail); err == nil {
		t.Fatal("nonfinite radiance was accepted")
	}
}

func writeDetails(t *testing.T, path string, f *frame.Frame) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range f.Pix {
		if err := binary.Write(out, binary.LittleEndian, []float32{p.Cover, p.Depth}); err != nil {
			t.Fatal(err)
		}
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
