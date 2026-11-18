package capture

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesDecodableImageAndKeepsExistingFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.png")
	if err := Write(path, image.NewRGBA(image.Rect(0, 0, 8, 6)), map[string]any{"native_window": false}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(file)
	file.Close()
	if err != nil || picture.Bounds().Dx() != 8 || picture.Bounds().Dy() != 6 {
		t.Fatalf("capture dimensions: %v", err)
	}
	before, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, image.NewRGBA(image.Rect(0, 0, 1, 1)), "replacement"); err == nil {
		t.Fatal("existing capture replaced")
	}
	after, err := os.ReadFile(path + ".json")
	if err != nil || string(before) != string(after) {
		t.Fatal("existing receipt changed")
	}
}

func TestWriteRefusesExistingReceiptBeforeCreatingImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.png")
	if err := os.WriteFile(path+".json", []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, image.NewRGBA(image.Rect(0, 0, 1, 1)), "replacement"); err == nil {
		t.Fatal("existing receipt accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("image created beside unrelated receipt")
	}
	before, err := os.ReadFile(path + ".json")
	if err != nil || string(before) != "keep me" {
		t.Fatal("unrelated receipt changed")
	}
}

func TestExistingImageWithoutReceiptIsNeverReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.png")
	if err := os.WriteFile(path, []byte("original image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, image.NewRGBA(image.Rect(0, 0, 1, 1)), "new receipt"); err == nil {
		t.Fatal("existing image accepted")
	}
	before, err := os.ReadFile(path)
	if err != nil || string(before) != "original image bytes" {
		t.Fatal("existing image changed")
	}
	if _, err := os.Stat(path + ".json"); !os.IsNotExist(err) {
		t.Fatal("failed capture left an orphan receipt")
	}
}
