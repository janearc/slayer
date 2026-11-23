package session

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/skyglow"
)

// flatCell is the size, in metres, of the bins the world's emitters are
// gathered into before the glow is solved: about the width of a street,
// so a lit tower's panels become a few sources and the glow is right at
// a distance of a few streets and more. Picked.
const flatCell = 25.0

// onlyOneFlatLayer is how many layers of flat air a set may say: the
// glow is solved for one air, so one.
const onlyOneFlatLayer = 1

// metresPerUnit is the world unit flat air takes: the scene's metres.
const metresPerUnit = 1.0

// evenCoverage is the coverage of a layer with no patches: all of it
// thick, as flat air is.
const evenCoverage = 1.0

// emitters is what shines in a world, as skyglow takes it, and a count of
// the pieces that shine and are left out.
type emitters struct {
	cluster *skyglow.Cluster
	// left is how many shining things or lamps the glow cannot take:
	// emission painted by a texture, a shape that is no mesh or sphere, a
	// spot, a beam, a sun.
	left int
}

// flatAir stages the set's flat layer of air, where it says one: the
// layer as a painted volume of haze, and the glow the world's own light
// leaves in it, solved by package skyglow. A set that says none is
// staged as it was, and this changes nothing.
//
// It refuses a second layer, a world unit that is not one, haze that is
// patchy, a layer below the ground, and a set that does not say
// volume-illumination, since the legacy lamp factor of a painted volume
// is no physical scattering and the glow is.
func (stage *staging) flatAir(read set.Set) error {
	layers := read.FlatAir()
	if len(layers) == nothing {
		return nil
	}
	air, mist, err := stage.flatAirOf(read, layers)
	if err != nil {
		return err
	}
	found := stage.shining()
	sources := found.cluster.Sources()
	region := flatRegion(air, sources)
	aerosol, err := stage.hazed(layers[0], air, mist, region)
	if err != nil {
		return err
	}
	if len(sources) == nothing {
		return nil
	}
	return stage.glowing(air, region, sources, found, aerosol)
}

// flatAirOf is the air the set's flat layer is, and the mist it is
// taken from, or why it cannot be.
func (stage *staging) flatAirOf(read set.Set, layers []set.AtmosphereLayer) (
	skyglow.Air, set.Mist, error) {
	name := stage.given.set
	if len(layers) > onlyOneFlatLayer {
		return skyglow.Air{}, set.Mist{}, fmt.Errorf("%s: flat air has "+
			"one layer, and this says %d", name, len(layers))
	}
	if said, has := read.World(); has && said.Unit() != metresPerUnit {
		return skyglow.Air{}, set.Mist{}, fmt.Errorf("%s: flat air is in "+
			"metres, and the world unit is %g", name, said.Unit())
	}
	if stage.world.VolumeIllumination.Samples() == nothing {
		return skyglow.Air{}, set.Mist{}, fmt.Errorf("%s: flat air needs "+
			"volume-illumination, so lamps scatter by the isotropic "+
			"phase and not the legacy factor", name)
	}
	mist, _ := read.Mist(layers[0].Weather())
	if _, _, cloud := mist.Cloud(); cloud {
		return skyglow.Air{}, set.Mist{}, fmt.Errorf("%s: flat air is "+
			"a layer from the ground, not a cloud deck", name)
	}
	base, level := layers[0].Base()
	air := skyglow.Air{Extinction: mist.Extinction(), Depth: mist.Depth(),
		Albedo: layers[0].Albedo(), Base: base}
	if !level {
		air.Base = flatGround
	}
	if mist.Coverage() != evenCoverage || air.Base < flatGround || !air.Valid() {
		return skyglow.Air{}, set.Mist{}, fmt.Errorf("%s: flat air is an "+
			"even layer, coverage 1, from the ground up, of an extinction "+
			"and an albedo above nought: %+v, coverage %g", name, air,
			mist.Coverage())
	}
	return air, mist, nil
}

// flatRegion is the region the air is solved over and drawn in: round the
// sources, or round the origin where nothing shines.
func flatRegion(air skyglow.Air, sources []skyglow.Source) skyglow.Region {
	if len(sources) == nothing {
		sources = []skyglow.Source{{}}
	}
	return skyglow.RegionAround(air, sources)
}

