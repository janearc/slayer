package stage

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// still is the moment a lamp is lit at: no lamp on a set pulses
const still = 0

// metresPerKilometre turns a body's kilometres, which its template is
// written in, into metres
const metresPerKilometre = 1e3

// a builder puts one of the lists a set's world says into the tracer's
// world, its pieces wearing the materials built already, each place it
// says moved by shift.
type builder func(world *thing.World, said set.World,
	worn map[string]*thing.Material, shift maths.Vec) error

// unmoved is the shift of no distance.
var unmoved maths.Vec

// moved is a place moved by a shift. A shift of nought leaves the place
// as it is to the bit, where adding it would turn a minus nought to a
// plain one.
func moved(at, shift maths.Vec) maths.Vec {
	if shift == unmoved {
		return at
	}
	return at.Add(shift)
}

// builders are the lists that put something into the world, by the key
// the set file writes each under. the materials and the motions are worn
// and ridden by others; builders that need world scale read it from said.
var builders = map[string]builder{
	"ocean": oceanSurface, "boxes": boxes, "channels": channels,
	"spheres": spheres, "lamps": lamps, "lamp-fields": crowds,
	"beams": beams, "spots": spots,
	"floods": floods, "sky": hang, "bodies": bodies, "suns": suns,
	"wisps": wisps, "fizzes": fizzes, "vapor-cavities": vaporCavities,
	"imports": imports,
}

// oceanSurface builds the world's explicitly sampled finite wave sheet, or
// where the set says it unbounded, a sea with no edge. The sea is the
// manufactory's (manufacture.Sea), put in the world where the set's ocean
// stands among its lists. Wave geometry and sea parameters are SI;
// placement is expressed in the world's units and scaled by its
// metre-per-unit value.
func oceanSurface(world *thing.World, said set.World,
	worn map[string]*thing.Material, shift maths.Vec) error {
	surface, has := said.OceanSurface()
	if !has {
		return nil
	}
	made, err := seaOf(said, surface, shift)
	if err != nil {
		return err
	}
	return made.Staged(world, seaWear(worn, said, surface))
}

// SeaFor is the manufactory's sea a set's world says, without its water
// made: the object a plan asks its box of. It is false where the world
// says no ocean.
func SeaFor(said set.World, shift maths.Vec) (manufacture.Sea, bool, error) {
	surface, has := said.OceanSurface()
	if !has {
		return manufacture.Sea{}, false, nil
	}
	made, err := seaOf(said, surface, shift)
	return made, err == nil, err
}

// seaWear is how the sea the set says is worn: a sea with no edge by its
// glint from its own bands, a sheet by its material and what the set asks
// of it.
func seaWear(worn map[string]*thing.Material, said set.World,
	surface set.WaveSurface) manufacture.Wear {
	return func(open *arid.Sea) (*thing.Material, error) {
		if surface.Unbounded() {
			material, err := seaGlazed(worn[surface.Material()], surface,
				open)
			if err != nil {
				return nil, fmt.Errorf("ocean glint: %w", err)
			}
			return material, nil
		}
		return sheetMaterial(worn, said, surface)
	}
}

// seaOf is the manufactory's sea of a set's ocean: what it is made of,
// where it stands and how the world's ocean says it was made.
func seaOf(said set.World, surface set.WaveSurface,
	shift maths.Vec) (manufacture.Sea, error) {
	scale := 1 / said.Unit()
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 {
		return manufacture.Sea{}, fmt.Errorf("ocean: world unit %g produces an invalid metre scale", said.Unit())
	}
	if surface.Unbounded() {
		return openSea(said, surface, shift, scale)
	}
	normalMode := surface.ShadingNormalApproximation()
	var surfaceWake *arid.SurfaceWake
	if authoredWake, hasWake := surface.SurfaceWake(); hasWake {
		middle, origin := surface.MapMiddle(), authoredWake.Origin()
		wake := arid.SurfaceWake{
			OriginEastMetres:     (origin[0] - middle[0]) * said.Unit(),
			OriginNorthMetres:    (origin[1] - middle[1]) * said.Unit(),
			HeadingRadians:       authoredWake.HeadingRadians(),
			SpeedMetresPerSecond: authoredWake.SpeedMetresPerSecond(),
			AmplitudeMetres:      authoredWake.AmplitudeMetres(),
			HalfWidthMetres:      authoredWake.HalfWidthMetres(),
			LengthMetres:         authoredWake.LengthMetres(),
			WavelengthMetres:     authoredWake.WavelengthMetres(),
			AddToSpectralShading: authoredWake.AddToSpectralShading(),
		}
		surfaceWake = &wake
	}
	model, err := SeaOf(surface)
	if err != nil {
		return manufacture.Sea{}, fmt.Errorf("ocean: %w", err)
	}
	sheet := &manufacture.Sheet{Extent: surface.Extent(),
		Samples: surface.Samples(), Normal: normalMode, Wake: surfaceWake}
	if continuation, has := surface.FlatMean(); has {
		sheet.Mean = &manufacture.MeanPlane{
			Level:    continuation.MeanLevelMetres(),
			Collar:   continuation.CollarWidthMetres(),
			Infinite: continuation.ExtendsToInfinity()}
	}
	return manufacture.Sea{Model: model, Seconds: surface.Seconds(),
		Seed: surface.Seed(), Sheet: sheet, At: seaAt(surface, shift, scale),
		Optics: sheetOptics(said, surface, normalMode),
		Motion: sheetMotion(surface, model, normalMode, surfaceWake)}, nil
}

