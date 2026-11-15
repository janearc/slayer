//go:build draw

package weather

import (
	"testing"
)

func TestGPUCacheReusesTheActualStoredWeather(t *testing.T) {
	dir := t.TempDir()
	first := small(t, 5)
	if _, err := first.Reach(dir, 3600, 3600, true); err != nil {
		t.Fatal(err)
	}
	second := small(t, 5)
	if n, err := second.Reach(dir, 3600, 3600, true); err != nil || n != 0 {
		t.Fatalf("%d %v", n, err)
	}
	if second.Clock() != first.Clock() {
		t.Fatal("cached clock changed")
	}
	for k, v := range first.vorticity {
		if second.vorticity[k] != v {
			t.Fatal("cached GPU vorticity changed")
		}
	}
}
