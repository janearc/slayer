package session

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/airlight"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/trace"
)

// with -set, the set file says the world and the film; these flags say
// how it is drawn, kept, proved and seen, and may be given with it: the
// size and the rays among them, which stand in place of the film's
var withSet = map[string]bool{"set": true, "w": true, "h": true,
	"n": true, "lake": true, "moon": true,
	"fetch-terrain": true, "refresh-terrain": true, "check-terrain": true,
	"moonglow": true, "giants": true, "lakeglow": true, "close": true,
	"rig":       true,
	"closeglow": true,
	"out":       true,
	"raw":       true,
	"prove":     true, "proof": true, "pixel": true, "many": true,
	"cpuprofile": true, "gpu": true, "at": true, "wide": true,
	"sky": true, "places": true, "sparks-for": true,
	"sparks-sphere": true, "shell": true, "shell-at": true,
	"fireball": true, "fireball-at": true, "fastlight": true,
	"vibes": true, "exact-sky": true}

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
	trial          *planTrial
	firebrandShape *mesh.Mesh
	unretained     string
	fronts         []retainedFront
	beds           []retainedBed
	weathers       []*weatherOnFilm
	deformations   []retainedDeformation
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
	near           *nearGround
	groundExtent   maths.Box
	west           float64
	north          float64
	lowest         float64
	groundlessAir  bool
	airAltitude    float64
	// airedSun is the exact value and insertion index emitted by aired; it
	// is never inferred by matching floating-point light values.
	airedSun *stagedDistantSource
	// lunarSource is the exact modeled Moon emitter inserted by moonlit.
	lunarSource *stagedDistantSource
	// airedSky is the exact atmosphere sky installed beside that source.
	airedSky thing.Sky
	// airFieldsProvider is a narrow staging seam for deterministic tests;
	// production leaves it nil and reads the pinned cache or solves the plan.
	airFieldsProvider func(string) (*airlight.Fields, airlight.Plan,
		cacheNote, error)
	// sight is the set's camera as a leveller sees it, levellers the
	// levellers made from it, by ladder, and sized the boxes of things a
	// levelled stand was measured by, once the first levelled thing asks.
	sight     *strategies.Eye
	levellers map[string]cities.Leveller
	sized     map[stocked]maths.Box
	// joined is the number each run of a contour stand is drawn from, by
	// the stand's name and the run's number among its sites.
	joined map[string][]uint64
	// facts is what is known of each design without making it, and rungs
	// how many things were made at each complexity, where scrooge is asked
	// for (see members_detail.go).
	facts map[memberKey]memberFact
	rungs rungTally
	// constructions is the row each kind and recipe asked for gave, or
	// the refusal (see recipe_construction.go).
	constructions map[constructKey]constructedRow
	// warm is the things made ahead of the loop that stands the set's
	// things, while that loop runs and nil at any other time (see
	// warm_things.go).
	warm *warmed
}

// standing is one thing as the stage stood it: the thing's name or its
// stand's, its kind, and where, east and north on the terrain's grid and
// the ground's height there, in metres.
type standing struct {
	of, kind            string
	east, north, height float64
}

// reportedKind identifies a plant by its authored species in reports. The
// construction category remains unchanged for dispatch and every other recipe.
func (stage *staging) reportedKind(kind, recipeName string) string {
	if recipe, found := stage.recipes.Recipe(recipeName); found {
		if recipe.IsPlant() {
			return recipe.PlantSpecies()
		}
		if recipe.IsAlga() {
			return recipe.AlgaSpecies()
		}
	}
	return kind
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
	return writeStood(path, stage.stood)
}

// writeStood writes a table of where things stood to a file: a header
// line, then a line each, of what it is of, its kind, east, north and
// height. both -places and Places write it.
func writeStood(path string, stood []standing) error {
	var lines strings.Builder
	lines.WriteString("of,kind,east,north,height\n")
	for _, each := range stood {
		fmt.Fprintf(&lines, "%q,%s,%.1f,%.1f,%.1f\n", each.of, each.kind,
			each.east, each.north, each.height)
	}
	return os.WriteFile(path, []byte(lines.String()), 0o644)
}

// refuseFantasy is the refusal of a set that states a fantasy and has no sky
// or ground for it to depart from, or nil when it has none to refuse.
func refuseFantasy(read set.Set, given asked) error {
	if !worldOnly(read) || len(read.Fantasy()) == nothing {
		return nil
	}
	return fmt.Errorf("%s: fantasy departs "+
		"from the hillside's sky and ground, and this set "+
		"has neither", given.set)
}

