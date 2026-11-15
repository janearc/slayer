package weather

import (
	"math"
	"os"
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

// still air reached at nothing, which takes no step, is reached again
// from the cache: its note names a checkpoint that was written.
func TestStillAirAtNothingIsReachedAgain(t *testing.T) {
	dir := t.TempDir()
	for range 2 {
		if _, err := small(t, 5).Reach(dir, 0, 1e5, false); err != nil {
			t.Fatal(err)
		}
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
