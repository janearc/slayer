package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
	"github.com/janearc/slayer/terrain"
)

// with -set, the set file says the world and the film; these flags say
// how it is drawn, kept, proved and seen, and may be given with it
var withSet = map[string]bool{"set": true, "out": true, "raw": true,
	"prove": true, "proof": true, "pixel": true, "many": true,
	"cpuprofile": true, "gpu": true, "at": true, "wide": true,
	"faces": true, "sky": true, "places": true}

// clashes is a flag given beside -set that says what the set file says,
// or nothing.
func clashes(flags *flag.FlagSet, given asked) string {
	clash := ""
	if given.set == "" {
		return clash
	}
	flags.Visit(func(each *flag.Flag) {
		if !withSet[each.Name] && clash == "" {
			clash = each.Name
		}
	})
	return clash
}

// the scene's own numbers, picked by looking, none measured: the colour
// of kilauea's ground, dark weathered basalt, and how many of a stand's
// kind are made before the rest repeat them
var basalt = maths.RGB(0.07, 0.065, 0.06)

const standVariety = 4

// staging is a set file's world being built: what was asked, the
// stock, where each named thing stands, where every thing stood, and
// the tally.
type staging struct {
	firebrandShape *mesh.Mesh
	unretained     string
	fronts         []retainedFront
	beds           []retainedBed
	fuels          map[string][]groveTree
	fires          []retainedFire
	dynamicThings  map[*thing.Thing]bool
	dynamicVolumes map[*thing.Volume]bool
	dynamicLamps   map[int]bool
	given          asked
	world          *thing.World
	have           stock
	made           tally
	places         map[string]maths.Vec
	radii          map[string]float64
	recipes        set.Set
	stood          []standing
	ground         terrain.Window
	groundExtent   maths.Box
	west           float64
	north          float64
	lowest         float64
}

// standing is one thing as the stage stood it: the thing's name or its
// stand's, its kind, and where, east and north on the terrain's grid and
// the ground's height there, in metres.
type standing struct {
	of, kind            string
	east, north, height float64
}

// note keeps where a thing of a name and a kind was stood.
func (stage *staging) note(of, kind string, place maths.Vec) {
	stage.stood = append(stage.stood, standing{of: of, kind: kind,
		east: stage.west + place.X, north: stage.north - place.Z,
		height: place.Y + stage.lowest})
}

// written writes where every thing stood, a line each, as a table of
// commas: what it is of, its kind, east, north and height.
func (stage *staging) written(path string) error {
	var lines strings.Builder
	lines.WriteString("of,kind,east,north,height\n")
	for _, each := range stage.stood {
		fmt.Fprintf(&lines, "%q,%s,%.1f,%.1f,%.1f\n", each.of, each.kind,
			each.east, each.north, each.height)
	}
	return os.WriteFile(path, []byte(lines.String()), 0o644)
}

// staged is the world a set file asks for, and what was asked, with the
// set's film and its first camera in it.
func staged(given asked) (*thing.World, tally, asked, error) {
	written, err := os.ReadFile(given.set)
	if err != nil {
		return nil, tally{}, given, err
	}
	read, err := set.Read(given.set, written)
	if err != nil {
		return nil, tally{}, given, err
	}
	given, err = filmed(read, given)
	if err != nil {
		return nil, tally{}, given, err
	}
	stage := &staging{given: given, world: thing.NewWorld(),
		have: stock{}, places: map[string]maths.Vec{},
		radii: map[string]float64{}, recipes: read}
	retained := false
	defer func() {
		if !retained {
			(&liveScene{beds: stage.beds}).closeBeds()
		}
	}()
	world, inSpace := read.Planet()
	err = stage.place(read, world, inSpace)
	if err == nil {
		err = stage.lasers(read)
	}
	if err != nil {
		return nil, tally{}, given, err
	}
	if given.placed != "" {
		if err := stage.written(given.placed); err != nil {
			return nil, tally{}, given, err
		}
	}
	shots, err := read.Shots(stage.where)
	if err != nil || len(shots) == 0 {
		return nil, tally{}, given, fmt.Errorf("%s: no camera: %v",
			given.set, err)
	}
	eye := shots[0].Eye()
	stage.given.eye = &eye
	if stage.given.overlay, err = stage.overlaid(read); err != nil {
		return nil, tally{}, given, err
	}
	if inSpace {
		if err := stage.planetLit(read, world, eye.Origin); err != nil {
			return nil, tally{}, given, err
		}
	}
	if film, _ := read.Film(); stage.given.frame != nil {
		rig, _ := film.Rig()
		if err := stage.focused(rig, eye); err != nil {
			return nil, tally{}, given, err
		}
	}
	stage.given.live = stage.retain()
	if len(stage.fronts) > 0 || len(stage.beds) > 0 {
		stage.world, stage.made, err = stage.given.live.frameAt(stage.given.at)
		if err != nil {
			return nil, tally{}, given, err
		}
	}
	retained = true
	return stage.world, stage.made, stage.given, nil
}

