package gpu

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// grey is the material a thing with none gets.
var grey = &thing.Material{Name: "grey", Colour: maths.Grey(0.8)}

// materialStruct is a material as the kernel reads it. a painted one's
// colour, and a shining one's glow, are left to the cpu, from place to
// place where the ray landed.
var materialStruct = structure{"Material", []field{
	floatField("colour", ""),
	floatField("glow", ""),
	floatField("mirror", "gloss, index, through, satin"),
	intField("info", "painted, shining, its rank among clear things"),
	floatField("metal_n", "a metal's n in red, green, blue"),
	floatField("metal_k", "its k in red, green, blue"),
	floatField("film", "a metal: one, or none; its film's index, thickness"),
	floatField("absorbs", "the film's absorption a metre, red, green, blue"),
	floatField("crinkle", "its wrinkles' size, slope and seed"),
	floatField("foam_origin", "wake origin east, north, heading, speed in object units"),
	floatField("foam_shape", "wake half-width, length, wavelength, maximum coverage"),
	floatField("foam_look", "aerated-water diffuse reflectance RGB and satin"),
	floatField("foam_extra", "authored breakup band"),
	intField("foam_seed", "low 32 bits of the authored ocean seed"),
	floatField("glint", "the mean-square slope of Beckmann's highlight, "+
		"both ways together; nought for GGX's by satin"),
}}

// In optical-state renders only, Material.info.w carries a named medium ID.
// clearMediumStruct is present only in an optical-state render. A row carries
// total extinction, unit scale, and the opt-in bounded scattering profile.
var clearMediumStruct = structure{"ClearMedium", []field{
	floatField("extinction_mpu", "RGB inverse metres, metres per world unit"),
	floatField("scatter_phase", "RGB inverse metres, phase asymmetry"),
	floatField("profile", "integration steps, maximum distance in metres"),
	floatField("moon_air", "validated lunar direction in air, water refractive index"),
	floatField("moon_light", "actual staged lunar irradiance RGB, tagged lamp index plus one"),
	floatField("moon_plane", "finite calm-water mean centre XYZ, half extent in world units"),
}}

// conductor is a material's metal and its film as the kernel reads
// them: none but zeros for a surface that is not a metal, which the
// kernel reads as Schlick's share, as Mirrors gives.
func conductor(made *thing.Material) values {
	if made.Metal == nil {
		return values{"metal_n": {0, 0, 0}, "metal_k": {0, 0, 0},
			"film": {0, 0, 0}, "absorbs": {0, 0, 0}}
	}
	n, k := made.Metal.N, made.Metal.K
	parts := values{"metal_n": {n[0], n[1], n[2]},
		"metal_k": {k[0], k[1], k[2]}, "film": {1, 0, 0},
		"absorbs": {0, 0, 0}}
	if film := made.Film; film != nil {
		a := film.Absorbs
		parts["film"] = []float64{1, film.Index, film.Thickness}
		parts["absorbs"] = []float64{a[0], a[1], a[2]}
	}
	return parts
}

// painting is a material's two textures as they are at the picture's
// time: its paint, the colour from place to place, and its shine, the
// glow; either nil where it has none.
type painting struct{ paint, shine texture.Texture }

