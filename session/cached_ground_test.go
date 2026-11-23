package session

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/terrain"
)

// fixtureModel is a copy of terrain's placed fixture in a test's folder.
func fixtureModel(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../terrain/testdata/fixture-plain.tif")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model.tif")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAPinsHashIsRememberedByItsFile(t *testing.T) {
	cachedHome(t)
	path := fixtureModel(t)
	want, err := diskcache.FileDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	cold, coldNote, err := rememberedHash(path)
	if err != nil || cold != want || coldNote.hit {
		t.Fatalf("cold %s, %v, %v", cold, coldNote, err)
	}
	warm, warmNote, err := rememberedHash(path)
	if err != nil || warm != want || !warmNote.hit ||
		!strings.Contains(warmNote.String(), "size and time") {
		t.Fatalf("warm %s, %v, %v", warm, warmNote, err)
	}
}

func TestAFileChangedInPlaceIsHashedAgain(t *testing.T) {
	cachedHome(t)
	path := fixtureModel(t)
	if _, _, err := rememberedHash(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	want, _ := diskcache.FileDigest(path)
	got, note, err := rememberedHash(path)
	if err != nil || note.hit || got != want {
		t.Fatalf("a changed file's old sha256 was served: %v, %v", note, err)
	}
}

func TestAFileThatIsNotThereIsRefused(t *testing.T) {
	cachedHome(t)
	gone := filepath.Join(t.TempDir(), "gone.tif")
	_, note, err := rememberedHash(gone)
	if err == nil || note.hit {
		t.Fatalf("a missing file was hashed: %v, %v", note, err)
	}
}

// fixtureCut is a window of the fixture: its first two points each way.
func fixtureCut(t *testing.T, path string) groundCut {
	t.Helper()
	raster, err := terrain.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	west, north, spacing := raster.Corner()
	return groundCut{west: west + spacing/2, north: north - spacing/2,
		step: spacing, across: 2, down: 2}
}

func TestTheGroundFromTheCacheIsTheGroundCut(t *testing.T) {
	cachedHome(t)
	path := fixtureModel(t)
	sum, _ := diskcache.FileDigest(path)
	cut := fixtureCut(t, path)
	fresh, err := cut.from(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, note, err := keptGround(path, sum, cut); err != nil || note.hit {
		t.Fatalf("cold: %v, %v", note, err)
	}
	warm, note, err := keptGround(path, sum, cut)
	if err != nil || !note.hit || !bytes.Equal(warm.Bytes(), fresh.Bytes()) {
		t.Fatalf("warm: %v, %v", note, err)
	}
}

func TestGroundOfAnotherModelOrWindowIsAMiss(t *testing.T) {
	cachedHome(t)
	path := fixtureModel(t)
	sum, _ := diskcache.FileDigest(path)
	cut := fixtureCut(t, path)
	if _, _, err := keptGround(path, sum, cut); err != nil {
		t.Fatal(err)
	}
	other := sum[:len(sum)-1] + "0"
	if other == sum {
		other = sum[:len(sum)-1] + "1"
	}
	smoother := cut
	smoother.smooth = cut.step
	for name, each := range map[string]struct {
		sum string
		cut groundCut
	}{"model": {other, cut}, "smoothing": {sum, smoother}} {
		_, note, err := keptGround(path, each.sum, each.cut)
		if err != nil || note.hit {
			t.Errorf("ground of another %s was served: %v, %v", name,
				note, err)
		}
	}
}

func TestGroundWithNoPinIsCutAndNotKept(t *testing.T) {
	cachedHome(t)
	path := fixtureModel(t)
	_, note, err := keptGround(path, "", fixtureCut(t, path))
	if err != nil || note.hit || note.keptErr == nil {
		t.Fatalf("ground with no pin: %v, %v", note, err)
	}
}

func TestGroundOfAModelThatIsNotThereIsRefused(t *testing.T) {
	cachedHome(t)
	path := fixtureModel(t)
	cut := fixtureCut(t, path)
	sum, _ := diskcache.FileDigest(path)
	gone := filepath.Join(t.TempDir(), "gone.tif")
	if _, _, err := keptGround(gone, sum, cut); err == nil {
		t.Fatal("a missing model was cut")
	}
}