// seaAt is where a sea's boundary stands: at the ocean's middle, moved by
// shift, and scaled by the world's metres a unit.
func seaAt(surface set.WaveSurface, shift maths.Vec,
	scale float64) maths.Subspace {
	return maths.Subspace{Origin: moved(surface.Middle(), shift),
		Scale: scale}
}

// sheetMaterial is the material a finite sheet wears: its glint, then the
// diffuse sky in-scattering and the wake foam the set asks of it.
func sheetMaterial(worn map[string]*thing.Material, said set.World,
	surface set.WaveSurface) (*thing.Material, error) {
	material, err := glazed(worn[surface.Material()], surface)
	if err != nil {
		return nil, fmt.Errorf("ocean glint: %w", err)
	}
	if skySamples := surface.DiffuseSkySamples(); skySamples > 0 {
		if material.ClearMedium == nil {
			return nil, fmt.Errorf("ocean diffuse-sky in-scattering needs a participating clear medium on material %q", surface.Material())
		}
		configured, err := material.ClearMedium.WithDiffuseSkyInscatter(
			skySamples)
		if err != nil {
			return nil, fmt.Errorf("ocean diffuse-sky in-scattering: %w", err)
		}
		own := *material
		own.ClearMedium = configured
		material = &own
	}
	if authoredWake, hasWake := surface.SurfaceWake(); hasWake {
		if authoredFoam, enabled := authoredWake.Foam(); enabled && authoredFoam.Coverage() > 0 {
			if surface.DirectSunMode() != "" || surface.DirectMoonMode() != "" || surface.DiffuseSkySamples() > 0 ||
				surface.ClearCloudSkySamples() > 0 || surface.InscatterSource() != "" ||
				material.ClearMedium.HasScattering() {
				return nil, fmt.Errorf("ocean wake foam cannot be combined with mean-plane or participating source approximations")
			}
			if material.Texture != nil || material.Shine != nil ||
				material.Glow != (maths.Light{}) {
				return nil, fmt.Errorf("ocean wake foam refuses painted, shining or emissive base material %q", surface.Material())
			}
			middle, origin := surface.MapMiddle(), authoredWake.Origin()
			reflectance := authoredFoam.Reflectance()
			patternWavelength := authoredFoam.PatternWavelengthMetres()
			if patternWavelength == 0 {
				patternWavelength = authoredWake.WavelengthMetres()
			}
			own := *material
			own.WakeFoam = &thing.WakeFoamResponse{
				OriginEast:  (origin[0] - middle[0]) * said.Unit(),
				OriginNorth: (origin[1] - middle[1]) * said.Unit(),
				Heading:     authoredWake.HeadingRadians(),
				Speed:       authoredWake.SpeedMetresPerSecond(),
				HalfWidth:   authoredWake.HalfWidthMetres() * authoredFoam.WidthScale(),
				Length:      authoredWake.LengthMetres(),
				Wavelength:  patternWavelength,
				Coverage:    authoredFoam.Coverage(), Band: authoredFoam.Band(),
				Reflectance: maths.Light{R: reflectance[0], G: reflectance[1], B: reflectance[2]},
				Satin:       authoredFoam.Satin(),
				FrozenTime:  surface.Motion() != "scene-clock",
				Seconds:     surface.Seconds(),
				Seed:        uint32(surface.Seed()),
			}
			material = &own
			if !own.WakeFoam.Valid() {
				return nil, fmt.Errorf("ocean wake foam parameters exceed their finite authored bounds")
			}
		}
	}
	return material, nil
}