// material is the index of a material in the materials buffer, added
// the first time it is seen, with its textures at the picture's time.
// what it does outside the visible, its Radiant, the visible picture
// does not ask, as slayer's tracer does not.
func (encoded *scene) material(made *thing.Material) int {
	if made == nil {
		made = grey
	}
	if index, ok := encoded.materialOf[made]; ok {
		return index
	}
	index := len(encoded.materialOf)
	encoded.materialOf[made] = index
	textures := painting{paint: ticked(made.Texture, encoded.at),
		shine: ticked(made.Shine, encoded.at)}
	encoded.paints = append(encoded.paints, textures)
	encoded.glossy = encoded.glossy || made.Gloss > 0
	encoded.cleared = encoded.cleared || made.Through > 0
	colour, glow := made.Colour, made.Glow
	row := values{
		"colour": {colour.R, colour.G, colour.B},
		"glow":   {glow.R, glow.G, glow.B},
		"mirror": {made.Gloss, made.Index, made.Through, made.Satin},
		"info": wholes(flag(textures.paint != nil),
			flag(textures.shine != nil), made.Rank)}
	if encoded.opticalState {
		row["info"] = append(row["info"], float64(encoded.mediumID(made.ClearMedium)))
	}
	for name, numbers := range conductor(made) {
		row[name] = numbers
	}
	row["crinkle"] = []float64{0, 0, 0}
	if wrinkles := made.Crinkle; wrinkles != nil {
		row["crinkle"] = []float64{wrinkles.Size, wrinkles.Slope,
			float64(wrinkles.Seed)}
	}
	row["foam_origin"] = []float64{0, 0, 0, 0}
	row["foam_shape"] = []float64{0, 0, 0, 0}
	row["foam_look"] = []float64{0, 0, 0, 0}
	row["foam_extra"] = []float64{0, 0, 0, 0}
	row["foam_seed"] = wholes(0)
	row["glint"] = []float64{made.SlopeVariance}
	if foam := made.WakeFoam; foam != nil && foam.Coverage > 0 {
		seconds := encoded.at.Seconds()
		if foam.FrozenTime {
			seconds = foam.Seconds
		}
		row["foam_origin"] = []float64{
			foam.OriginEast + math.Sin(foam.Heading)*foam.Speed*seconds,
			foam.OriginNorth + math.Cos(foam.Heading)*foam.Speed*seconds,
			foam.Heading, foam.Speed,
		}
		row["foam_shape"] = []float64{foam.HalfWidth, foam.Length,
			foam.Wavelength, foam.Coverage}
		row["foam_look"] = []float64{foam.Reflectance.R,
			foam.Reflectance.G, foam.Reflectance.B, foam.Satin}
		row["foam_extra"] = []float64{foam.Band}
		row["foam_seed"] = wholes(int(foam.Seed))
	}
	encoded.materials.row(materialStruct, row)
	return index
}

// surface is material for a surface: a painted or shining one hands its
// places back for the cpu to paint, where a volume's paint and shine are
// baked into its grid.
func (encoded *scene) surface(made *thing.Material) int {
	material := encoded.material(made)
	if encoded.paints[material] != (painting{}) {
		encoded.painted = true
	}
	return material
}

// ticked is a texture as it is at a moment: itself unless it moves, as
// slayer's tracer ticks it.
func ticked(painted texture.Texture, moment time.Duration) texture.Texture {
	if moving, ok := painted.(texture.Timed); ok {
		return moving.Tick(moment)
	}
	return painted
}

// things puts every seen thing in the world in its place at time t:
// spheres as a centre and a radius, and meshes each once in their own
// space, under trees of their own, standing wherever a thing stands,
// with a tree over the standings. The meshes' faces are packed once all
// are placed, or taken whole from the last frame's (see keptPool). The
// trees of slayer's meshes are grown first, all at once (see growTrees).
func (encoded *scene) things(world *thing.World, at time.Duration) error {
	encoded.growTrees(world)
	for _, placed := range world.Things {
		if placed.Hidden() {
			continue
		}
		if err := encoded.place(placed, at); err != nil {
			return err
		}
	}
	if err := encoded.packPool(); err != nil {
		return err
	}
	memoryStage("pool-after")
	return encoded.instancesOf()
}

