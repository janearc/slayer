package job

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bufferedSet = `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 2]}
      orientation: {look_at: [1, 0, 0]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      first: {material: matte, middle: [10, 20, 3], size: [2, 4, 6], source: eye}
`

func TestStillFromFreezesBufferWithoutChangingDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "set.yaml")
	if err := os.WriteFile(path, []byte(bufferedSet), 0600); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(bufferedSet, "[2, 4, 6]", "[6, 4, 6]", 1)
	saved, err := StillFrom(path, []byte(bufferedSet))
	if err != nil {
		t.Fatal(err)
	}
	edited, err := StillFrom(path, []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	if saved.PictureKey() == edited.PictureKey() {
		t.Fatal("different buffered geometry shares picture key")
	}
	read, has := edited.ImportedSet()
	if !has {
		t.Fatal("buffer was not retained for staging")
	}
	world, has := read.World()
	if !has || world.Boxes()[0].Size().X != 6 {
		t.Fatal("buffer geometry was not retained")
	}
	if err := os.WriteFile(path, []byte("not a set"), 0600); err != nil {
		t.Fatal(err)
	}
	read, has = edited.ImportedSet()
	world, _ = read.World()
	if !has || world.Boxes()[0].Size().X != 6 {
		t.Fatal("saved file replaced retained buffer")
	}
}

func TestStillFromRefusesInvalidSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "set.yaml")
	if _, err := StillFrom(path, []byte("scene: [")); err == nil {
		t.Fatal("invalid set accepted")
	}
	if _, err := StillFrom("", []byte(bufferedSet)); err == nil {
		t.Fatal("empty path accepted")
	}
}
