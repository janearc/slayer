package main

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"math"
	"time"
)

type fuelGroup struct {
	field   *flame.Field
	indices []int
	anchor  maths.Vec
}
type retainedBed struct {
	model      fuelbed.Evolution
	spec       fuelbed.Spec
	groups     []fuelGroup
	tiles      []*mesh.Mesh
	lightCells int
}

// prepareBeds retains one fuel solver and immutable display geometry.
func (stage *staging) prepareBeds(read set.Set) error {
	for _, b := range read.FuelBeds() {
		width, depth, cell := b.Width(), b.Depth(), b.Cell()
		x0, z0 := b.East()-stage.west-float64(width)*cell/2, stage.north-b.North()-float64(depth)*cell/2
		w, h, spacing := stage.ground.Size()
		if x0 < 0 || z0 < 0 || x0+float64(width)*cell > float64(w-1)*spacing || z0+float64(depth)*cell > float64(h-1)*spacing {
			return fmt.Errorf("fuel bed is off the ground")
		}
		count := width*depth/64 + len(stage.world.Volumes)
		for _, front := range stage.fronts {
			count += front.peakVolumes
		}
		if count > gpu.VolumeLimit() {
			return fmt.Errorf("fuel bed exceeds GPU volume budget")
		}
		spec := fuelbed.Grass(fuelbed.Spec{Width: width, Depth: depth, Cell: cell, Step: b.Step(), Ambient: b.Ambient(), Initial: make([]fuelbed.Cell, width*depth)})
		bed := retainedBed{spec: spec}
		row, _ := read.Recipe(b.Recipe())
		fire, _ := row.Fire()
		bed.lightCells = fire.LightCells
		for z := 0; z < depth; z++ {
			for x := 0; x < width; x++ {
				i := z*width + x
				dx, dz := (float64(x)+.5-float64(width)/2)*cell, (float64(z)+.5-float64(depth)/2)*cell
				spec.Initial[i] = fuelbed.Cell{Excess: (b.Hot() - b.Ambient()) * math.Exp(-(dx*dx+dz*dz)/(b.Radius()*b.Radius())), Fuel: b.Fuel()}
				if b.Surface() == "terrain" {
					continue
				}
				corners := []maths.Vec{}
				for _, p := range [][2]float64{{0, 0}, {0, 1}, {1, 1}, {1, 0}} {
					px, pz := x0+(float64(x)+p[0])*cell, z0+(float64(z)+p[1])*cell
					corners = append(corners, maths.V(px, stage.groundAt(px, pz)+.015, pz))
				}
				tile, err := mesh.New(corners, [][3]int{{0, 1, 2}, {0, 2, 3}}, nil, false)
				if err != nil {
					return err
				}
				bed.tiles = append(bed.tiles, tile)
			}
		}
		// Terrain places the flames; the PDE has no slope feedback.
		// Jitter stays inside cells without moving the stencil.
		for gz := 0; gz < depth; gz += 8 {
			for gx := 0; gx < width; gx += 8 {
				group := fuelGroup{anchor: maths.V(x0+float64(gx)*cell, 0, z0+float64(gz)*cell)}
				var patches []flame.Ignition
				for z := gz; z < gz+8; z++ {
					for x := gx; x < gx+8; x++ {
						index := z*width + x
						at := maths.V(x0+(float64(x)+.5)*cell, 0, z0+(float64(z)+.5)*cell)
						at.X += (float64(stir(stage.given.seed, index+2001)%10000)/10000 - .5) * cell * .7
						at.Z += (float64(stir(stage.given.seed, index+4001)%10000)/10000 - .5) * cell * .7
						at.Y = stage.groundAt(at.X, at.Z) + .025

						patch := fire.Flame
						patch.Seed = stir(stage.given.seed, index+1)
						patches = append(patches, flame.Ignition{At: at.Sub(group.anchor), Fire: patch})
						group.indices = append(group.indices, index)
					}
				}
				field, err := flame.NewField(patches, stage.given.exposure)
				if err != nil {
					return err
				}
				group.field = field
				bed.groups = append(bed.groups, group)
			}
		}
		var err error
		east, south := b.HeatTransport()
		transport := fuelbed.Transport{East: east, South: south}
		if stage.given.onGPU {
			bed.model, err = fuelbed.NewMetalAdvected(spec, transport)
		} else {
			bed.model, err = fuelbed.NewAdvected(spec, transport)
		}
		if err != nil {
			return err
		}
		stage.beds = append(stage.beds, bed)
	}
	return nil
}

// addAt derives flames and spent-fuel reflectance from the current state.
func (bed *retainedBed) addAt(world *thing.World, made *tally, at time.Duration) error {
	state, err := bed.model.At(at.Seconds())
	if err != nil {
		return err
	}
	made.things += len(bed.tiles)
	for i, tile := range bed.tiles {
		c := state[i]
		// Authored fuel reflectance, without charcoal chemistry.
		colour := maths.RGB(.25, .17, .055).Scale(c.Fuel).Add(maths.RGB(.012, .01, .008).Scale(1 - c.Fuel))
		world.Add(thing.Thing{Shape: tile, Material: &thing.Material{Name: "surface fuel", Colour: colour}})
	}
	for _, group := range bed.groups {
		strengths := make([]float64, len(group.indices))
		for i, index := range group.indices {
			strengths[i] = math.Min(1, fuelbed.Reaction(state[index], bed.spec.Activation)/.35)
			// Display cutoff only; heat and fuel still evolve.
			if strengths[i] < .002 {
				strengths[i] = 0
			}

		}
		field, err := group.field.WithStrengths(strengths)
		if err != nil {
			return err
		}
		field = field.Tick(at).(*flame.Field)
		pieces := field.Pieces()
		if len(pieces) == 0 {
			continue
		}
		made.burning += len(pieces)
		world.AddVolume(thing.Volume{Medium: field, Material: &thing.Material{Name: "surface flame", Shine: field.Shine()}, Motion: thing.Still(maths.Subspace{Origin: group.anchor})})
		// One finite lamp per group retains sampled power.
		// Its position and radius approximate local lighting.
		total := maths.Light{}
		centre := maths.Vec{}
		weight := 0.0
		for _, source := range field.Sources(bed.lightCells) {
			v := source.Strength.Luminance()
			total = total.Add(source.Strength)
			centre = centre.Add(source.At.Scale(v))
			weight += v
		}
		if weight > 0 {
			lamp, err := emit.NewVolume(emit.Lit{At: centre.Scale(1 / weight).Add(group.anchor), Strength: total}, measure.Metres(4*bed.spec.Cell))
			if err != nil {
				return err
			}
			world.Lamps = append(world.Lamps, lamp)
		}
	}
	return nil
}

// closeBeds releases the solvers after a sequence finishes.
func (live *liveScene) closeBeds() {
	for _, bed := range live.beds {
		if closer, ok := bed.model.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}