// place puts one thing where it is at time t: a sphere among the
// spheres, or a mesh standing there.
func (encoded *scene) place(placed *thing.Thing, at time.Duration) error {
	if material := placed.Material; material != nil && material.IndependentEmission {
		return unsupported{"independent surface emission is CPU-only"}
	}
	if material := placed.Material; material != nil {
		if _, covered := material.Texture.(texture.Cover); covered {
			return unsupported{"a ground cover, which only the processor " +
				"renderer (--renderer cpu) reads by slope"}
		}
	}
	if material := placed.Material; material != nil && material.WakeFoam != nil && material.WakeFoam.Coverage > 0 &&
		(material.Texture != nil || material.Shine != nil ||
			material.Glow != (maths.Light{})) {
		return unsupported{"wake foam on a painted, shining or emissive material"}
	}
	if material := placed.Material; material != nil && material.WakeFoam != nil &&
		!material.WakeFoam.Valid() {
		return unsupported{"wake foam with invalid or out-of-range parameters"}
	}
	material := encoded.surface(placed.Material)
	space, clear := placed.At(at), placed.Clear()
	switch made := placed.Shape.(type) {
	case shape.Plane:
		encoded.plane(space, material, clear)
		return nil
	case shape.Sphere:
		encoded.sphere(space, made.Radius, material, clear)
		return nil
	case globe.Globe:
		encoded.sphere(space, made.Radius(), material, clear)
		return nil
	case *shape.Mesh:
		// a mesh shaded smooth is bent by its corners' normals, as
		// slayer's tracer bends it. its fields are public and can be
		// changed after it is made, so one whose normals are no longer
		// one to a vertex is refused here and not read past their end
		if made.Smooth && len(made.Normals) > 0 &&
			len(made.Normals) != len(made.Vertices) {
			return unsupported{"a smooth mesh without a normal for every vertex"}
		}
		grown := func() pendingMesh { return encoded.shapeTree(made) }
		encoded.stand(made, grown, space, material, clear)
		return nil
	case *mesh.Mesh:
		if err := restPaintRefusal(made, placed.Material); err != nil {
			return err
		}
		grown := func() pendingMesh { return encoded.slayersTree(made) }
		encoded.stand(made, grown, space, material, clear)
		return nil
	case shape.Tube:
		return encoded.tube(made, space, material, clear)
	case *shape.MeshGroup:
		return encoded.group(made, space, material, clear)
	case *shape.HeightField:
		encoded.field(made, space, material, clear)
		return nil
	case *arid.Sea:
		return encoded.sea(made.Pack(), space, material, clear)
	case *arid.InfiniteMeanContinuation:
		if level, ok := made.ExactPlaneLevel(); ok && encoded.oceanMoon != nil &&
			encoded.oceanMoon.Unbounded && placed == encoded.oceanMoon.Boundary {
			shifted := maths.NewSubspace(space.Out(maths.V(0, level, 0)), space.Orientation, space.Scale)
			encoded.plane(shifted, material, clear)
			return nil
		}
	}
	kind := fmt.Sprintf("a shape of kind %T", placed.Shape)
	return unsupported{kind}
}

// sphereStruct is a sphere as the kernel reads it: where it is and how
// big in the world, and the rows that carry a point of the world into
// its own space, where its paint and shine are asked.
var sphereStruct = structure{"Sphere", []field{
	floatField("centre", "radius in w"),
	intField("info", "material, clear, primitive kind: 0 sphere, 1 plane"),
	floatField("into_x", "the world into it, a row each, shift in w"),
	floatField("into_y", ""),
	floatField("into_z", ""),
}}

// sphere appends one sphere, placed. a globe is one too, which the
// kernel meets by the same stable quadratic as globe.Hit.
func (encoded *scene) sphere(space maths.Subspace, radius float64,
	material int, clear bool) {
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	centre := space.Out(maths.Vec{})
	given := values{
		"centre": {centre.X, centre.Y, centre.Z, radius * scale},
		"info":   wholes(material, flag(clear))}
	into(space, given)
	encoded.boundSphere(centre, radius*scale)
	encoded.spheres.row(sphereStruct, given)
	encoded.counts.spheres++
}

// plane shares the analytic primitive buffer; info.z distinguishes it.
// Centre holds its world normal and signed offset, with local texture rows.
func (encoded *scene) plane(space maths.Subspace, material int, clear bool) {
	normal := space.Orientation.Out(maths.Up).Unit()
	origin := space.Out(maths.Vec{})
	given := values{
		"centre": {normal.X, normal.Y, normal.Z, normal.Dot(origin)},
		"info":   wholes(material, flag(clear), 1),
	}
	into(space, given)
	encoded.boundPlane()
	encoded.spheres.row(sphereStruct, given)
	encoded.counts.spheres++
}

// into writes the rows that carry a point of the world into a space's own: its
// axes in the world over its scale, each shifted so the space's origin goes to
// zero.
func into(space maths.Subspace, given values) {
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	origin := space.Out(maths.Vec{})
	for index, axis := range axes() {
		row := space.Orientation.Out(axis).Unit().Scale(1 / scale)
		given["into_"+xyzName(index)] = append(point(row),
			-row.Dot(origin))
	}
}

