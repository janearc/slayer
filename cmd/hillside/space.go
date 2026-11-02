package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
)

// darkness is the sky in space: no light from anywhere but what is
// there. stars are not drawn yet.
type darkness struct{}

// Light is none, every way.
func (darkness) Light(maths.Vec) maths.Light { return maths.Light{} }

// spaced stages a set with a planet and no ground: the planet at the
// middle, where its cameras find it, against darkness.
func (stage *staging) spaced(made set.Planet) {
	stage.places[made.Name()] = maths.Vec{}
	stage.world.Sky = darkness{}
}

// dusk stages the sky over a ground set with a planet in it: the
// twilight the set's sun lights from below the horizon, and the planet
// where the set puts it, so its cameras find it.
//
// the earth's air is not yet between the ground and the planet or its
// rings, so they are seen as from above the air: said, until it is.
func (stage *staging) dusk(read set.Set, made set.Planet) error {
	sun, has := read.Sunlit()
	if !has {
		return fmt.Errorf("%s: a planet needs a sun to light it",
			stage.given.set)
	}
	twilight, err := stage.twilit(sun)
	if err != nil {
		return err
	}
	stage.world.Sky = moonlight.Skies{twilight}
	azimuth, altitude, distance, _ := made.InSky()
	stage.places[made.Name()] = skyward(azimuth, altitude).Scale(distance)
	return stage.earthed()
}

// earthed stands the set on the earth: a globe of its radius, its top at
// the sea, under the ground's model. it is the earth the twilight's sun
// has set behind, so the star's lamp is hidden from the ground, the
// trees and the smoke, and only what stands high enough sees it.
func (stage *staging) earthed() error {
	ball, err := globe.New(earthRadius)
	if err != nil {
		return err
	}
	centre := maths.V(0, -earthRadius-under, 0)
	stage.world.Add(thing.Thing{Shape: ball,
		Material: &thing.Material{Name: "earth", Colour: soil},
		Motion:   thing.Still(maths.Subspace{Origin: centre})})
	stage.made.things++
	return nil
}

// planetLit is the set's planet as its first camera sees it: a globe
// whose face is worked from its air under its star, which shines from
// the set's sun's way at the planet's own distance from it.
func (stage *staging) planetLit(read set.Set, made set.Planet,
	eye maths.Vec) error {
	sun, has := read.Sunlit()
	if !has {
		return fmt.Errorf("%s: a planet needs a sun to light it",
			stage.given.set)
	}
	body, err := stage.body(made)
	if err != nil {
		return err
	}
	middle := stage.places[made.Name()]
	pole := skyward(made.Pole())
	plane := maths.Upright(pole)
	from := eye.Sub(middle)
	body, err = body.Seen(maths.V(from.Dot(plane.X), from.Dot(plane.Y),
		from.Dot(plane.Z)), rayWide(read, stage.given))
	if err != nil {
		return err
	}
	star := skyward(sun.Where())
	light := starlight(made.FromStar())
	face := planet.NewFace(body, pole, star, eye.Sub(middle), light,
		stage.given.exposure)
	ball, err := globe.New(body.Radius())
	if err != nil {
		return err
	}
	stage.world.Add(thing.Thing{Shape: ball,
		Material: &thing.Material{Name: made.Name(), Shine: face},
		Motion:   thing.Still(maths.Subspace{Origin: middle})})
	stage.made.things++
	if err := stage.ringed(body, made, pole, middle); err != nil {
		return err
	}
	lamp, err := emit.NewDistant(star, seenAs(light, stage.given.exposure))
	if err != nil {
		return err
	}
	stage.world.Lamp(lamp)
	stage.made.lamps++
	return nil
}

// rayWide is how wide one ray of the picture is, in radians: the rig's
// frame over its focal length, as its lens is focused far, or the lens's
// width, over the rays across.
func rayWide(read set.Set, given asked) float64 {
	fine := float64(given.width * given.samples)
	if film, has := read.Film(); has {
		if rig, has := film.Rig(); has {
			width, _ := rig.Frame()
			focal, _, _ := rig.Lens()
			return width / focal / fine
		}
	}
	return given.wide * maths.Degree / fine
}

// ringThickness is how thick a ring's slab is drawn, in metres: real
// rings are tens of metres thick, and a slab this thick is one a march
// finds, its depth through kept as the ring's. picked.
const ringThickness = 2e5