// sheetOptics is how the world's ocean says a finite sheet was made: the
// metres a unit is, the sun and moon routes, the interface events and
// sources the set asks, the continuation and the shading normal. The
// boundary, wave motion, mean and extent are put in as the sea is staged.
func sheetOptics(said set.World, surface set.WaveSurface,
	normalMode string) thing.OceanSurface {
	infiniteMeanPlane := false
	if continuation, has := surface.FlatMean(); has {
		infiniteMeanPlane = continuation.ExtendsToInfinity()
	}
	return thing.OceanSurface{MetresPerUnit: said.Unit(),
		DirectSunMeanPlane:         surface.DirectSunMode() == "mean-plane",
		DirectMoonMeanPlane:        surface.DirectMoonMode() == "mean-plane",
		InterfaceEvents:            surface.InterfaceEvents(),
		SpotInscatter:              surface.InscatterSource() == "spot-lamps",
		InfiniteMeanPlane:          infiniteMeanPlane,
		ClearCloudSkySamples:       surface.ClearCloudSkySamples(),
		ClearCloudWaterStepMode:    surface.ClearCloudWaterStepMode(),
		SmoothNormalApproximation:  normalMode == arid.SmoothNormalBilinearVertexSlopes,
		ShadingNormalApproximation: normalMode,
		SunLampIndex:               -1, FillLampIndex: -1}
}

// sheetMotion is the recipe of a sheet that moves with the scene clock, or
// nil for one that stands at its moment.
func sheetMotion(surface set.WaveSurface, model oceantex.Sea,
	normalMode string, wake *arid.SurfaceWake) *thing.OceanWaveConfig {
	if surface.Motion() != "scene-clock" {
		return nil
	}
	motion := thing.OceanWaveConfig{WindMetresPerSecond: model.Wind,
		FetchMetres: model.Fetch, DepthMetres: model.Depth,
		TowardRadians: model.Toward,
		ExtentMetres:  surface.Extent(), Samples: surface.Samples(),
		Seed: surface.Seed(), SceneClock: true,
		ShadingNormal: normalMode}
	if wake != nil {
		copy := thing.OceanSurfaceWakeConfig(*wake)
		motion.SurfaceWake = &copy
	}
	if continuation, has := surface.FlatMean(); has {
		motion.FlatMean = true
		motion.InfiniteMeanPlane = continuation.ExtendsToInfinity()
		motion.MeanLevelMetres = continuation.MeanLevelMetres()
		motion.CollarWidthMetres = continuation.CollarWidthMetres()
	}
	return &motion
}

// glazed is the material the ocean sheet wears: the one the world
// names, itself, when the set says no glint, so nothing about a sea
// that says none changes; and otherwise the sheet's own copy of it,
// which other things wearing that material do not share.
//
// An artistic glint sets the copy's Satin to the authored roughness, a
// look chosen by eye for the tracer's GGX lobe and not a measured sea.
// A cox-munk glint sets its slope variance to the slope the sea leaves
// below one cell, for Beckmann's lobe: see arid.GlintSlopes.
func glazed(worn *thing.Material,
	surface set.WaveSurface) (*thing.Material, error) {
	glint := surface.Glint()
	switch glint.Model() {
	case set.GlintArtistic:
		own := *worn
		own.Satin = glint.Roughness()
		return &own, nil
	case set.GlintCoxMunk:
		model, err := SeaOf(surface)
		if err != nil {
			return nil, err
		}
		budget, err := arid.GlintSlopes(model, surface.Extent(),
			surface.Samples())
		if err != nil {
			return nil, err
		}
		own := *worn
		own.SlopeVariance = budget.Unresolved
		return &own, nil
	}
	return worn, nil
}

// boxes puts each box in, as triangles about its middle.
func boxes(world *thing.World, said set.World,
	worn map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.Boxes() {
		size := each.Size()
		shaped, err := box(size.X, size.Y, size.Z)
		if err != nil {
			return fmt.Errorf("box %s: %w", each.Name(), err)
		}
		world.Add(thing.Thing{Shape: shaped, Material: worn[each.Material()],
			Motion: thing.Still(boxPlace(each, shift))})
	}
	return nil
}

