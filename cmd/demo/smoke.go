package main

import (
	"image"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/trace"
)

// drawSmoke uses the actual milestone scene, camera and development. This is a
// preview, not smokecheck's terminal/reference comparison or a passing verdict.
func drawSmoke(scene smoke.Smoke, given preview) (*image.RGBA, error) {
	world, eye := scene.Scene()
	lens, err := film.PinholeOf(scene.Wide)
	if err != nil {
		return nil, err
	}
	frame, err := trace.Sampled(world, eye, *lens, nothing,
		given.width, given.height, max(given.rays, scene.Samples))
	if err != nil {
		return nil, err
	}
	picture := scene.Tone.Develop(frame)
	if scene.Gauge.Width > nothing && scene.Gauge.Height > nothing {
		picture = colour.Masked(picture, scene.Gauge.Aspect(), one)
	}
	return picture, nil
}
