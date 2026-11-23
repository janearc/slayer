package weathering

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// Jointed is a texture that knows which of its points are mortar: brick
// is, and weathering lodges more of some agents in the joints.
type Jointed interface {
	InJoint(point maths.Vec) bool
}

// Weathered is a surface with a layer of agents over it. It paints what
// its base paints, and then what each agent lays where its rule lets it.
// It is read by the CPU tracer through Facing, which hands it the way the
// surface faces; asked by At alone, with no way, it can only paint its
// base.
type Weathered struct {
	// Base is what the surface looked like before. Nil is Colour, flat.
	Base   texture.Texture
	Colour maths.Light
	// Name is the surface's material.
	Name string
	// Layer is the agents, shared by every surface they weather.
	Layer *Layer
}

// compile-time check that Weathered is read by its way.
var _ texture.Oriented = Weathered{}

// At is the surface's base at a point in its own space.
func (weathered Weathered) At(point maths.Vec) maths.Light {
	if weathered.Base == nil {
		return weathered.Colour
	}
	return weathered.Base.At(point)
}

// Facing is the surface at a point: its place in its own space, in the
// world, and the way the surface faces there, out of it.
func (weathered Weathered) Facing(local, world,
	normal maths.Vec) maths.Light {
	site := &Site{Point: world, Normal: normal, Name: weathered.Name}
	if joined, jointed := weathered.Base.(Jointed); jointed {
		site.Joint = joined.InJoint(local)
	}
	return weathered.Layer.Lay(weathered.At(local), site)
}

// Lay is the base colour at a site with every agent's work laid on it,
// in the agents' order.
func (layer Layer) Lay(base maths.Light, site *Site) maths.Light {
	colour := base
	for _, agent := range layer.Agents {
		amount := layer.Amount(agent, site)
		if !(amount > 0) {
			continue
		}
		laying := layer.paint(agent, site)
		if agent.Kind == Glitter {
			colour = colour.Add(laying.Scale(amount))
			continue
		}
		colour = mix(colour, laying, amount)
	}
	return colour
}

// mix is from, and to as far as share of the way.
func mix(from, to maths.Light, share float64) maths.Light {
	return from.Scale(1 - share).Add(to.Scale(share))
}

// paint is the colour an agent lays at a site: its own, or for the kinds
// that lay two, one of them by a number drawn for the mark's cell, or
// for brick the bare face where it has flaked and the crack's colour
// elsewhere.
func (layer Layer) paint(agent Agent, site *Site) maths.Light {
	switch agent.Kind {
	case Graffiti:
		_, colour := layer.mark(agent, site)
		return colour
	case Festival:
		at := site.Point.Scale(1 / math.Max(agent.Grain, tiny))
		cellX := int64(math.Floor(at.X))
		cellY := int64(math.Floor(at.Y))
		cellZ := int64(math.Floor(at.Z))
		seed := layer.Seed + salt(agent) + 1
		return mix(agent.Colour, agent.Other,
			noise.Hash3(seed, cellX, cellY, cellZ))
	case Spall:
		if layer.flaked(agent, site) > cellMiddle {
			return agent.Other
		}
	}
	return agent.Colour
}