// hazed puts the layer in the world as a painted volume of haze over the
// region, with the aerosol's wavelength dependence and the mist's lobe.
func (stage *staging) hazed(layer set.AtmosphereLayer, air skyglow.Air,
	mist set.Mist, region skyglow.Region) (*medium.Aerosol, error) {
	weather := medium.NewWeather(air.Extinction, air.Depth, mist.Coverage(),
		mist.Seed(), maths.Vec{})
	weather.Asymmetry = mist.Asymmetry()
	bounds := maths.Box{Min: region.Low, Max: region.High}
	floor := func(float64, float64) float64 { return flatGround }
	deck, err := medium.NewDeck(weather, bounds, floor, air.Base)
	if err != nil {
		return nil, err
	}
	aerosol, err := medium.NewAerosol(deck, medium.DefaultExponent)
	if err != nil {
		return nil, err
	}
	material := thing.Diffuse(layer.Weather(), maths.Grey(layer.Albedo()))
	stage.world.AddVolume(thing.NewVolume(aerosol, material))
	return aerosol, nil
}

// glowing solves the glow of the sources in the air and gives it to the
// world, and tells the progress line what it came to.
func (stage *staging) glowing(air skyglow.Air, region skyglow.Region,
	sources []skyglow.Source, found *emitters, aerosol *medium.Aerosol) error {
	field, err := skyglow.Solve(air, region, sources,
		skyglow.DefaultOptions())
	if err != nil {
		return fmt.Errorf("%s: %w", stage.given.set, err)
	}
	field.Medium = aerosol
	stage.world.Glow = field
	told := fmt.Sprintf("%s; %d emitters left out", field.Report().Summary(),
		found.left)
	stage.given.cached = append(stage.given.cached,
		cacheNote{what: told, told: true})
	return nil
}

// shining gathers what shines in the world: its glowing surfaces and its
// point lamps. A thing drawn but casting no shadow is a star or the moon
// drawn as a ball, which lights the air by its own distant lamp and not
// as part of the city, so it is passed over.
func (stage *staging) shining() *emitters {
	found := &emitters{cluster: skyglow.NewCluster(flatCell)}
	for _, each := range stage.world.Things {
		material := each.Material
		if each.Hidden() || each.Clear() || material == nil {
			continue
		}
		if material.Shine != nil {
			found.left++
			continue
		}
		if material.Glow != (maths.Light{}) {
			found.surface(each.Shape, material.Glow, each.At(0).Out)
		}
	}
	for _, each := range stage.world.Lamps {
		found.lamp(each)
	}
	return found
}

// lamp adds a lamp that shines every way from a point, which the tracer
// scatters itself, and counts any other as left out.
func (found *emitters) lamp(each thing.Lamp) {
	bulb, point := each.(emit.Lit)
	if !point || bulb.Width > nothing {
		found.left++
		return
	}
	found.cluster.Point(bulb.At, bulb.Strength, true)
}

// surface adds a shape that shines with a radiance, placed in the world by
// out: each triangle of a mesh as a panel, a sphere as a point, and any
// other shape counted as left out.
func (found *emitters) surface(solid shape.Shape, radiance maths.Light,
	out func(maths.Vec) maths.Vec) {
	switch made := solid.(type) {
	case *mesh.Mesh:
		for face := range made.FaceCount() {
			first, second, third := made.Corners(face)
			found.panel(first, second, third, radiance, out)
		}
	case *shape.Mesh:
		for _, face := range made.Faces {
			found.panel(made.Vertices[face[0]], made.Vertices[face[1]],
				made.Vertices[face[2]], radiance, out)
		}
	case *shape.MeshGroup:
		for index := range made.MemberCount() {
			member := made.Member(index)
			found.surface(member.Mesh, radiance, func(at maths.Vec) maths.Vec {
				return out(member.Space.Out(at))
			})
		}
	case shape.Sphere:
		found.sphere(made, radiance, out)
	default:
		found.left++
	}
}

// panel adds one triangle of a mesh, in the mesh's own space, placed in
// the world by out.
func (found *emitters) panel(first, second, third maths.Vec,
	radiance maths.Light, out func(maths.Vec) maths.Vec) {
	found.cluster.Panel(out(first), out(second), out(third), radiance)
}

// sphere adds a glowing sphere as a point that shines every way: its
// radiance times its disc, which is its intensity along any way.
func (found *emitters) sphere(made shape.Sphere, radiance maths.Light,
	out func(maths.Vec) maths.Vec) {
	middle := out(maths.Vec{})
	radius := out(maths.V(made.Radius, 0, 0)).Sub(middle).Len()
	found.cluster.Point(middle, radiance.Scale(math.Pi*radius*radius), false)
}