// staged is the world a set file asks for, and what was asked, with the
// set's film and its first camera in it.
func staged(given asked) (*thing.World, tally, asked, error) {
	read, err := given.readSet()
	if err != nil {
		return nil, tally{}, given, err
	}
	if err := refuseTerrainOpticalUnit(read); err != nil {
		return nil, tally{}, given, err
	}
	if authored, has := read.World(); has {
		if surface, ocean := authored.OceanSurface(); ocean {
			report, err := oceanSpectrumEstimate(surface)
			if err != nil {
				return nil, tally{}, given, fmt.Errorf("%s: %w", given.set, err)
			}
			given.oceanSpectrum = &report
		}
	}
	given, err = filmed(read, given)
	if err == nil {
		err = refuseFantasy(read, given)
	}
	if err != nil {
		return nil, tally{}, given, err
	}
	if worldOnly(read) {
		return stagedWorld(read, given).results()
	}
	given, departures := fantasied(read, given)
	stage := &staging{given: given, world: thing.NewWorld(),
		have: stock{}, places: map[string]maths.Vec{},
		radii: map[string]float64{}, recipes: read}
	stage.world.VolumeIllumination = read.VolumeIllumination()
	retained := false
	defer func() {
		if !retained {
			(&liveScene{beds: stage.beds}).closeBeds()
		}
	}()
	world, inSpace := read.Planet()
	firstLamp := len(stage.world.Lamps)
	err = stage.place(read, world, inSpace)
	if err == nil && len(stage.deformations) > 0 {
		err = deformingExclusionRefusal(read)
	}
	if err == nil {
		err = stage.tagOceanDirectMoon(read)
	}
	if err == nil {
		err = stage.tagOceanDirectSun(read, firstLamp)
	}
	if err == nil {
		err = stage.stepped("lasers", stage.lasers, read)
	}
	if err == nil && given.lake > 0 {
		err = stage.flooded(given.lake)
	}
	if err == nil {
		err = stage.stepped("halves", stage.halved, read)
	}
	if err == nil {
		err = stage.stepped("weathering", stage.weathered, read)
	}
	if err == nil {
		err = stage.stepped("flat air", stage.flatAir, read)
	}
	if err != nil {
		return nil, tally{}, given, err
	}
	if given.placed != "" {
		if err := stage.written(given.placed); err != nil {
			return nil, tally{}, given, err
		}
	}
	eye, err := stage.eye(read)
	if err != nil {
		return nil, tally{}, given, err
	}
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
	if len(stage.deformations) > 0 {
		if err := deformingRefusal(stage.given, false); err != nil {
			return nil, tally{}, given, err
		}
		// Applies to the older sequence path as well as Session films.
		stage.given.sections.keep = false
	}
	stage.given.live = stage.retain()
	if len(stage.fronts) > 0 || len(stage.beds) > 0 || len(stage.deformations) > 0 {
		stage.world, stage.made, err = stage.given.live.frameAt(stage.given.at)
		if err != nil {
			return nil, tally{}, given, err
		}
	}
	retained = true
	stage.given.departures = append(departures, stage.given.departures...)
	stage.given.stood, stage.given.stoodKept = stage.stood, true
	stage.given.authoredOrigins = stage.places
	return stage.world, stage.made, stage.given, nil
}

