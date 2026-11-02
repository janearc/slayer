// Package torture holds slayer's tracer to laws that hold forevery scene, over
// scenes made at random from the harness's own parts: no NaN, no negative
// light, no light made from nothing, the same frame every time, and the same
// picture at any scale.
package torture_test

import (
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/burning-trousers/scenes"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/trace"
)

// arranged is the scene a seed makes, lit or by its sky alone.
func arranged(seed uint64, lit bool) scenes.Scene {
	return scenes.Arranged(seed, lit)
}

// drawn is a scene drawn at a scale, a pixel for each look.
func drawn(t *testing.T, made scenes.Scene, by float64) []frame.Pixel {
	t.Helper()
	rays := made.Rays(by)
	picture, err := trace.Render(made.World(by), maths.Subspace{},
		fake.NewAimed(rays...), 0, len(rays), 1)
	if err != nil {
		t.Fatal(err)
	}
	return picture.Pix
}