// boxPlace is where a box stands: its middle, moved by shift, and turned
// about the vertical by its heading. A heading is clockwise seen from above,
// and a turn about the vertical axis is counterclockwise, so it is negated,
// as a thing's heading is. A box with no heading keeps the zero turn it
// always had.
func boxPlace(each set.Box, shift maths.Vec) maths.Subspace {
	var turn maths.Orientation
	if heading, said := each.Heading(); said {
		turn = maths.Rotation(maths.YAxis, -heading*maths.Degree)
	}
	return maths.NewSubspace(moved(each.Middle(), shift), turn, 0)
}

// box is a box of triangles about its own middle, each face turning
// counterclockwise seen from outside.
func box(wide, tall, deep float64) (*mesh.Mesh, error) {
	x, y, z := wide/2, tall/2, deep/2
	corners := []maths.Vec{
		maths.V(-x, -y, -z), maths.V(x, -y, -z), maths.V(x, y, -z),
		maths.V(-x, y, -z), maths.V(-x, -y, z), maths.V(x, -y, z),
		maths.V(x, y, z), maths.V(-x, y, z)}
	faces := [][3]int{{0, 3, 2}, {0, 2, 1}, {4, 5, 6}, {4, 6, 7},
		{0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5},
		{0, 1, 5}, {0, 5, 4}, {3, 7, 6}, {3, 6, 2}}
	return mesh.New(corners, faces, nil, false)
}

// spheres puts each sphere in, still at its middle or carried by its
// motion from there.
func spheres(world *thing.World, said set.World,
	worn map[string]*thing.Material, shift maths.Vec) error {
	runs := map[string]set.Motion{}
	for _, each := range said.Motions() {
		runs[each.Name()] = each
	}
	for _, each := range said.Spheres() {
		start := moved(each.Middle(), shift)
		var moving thing.Motion = thing.Still(
			maths.Subspace{Origin: start})
		if name, carried := each.Motion(); carried {
			moving = keyed(runs[name], start, shift)
		}
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: each.Radius()},
			Material: worn[each.Material()], Motion: moving})
	}
	return nil
}

// keyed is a motion's run, from a thing's place before it, each place in
// the run moved by shift.
func keyed(run set.Motion, base, shift maths.Vec) thing.Keyed {
	length, ease, hold := run.Run()
	var frames []thing.Keyframe
	for _, each := range run.Frames() {
		frames = append(frames, thing.Keyframe{Share: each.Share,
			At: moved(each.At, shift)})
	}
	return thing.Keyed{Frames: frames, Length: length, Ease: ease,
		Hold: hold, Base: base}
}

// lamps lights each bulb, a black body's light from beyond its radius.
func lamps(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	return lit(world, said.Lamps(), shift)
}

// crowds lights each lamp the set's lamp-fields scatter, as lamps does.
func crowds(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	return lit(world, said.LampFields(), shift)
}

// lit puts each bulb in the world. A black body is worked out once for
// each temperature and strength: a crowd says one.
func lit(world *thing.World, bulbs []set.Lamp, shift maths.Vec) error {
	type shine struct{ heat, strength float64 }
	made := map[shine]bands.Light{}
	for _, each := range bulbs {
		heat, strength := each.Shine()
		spectrum, found := made[shine{heat, strength}]
		var err error
		if !found {
			spectrum, _, err = blackbody(heat, strength)
			made[shine{heat, strength}] = spectrum
		}
		if err != nil {
			return fmt.Errorf("lamp %s: %w", each.Name(), err)
		}
		at := moved(each.At(), shift)
		bulb := emit.Point{At: at, Light: spectrum,
			Beyond: each.Radius()}
		world.Lamp(bulb.When(still))
	}
	return nil
}

// beams stages each finite Gaussian source at its written peak strength.
// Radii and reaches are world-coordinate distances, matching the beam's
// coordinates in the tracer; the measure wrapper carries those numbers
// without rescaling them by world.unit.
func beams(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.Beams() {
		heat, strength := each.Shine()
		spectrum, _, err := blackbody(heat, strength)
		if err != nil {
			return fmt.Errorf("beam %s: %w", each.Name(), err)
		}
		point := emit.Point{At: moved(each.At(), shift),
			Toward: each.Aim(), Light: spectrum}.When(still)
		radius, reach, divergence := each.Envelope()
		beam, err := emit.NewBeam(point, measure.Metres(radius),
			measure.Metres(reach), measure.Degrees(divergence))
		if err != nil {
			return fmt.Errorf("beam %s: %w", each.Name(), err)
		}
		world.Lamp(beam)
	}
	return nil
}