// place stages what the set stands on: in space, its planet and the
// darkness; on the ground, its terrain, its light and its stands, and
// a planet in its sky where the set says one stands.
func (stage *staging) place(read set.Set, world set.Planet,
	planeted bool) error {
	_, grounded := read.Ground()
	_, _, _, inSky := world.InSky()
	switch {
	case planeted && !grounded && inSky:
		return fmt.Errorf("%s: a planet in a sky needs ground under it",
			stage.given.set)
	case planeted && grounded && !inSky:
		return fmt.Errorf("%s: a planet over terrain says where it "+
			"stands in the sky", stage.given.set)
	case planeted && !grounded:
		stage.spaced(world)
		return nil
	}
	if err := stage.terrain(read); err != nil {
		return err
	}
	if planeted {
		if err := stage.dusk(read, world); err != nil {
			return err
		}
	} else if err := stage.light(read); err != nil {
		return err
	}
	if err := stage.stand(read); err != nil {
		return err
	}
	if err := stage.atmosphere(read); err != nil {
		return err
	}
	if err := stage.prepareFronts(read); err != nil {
		return err
	}
	return stage.prepareBeds(read)
}

// filmed is what was asked with the set's film and light in it.
func filmed(read set.Set, given asked) (asked, error) {
	film, has := read.Film()
	if !has {
		return given, fmt.Errorf("%s says no film", given.set)
	}
	given.width, given.height, given.samples = film.Size()
	given.exposure = film.Exposure()
	given.night = read.Light() == "moon"
	given.cachedLighting = read.VolumeLighting() == "cached"
	if read.Renderer() != "" {
		given.nativeMeshes = read.Renderer() == "gpu-native"
		given.nativeShadows = read.Renderer() == "gpu-native-shadows"
		given.onGPU = read.Renderer() == "gpu" || given.nativeShadows || given.nativeMeshes
	}
	if read.HasSeed() {
		given.seed = read.Seed()
	}
	if film.Wide() > 0 {
		given.wide = film.Wide()
	}
	if rig, has := film.Rig(); has {
		if given.cachedLighting {
			return given, fmt.Errorf(
				"volume-lighting: cached requires a pinhole film")
		}
		return rigged(rig, given)
	}
	return given, nil
}

// where is where a named thing stands, for the set's cameras.
func (stage *staging) where(name string) (maths.Vec, bool) {
	at, found := stage.places[name]
	return at, found
}