// flag is one for true.
func flag(set bool) int {
	if set {
		return 1
	}
	return 0
}

// meshFaces is a shape.Mesh's faces in its own space. what a face is
// made of, and whether it is clear, is the standing's.
func meshFaces(made *shape.Mesh) []triangle {
	smooth := made.Smooth && len(made.Normals) > 0
	faces := make([]triangle, 0, len(made.Faces))
	for _, face := range made.Faces {
		one := triangle{smooth: smooth, unlifted: smooth}
		for corner, index := range face {
			one.corners[corner] = made.Vertices[index]
			if smooth {
				one.normals[corner] = made.Normals[index]
			}
		}
		faces = append(faces, one)
	}
	return faces
}

// threeOf is three corners as a face's.
func threeOf(one, two, three maths.Vec) [3]maths.Vec {
	return [3]maths.Vec{one, two, three}
}

// triangleStruct and nodeStruct are a face and a box of the tree as the
// kernel reads them. a face shaded smooth has its bend's place in
// corner_a's w, and every other face minus one.
var triangleStruct = structure{"Triangle", []field{
	floatField("corner_a", "in its mesh's own space, its bend in w"),
	floatField("corner_b", ""),
	floatField("corner_c", ""),
}}

// bendStruct is the normals at a smooth face's corners, in the order of
// its corners, as the kernel reads them.
var bendStruct = structure{"Bend", []field{
	floatField("normal_a", "in its mesh's own space, one long"),
	floatField("normal_b", ""),
	floatField("normal_c", ""),
}}

// A bend's place rides in the fourth number of a face's first corner, a
// float. As a float's value it names every whole number only below 2^24,
// so a scene of more faces shaded smooth could not be drawn. So its bits
// carry the place instead: the place plus bendBias, the bits of the least
// float that is not subnormal, so no device flushes it to nought and every
// place reads back exactly.
//
// A face shaded flat keeps -1, a float's value,
// and the kernel's bend_of reads the two apart by sign.

// bendBias is the bits of the least positive float that is not
// subnormal.
const bendBias = 0x00800000

// mostBends is the most faces shaded smooth a scene may have: the places
// whose bits, biased, stay below infinity's.
const mostBends = 0x7f800000 - bendBias

// bendNumber is a face's bend's place as the number the kernel reads: -1
// for a face shaded flat, or else the float whose bits are the place
// plus bendBias.
func bendNumber(bend int) float64 {
	if bend < 0 {
		return -1
	}
	return float64(math.Float32frombits(uint32(bend) + bendBias))
}

var nodeStruct = structure{"Node", []field{
	floatField("low", ""),
	floatField("high", ""),
	intField("info", "first, count"),
}}

// wholes is whole numbers as a field's numbers.
func wholes(numbers ...int) []float64 {
	found := make([]float64, len(numbers))
	for index, number := range numbers {
		found[index] = float64(number)
	}
	return found
}

// point is a place as three numbers.
func point(at maths.Vec) []float64 { return []float64{at.X, at.Y, at.Z} }

// tree appends the triangles in the tree's order, the bends of those
// shaded smooth, and the tree's boxes. It refuses more smooth faces
// than a bend's place can name.
func (encoded *scene) tree(grown tree) error {
	return encoded.treeWithin(grown, mostBends)
}

// treeWithin uses the same packing path under an explicit smooth-face cap.
// Small caps let boundary checks run without allocating millions of faces.
func (encoded *scene) treeWithin(grown tree, bendLimit int) error {
	return encoded.packWithin([][]triangle{grown.triangles}, grown.nodes,
		bendLimit)
}

