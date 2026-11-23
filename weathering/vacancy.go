package weathering

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the draws a dwelling's state is made of, which keep a dwelling's own
// noise apart from the noise of its marks
const (
	// stateDraw tells which state an empty dwelling is in.
	stateDraw = 1
	// shardDraw tells which corner of a pane is broken, and shardSize
	// how much of it, by the draws that follow.
	shardDraw = 2
	// slantDraw is the first draw of a board's slant.
	slantDraw = 3
)

// across is the way along the fronts of a vacancy agent's dwellings: level,
// and square to the way the fronts face, which is the agent's Facing. It
// is nought where the agent says no way the fronts face.
func (layer Layer) across(agent Agent) maths.Vec {
	return layer.Up.Cross(agent.Facing).Unit()
}

// outward is the way the fronts of a vacancy agent's dwellings face, level.
func (layer Layer) outward(agent Agent) maths.Vec {
	alongUp := agent.Facing.Dot(layer.Up)
	return agent.Facing.Sub(layer.Up.Scale(alongUp)).Unit()
}

// vacant is whether the dwelling at a place stands empty, by one vacancy
// agent. A dwelling is a cell along the fronts the agent's Facing says,
// Cell wide and Storey tall, and Strength of them stand empty by the
// agent's age, a share that only rises. Past Reach of the agent's place
// none does; no Reach is anywhere.
func (layer Layer) vacant(agent Agent, point maths.Vec) bool {
	if agent.Kind != Vacancy || !(agent.Cell > 0) ||
		layer.across(agent) == (maths.Vec{}) {
		return false
	}
	away := point.Sub(agent.At)
	alongUp := away.Dot(layer.Up)
	away = away.Sub(layer.Up.Scale(alongUp))
	if agent.Reach > 0 && away.Len() > agent.Reach {
		return false
	}
	row, storey := layer.dwelling(agent, point)
	share := agent.Strength * grown(layer.Age, agent.Settle)
	return noise.Hash3(layer.Seed+salt(agent), row, storey, 0) < share
}

// dwelling is which dwelling a place is in: its number along the fronts
// and its storey, or nought for the storey where the agent says none.
func (layer Layer) dwelling(agent Agent, point maths.Vec) (int64, int64) {
	along := layer.across(agent)
	storey := int64(0)
	if agent.Storey > 0 {
		storeysUp := math.Floor((point.Dot(layer.Up) - layer.Ground) /
			agent.Storey)
		storey = int64(storeysUp)
	}
	cellsAlong := math.Floor(point.Dot(along) / agent.Cell)
	return int64(cellsAlong), storey
}

// empty is whether any vacancy agent says the dwelling at a place stands
// empty.
func (layer Layer) empty(point maths.Vec) bool {
	return slices.ContainsFunc(layer.Agents, func(agent Agent) bool {
		return layer.vacant(agent, point)
	})
}

// Dimmed is the glow of a lamp behind a pane, out wherever the dwelling
// it lights stands empty.
type Dimmed struct {
	// Light is the glow, Place where the lamp's thing stands, and Layer
	// the agents that empty dwellings.
	Light maths.Light
	Place maths.Subspace
	Layer *Layer
}

// At is the lamp's glow at a point of its thing, or none if empty.
func (dimmed Dimmed) At(point maths.Vec) maths.Light {
	if dimmed.Layer.empty(dimmed.Place.Out(point)) {
		return maths.Light{}
	}
	return dimmed.Light
}

// pane is one window's glass, in world space: its middle, the way along
// it and up it, the way it faces out, and how wide and tall it is.
type pane struct {
	middle, across, outward maths.Vec
	wide, tall              float64
}

// panesOf is each pane of a thing that faces the way the agent's fronts
// do: each member of a group, or the whole shape.
func (layer Layer) panesOf(each *thing.Thing, agent Agent) []pane {
	place := each.At(0)
	group, grouped := each.Shape.(*shape.MeshGroup)
	if !grouped {
		if one, wall := layer.paneOf(agent, each.Shape, place.Out); wall {
			return []pane{one}
		}
		return nil
	}
	var found []pane
	for _, member := range group.Members() {
		out := func(point maths.Vec) maths.Vec {
			return place.Out(member.Space.Out(point))
		}
		if one, wall := layer.paneOf(agent, member.Mesh, out); wall {
			found = append(found, one)
		}
	}
	return found
}

// paneOf is the pane that one mesh is, taken through out into the world,
// and whether it is one: it has a face, and it is wider along the fronts
// than it is deep, which a pane on a side wall is not. Its middle stands
// on its outer face.
func (layer Layer) paneOf(agent Agent, of shape.Shape,
	out func(maths.Vec) maths.Vec) (pane, bool) {
	outward := layer.outward(agent)
	axes := [3]maths.Vec{layer.across(agent), layer.Up, outward}
	var least, most [3]float64
	for index := range axes {
		least[index], most[index] = math.Inf(1), math.Inf(-1)
	}
	triangles(of, out, func(first, second, third maths.Vec) {
		for _, corner := range [3]maths.Vec{first, second, third} {
			for index, axis := range axes {
				least[index] = min(least[index], corner.Dot(axis))
				most[index] = max(most[index], corner.Dot(axis))
			}
		}
	})
	wide, deep := most[0]-least[0], most[2]-least[2]
	if !(wide > deep) || !(wide > 0) {
		return pane{}, false
	}
	middle := axes[0].Scale((least[0] + most[0]) / 2).
		Add(axes[1].Scale((least[1] + most[1]) / 2)).
		Add(outward.Scale(most[2]))
	return pane{middle: middle, across: axes[0], outward: outward,
		wide: wide, tall: most[1] - least[1]}, true
}