// light is the sky and its light: the full moon where the set says
// where it stands, or else the sun or the moon as hillside's own are,
// picked.
func (stage *staging) light(read set.Set) error {
	if read.Light() == "none" {
		stage.world.Sky = sky.Gradient{}
		return nil
	}
	if moon, has := read.Moonlit(); has && read.Light() == "moon" {
		way, err := stage.moonlit(moon)
		if err != nil {
			return err
		}
		return stage.skies(read, way)
	}
	if sun, has := read.Sunlit(); has && read.Light() == "sun" {
		azimuth, altitude := sun.Where()
		if altitude <= 0 {
			return fmt.Errorf("%s: daylight sun must be above the horizon", stage.given.set)
		}
		stage.world.Sky = skyFor(stage.given)
		for index, source := range []struct {
			toward maths.Vec
			power  float64
		}{
			{skyward(azimuth, altitude), day}, {fillWay, day * fill},
		} {
			colour := maths.Grey(source.power)
			if index == placementZero {
				colour = tintedSun(sun, source.power)
			}
			lamp, err := emit.NewDistant(source.toward, colour)
			if err != nil {
				return err
			}
			stage.world.Lamp(lamp)
			stage.made.lamps++
		}
		return nil
	}
	level := day
	stage.world.Sky = skyFor(stage.given)
	if stage.given.night {
		level = moonlit
	}
	err := lamps(stage.world, stage.given, level)
	stage.made.lamps = len(stage.world.Lamps)
	return err
}

// skyward is the way to a body at an azimuth and an altitude, in
// degrees: x east, y up, and north toward minus z.
func skyward(azimuth, altitude float64) maths.Vec {
	round, up := azimuth*maths.Degree, altitude*maths.Degree
	return maths.V(math.Sin(round)*math.Cos(up), math.Sin(up),
		-math.Cos(round)*math.Cos(up))
}

// moonlit is the full moon where the set puts it: its face, glowing as
// its geometric albedo says, and its light from its way, each through
// the air from the ground's lowest height; and its way. a moon at or
// below the horizon is refused.
func (stage *staging) moonlit(moon set.Moon) (maths.Vec, error) {
	azimuth, altitude := moon.Where()
	if !(altitude > 0) {
		return maths.Vec{}, fmt.Errorf("%s: the moon is not above the "+
			"horizon", stage.given.set)
	}
	way := skyward(azimuth, altitude)
	height := measure.Metres(stage.lowest)
	rise, exposure := measure.Radians(altitude*maths.Degree),
		stage.given.exposure
	if moon.Disc() {
		face, err := moonlight.Disc().Through(height, rise)
		if err != nil {
			return way, err
		}
		ball, err := globe.New(moonlight.Moon().Radius().InMetres())
		if err != nil {
			return way, err
		}
		far := moonlight.Moon().Distance().InMetres()
		material, err := stage.faced(moon, way, face.Picture(exposure))
		if err != nil {
			return way, err
		}
		// The visual disc must not block its paired distant light.
		stage.world.Add(thing.Thing{Shape: ball, Material: material,
			Attrs:  thing.Attrs{Clear: true},
			Motion: thing.Still(maths.Subspace{Origin: way.Scale(far)})})
	}
	arriving, err := moonlight.Full().Through(height, rise)
	if err != nil {
		return way, err
	}
	lamp, err := emit.NewDistant(way, arriving.Picture(exposure))
	if err != nil {
		return way, err
	}
	stage.world.Lamp(lamp)
	stage.made.lamps++
	return way, nil
}

// skies is the sky the set's lights make, seen from the ground's
// lowest height: the full moon's, from its way, and the twilight a sun
// below the horizon lights, where the set says one stands.
func (stage *staging) skies(read set.Set, moon maths.Vec) error {
	height, exposure := measure.Metres(stage.lowest), stage.given.exposure
	sky, err := moonlight.NewSky(height, moon, moonlight.Full(), exposure)
	if err != nil {
		return err
	}
	all := moonlight.Skies{sky}
	if sun, has := read.Sunlit(); has {
		dusk, err := stage.twilit(sun)
		if err != nil {
			return err
		}
		all = append(all, dusk)
	}
	stage.world.Sky = all
	return nil
}

