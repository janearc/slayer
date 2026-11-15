//go:build draw

package named_test

import (
	"testing"
)

// The moonrise builds with another tree and draws at a tiny size, the
// moon lit in the middle.
func TestTheMoonriseBuildsAndDraws(t *testing.T) {
	scene := mustGet(t, "moonrise")
	kinds := mustParameter(t, scene, "kind").Choices()
	built := mustBuild(t, "moonrise", map[string]any{"seed": few,
		"kind": kinds[none]})
	if built.Timed() {
		t.Error("the moonrise keeps time, and has no lamp that does")
	}
	drawsTiny(t, "moonrise", built)
}