// ringed puts each of the body's rings in the world as a slab about its
// middle, its plane square to the pole.
func (stage *staging) ringed(body planet.Body, made set.Planet, pole,
	middle maths.Vec) error {
	tipped := thing.Still(maths.Subspace{Origin: middle,
		Orientation: maths.Upright(pole)})
	watts, band, glowing := made.Xrays()
	xrays, err := planet.NewXrays(watts, band)
	if glowing && err != nil {
		return err
	}
	for _, ring := range body.Rings() {
		slab, err := planet.NewRingSlab(ring, ringThickness)
		if err != nil {
			return err
		}
		stage.world.AddVolume(thing.Volume{Medium: slab, Motion: tipped})
		if !glowing {
			continue
		}
		glow, shine, err := planet.NewRingGlow(ring, ringThickness,
			body.Radius(), xrays, stage.given.exposure)
		if err != nil {
			return err
		}
		stage.world.AddVolume(thing.Volume{Medium: glow, Motion: tipped,
			Material: &thing.Material{Name: "glow", Shine: shine}})
	}
	return nil
}

// seenAs is light in libtheme's bands as the tracer's three channels, at
// an exposure.
func seenAs(light bands.Light, exposure float64) maths.Light {
	red, green, blue := srgb.Light(observer.Swatch(light.Scale(exposure)))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}

// starlight is the sun's light at a distance in astronomical units, in
// libtheme's bands: a black body at the sun's temperature, falling as
// the distance squared.
func starlight(distance float64) bands.Light {
	spectrum := map[int]float64{}
	for nm, power := range moonlight.Sunlight() {
		spectrum[nm] = power / (distance * distance)
	}
	return bands.Split(spectrum)
}

// body is the planet as the planet package makes it, from its columns
// and zones, with methane's absorption read from the shelf.
func (stage *staging) body(made set.Planet) (planet.Body, error) {
	gases, err := stage.gases()
	if err != nil {
		return planet.Body{}, err
	}
	var zones []planet.Zone
	for _, each := range made.Zones() {
		south, north, name := each.Latitudes()
		column, err := columnOf(made.Column(name), gases)
		if err != nil {
			return planet.Body{}, err
		}
		zone, err := planet.NewZone(south, north, column)
		if err != nil {
			return planet.Body{}, err
		}
		zones = append(zones, zone)
	}
	radius, gravity, deck := made.Size()
	body, err := planet.NewBody(radius, gravity, deck, zones)
	if err != nil {
		return planet.Body{}, err
	}
	rings, err := ringsOf(made, body.Mass())
	if err != nil {
		return planet.Body{}, err
	}
	ringed, err := body.Ringed(rings)
	if err != nil {
		return planet.Body{}, err
	}
	stormy, err := stormed(ringed, made, gases)
	if err != nil {
		return planet.Body{}, err
	}
	return weathered(stormy, made, gases)
}

// ringsOf is the set's rings as the planet package makes them, under a
// planet of a mass in kilograms.
func ringsOf(made set.Planet, mass float64) ([]planet.Ring, error) {
	var rings []planet.Ring
	for _, each := range made.Rings() {
		ring, err := ringOf(made, each, mass)
		if err != nil {
			return nil, err
		}
		rings = append(rings, ring)
	}
	return rings, nil
}

// ringOf is one ring as the planet package makes it: tinted where the
// set says, with the planet's ringlets, its own grains and moonlets,
// and glowing by the phosphors it is made of.
func ringOf(made set.Planet, each set.Ring, mass float64) (planet.Ring,
	error) {
	inner, outer := each.Reach()
	depth, albedo, lean := each.Optics()
	ring, err := planet.NewRing(inner, outer, depth, albedo, lean)
	if blue, tinted := each.Blue(); err == nil && tinted {
		ring, err = ring.Tinted(blue)
	}
	if seed, strength, width, textured := made.Ringlets(); err == nil &&
		textured {
		ring, err = ring.Ringleted(seed, strength, width)
	}
	if grains, clumped := each.Grained(); err == nil && clumped {
		ring, err = ring.Grained(grains.Clumps())
	}
	seed, count, least, most, slope, opens, mooned := each.Moonlets()
	if err == nil && mooned {
		ring, err = ring.Cleared(seed, count, least, most, slope, opens,
			mass)
	}
	if err != nil {
		return planet.Ring{}, err
	}
	return ring.Glowing(phosphorsOf(each.MadeOf()))
}