// spots lights each spot, the colour libtheme gives a black body at its
// temperature, so many times over.
func spots(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.Spots() {
		heat, strength := each.Shine()
		cone, soft := each.Cone()
		at := moved(each.At(), shift)
		world.Lamp(film.Spot{At: at, Aim: each.Aim(),
			Strength: swatches.Light(swatch.Planckian(heat)).Scale(strength),
			Cone:     cone * maths.Degree, Soft: soft * maths.Degree})
	}
	return nil
}

// floods lights each flood, its colour so many times over.
func floods(world *thing.World, said set.World,
	_ map[string]*thing.Material, _ maths.Vec) error {
	for _, each := range said.Floods() {
		world.Lamp(film.Flood{
			Strength: light(each.Colour()).Scale(each.Strength())})
	}
	return nil
}

// hang hangs the sky: a gradient, or the stars. A world with a sky
// already keeps it.
func hang(world *thing.World, said set.World,
	_ map[string]*thing.Material, _ maths.Vec) error {
	if world.Sky != nil {
		return nil
	}
	hung, _ := said.Sky()
	if stars, starry := hung.Starry(); starry {
		share, size, bright := stars.Look()
		world.Sky = sky.Stars{Seed: stars.Seed(), Share: share, Size: size,
			Bright: bright}
		return nil
	}
	horizon, zenith, ground := hung.Light()
	world.Sky = sky.Gradient{Horizon: light(horizon), Zenith: light(zenith),
		Ground: light(ground)}
	return nil
}

// bodies places each body from its template: its middle at a place, at
// the radius the set says or its own true size in the world's unit, its
// moons' ground tiled so finely, and its axis and aurora as the set says
// or its template has them.
func bodies(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.Bodies() {
		body, err := template.Named(each.Template())
		if err != nil {
			return fmt.Errorf("body %s: %w", each.Name(), err)
		}
		if tilt, tilted := each.Tilt(); tilted {
			body = body.Tilting(measure.Degrees(tilt))
		}
		if shown, says := each.Aurora(); says && !shown {
			body = body.WithoutAurora()
		}
		scale := metresPerKilometre / said.Unit()
		if radius, sized := each.Radius(); sized {
			scale = radius / body.Radius()
		}
		middle := moved(each.Middle(), shift)
		if err := body.Place(world, middle, scale,
			each.Detail()); err != nil {
			return fmt.Errorf("body %s: %w", each.Name(), err)
		}
	}
	return nil
}

// suns lights each sun, one way off.
func suns(world *thing.World, said set.World,
	_ map[string]*thing.Material, _ maths.Vec) error {
	for _, each := range said.Suns() {
		toward, err := along(each.Toward())
		if err != nil {
			return fmt.Errorf("sun %s: %w", each.Name(), err)
		}
		world.Lamp(thing.Sun(toward, light(each.Colour())))
	}
	return nil
}

// along is a way as the world's vector, of length one. It refuses a way
// too short to say which way it points.
func along(way maths.Vec) (maths.Vec, error) {
	pointed, err := measure.NewDirection(way.X, way.Y, way.Z)
	if err != nil {
		return maths.Vec{}, err
	}
	unit := pointed.Times(measure.Metres(1))
	return maths.V(unit.AlongX().InMetres(), unit.AlongY().InMetres(),
		unit.AlongZ().InMetres()), nil
}

// wisps lets each wisp of gas in, still at its middle: a ball made
// ragged by noise, which gives back its colour of the light that
// crosses it.
func wisps(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.Wisps() {
		radius, thick := each.Ball()
		seed, scale, amount := each.Ragged()
		at := maths.Subspace{Origin: moved(each.Middle(), shift)}
		world.AddVolume(thing.Volume{
			Medium: medium.Wispy{Of: medium.Puff{Radius: radius,
				Thick: thick}, Seed: seed, Scale: scale, Amount: amount},
			Material: &thing.Material{Name: each.Name(),
				Colour: light(each.Colour())},
			Motion: thing.Still(at),
		})
	}
	return nil
}