// packWithin packs runs of faces, one after another as if they were one
// list, the bends of those shaded smooth, and the boxes, under a cap on
// smooth faces. The pool packs each tree's faces where the tree holds
// them, so they are never copied into one list of float64 first.
func (encoded *scene) packWithin(runs [][]triangle, nodes []node,
	bendLimit int) error {
	faces, bends := facesAndBends(runs)
	if bends > bendLimit {
		return unsupported{fmt.Sprintf("more than %d faces shaded "+
			"smooth", bendLimit)}
	}
	if !allFinite(runs) {
		return unsupported{"a mesh face outside finite float32 coordinates"}
	}
	if err := encoded.geometryCapacity(faces, bends,
		len(nodes)); err != nil {
		return err
	}
	encoded.reserveGeometry(faces, bends, len(nodes))
	// All index/coordinate/byte checks precede either output reservation.
	encoded.packedFaces(runs)
	for _, box := range nodes {
		encoded.packedNode(box)
	}
	encoded.counts.triangles = faces
	encoded.counts.nodes = len(nodes)
	return nil
}

// facesAndBends is how many faces the runs hold, and how many of them
// are shaded smooth.
func facesAndBends(runs [][]triangle) (faces, bends int) {
	for _, run := range runs {
		faces += len(run)
		for index := range run {
			if run[index].smooth {
				bends++
			}
		}
	}
	return faces, bends
}

// allFinite is whether every face of the runs fits float32.
func allFinite(runs [][]triangle) bool {
	for _, run := range runs {
		for index := range run {
			if !finiteFace(run[index]) {
				return false
			}
		}
	}
	return true
}

// packedFaces packs the runs' faces in order, each smooth one with the
// place of its bend, counted across every run.
func (encoded *scene) packedFaces(runs [][]triangle) {
	bends := 0
	for _, run := range runs {
		for _, face := range run {
			bend := -1
			if face.smooth {
				bend = bends
				bends++
			}
			encoded.packedFace(face, bend)
		}
	}
}

// finiteFace keeps fast-math kernels from receiving nonfinite mesh data.
func finiteFace(face triangle) bool {
	finite := func(point maths.Vec) bool {
		return fits(point.X) && fits(point.Y) && fits(point.Z)
	}
	for _, corner := range face.corners {
		if !finite(corner) {
			return false
		}
	}
	if face.smooth {
		for _, normal := range face.normals {
			if !finite(normal) {
				return false
			}
		}
	}
	return true
}

// lampStruct is a lamp as the kernel reads it. a width of zero is a lamp that
// shines every way, and its way is then not read. a distant lamp has a beyond
// of minus one, and at is the way toward it.
var lampStruct = structure{"Lamp", []field{
	floatField("at", "beyond in w, or the way to a distant lamp"),
	floatField("strength", "the soft edge of its cone in w"),
	floatField("toward", "the way of its cone, its width in w"),
	floatField("extent", "sphere radius in x; "+
		"beam exponent/radius/slope/reach in xyzw"),
	floatField("second", "a polar lamp's second pole in xyz; "+
		"the lamp's kind in w, nought for the rest"),
}}

// the kinds of lamp that are told apart by the w of second. an emit.Beams
// is a polar lamp: its at and beyond, its strength, its soft edge, its
// width and its first pole are the cone lamp's, and second holds its
// other pole. a film.Flood is a flood: its at holds the way up of its
// dome and its strength is its strength. any other lamp has no kind.
const (
	polarKind = 1
	floodKind = 2
)

// worldUp is the way up a film.Flood with no Up of its own means, as
// film.Flood's Light does.
var worldUp = maths.V(0, 1, 0)

// distantBeyond is the beyond that marks a lamp as distant
const distantBeyond = -1

// distantOf is a lamp that lies a way from everything, emit's or a
// film.Sun, as the kernel reads one: the way to it, the light it lands,
// and, for a sun drawn as a disc, its width, for the kernel to soften
// its shadows across; for any other lamp, not distant.
func distantOf(lamp thing.Lamp) (toward maths.Vec, light maths.Light,
	disc float64, distant bool) {
	switch source := lamp.(type) {
	case emit.Distant:
		toward, _, light = source.Light(maths.Vec{}, maths.Vec{})
		return toward, light, 0, true
	case film.Sun:
		toward, _, light = source.Light(maths.Vec{}, maths.Vec{})
		if source.Disc && source.Size > 0 {
			disc = source.Size
		}
		return toward, light, disc, true
	}
	return maths.Vec{}, maths.Light{}, 0, false
}

