package file

import (
	"crypto/sha256"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestSequenceDigestDescribesCommittedBytesAndNeverAFailedFrame(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 8, 8))
	picture.SetRGBA(3, 5, color.RGBA{R: 200, G: 41, A: 255})
	path := filepath.Join(t.TempDir(), "frame.png")
	got, err := SequencePNGHash(path, picture)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != sha256.Sum256(raw) {
		t.Fatal("digest is not the committed PNG")
	}
	// The expected digest is independent of later file tampering. The
	// resume reader can detect it instead of recording corrupted bytes.
	if err := os.WriteFile(path, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if got == sha256.Sum256([]byte("tampered")) {
		t.Fatal("expected digest followed external bytes")
	}
	for _, test := range []struct {
		path  string
		image *image.RGBA
	}{
		{path, nil}, {path, image.NewRGBA(image.Rect(0, 0, 0, 0))},
		{filepath.Join(t.TempDir(), "missing", "frame.png"), picture},
		{t.TempDir(), picture},
	} {
		sum, err := SequencePNGHash(test.path, test.image)
		if err == nil || sum != ([sha256.Size]byte{}) {
			t.Fatalf("failed frame yielded a digest: %x %v", sum, err)
		}
	}
	raw, err = os.ReadFile(path)
	if err != nil || string(raw) != "tampered" {
		t.Fatal("encoding refusal replaced the previous file", err)
	}
}