// fizzes lets each fizz in, standing still on its foot: the mist of
// bubbles too small to see, which gives back its colour of the light
// that crosses it. The column does not move; its churn climbs at the
// fizz's rise, when the tracer asks what it is at a moment.
func fizzes(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.Fizzes() {
		radius, height, thick := each.Column()
		seed, churn := each.Churned()
		at := maths.Subspace{Origin: moved(each.Foot(), shift)}
		if axis, explicit := each.Axis(); explicit {
			at.Orientation = columnOrientation(axis)
		}
		field := medium.Medium(medium.Fizz{Radius: radius, Height: height,
			Thick: thick, Rise: each.Rise(), Churn: churn, Seed: seed})
		if each.ClearWater() {
			phase, explicit := each.PhaseAsymmetry()
			cloud, err := authoredClearCloud(field, phase, explicit)
			if err != nil {
				return fmt.Errorf("fizz %q: %w", each.Name(), err)
			}
			field = cloud
		}
		world.AddVolume(thing.Volume{
			Medium: field,
			Material: &thing.Material{Name: each.Name(),
				Colour: light(each.Colour())},
			Motion: thing.Still(at),
		})
	}
	return nil
}

// columnOrientation turns an existing local +Y column into an authored flow
// direction. A downstream direction may include a small world-up component;
// this is a chosen plume trajectory, not a buoyant-rise or flow solution.
func columnOrientation(axis maths.Vec) maths.Orientation {
	turn := maths.Up.Cross(axis)
	if turn.Len() == 0 {
		if maths.Up.Dot(axis) < 0 {
			return maths.Rotation(maths.XAxis, math.Pi)
		}
		return maths.Orientation{}
	}
	return maths.Rotation(turn, math.Atan2(turn.Len(), maths.Up.Dot(axis)))
}

// authoredClearCloud builds a clear cloud, with an explicit Henyey-Greenstein
// phase when the authored scene gives one.
func authoredClearCloud(field medium.Medium, phase float64, explicit bool) (medium.ClearCloud, error) {
	if explicit {
		return medium.NewClearCloudWithHenyeyGreenstein(field, phase)
	}
	return medium.NewClearCloud(field)
}

// vaporCavities adds one compact phase-locked tip density field per
// authored propeller. Its colour and clear-water switch remain independent
// from ordinary rising fizz.
func vaporCavities(world *thing.World, said set.World,
	_ map[string]*thing.Material, shift maths.Vec) error {
	for _, each := range said.VaporCavities() {
		tipRadius, rpm, slipstream, length := each.Controls()
		coreRadius, density := each.Core()
		pressureDeficit, recovery := each.PressureRecovery()
		fragmentSeed, fragmentScale, fragmentContrast := each.Fragments()
		wander, wanderScale, spreadStart, spread := each.Dispersal()
		field, err := medium.NewPropellerCavitation(medium.PropellerCavitationConfig{
			Blades: each.Blades(),
			Origin: moved(each.Origin(), shift), Axis: each.Axis(),
			TipRadius: tipRadius, RPM: rpm, SlipstreamMPS: slipstream,
			LengthM: length, CoreRadiusM: coreRadius, DensityPerM: density,
			PressureDeficitPa: pressureDeficit, RecoveryPaPerM: recovery,
			VisualOnset:    each.VisualOnset(),
			FragmentScaleM: fragmentScale, FragmentContrast: fragmentContrast,
			FragmentSeed: fragmentSeed, WanderM: wander, WanderScaleM: wanderScale, SpreadStartM: spreadStart, SpreadPerM: spread,
		})
		if err != nil {
			return fmt.Errorf("vapor-cavities %q: %w", each.Name(), err)
		}
		var volume medium.Medium = field
		if each.ClearWater() {
			phase, explicit := each.PhaseAsymmetry()
			cloud, err := authoredClearCloud(volume, phase, explicit)
			if err != nil {
				return fmt.Errorf("vapor-cavities %q: %w", each.Name(), err)
			}
			volume = cloud
		}
		world.AddVolume(thing.Volume{
			Medium:   volume,
			Material: &thing.Material{Name: each.Name(), Colour: light(each.Colour())},
			Motion:   thing.Still(maths.Subspace{}),
		})
	}
	return nil
}
