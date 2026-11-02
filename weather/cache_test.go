package weather

import (
	"math"
	"os"
	"slices"
	"testing"
)

// small is a small, quick weather for the cache's tests.
func small(t *testing.T, seed uint64) *Weather {
	t.Helper()
	grid, err := NewGrid(32, 16, 70*math.Pi/180, 6e7)
	if err != nil {
		t.Fatal(err)
	}
	stirring, err := NewStirring(4e-17, 6e6, 3e-5)
	if err != nil {
		t.Fatal(err)
	}
	made, err := New(grid, 18000, 3e6, 2e6, stirring, seed)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// weather reached through the cache, once from nothing and once from a
// checkpoint left halfway, is the weather run straight through, to the
// bit: the checkpoint keeps the storms' draws as well as the air.
func TestTheCacheResumesToTheBit(t *testing.T) {
	dir := t.TempDir()
	straight := small(t, 5)
	straight.Run(4e5)
	first := small(t, 5)
	if _, err := first.Reach(dir, 2e5, 1e5, false); err != nil {
		t.Fatal(err)
	}
	second := small(t, 5)
	steps, err := second.Reach(dir, 4e5, 1e5, false)
	if err != nil {
		t.Fatal(err)
	}
	if steps == 0 {
		t.Fatal("took no steps past the checkpoint")
	}
	if second.clock != straight.clock ||
		!slices.Equal(second.vorticity, straight.vorticity) ||
		!slices.Equal(second.origin, straight.origin) {
		t.Error("the resumed weather is not the weather run straight")
	}
}

// a second reach of a moment the cache holds takes no step at all.
func TestAReachedMomentIsReadNotRun(t *testing.T) {
	dir := t.TempDir()
	if _, err := small(t, 5).Reach(dir, 2e5, 1e5, false); err != nil {
		t.Fatal(err)
	}
	steps, err := small(t, 5).Reach(dir, 2e5, 1e5, false)
	if err != nil {
		t.Fatal(err)
	}
	if steps != 0 {
		t.Errorf("%d steps for a moment the cache holds", steps)
	}
}

// another seed is other weather, kept apart in the cache.
func TestOtherWeatherHasItsOwnShelf(t *testing.T) {
	dir := t.TempDir()
	if _, err := small(t, 5).Reach(dir, 1e5, 1e5, false); err != nil {
		t.Fatal(err)
	}
	if _, err := small(t, 6).Reach(dir, 1e5, 1e5, false); err != nil {
		t.Fatal(err)
	}
	shelves, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(shelves) != 2 {
		t.Errorf("%d shelves for two weathers", len(shelves))
	}
	if small(t, 5).key(false) == small(t, 5).key(true) {
		t.Error("the gpu's weather shares the cpu's key")
	}
}
