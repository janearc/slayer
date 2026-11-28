package placement

import (
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// LightLimit bounds snapshot size and overlay drawing for dense lamp fields.
const LightLimit = 64

// DisplayLight describes a read-only marker, with a staged ordinal when
// authoring identity is unavailable. Distant sources have no position.
type DisplayLight struct {
	Kind    string
	Ordinal int
	Author  thing.AuthorRef
	At, Way maths.Vec
}

// Valid reports whether a marker has the finite data its kind needs.
func (light DisplayLight) Valid() bool {
	if light.Ordinal <= 0 || (light.Author.Name == "") != (light.Author.Collection == "") {
		return false
	}
	at, way := light.At, light.Way
	if !finiteLightVector(at) || !finiteLightVector(way) {
		return false
	}
	switch light.Kind {
	case "Point":
		return way == (maths.Vec{})
	case "Spot":
		return way != (maths.Vec{})
	case "Sun", "Moon", "Distant":
		return at == (maths.Vec{}) && way != (maths.Vec{})
	}
	return false
}

func finiteLightVector(v maths.Vec) bool {
	return !math.IsInf(v.X, 0) && !math.IsNaN(v.X) &&
		!math.IsInf(v.Y, 0) && !math.IsNaN(v.Y) &&
		!math.IsInf(v.Z, 0) && !math.IsNaN(v.Z)
}

// WorkingLights prefers authored lights and direction-only sources when a
// world has too many lamps to label. It reads the staged emitter types.
func WorkingLights(world *thing.World) []DisplayLight {
	var lights, others []DisplayLight
	for i, lamp := range world.Lamps {
		author := world.LampAuthor(i)
		light := describeLight(lamp, author)
		light.Ordinal, light.Author = i+1, author
		if !light.Valid() {
			continue
		}
		if light.preferred() {
			lights = append(lights, light)
			if len(lights) == LightLimit {
				return lights
			}
		} else if len(others) < LightLimit {
			others = append(others, light)
		}
	}
	return append(lights, others[:min(len(others), LightLimit-len(lights))]...)
}

func (light DisplayLight) preferred() bool {
	return light.Author.Name != "" || light.Kind == "Sun" || light.Kind == "Moon" || light.Kind == "Distant"
}

func describeLight(lamp thing.Lamp, author thing.AuthorRef) DisplayLight {
	switch source := lamp.(type) {
	case emit.Lit:
		if source.Width > 0 {
			return DisplayLight{Kind: "Spot", At: source.At, Way: source.Toward}
		}
		return DisplayLight{Kind: "Point", At: source.At}
	case film.Spot:
		return DisplayLight{Kind: "Spot", At: source.At, Way: source.Aim}
	case film.Sun:
		return DisplayLight{Kind: "Sun", Way: source.Toward}
	case emit.Distant:
		toward, _, _ := source.Light(maths.Vec{}, maths.Vec{})
		if author.Collection == "scene" && author.Name == "sun" {
			return DisplayLight{Kind: "Sun", Way: toward}
		}
		if author.Collection == "scene" && author.Name == "moon" {
			return DisplayLight{Kind: "Moon", Way: toward}
		}
		return DisplayLight{Kind: "Distant", Way: toward}
	}
	return DisplayLight{}
}

// addSpaceLight retains a scene-level direction alongside local world lamps.
func (scene *DisplayScene) addSpaceLight(read set.Set, worldLamps int) {
	worldOnly := stage.WorldOnly(read)
	lamp := spaceLight(read, worldOnly)
	if lamp.Kind == "" {
		return
	}
	lamp.Ordinal = 1
	if worldOnly {
		lamp.Ordinal += worldLamps
	}
	if !lamp.Valid() {
		return
	}
	insert := 0
	if worldOnly {
		// The producer appends the moon after the authored world. Keep its
		// spot among preferred sources before the anonymous local lamps.
		for insert < len(scene.Lights) && scene.Lights[insert].preferred() {
			insert++
		}
		if insert == LightLimit {
			return
		}
	} else {
		for i := range scene.Lights {
			scene.Lights[i].Ordinal++
		}
	}
	scene.Lights = slices.Insert(scene.Lights, insert, lamp)
	scene.Lights = scene.Lights[:min(len(scene.Lights), LightLimit)]
}

func spaceLight(read set.Set, worldOnly bool) DisplayLight {
	if !worldOnly {
		if _, way, ok := stage.SpaceSun(read); ok {
			return DisplayLight{Kind: "Sun", Way: way.Unit(), Author: thing.AuthorRef{Collection: "scene", Name: "sun"}}
		}
	}
	if read.Light() == "moon" {
		return spaceMoon(read, worldOnly)
	}
	return DisplayLight{}
}

// spaceMoon checks the same direct source as the producer. Disc, twilight
// and world-only sky or ocean checks stay with full staging.
func spaceMoon(read set.Set, worldOnly bool) DisplayLight {
	moon, has := read.Moonlit()
	_, twilight := read.Sunlit()
	if !has || moon.Disc() || twilight {
		return DisplayLight{}
	}
	if worldOnly {
		world, _ := read.World()
		_, sky := world.Sky()
		_, ocean := world.OceanSurface()
		if sky || ocean {
			return DisplayLight{}
		}
	}
	way, rise, above := stage.Moonward(moon)
	if !above {
		return DisplayLight{}
	}
	film, _ := read.Film()
	lamp, err := stage.DirectMoon(way, measure.Metres(0), rise, film.Exposure())
	if err != nil {
		return DisplayLight{}
	}
	author := thing.AuthorRef{Collection: "scene", Name: "moon"}
	marker := describeLight(lamp, author)
	marker.Author = author
	return marker
}

// LightProjection is a marker's normalized screen position and the
// screen-space direction of a spot's aim, with y increasing downwards.
type LightProjection struct {
	X, Y, AimX, AimY float64
}

// ProjectLight projects positions or directions through the working camera.
func (scene *DisplayScene) ProjectLight(light DisplayLight) (LightProjection, bool) {
	point := scene.Eye.In(light.At)
	way := scene.Eye.Orientation.In(light.Way)
	if light.Kind == "Sun" || light.Kind == "Moon" || light.Kind == "Distant" {
		point = way
	} else if depth := -point.Z; depth < DisplayNear || depth > DisplayFar {
		return LightProjection{}, false
	}
	if !finiteLightVector(point) || point.Z >= 0 {
		return LightProjection{}, false
	}
	x, y := scene.project(point)
	if !(x >= 0 && x <= 1 && y >= 0 && y <= 1) {
		return LightProjection{}, false
	}
	projected := LightProjection{X: x, Y: y}
	if light.Kind == "Spot" {
		// Perspective derivative at the lamp, so arrow length need not
		// imply a beam reach or cross the near plane.
		depth := -point.Z
		projected.AimX = way.X + (point.X/depth)*way.Z
		projected.AimY = -way.Y - (point.Y/depth)*way.Z
	}
	return projected, true
}