// twilit is the sky a sun below the horizon lights. a sun the earth does not
// hide from the ground's lowest height is refused, since its light on the
// ground and its disc are not built yet.
//
// from a height the horizon dips: the sun must stand lower than the dip, the
// angle whose cosine is the earth's radius over the radius there. ground below
// the sea is taken at the sea, where the dip is zero.
func (stage *staging) twilit(sun set.Sun) (*moonlight.Sky, error) {
	azimuth, altitude := sun.Where()
	out := earthRadius / (earthRadius + math.Max(0, stage.lowest))
	dip := math.Acos(out) / maths.Degree
	if !(altitude < -dip) {
		return nil, fmt.Errorf("%s: the sun at %g degrees is not "+
			"hidden by the earth, whose horizon dips %.2f degrees "+
			"here", stage.given.set, altitude, dip)
	}
	return moonlight.NewSky(measure.Metres(stage.lowest),
		skyward(azimuth, altitude), moonlight.Sunlight(),
		stage.given.exposure)
}

// terrain is the set's ground from the model its source key pins,
// checked as pinned on the reader's shelf before it is read.
func (stage *staging) terrain(read set.Set) error {
	ground, has := read.Ground()
	if !has {
		return fmt.Errorf("%s says no terrain", stage.given.set)
	}
	shelf, err := pinned(ground.From(), filepath.Dir(stage.given.set))
	if err != nil {
		return err
	}
	raster, err := terrain.Open(filepath.Join(shelf, ground.File()))
	if err != nil {
		return err
	}
	defer raster.Close()
	west, north, across, down, step := ground.Window()
	stage.west, stage.north = west, north
	stage.ground, err = raster.Window(west, north,
		int(across/step)+1, int(down/step)+1, step)
	if err == nil {
		stage.ground, err = stage.ground.Smoothed(ground.Smooth())
	}
	if err != nil {
		return err
	}
	surface, lowest, err := stage.ground.Ground()
	if err != nil {
		return err
	}
	stage.lowest = lowest
	stage.groundExtent = surface.Bounds()
	stage.given.floor = lowest
	stage.world.Add(thing.Thing{Shape: surface,
		Material: &thing.Material{Name: "basalt", Colour: basalt}})
	return nil
}

// pinned is the shelf, once the source is found pinned on it, in the
// sources.yaml nearest the set file: in its directory or one above.
func pinned(key, near string) (string, error) {
	shelf, _, err := pinnedSource(key, near)
	return shelf, err
}

// pinnedSource returns the exact catalogue used for the pin check as well
// as the shelf, so sequence fingerprints do not repeat the path search.
func pinnedSource(key, near string) (string, string, error) {
	asset, err := pinnedAssetAt(key, near)
	return asset.shelf, asset.catalogue, err
}

// pinnedAsset keeps the checked path and catalogue from one lookup.
type pinnedAsset struct{ shelf, catalogue, path string }

// pinnedAssetAt checks exactly the catalogue entry that supplies the path.
func pinnedAssetAt(key, near string) (pinnedAsset, error) {
	shelf, err := sources.Here()
	if err != nil {
		return pinnedAsset{}, err
	}
	listed, err := nearest(near, "sources.yaml")
	if err != nil {
		return pinnedAsset{}, err
	}
	list, err := sources.Read(listed)
	if err != nil {
		return pinnedAsset{}, err
	}
	source, found := list.Find(key)
	if !found {
		return pinnedAsset{}, fmt.Errorf("%s is not in sources.yaml", key)
	}
	kept := sources.NewShelf(shelf, nil)
	if _, err := kept.Check(source); err != nil {
		return pinnedAsset{}, err
	}
	return pinnedAsset{shelf: shelf, catalogue: listed, path: kept.Path(source)}, nil
}

// faced is the moon's material: its glow, and its face where the set
// names a mosaic, read from the shelf once its source is found pinned.
func (stage *staging) faced(moon set.Moon, way maths.Vec,
	glow maths.Light) (*thing.Material, error) {
	material := &thing.Material{Name: "moon", Glow: glow}
	if moon.Face() == "" {
		return material, nil
	}
	asset, err := pinnedAssetAt(moon.Face(), filepath.Dir(stage.given.set))
	if err != nil {
		return nil, err
	}
	face, err := moonlight.NewFace(asset.path, way, glow)
	if err != nil {
		return nil, err
	}
	material.Shine = face
	return material, nil
}