// spotAsLit is a film.Spot as the point emitter the kernel already
// draws: at the spot's place, as strong a unit away, and kept to the
// spot's cone, whole within the cone less its soft edge and fading
// across that edge.
//
// The two are one lamp: both fall by the inverse
// square, and the spot's edge is the emitter's smooth step written the
// other way up, so the kernel needs to know nothing new. refuseLamp
// keeps out the spots this cannot say.
func spotAsLit(spot film.Spot) emit.Lit {
	return emit.Lit{At: spot.At, Strength: spot.Strength,
		Toward: spot.Aim.Unit(), Width: spot.Cone - spot.Soft,
		Soft: spot.Soft}
}

// beamZero is defined as no arriving light, or no excluded launch interior.
const beamZero = 0

// lampsOf appends every lamp: a point emitter at its time, a distant
// lamp, the same way and light at every point, or a flood, which has no
// way at all.
func (encoded *scene) lampsOf(world *thing.World) {
	encoded.counts.beams = beamZero
	for _, lamp := range world.Lamps {
		if beam, ok := lamp.(*emit.Beam); ok {
			encoded.beamOf(beam)
			encoded.counts.beams++
			continue
		}
		if encoded.ambientOf(lamp) {
			continue
		}
		if toward, light, disc, distant := distantOf(lamp); distant {
			way := append(point(toward), distantBeyond)
			encoded.lamps.row(lampStruct, values{
				"at":       way,
				"strength": {light.R, light.G, light.B},
				"toward":   {}, "extent": {disc}, "second": {}})
			continue
		}
		var lit emit.Lit
		radius := 0.0
		switch source := lamp.(type) {
		case emit.Volume:
			lit, radius = source.Point(), source.Radius().InMetres()
		case film.Spot:
			lit = spotAsLit(source)
		default:
			lit = lamp.(emit.Lit)
		}
		strength := lit.Strength
		encoded.lamps.row(lampStruct, values{
			"at": append(point(lit.At), lit.Beyond),
			"strength": {strength.R, strength.G, strength.B,
				lit.Soft},
			"toward": append(point(lit.Toward), lit.Width),
			"extent": {radius}, "second": {}})
	}
	encoded.counts.lamps = len(world.Lamps)
	encoded.crowd = lampCrowdOf(world.Lamps)
	encoded.lists = lampListsOf(encoded.lamps.bytes)
}

// ambientOf appends a lamp that has no place of its own, a film.Flood, or
// a polar emitter, as the kernel reads one, and says whether it was one.
func (encoded *scene) ambientOf(lamp thing.Lamp) bool {
	switch source := lamp.(type) {
	case film.Flood:
		up := source.Up
		if up == (maths.Vec{}) {
			up = worldUp
		}
		strength := source.Strength
		encoded.lamps.row(lampStruct, values{
			"at":       point(up.Unit()),
			"strength": {strength.R, strength.G, strength.B},
			"toward":   {}, "extent": {},
			"second": {beamZero, beamZero, beamZero, floodKind}})
		return true
	case emit.Beams:
		encoded.polarOf(source)
		return true
	}
	return false
}

// polarOf appends beams as a lamp with two cones: where its poles point
// at the time its clock holds, which a thing.Lamp has no time to say, so
// it is read here, once, as the tracer reads it for every point of a
// picture drawn at that time.
func (encoded *scene) polarOf(beams emit.Beams) {
	north, south := beams.Poles(*beams.Clock)
	strength := beams.Strength()
	encoded.lamps.row(lampStruct, values{
		"at":       append(point(beams.At), beams.Beyond),
		"strength": {strength.R, strength.G, strength.B, beams.Soft},
		"toward":   append(point(north), beams.Width),
		"extent":   {},
		"second":   append(point(south), polarKind)})
}

// beamOf keeps collimated sources distinct from inverse-square point lamps.
// Extent's positive y selects a beam, whose x is its Gaussian exponent.
func (encoded *scene) beamOf(beam *emit.Beam) {
	lit := beam.Point()
	strength := lit.Strength
	encoded.lamps.row(lampStruct, values{
		"at":       point(lit.At),
		"strength": {strength.R, strength.G, strength.B},
		"toward":   point(lit.Toward),
		"extent": {emit.BeamExponent, beam.Radius().InMetres(),
			beam.Slope(), beam.Reach().InMetres()},
		"second": {},
	})
}

