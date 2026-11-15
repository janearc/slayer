//go:build draw

package weather

import (
	"slices"
	"testing"
)

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