// nearest is a file of a name in a directory or the nearest above it.
func nearest(dir, name string) (string, error) {
	at, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(at, name)); err == nil {
			return filepath.Join(at, name), nil
		}
		above := filepath.Dir(at)
		if above == at {
			return "", fmt.Errorf("no %s in %s or above it", name, dir)
		}
		at = above
	}
}

// at is a point of the scene for an easting and a northing in the
// model's metres, on the ground there.
func (stage *staging) at(east, north float64) (maths.Vec, error) {
	across, down, step := stage.ground.Size()
	column := int(math.Round((east - stage.west) / step))
	row := int(math.Round((stage.north - north) / step))
	if column < 0 || row < 0 || column >= across || row >= down {
		return maths.Vec{}, fmt.Errorf("%.1f e, %.1f n is off the ground",
			east, north)
	}
	height := stage.ground.At(column, row) - stage.lowest
	return maths.V(east-stage.west, height, stage.north-north), nil
}

// stand stands every thing and every stand the set names on its ground.
func (stage *staging) stand(read set.Set) error {
	for i, each := range read.Things() {
		east, north := each.At()
		place, err := stage.at(east, north)
		if err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
		place.Y += each.Height()
		stage.places[each.Name()] = place
		stage.note(each.Name(), each.Kind(), place)
		_, err = stage.putRecipe(each.Kind(), each.Recipe(), each.Complexity(),
			each.Seed(), uint64(i), place, each.Alight(), each.Name())
		if err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
	}
	for _, each := range read.Stands() {
		if err := stage.scatter(each); err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
	}
	for _, each := range read.Texts() {
		if err := stage.letters(each); err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
	}
	return stage.ecosystems(read)
}

// tries is how many places a seed offers a stand's thing before the
// stand is said to have no room in its band: enough for a band a
// fraction of a percent of a large window, and cheap beside a picture
const tries = 4096

// scatter stands a stand's things at places its seed picks over the
// ground, each where the ground is between the stand's heights, the
// first of them alight as many as it says.
func (stage *staging) scatter(each set.Stand) error {
	count, seed := each.Count()
	places := rand.New(rand.NewPCG(seed, 1))
	var grove []groveTree
	for i := range count {
		place, err := stage.inBand(each, places)
		if err != nil {
			return err
		}
		stage.note(each.Name(), each.Kind(), place)
		pick := seed*standVariety + uint64(i%standVariety)
		var bounds maths.Box
		bounds, err = stage.putRecipe(each.Kind(), each.Recipe(), each.Complexity(),
			pick, uint64(i), place, i < each.Alight(), "")
		if err != nil {
			return err
		}
		grove = append(grove, groveTree{at: place, bounds: bounds})
	}
	if stage.fuels == nil {
		stage.fuels = map[string][]groveTree{}
	}
	stage.fuels[each.Name()] = grove
	return stage.grove(each, grove)
}

// inBand is a place the generator offers on the ground, short of its
// edge and inside the stand's wedge if it has one, where the ground is
// between the stand's heights.
func (stage *staging) inBand(each set.Stand,
	places *rand.Rand) (maths.Vec, error) {
	above, below := each.Band()
	offer, err := stage.offering(each, places)
	if err != nil {
		return maths.Vec{}, err
	}
	for range tries {
		place, err := stage.at(offer())
		if err != nil {
			continue
		}
		height := place.Y + stage.lowest
		if height > above && height < below && stage.allowed(each, place) {
			return place, nil
		}
	}
	return maths.Vec{}, fmt.Errorf("no ground between %v and %v m in %d "+
		"tries", above, below, tries)
}