// vacate boards the windows of empty dwellings, or breaks them, by the
// agent: each pane of a thing whose material the agent names, in an empty
// dwelling, is boarded in planks across it, or has a corner of it missing
// as a dark shard, or, for the rest, is left to its dark room.
func (layer *Layer) vacate(world *thing.World, agent Agent) {
	var boards, shards builder
	for _, each := range slices.Clone(world.Things) {
		if !holdsStill(each) || each.Material == nil ||
			!named(each.Material.Name, agent.On) {
			continue
		}
		for _, one := range layer.panesOf(each, agent) {
			if !layer.vacant(agent, one.middle) {
				continue
			}
			row, storey := layer.dwelling(agent, one.middle)
			state := noise.Hash3(layer.Seed+salt(agent)+stateDraw, row,
				storey, 0)
			switch {
			case state < agent.Boarded:
				layer.plank(&boards, agent, one)
			case state < agent.Boarded+agent.Broken:
				layer.shard(&shards, agent, one)
			}
		}
	}
	boards.add(world, &thing.Material{Name: "vacancy boards",
		Colour: agent.Colour})
	shards.add(world, &thing.Material{Name: "vacancy broken",
		Colour: agent.Other})
}

// builder collects the triangles of many small pieces into one mesh.
type builder struct {
	points []maths.Vec
	faces  [][3]int
}

// quad adds a four sided piece, its corners going round as seen from
// outside.
func (collected *builder) quad(first, second, third, fourth maths.Vec) {
	start := len(collected.points)
	collected.points = append(collected.points, first, second, third, fourth)
	collected.faces = append(collected.faces, [3]int{start, start + 1,
		start + 2}, [3]int{start, start + 2, start + 3})
}

// triangle adds a three sided piece.
func (collected *builder) triangle(first, second, third maths.Vec) {
	start := len(collected.points)
	collected.points = append(collected.points, first, second, third)
	collected.faces = append(collected.faces, [3]int{start, start + 1,
		start + 2})
}

// add puts what was collected in the world as one thing in a material,
// where the world's own place is the place; nothing collected is nothing
// added.
func (collected *builder) add(world *thing.World, material *thing.Material) {
	if len(collected.faces) == 0 {
		return
	}
	made, err := mesh.New(collected.points, collected.faces, nil, false)
	if err != nil {
		return
	}
	world.Add(thing.Thing{Shape: made, Material: material,
		Motion: thing.Still(maths.Subspace{})})
}

// plank boards a pane: boards across it, as many as fit its height at
// the agent's Plank, each Share of its slot tall, standing Proud off the
// glass, and each end a little higher or lower, up to Slant, by a draw.
func (layer *Layer) plank(into *builder, agent Agent, one pane) {
	fit := math.Round(one.tall / math.Max(agent.Plank, tiny))
	count := max(1, int(fit))
	slot := one.tall / float64(count)
	bottom := one.middle.Dot(layer.Up) - one.tall/2
	left := one.middle.Sub(one.across.Scale(one.wide / 2)).
		Add(one.outward.Scale(agent.Proud))
	right := left.Add(one.across.Scale(one.wide))
	for board := range count {
		low := layer.Up.Scale(bottom + float64(board)*slot -
			left.Dot(layer.Up))
		high := low.Add(layer.Up.Scale(slot * agent.Share))
		tilt := func(end int64) maths.Vec {
			key := int64(one.middle.Dot(one.across) * 100)
			drawn := noise.Hash3(layer.Seed+salt(agent)+slantDraw,
				int64(board), end, key)
			return layer.Up.Scale((drawn - cellMiddle) * agent.Slant)
		}
		lowLeft := left.Add(low).Add(tilt(0))
		lowRight := right.Add(low).Add(tilt(1))
		highRight := right.Add(high).Add(tilt(1))
		highLeft := left.Add(high).Add(tilt(0))
		into.quad(lowLeft, lowRight, highRight, highLeft)
	}
}

// shard breaks a pane: a corner of it, chosen by a draw, gone to the dark
// behind it, as a triangle whose sides are a draw's share of the pane's
// width and height, no more than the pane's Share of either.
func (layer *Layer) shard(into *builder, agent Agent, one pane) {
	key := int64(one.middle.Dot(one.across) * 100)
	side := func(draw int64) float64 {
		return agent.Share * noise.Hash3(layer.Seed+salt(agent)+shardDraw,
			draw, key, 0)
	}
	flip := func(draw int64) float64 {
		drawn := noise.Hash3(layer.Seed+salt(agent)+shardDraw, draw, key, 1)
		return 1 - 2*math.Floor(2*drawn)
	}
	across, up := flip(0), flip(1)
	corner := one.middle.Add(one.across.Scale(across * one.wide / 2)).
		Add(layer.Up.Scale(up * one.tall / 2)).
		Add(one.outward.Scale(agent.Proud))
	acrossReach := side(2)
	upReach := side(3)
	acrossEdge := corner.Sub(one.across.Scale(across * one.wide * acrossReach))
	upEdge := corner.Sub(layer.Up.Scale(up * one.tall * upReach))
	into.triangle(corner, acrossEdge, upEdge)
}