// fieldStruct is a height field as the kernel reads it: where it is,
// which samples are its, and the sizes its march bounds itself by.
var fieldStruct = structure{"Field", []field{
	floatField("origin", "scale in w"),
	floatField("x", "the field's axes in the world"),
	floatField("y", ""),
	floatField("z", ""),
	intField("data", "first sample, samples a side, material, clear"),
	floatField("size", "extent, bottom, top, slant"),
	intField("normal", "x slope offset, z slope offset, interpolated-slope flag"),
	floatField("sides", "a sea with no edge: its bands' sides in metres"),
	floatField("sea", "a sea with no edge: its touch in metres, its most "+
		"steps, its bands, one if its far part is its mean plane "+
		"(scrooge); nought for a height field"),
	floatField("slopes", "a sea with no edge: its bands' slope variances"),
	floatField("shown", "a sea with no edge: the samples a pixel covers "+
		"where a band starts to fade from it, and where it is gone"),
}}

// field appends one height field: its placement, its samples flat
// after whatever heights are already packed, and the measured sizes
// shape.HeightField marches by on the cpu, so the two march alike.
func (encoded *scene) field(made *shape.HeightField,
	space maths.Subspace, material int, clear bool) {
	samples, n, extent, bottom, top, slant := made.Pack()
	first := len(encoded.heights.bytes) / 4
	for _, height := range samples {
		encoded.heights.word(math.Float32bits(float32(height)))
	}
	var normal []float64
	if slopeX, slopeZ, ok := made.ApproximateShadingSlopes(); ok {
		xOffset := len(encoded.heights.bytes) / 4
		for _, slope := range slopeX {
			encoded.heights.word(math.Float32bits(float32(slope)))
		}
		zOffset := len(encoded.heights.bytes) / 4
		for _, slope := range slopeZ {
			encoded.heights.word(math.Float32bits(float32(slope)))
		}
		normal = []float64{float64(xOffset), float64(zOffset), 1}
	} else {
		normal = []float64{0, 0, 0}
	}
	given := placedAt(space)
	given["data"] = wholes(first, n, material, flag(clear))
	given["size"] = []float64{extent, bottom, top, slant}
	given["normal"] = wholes(int(normal[0]), int(normal[1]), int(normal[2]))
	given["sides"], given["sea"] = nil, nil
	given["slopes"], given["shown"] = nil, nil
	encoded.fields.row(fieldStruct, given)
	encoded.counts.fields++
}

// headerStruct is the header as the kernel reads it.
var headerStruct = structure{"Header", []field{
	floatField("origin", "the eye"),
	floatField("right", "the eye's axes in the world"),
	floatField("up", ""),
	floatField("back", ""),
	floatField("lens", "half the view's tangent, aspect, width, height"),
	intField("counts", "spheres, triangles, nodes, lamps"),
	intField("more", "volumes, strides, shades, trees"),
	floatField("tuning", "lift, faint, evenly, spread"),
	intField("tops", "mesh standings, the boxes of their tree"),
	intField("look", "whether rays are given in x; narrow beam count "+
		"in y; height fields in z; one in w if a ray's march is "+
		"refined inside each beam's cylinder (scrooge), nought if "+
		"inside the union of the beams' boxes"),
	intField("lighting", "source volume during smoke preparation; "+
		"fast light's bake volume and lamp in x and y, and for its "+
		"trace the samples a pixel's side in x where the sky's looks "+
		"are shared, the grid lamps' bits in z, "+
		"one past the beam tables' first number in w"),
	intField("environment", "isotropic volume sky samples, zero keeps legacy lamps"),
	floatField("bounds", "centre and radius of the sphere holding every standing mesh, world metres"),
	floatField("sky_low", "incident grid bounds in world metres"),
	floatField("sky_high", ""),
	intField("sky_grid", "incident grid vertex counts and RGB float offset"),
	floatField("close_origin", "the eye where the shutter closes"),
	floatField("close_right", "that eye's axes in the world"),
	floatField("close_back", ""),
	floatField("optics", "a thin lens: aperture's reach, focus "+
		"distance, width and height seen sharp at the focus"),
	intField("exposure", "whether the shutter is open in x; whether "+
		"the aperture is in y; samples along a pixel's side in z; "+
		"the frame's seed in w"),
	floatField("spread", "whether a pixel's samples start spread over "+
		"it in x; samples along a pixel's side in y; half a "+
		"stratum's width, as a share in 32 bits, in z; the width of "+
		"the filter's slope, in pixels, in w"),
	floatField("glow_low", "the world's glow's box in world metres"),
	floatField("glow_high", ""),
	intField("glow_grid", "its cells across x, y, z and its first "+
		"RGB number in the boxes; -1 for no glow"),
	intField("analytic", "where the sphere and conifer tree's nodes "+
		"begin in the tops, how many leaves with no box come first, "+
		"and how many nodes of the tree follow them"),
	intField("lamp_tree", "the nodes of the lamp tree in x, and the "+
		"lamps outside it in y; both kept after the lamps' rows, "+
		"nought for a world drawn without one"),
	intField("lamp_lists", "how many floods, cones and beams are "+
		"listed after the lamp tree and the lamps outside it, in "+
		"x, y and z; nought for a world with none"),
}}