// offering is how the generator offers a stand places, east and north:
// evenly over the ground short of its edge, or evenly over the ground
// of its wedge, seen from where its thing stands.
func (stage *staging) offering(each set.Stand,
	places *rand.Rand) (func() (float64, float64), error) {
	of, from, to, near, far, within := each.Within()
	if !within {
		across, down, step := stage.ground.Size()
		wide := float64(across-1) * step
		deep := float64(down-1) * step
		return func() (float64, float64) {
			return stage.west + step + places.Float64()*(wide-2*step),
				stage.north - step - places.Float64()*(deep-2*step)
		}, nil
	}
	seen, known := stage.places[of]
	if !known {
		return nil, fmt.Errorf("within sight of %s, which stands nowhere "+
			"on the ground", of)
	}
	east, north := stage.west+seen.X, stage.north-seen.Z
	turn := math.Mod(to-from+360, 360)
	if turn == 0 {
		turn = 360
	}
	return func() (float64, float64) {
		bearing := (from + places.Float64()*turn) * math.Pi / 180
		out := math.Sqrt(near*near + places.Float64()*(far*far-near*near))
		return east + out*math.Sin(bearing), north + out*math.Cos(bearing)
	}, nil
}

// putRecipe keeps named recipes and complexity levels in separate stock.
// An optical recipe's pointing is already baked into its assembly.
func (stage *staging) putRecipe(kind, recipe, detail string, seed, index uint64,
	place maths.Vec, burning bool, name string) (maths.Box, error) {
	if kind == "fire" {
		return stage.putFireRecipe(recipe, seed, index, place, name)
	}
	identity := kind
	if recipe != "" {
		identity = "recipe:" + recipe
	}
	key := stocked{name: identity, pick: int(seed), detail: detail}
	one, known := stage.have[key]
	if !known {
		fresh, err := stage.constructed(kind, recipe, detail, seed)
		if err != nil {
			return maths.Box{}, err
		}
		one = *fresh
		stage.have[key] = one
		stage.made.kept += one.stored
	}
	turn := stage.placementTurn(kind, name, seed, index)
	at := maths.Subspace{Origin: place,
		Orientation: maths.About(maths.YAxis, turn), Scale: 1}
	if name != "" {
		stage.radii[name] = horizontalRadius(one.bounds)
	}
	return one.bounds, stage.add(one, kind, at, seed, burning)
}

// allowed leaves clearance beyond the excluded object's enclosing circle.
func (stage *staging) allowed(each set.Stand, place maths.Vec) bool {
	name, clearance := each.Exclude()
	if name == "" {
		return true
	}
	centre := stage.places[name]
	return math.Hypot(place.X-centre.X, place.Z-centre.Z) >=
		clearance+stage.radii[name]
}

// horizontalRadius encloses every yaw of the physical extent.
func horizontalRadius(box maths.Box) float64 {
	x := math.Max(math.Abs(box.Min.X), math.Abs(box.Max.X))
	z := math.Max(math.Abs(box.Min.Z), math.Abs(box.Max.Z))
	return math.Hypot(x, z)
}

// add stands a kept thing at a place with its lamps and its fire.
func (stage *staging) add(one kept, kind string, at maths.Subspace,
	seed uint64, burning bool) error {
	if stage.made.kept > stage.given.faces {
		return fmt.Errorf("%w: past -faces of %d", errTooMany,
			stage.given.faces)
	}
	for _, each := range stood(one.things, at) {
		stage.world.Add(each)
	}
	if kind == "campfire" || kind == "bonfire" {
		stage.unretained = "legacy fire lamps are sampled at construction; use fire recipes"
	}
	stage.made.things++
	stage.made.faces += one.faces
	lit, err := one.lamps(at)
	if err != nil {
		return err
	}
	for _, lamp := range lit {
		stage.world.Lamp(lamp)
		stage.made.lamps++
	}
	if !burning {
		return nil
	}
	stage.made.burning++
	flames, err := alight(stage.world, one, at, seed, stage.given)
	stage.made.lamps += flames
	return err
}