// phosphorsOf is what a ring is made of, less its ice, which glows not.
func phosphorsOf(stuff map[string]float64) map[string]float64 {
	delete(stuff, "ice")
	return stuff
}

// stormed is the body with the set's storms, and their churn, if it says
// any.
func stormed(body planet.Body, made set.Planet,
	gases map[string]planet.Gas) (planet.Body, error) {
	written, has := made.Stormy()
	if !has {
		return body, nil
	}
	if name, strength := written.Churn(); name != "" {
		column, err := columnOf(made.Column(name), gases)
		if err == nil {
			body, err = body.Churned(column, strength)
		}
		if err != nil {
			return planet.Body{}, err
		}
	}
	var spots []planet.Spot
	for _, each := range written.Spots() {
		spot, err := planet.NewSpot(each.Where())
		if err != nil {
			return planet.Body{}, err
		}
		spots = append(spots, spot)
	}
	seed, turbulence, eddies, stretch, edge, folds := written.Flow()
	storms, err := planet.NewStorms(seed, turbulence, eddies, stretch, edge,
		folds, spots)
	if err != nil {
		return planet.Body{}, err
	}
	if shear, every, has := written.Jets(); has {
		if storms, err = storms.Jetted(shear, every); err != nil {
			return planet.Body{}, err
		}
	}
	seed, count, smallest, largest, slope, has := written.Swarm()
	if has {
		storms, err = storms.Swarmed(seed, count, smallest, largest, slope)
		if err != nil {
			return planet.Body{}, err
		}
	}
	return marked(body.Stormy(storms), made, written, gases)
}

// marked is the body with the columns its vortices show, if the set's
// storms name them.
func marked(body planet.Body, made set.Planet, written set.Storms,
	gases map[string]planet.Gas) (planet.Body, error) {
	capName, clearingName := written.Marks()
	if capName == "" && clearingName == "" {
		return body, nil
	}
	cap, err := columnOf(made.Column(capName), gases)
	if err != nil {
		return planet.Body{}, err
	}
	clearing, err := columnOf(made.Column(clearingName), gases)
	if err != nil {
		return planet.Body{}, err
	}
	return body.Marked(cap, clearing)
}

// columnOf is a column of set layers as planet layers, pressures from
// bar to pascals.
func columnOf(layers []set.Layer, gases map[string]planet.Gas) (
	[]planet.Layer, error) {
	var column []planet.Layer
	for _, each := range layers {
		top, bottom := each.Pressures()
		var named []planet.Gas
		var shares []float64
		for name, share := range each.Gases() {
			named = append(named, gases[name])
			shares = append(shares, share)
		}
		depth, albedo, lean := each.Haze()
		haze, err := planet.NewHaze(depth, albedo, lean)
		if blue, tinted := each.HazeBlue(); err == nil && tinted {
			haze, err = haze.Tinted(blue)
		}
		if err != nil {
			return nil, err
		}
		layer, err := planet.NewLayer(top*1e5, bottom*1e5, named, shares,
			haze)
		if err != nil {
			return nil, err
		}
		column = append(column, layer)
	}
	return column, nil
}

// gases is every gas a set may name, methane with its absorption read
// from karkoschka's table on the shelf.
func (stage *staging) gases() (map[string]planet.Gas, error) {
	path, err := stage.shelved("karkoschka-1994-methane")
	if err != nil {
		return nil, err
	}
	table, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer table.Close()
	absorbing, err := planet.MethaneAbsorbing(table)
	if err != nil {
		return nil, err
	}
	return map[string]planet.Gas{"neon": planet.Neon(),
		"krypton": planet.Krypton(), "xenon": planet.Xenon(),
		"methane": planet.Methane(absorbing)}, nil
}

// shelved is where a source is on the reader's shelf, once it is found
// there as pinned.
func (stage *staging) shelved(key string) (string, error) {
	dir := filepath.Dir(stage.given.set)
	shelf, err := pinned(key, dir)
	if err != nil {
		return "", err
	}
	listed, err := nearest(dir, "sources.yaml")
	if err != nil {
		return "", err
	}
	list, err := sources.Read(listed)
	if err != nil {
		return "", err
	}
	source, _ := list.Find(key)
	return sources.NewShelf(shelf, nil).Path(source), nil
}