// place stages what the set stands on: set in space, its craft afloat;
// a planet alone, the planet and the darkness; on the ground, its
// terrain, its light and its stands, and a planet in its sky where the
// set says one stands.
func (stage *staging) place(read set.Set, world set.Planet,
	planeted bool) error {
	_, grounded := read.Ground()
	_, _, _, inSky := world.InSky()
	switch {
	case read.InSpace():
		return stage.afloat(read)
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
	if reference, has := read.GroundlessAir(); has {
		return stage.placeGroundlessAir(read, reference)
	}
	if err := stage.stepped("terrain", stage.terrain, read); err != nil {
		return err
	}
	if elevation, floods := stillWater(read); floods {
		if err := stage.flooded(elevation); err != nil {
			return err
		}
	}
	if err := stage.stagedLight(read, world, planeted); err != nil {
		return err
	}
	if err := stage.stand(read); err != nil {
		return err
	}
	if err := stage.stepped("atmosphere", stage.atmosphere,
		read); err != nil {
		return err
	}
	if err := stage.stepped("plumes", stage.plumes, read); err != nil {
		return err
	}
	if err := stage.stepped("fronts", stage.prepareFronts,
		read); err != nil {
		return err
	}
	return stage.stepped("beds", stage.prepareBeds, read)
}

// stagedLight stages the set's light, a planet's dusk where it stands
// under one, as a step of its own.
func (stage *staging) stagedLight(read set.Set, world set.Planet,
	planeted bool) error {
	if err := stage.given.steps.step("light", nothing, nothing); err != nil {
		return err
	}
	if planeted {
		return stage.dusk(read, world)
	}
	return stage.light(read)
}

// stepped is a step of staging done once staging is come to it and not
// told to stop.
func (stage *staging) stepped(what string, do func(set.Set) error,
	read set.Set) error {
	if err := stage.given.steps.step(what, nothing, nothing); err != nil {
		return err
	}
	return do(read)
}

// stillWater is the level of the still-water globe a set asks for, and
// whether it asks: a terrain `sea` with no `ocean`. An ocean is the set's
// sea, and a flat sea under it is none, so the terrain's `sea` then stages
// no globe, and its level stays for whatever else reads it.
func stillWater(read set.Set) (float64, bool) {
	ground, has := read.Ground()
	if !has {
		return 0, false
	}
	elevation, said := ground.Sea()
	if !said {
		return 0, false
	}
	if world, has := read.World(); has {
		if _, ocean := world.OceanSurface(); ocean {
			return 0, false
		}
	}
	return elevation, true
}

// filmed is what was asked with the set's film and light in it.
func filmed(read set.Set, given asked) (asked, error) {
	film, has := read.Film()
	if !has && !(worldOnly(read) && saysFilm(film)) {
		return given, fmt.Errorf("%s says no film", given.set)
	}
	if has {
		given = given.sized(film.Size())
	}
	if !given.said.exposure {
		given.exposure = film.Exposure()
	}
	given.print = stagedPrint(film)
	given.sections.over, given.sections.size, given.sections.keep =
		film.Sections()
	given.sections.drift = film.Drift()
	given.filter = film.Filter()
	given.night = read.Light() == "moon"
	given.cachedLighting = read.VolumeLighting() == "cached"
	given = given.resolved(read.Flag)
	if read.Renderer() != "" && !given.said.renderer {
		given = given.renderedBy(read.Renderer())
	}
	if read.HasSeed() {
		given.seed = read.Seed()
	}
	if film.Wide() > 0 && !given.said.wide {
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

// renderedBy is what was asked, drawn by the renderer of a name: the
// cpu, or the gpu by its own traversal, its own shadows or its own
// meshes and shadows. A name it does not know is the cpu; the set's
// schema and the job's take only these four.
func (given asked) renderedBy(name string) asked {
	given.nativeMeshes = name == "gpu-native"
	given.nativeShadows = name == "gpu-native-shadows"
	given.onGPU = name == "gpu" || given.nativeShadows ||
		given.nativeMeshes
	return given
}

// eye is where the set's camera stands: the one the job names, or else
// the first, every camera of the set made ready as before.
func (stage *staging) eye(read set.Set) (maths.Subspace, error) {
	if stage.given.camera != "" {
		shot, err := namedShot(read, stage.given.camera, stage.where)
		if err != nil {
			return maths.Subspace{}, fmt.Errorf("%s: %w", stage.given.set,
				err)
		}
		stage.given = stage.given.lensOf(shot)
		if err := stage.given.withShotMedia(shot, worldMetresPerUnit(read)); err != nil {
			return maths.Subspace{}, fmt.Errorf("%s: %w", stage.given.set, err)
		}
		if err := stage.given.resolveOceanInitialMedium(stage.world); err != nil {
			return maths.Subspace{}, fmt.Errorf("camera %s: %w", shot.Name(), err)
		}
		return shot.Eye(), nil
	}
	shots, err := read.Shots(stage.where)
	if err != nil || len(shots) == 0 {
		return maths.Subspace{}, fmt.Errorf("%s: no camera: %v",
			stage.given.set, err)
	}
	stage.given = stage.given.lensOf(shots[0])
	if err := stage.given.withShotMedia(shots[0], worldMetresPerUnit(read)); err != nil {
		return maths.Subspace{}, fmt.Errorf("%s: %w", stage.given.set, err)
	}
	if err := stage.given.resolveOceanInitialMedium(stage.world); err != nil {
		return maths.Subspace{}, fmt.Errorf("camera %s: %w", shots[0].Name(), err)
	}
	return shots[0].Eye(), nil
}

// resolveOceanInitialMedium connects the camera's authored named medium to
// the ocean boundary's configured copy. The ocean-level sky sample budget is
// attached only to that boundary; matching base coefficients, units, bounds,
// optical index and rank must still agree before the camera stack adopts it.
func (given *asked) resolveOceanInitialMedium(world *thing.World) error {
	if given == nil || world == nil || world.OceanSurface == nil ||
		world.OceanSurface.Boundary == nil ||
		world.OceanSurface.Boundary.Material == nil {
		return nil
	}
	material := world.OceanSurface.Boundary.Material
	wanted := material.ClearMedium
	if wanted == nil || wanted.DiffuseSkyInscatterSamples() == 0 {
		return nil
	}
	for index := range given.initialMedia {
		initial := &given.initialMedia[index]
		got := initial.ClearMedium
		if got == nil || got.Name() != wanted.Name() {
			continue
		}
		if initial.Index != material.Dense() || initial.Rank != material.Rank ||
			!sameClearMediumBase(got, wanted) {
			return fmt.Errorf("initial ocean medium %q does not match its configured boundary", wanted.Name())
		}
		if got.DiffuseSkyInscatterSamples() != 0 &&
			got.DiffuseSkyInscatterSamples() != wanted.DiffuseSkyInscatterSamples() {
			return fmt.Errorf("initial ocean medium %q has %d sky samples, boundary has %d", wanted.Name(), got.DiffuseSkyInscatterSamples(), wanted.DiffuseSkyInscatterSamples())
		}
		initial.ClearMedium = wanted
	}
	return nil
}

// sameClearMediumBase reports whether two clear media share their optical
// base: name, absorption and scattering.
func sameClearMediumBase(one, other *thing.ClearMedium) bool {
	return one != nil && other != nil && one.Name() == other.Name() &&
		one.Absorption() == other.Absorption() &&
		one.Scattering() == other.Scattering() &&
		one.Extinction() == other.Extinction() &&
		one.Phase() == other.Phase() &&
		one.MetresPerUnit() == other.MetresPerUnit() &&
		one.InscatterSteps() == other.InscatterSteps() &&
		one.MaxInscatterDistanceMetres() ==
			other.MaxInscatterDistanceMetres()
}

// withShotMedia carries a set camera's explicitly authored start stack to
// tracing. A moving camera would leave its declared containing medium, so
// that combination is refused at stage for either backend.
func (given *asked) withShotMedia(shot set.Shot,
	metresPerUnit float64) error {
	media := shot.InitialMedia()
	if len(media) == 0 {
		return nil
	}
	withMedia := *given
	withMedia.initialMedia = make([]trace.InitialMedium, len(media))
	if err := refuseGPUInitialMediaAperture(withMedia); err != nil {
		return fmt.Errorf("camera %s: %w", shot.Name(), err)
	}
	if given.cameraPath {
		return fmt.Errorf("camera %s has initial media and a moving job camera; a fixed start medium cannot follow camera motion", shot.Name())
	}
	converted := make([]trace.InitialMedium, len(media))
	for index, each := range media {
		converted[index] = trace.InitialMedium{Index: each.Index,
			Rank: each.Rank, Source: each.WaterOpticalSource()}
		converted[index].WaterComposition, _ = each.WaterComposition()
		if name, extinction, optical := each.OpticalMedium(); optical {
			absorption, scattering, phaseG, steps, maxDistance, participating :=
				each.ParticipatingMedium()
			var medium *thing.ClearMedium
			var err error
			if participating {
				medium, err = thing.NewClearMediumWithScattering(name,
					maths.RGB(absorption[0], absorption[1], absorption[2]),
					maths.RGB(scattering[0], scattering[1], scattering[2]),
					phaseG, metresPerUnit, steps, maxDistance)
			} else {
				medium, err = thing.NewClearMedium(name, maths.RGB(
					extinction[0], extinction[1], extinction[2]), metresPerUnit)
			}
			if err != nil {
				return fmt.Errorf("camera %s initial medium %s: %w",
					shot.Name(), name, err)
			}
			if water, hasWater := each.WaterComposition(); hasWater {
				medium, err = medium.WithWaterComposition(water, each.WaterOpticalSource())
				if err != nil {
					return fmt.Errorf("camera %s initial medium %s: %w",
						shot.Name(), name, err)
				}
			}
			converted[index].ClearMedium = medium
		}
	}
	given.initialMedia = converted
	return nil
}

// worldMetresPerUnit is how far one coordinate unit of the set's world
// travels in metres. A set without a world uses the renderer's metre unit.
func worldMetresPerUnit(read set.Set) float64 {
	world, has := read.World()
	if !has {
		return 1
	}
	return world.Unit()
}

// refuseTerrainOpticalUnit keeps a metres-authored terrain frame from being
// combined with a world whose coordinates claim a different metres scale.
// The terrain path currently rebases world-half coordinates in metres but
// does not scale ordinary world-half shapes.
func refuseTerrainOpticalUnit(read set.Set) error {
	if _, grounded := read.Ground(); !grounded {
		return nil
	}
	world, has := read.World()
	if !has || world.Unit() == 1 {
		return nil
	}
	for _, material := range world.Materials() {
		clear, has := material.Clear()
		if !has {
			continue
		}
		if _, _, optical := clear.OpticalMedium(); optical {
			return fmt.Errorf("terrain-backed named clear media require world.unit 1 because terrain and world-half coordinates are metres; got %g", world.Unit())
		}
	}
	return nil
}

// lensOf is what was asked, seen through the camera's own lens where it
// says how wide one is and the job did not say a width of its own.
func (given asked) lensOf(shot set.Shot) asked {
	if wide := shot.Lens().Wide; wide > 0 && !given.said.wide {
		given.wide = wide / maths.Degree
	}
	return given
}

// namedShot is the set's camera of a name, made ready. A name the set
// has no camera by is refused, with the names it has.
func namedShot(read set.Set, name string, where set.Where) (set.Shot,
	error) {
	names := read.Cameras()
	if !slices.Contains(names, name) {
		return set.Shot{}, fmt.Errorf("the set has no camera %q; its "+
			"cameras are %s", name, strings.Join(names, ", "))
	}
	return read.Shot(name, where)
}

// where is where a named thing stands, for the set's cameras.
func (stage *staging) where(name string) (maths.Vec, bool) {
	at, found := stage.places[name]
	return at, found
}

// sunDown is whether the set is lit by a sun that its place, date and
// clock time put at or below the horizon. The sun is down: it lights
// nothing, and the set has the other light it says.
func sunDown(read set.Set) bool {
	sun, has := read.Sunlit()
	return has && read.Light() == "sun" && sun.Down()
}

// light is the sky and its light: the full moon where the set says
// where it stands; the sun through the air the set names, computed; or
// else the sun or the moon as hillside's own are, picked.
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
	if sunDown(read) {
		stage.world.Sky = sky.Gradient{}
		return nil
	}
	if read.Air() != "" {
		return stage.aired(read)
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
		ball, err := globe.New(moonlight.Moon().Radius().InMetres() *
			stage.given.moonScale)
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
	stage.lunarSource = &stagedDistantSource{index: len(stage.world.Lamps), lamp: lamp}
	stage.world.Lamp(lamp)
	stage.made.lamps++
	return way, nil
}

// skies is the sky the set's lights make, seen from the ground's
// lowest height: the full moon's, from its way, and the twilight a sun
// below the horizon lights, where the set says one stands.
func (stage *staging) skies(read set.Set, moon maths.Vec) error {
	height, exposure := measure.Metres(stage.lowest), stage.given.exposure
	sky, note, err := keptSky(height, moon, moonlight.Full(), exposure,
		"the moonlit sky")
	stage.given.cached = append(stage.given.cached, note)
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
	sky, note, err := keptSky(measure.Metres(stage.lowest),
		skyward(azimuth, altitude), moonlight.Sunlight(),
		stage.given.exposure, "the twilit sky")
	stage.given.cached = append(stage.given.cached, note)
	return sky, err
}

// terrain is the set's ground from the model its source key pins,
// checked as pinned on the reader's shelf before it is read.
func (stage *staging) terrain(read set.Set) error {
	ground, has := read.Ground()
	if !has {
		return fmt.Errorf("%s says no terrain", stage.given.set)
	}
	asset, err := pinnedAssetAt(ground.From(), ground.File(),
		filepath.Dir(stage.given.set), fetchFlag)
	if err != nil {
		return err
	}
	stage.given.cached = append(stage.given.cached, asset.checked)
	if err := stage.cutGround(ground, asset); err != nil {
		return err
	}
	if finer, nearer := ground.Near(); nearer {
		if err := stage.cutNear(ground, finer, asset); err != nil {
			return err
		}
	}
	stage.lowest = stage.groundLowest()
	stage.given.floor = stage.lowest
	stage.groundExtent, err = stage.addGround(groundLook(ground,
		stage.lowest))
	return err
}

// groundLook is what the ground is made of: the cover the set says, laid
// by rule over the ground whose lowest height is lowest, or the
// reflectance the set says, or else hillside's own basalt.
func groundLook(ground set.Terrain, lowest float64) *thing.Material {
	if level, layers, covered := ground.Cover(); covered {
		name, source := surfacesNamed(layers)
		return &thing.Material{Name: name, Source: source,
			Colour:  surfaceLight(layers[0]),
			Texture: coveredGround(level, lowest, layers)}
	}
	red, green, blue, said := ground.Reflectance()
	if !said {
		return &thing.Material{Name: "basalt", Colour: basalt}
	}
	return &thing.Material{Name: "ground",
		Colour: maths.RGB(red, green, blue)}
}

// cutGround is the ground cut from the pinned model, by its own metres or
// laid on a sphere about the place the set says, and kept as stage.ground.
// Ground on a sphere puts a sentence about how much of it is made in the
// tally.
func (stage *staging) cutGround(ground set.Terrain,
	asset *pinnedAsset) error {
	west, north, across, down, step := ground.Window()
	stage.west, stage.north = west, north
	cut := groundCut{west: west, north: north, step: step,
		smooth: ground.Smooth(), across: int(across/step) + 1,
		down: int(down/step) + 1}
	latitude, longitude, placed := ground.Place()
	if !placed {
		var note cacheNote
		var err error
		stage.ground, note, err = keptGround(asset.path, asset.sum, cut)
		stage.given.cached = append(stage.given.cached, note)
		return err
	}
	seed, detailed := ground.Detail()
	slope, longest, relieved := ground.Relief()
	cut.place = &groundPlace{latitude: latitude, longitude: longitude,
		across: across, down: down, detailed: detailed, seed: seed,
		source: ground.From(), relieved: relieved, slope: slope,
		longest: longest}
	found, note, err := keptPlace(asset.path, asset.sum, cut)
	stage.given.cached = append(stage.given.cached, note)
	if err != nil {
		return err
	}
	stage.ground = found.window
	stage.made.ground = found.sentence(ground.From())
	return nil
}

// pinnedSource returns the exact catalogue used for the pin check as well
// as the shelf, so sequence fingerprints do not repeat the path search.
func pinnedSource(key, file, near string) (string, string, error) {
	asset, err := pinnedAssetAt(key, file, near, fetchFlag)
	if err != nil {
		return "", "", err
	}
	return asset.shelf, asset.catalogue, nil
}

// pinnedAsset keeps the checked path and catalogue from one lookup,
// the sha256 the file was checked by and how it was had.
type pinnedAsset struct {
	shelf, catalogue, path, sum string
	checked                     cacheNote
}

// pinnedAssetAt is the one place a stage reads an asset from the shelf:
// the terrain, the moon's face, and the tables a planet reads. it finds
// the shelf and the sources.yaml nearest the set, and asks the shelf for
// the dataset's file (the source's own when file is empty) as pinned.
//
// it downloads nothing: a file that is not there is one error naming the
// dataset, the path and flag, the flag that fetches it, or where flag is
// empty the verb of the sources command.
func pinnedAssetAt(key, file, near, flag string) (*pinnedAsset, error) {
	shelf, err := sources.Here()
	if err != nil {
		return nil, err
	}
	listed, err := nearest(near, listFile)
	if err != nil {
		return nil, err
	}
	list, err := sources.Read(listed)
	if err != nil {
		return nil, err
	}
	source, found := list.Find(key)
	if !found {
		return nil, fmt.Errorf("%s is not in %s", key, listFile)
	}
	made := &pinnedAsset{shelf: shelf, catalogue: listed}
	hashedPath := ""
	hashed := sources.NewShelf(shelf, nil).HashedBy(func(path string) (
		string, error) {
		sum, note, err := rememberedHash(path)
		made.sum, made.checked, hashedPath = sum, note, path
		return sum, err
	})
	if made.path, err = hashed.Reads(source, file, flag); err != nil {
		return nil, err
	}
	if hashedPath != made.path {
		// the pin checked is another file's: it says nothing of this one
		made.sum = ""
	}
	return made, nil
}

// faced is the moon's material: its glow, and its face where the set
// names a mosaic, read from the shelf once its source is found pinned.
func (stage *staging) faced(moon set.Moon, way maths.Vec,
	glow maths.Light) (*thing.Material, error) {
	glow = glow.Scale(stage.given.moonGlow)
	material := &thing.Material{Name: "moon", Glow: glow}
	if moon.Face() == "" {
		return material, nil
	}
	asset, err := pinnedAssetAt(moon.Face(), "",
		filepath.Dir(stage.given.set), fetchFlag)
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
	if stage.near != nil && stage.near.holds(east-stage.west,
		stage.north-north) {
		x, z := east-stage.west, stage.north-north
		return maths.V(x, stage.near.height(x, z, stage.lowest), z), nil
	}
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

// thingPlace is where a thing the set names stands: the place it gives
// outright, or its east and north on the ground and its height above.
func (stage *staging) thingPlace(each set.Thing) (maths.Vec, error) {
	place, absolute := stage.absolutePosition(each)
	if absolute {
		return place, nil
	}
	east, north := each.At()
	place, err := stage.at(east, north)
	if err != nil {
		return maths.Vec{}, fmt.Errorf("%s: %w", each.Name(), err)
	}
	place.Y += each.Height()
	return place, nil
}

// stand stands every thing and every stand the set names on its ground.
func (stage *staging) stand(read set.Set) error {
	things := read.Things()
	if err := stage.trialSighted(read); err != nil {
		return err
	}
	if scrooge.Asked() {
		// a camera may stand off a thing the set names later than the
		// first thing judged by it, so every thing's place is known
		// before any is made
		for _, each := range things {
			place, err := stage.thingPlace(each)
			if err != nil {
				return err
			}
			stage.places[each.Name()] = place
		}
	}
	stage.warm = stage.warmThings(things)
	for i, each := range things {
		if err := stage.given.steps.step("things", i,
			len(things)); err != nil {
			return err
		}
		if stage.trialUnseen(each.Name()) {
			continue
		}
		if _, scaled := each.Scale(); scaled && (stage.villagey(each) || each.Kind() == fireKind || each.Alight() || strings.HasPrefix(each.Name(), "rig ")) {
			return fmt.Errorf("%s: scale is not supported for terrain-founded villages, fire, burning things or special stage rigs", each.Name())
		}
		place, err := stage.thingPlace(each)
		if err != nil {
			return err
		}
		stage.places[each.Name()] = place
		stage.note(each.Name(), stage.reportedKind(each.Kind(), each.Recipe()), place)
		start := len(stage.world.Things)
		stood, err := stage.groundedThing(each, uint64(i), place)
		if !stood && err == nil {
			scale, _ := each.Scale()
			_, err = stage.putRecipeFacingScaled(each.Kind(), each.Recipe(),
				each.Complexity(), each.Seed(), uint64(i), place,
				each.Alight(), each.Name(), nil, scale)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
		stage.recordThingAuthor(start, each.Name())
	}
	stage.warmNote()
	stage.warm = nil
	stands := read.Stands()
	for i, each := range stands {
		if err := stage.given.steps.step("stands", i,
			len(stands)); err != nil {
			return err
		}
		start := len(stage.world.Things)
		if err := stage.scatter(each); err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
		stage.recordAuthor(start, "scene.set.stands", each.Name())
	}
	stage.rungNote()
	for _, each := range read.Texts() {
		if err := stage.letters(each); err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
	}
	return stage.stepped("ecosystems", stage.ecosystems, read)
}

// recordThingAuthor retains a named placement's static surfaces as one group.
// Deforming parts are rebuilt at frame time and cannot yet retain identity.
func (stage *staging) recordThingAuthor(start int, name string) {
	stage.recordAuthor(start, "scene.set.things", name)
}

// recordAuthor retains static surfaces under the named authoring group.
func (stage *staging) recordAuthor(start int, collection, name string) {
	for _, part := range stage.world.Things[start:] {
		if !stage.dynamicThings[part] {
			stage.world.RecordAuthor(part, thing.AuthorRef{Collection: collection, Name: name})
		}
	}
}

// absolutePosition turns a named east/north/up placement in the map's
// metres into the terrain-local right-handed frame. A false result keeps
// the historical terrain-sampled placement path.
func (stage *staging) absolutePosition(each set.Thing) (maths.Vec, bool) {
	position, has := each.Position()
	if !has {
		return maths.Vec{}, false
	}
	if stage.groundlessAir {
		return maths.V(position.East(), position.Up(),
			-position.North()), true
	}
	return maths.V(position.East()-stage.west,
		position.Up()-stage.lowest,
		stage.north-position.North()), true
}

// tries is how many places a seed offers a stand's thing before the
// stand is said to have no room in its band: enough for a band a
// fraction of a percent of a large window, and cheap beside a picture
const tries = 4096

// scatter stands a stand's things at places its seed picks over the
// ground, each where the ground is between the stand's heights, the
// first of them alight as many as it says. A stand of a kind that floats
// stands on the sea instead, where the ground is under it.
func (stage *staging) scatter(each set.Stand) error {
	if stage.groundlessAir {
		return stage.groundlessStand(each)
	}
	if waterline, floats := afloatKind(each.Kind()); floats {
		return stage.floated(each, waterline)
	}
	if each.Habit() != "" {
		return stage.habited(each, stage.onTerrain(each), true)
	}
	return stage.scattered(each, stage.inBand)
}

// scattered stands a stand's things at the places a chooser picks: on
// the terrain's ground for a scene with one, on the flat world ground for
// a scene in space. Each thing's place comes from a stream of its own, so no
// thing's place depends on the others (see members.go).
func (stage *staging) scattered(each set.Stand, choose chooser) error {
	count, seed := each.Count()
	var grove []groveTree
	for i := range count {
		place, err := memberPlace(each, choose, seed, i)
		if err != nil {
			return err
		}
		if stage.trialMemberUnseen(each, seed, place) {
			continue
		}
		stage.note(each.Name(), stage.reportedKind(each.Kind(), each.Recipe()), place)
		pick := standPick(each, seed, i)
		var bounds maths.Box
		bounds, err = stage.putRecipe(each.Kind(), each.Recipe(), each.Complexity(),
			pick, uint64(i), place, i < each.Alight(), "")
		if err != nil {
			return err
		}
		grove = append(grove, groveTree{at: place, bounds: bounds})
	}
	return stage.kept(each, grove)
}

// kept keeps the things a stand stood as the fuel the stand is, and hangs
// the stand's weather over them.
func (stage *staging) kept(each set.Stand, grove []groveTree) error {
	if stage.fuels == nil {
		stage.fuels = map[string][]groveTree{}
	}
	stage.fuels[each.Name()] = grove
	return stage.grove(each, grove)
}

// inBand is a place the generator offers on the ground, short of its
// edge and inside the stand's wedge if it has one, where the ground is
// between the stand's heights and no steeper than it allows.
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
		if height > above && height < below && stage.allowed(each, place) &&
			stage.gentle(each, place) {
			return place, nil
		}
	}
	return maths.Vec{}, noGround(each, above, below)
}

// noGround is the refusal of a stand no offered place would take: the
// heights it asked for, and the slope too where it said a steepest one.
// A stand that says none allows any slope, and its refusal names none.
func noGround(each set.Stand, above, below float64) error {
	if steepest := each.Steepest(); !math.IsInf(steepest, 1) {
		return fmt.Errorf("no ground between %v and %v m and no steeper "+
			"than %v in %d tries", above, below, steepest, tries)
	}
	return fmt.Errorf("no ground between %v and %v m in %d tries", above,
		below, tries)
}

// gentle is whether the ground under a place climbs no more steeply than
// the stand allows, read at the ground's nearest point; a stand that says
// no steepest grows on ground of any slope.
func (stage *staging) gentle(each set.Stand, place maths.Vec) bool {
	steepest := each.Steepest()
	if math.IsInf(steepest, 1) {
		return true
	}
	across, down, step := stage.ground.Size()
	column := min(max(int(math.Round(place.X/step)), 0), across-1)
	row := min(max(int(math.Round(place.Z/step)), 0), down-1)
	return stage.ground.Rise(column, row) <= steepest
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
// An optical recipe's pointing is already baked into its assembly. It is
// turned as the stage turns what it stands, and not as a habit would.
func (stage *staging) putRecipe(kind, recipe, detail string, seed, index uint64,
	place maths.Vec, burning bool, name string) (maths.Box, error) {
	return stage.putRecipeFacing(kind, recipe, detail, seed, index, place,
		burning, name, nil)
}

// stockOf is a thing of a kind or a recipe at a complexity and seed: the
// stock's, or made now and kept. improv/moonlake's giants, the rim's named
// trees, and its close spruces, tall enough to rise past the raised
// camera's eye, are grown taller, not magnified, so their limbs stay limbs
// and their needles needles (2026-11-10). The close ones, seen near, have
// branches on their branches.
func (stage *staging) stockOf(kind, recipe, detail string, seed uint64,
	name string) (kept, error) {
	taller, branched := stage.growth(name)
	key := stocked{name: stockIdentity(kind, recipe, taller, branched),
		pick: int(seed), detail: detail}
	if one, known := stage.have[key]; known {
		return one, nil
	}
	fresh, err := stage.wholeMade(wholeKey{kind: kind, recipe: recipe,
		detail: detail, seed: seed, taller: taller, branched: branched})
	if err != nil {
		return kept{}, err
	}
	stage.have[key] = fresh
	stage.made.kept += fresh.stored
	return fresh, nil
}

// growth is how much taller a thing of a name is grown, and whether its
// branches branch: the rim's giants and the far tree, and the close trees.
func (stage *staging) growth(name string) (taller float64, branched bool) {
	taller = 1.0
	if strings.HasPrefix(name, "rim ") || name == "the far tree" {
		taller = stage.given.giants
	}
	if strings.HasPrefix(name, "close ") {
		taller, branched = stage.given.closeScale, stage.given.closeScale != 1
	}
	return taller, branched
}

// stockIdentity is what a thing is kept in the stock by: its kind or its
// recipe, and how much taller and whether branched, where it is.
func stockIdentity(kind, recipe string, taller float64,
	branched bool) string {
	identity := kind
	if recipe != "" {
		identity = "recipe:" + recipe
	}
	if taller != 1 {
		identity = fmt.Sprintf("%s, %g times taller", identity, taller)
	}
	if branched {
		identity += ", branched"
	}
	return identity
}

// putRecipeFacing is putRecipe, turned to a given turn about the vertical,
// in radians, where one is given.
func (stage *staging) putRecipeFacing(kind, recipe, detail string,
	seed, index uint64, place maths.Vec, burning bool, name string,
	facing *float64) (maths.Box, error) {
	return stage.putRecipeFacingScaled(kind, recipe, detail, seed, index, place, burning, name, facing, 1)
}

// putRecipeFacingScaled stands a thing with an explicit uniform size factor.
func (stage *staging) putRecipeFacingScaled(kind, recipe, detail string,
	seed, index uint64, place maths.Vec, burning bool, name string,
	facing *float64, scale float64) (maths.Box, error) {
	if kind == fireKind {
		return stage.putFireRecipe(recipe, seed, index, place, name)
	}
	turn := stage.placementTurn(kind, name, seed, index)
	if facing != nil {
		turn = *facing
	}
	at := turnedAt(place, turn)
	if scale != 1 {
		at.Scale = scale
	}
	detail, err := stage.memberDetail(kind, recipe, detail, seed, name, at,
		burning)
	if err != nil {
		return maths.Box{}, err
	}
	one, err := stage.levelledStock(kind, recipe, detail, seed, name, at)
	if err != nil {
		return maths.Box{}, err
	}
	if scale != 1 && one.deform != nil {
		return maths.Box{}, fmt.Errorf("scale is not yet supported for deforming geometry")
	}
	// improv/moonlake's stage rigs: a stage light magnified to stand as a
	// rig the far lasers come out of (2026-11-10: "some kind of
	// device... right now they're just coming out of thin air"), its own
	// lamp put out, since the show's beams are lightshow's
	if strings.HasPrefix(name, "rig ") {
		at.Scale = stage.given.rig
		one.lamps = func(maths.Subspace) ([]emit.Lit, error) { return nil, nil }
	}
	if name != "" {
		stage.radii[name] = horizontalRadius(one.bounds) * scale
	}
	before := len(stage.world.Things)
	err = stage.add(one, kind, at, seed, burning)
	// -closeglow: the close spruces glow and nothing else, a mask of them
	if err == nil && stage.given.closeGlow && strings.HasPrefix(name, "close ") {
		glow := &thing.Material{Name: "close mask", Glow: maths.Grey(1)}
		for _, each := range stage.world.Things[before:] {
			each.Material = glow
		}
	}
	return one.bounds, err
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
	if one.deform != nil && burning {
		return fmt.Errorf("deforming geometry cannot be set alight")
	}
	lit, err := one.lamps(at)
	if err != nil {
		return err
	}
	if one.deform != nil && len(lit) > 0 {
		return fmt.Errorf("deforming geometry cannot carry lights")
	}
	if one.deform != nil {
		if err := deformingRefusal(stage.given, false); err != nil {
			return err
		}
		dynamic := retainedDeformation{build: one.deform, place: at}
		parts, err := dynamic.thingsAt(stage.given.at)
		if err != nil {
			return err
		}
		dynamic.initialFaces = facesOf(parts)
		if stage.dynamicThings == nil {
			stage.dynamicThings = map[*thing.Thing]bool{}
		}
		if stage.dynamicVolumes == nil {
			stage.dynamicVolumes = map[*thing.Volume]bool{}
		}
		if stage.dynamicLamps == nil {
			stage.dynamicLamps = map[int]bool{}
		}
		for _, part := range parts {
			stage.dynamicThings[stage.world.Add(part)] = true
		}
		stage.deformations = append(stage.deformations, dynamic)
		stage.made.faces += dynamic.initialFaces
	} else {
		for _, each := range stood(one.things, at) {
			stage.world.Add(each)
		}
		stage.made.faces += one.faces
	}
	stage.made.things++
	if len(lit) > nothing {
		stage.unretained = "legacy fire lamps are sampled at construction; use fire recipes"
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