// camera makes the header: the eye's place and axes, the lens, and how
// many of everything there are. Its shutter is closed; a drawer that
// holds one open writes it over the header after (see exposure.go).
func (encoded *scene) camera(eye maths.Subspace, lens film.Sensor,
	width, height int) {
	counts := encoded.counts
	given := viewValues(eye, lens, shutter{}, width, height)
	for name, numbers := range (values{
		"counts": wholes(counts.spheres, counts.triangles, counts.nodes,
			counts.lamps),
		"more": wholes(counts.volumes, strides, shades,
			counts.trees),
		"tuning": {lift, faint, evenly, encoded.spread},
		"tops":   wholes(counts.instances, counts.tops),
		"analytic": wholes(encoded.analyticAt.first,
			encoded.analyticAt.loose, encoded.analyticAt.nodes),
		"look": wholes(flag(encoded.looked), encoded.counts.beams,
			counts.fields, flag(scrooge.Asked())),
		"lighting": wholes(beamZero), "environment": wholes(encoded.volumeSky),
		"lamp_tree": wholes(len(encoded.crowd.nodes),
			len(encoded.crowd.others)),
		"lamp_lists": wholes(len(encoded.lists.floods),
			len(encoded.lists.cones), len(encoded.lists.beams)),
		"bounds": encoded.standingBounds()}) {
		given[name] = numbers
	}
	encoded.incidentHeader(given)
	encoded.glowHeader(given)
	encoded.header.row(headerStruct, given)
}

// cameraValues is the header's fields that the eye and the lens make:
// the eye's place and axes, and the lens for a picture width by height
// samples. A camera move changes these and nothing else in a scene.
func cameraValues(eye maths.Subspace, lens film.Pinhole,
	width, height int) values {
	wide, pixelShape := lens.Wide, lens.Pixel
	if wide == 0 {
		wide = 60 * maths.Degree
	}
	if pixelShape == 0 {
		pixelShape = 1
	}
	axes := eye.Orientation
	aspect := float64(width) * pixelShape / float64(height)
	return values{
		"origin": point(eye.Out(maths.Vec{})),
		"right":  point(axes.Out(maths.XAxis)),
		"up":     point(axes.Out(maths.YAxis)),
		"back":   point(axes.Out(maths.ZAxis)),
		"lens": {math.Tan(wide / 2), aspect, float64(width),
			float64(height)},
	}
}

// slayer's tracer's own numbers (trace), so the two march alike
const (
	// lift is the share of the world's spread, or of a point's distance
	// from the origin where that is more, that a ray leaving a surface
	// starts off it
	lift = 1e-6
	// strides is how many steps a ray takes across a volume, and shades
	// how many toward a lamp from each
	strides = 48
	shades  = 10
	faint   = 1e-3
	evenly  = 0.5
)
