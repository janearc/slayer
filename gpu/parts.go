package gpu

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
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
	floatField("mirror", "gloss, index"),
	intField("info", "painted, shining"),
}}

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
	colour, glow := made.Colour, made.Glow
	encoded.materials.row(materialStruct, values{
		"colour": {colour.R, colour.G, colour.B},
		"glow":   {glow.R, glow.G, glow.B},
		"mirror": {made.Gloss, made.Index},
		"info": wholes(flag(textures.paint != nil),
			flag(textures.shine != nil))})
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
// with a tree over the standings.
func (encoded *scene) things(world *thing.World, at time.Duration) error {
	for _, placed := range world.Things {
		if placed.Hidden() {
			continue
		}
		if err := encoded.place(placed, at); err != nil {
			return err
		}
	}
	if err := encoded.tree(encoded.faces); err != nil {
		return err
	}
	return encoded.instancesOf()
}

// place puts one thing where it is at time t: a sphere among the
// spheres, or a mesh standing there.
func (encoded *scene) place(placed *thing.Thing, at time.Duration) error {
	material := encoded.surface(placed.Material)
	space, clear := placed.At(at), placed.Clear()
	switch made := placed.Shape.(type) {
	case shape.Sphere:
		encoded.sphere(space, made.Radius, material, clear)
		return nil
	case globe.Globe:
		encoded.sphere(space, made.Radius(), material, clear)
		return nil
	case *shape.Mesh:
		if made.Smooth && len(made.Normals) > 0 {
			return unsupported{"a smooth mesh"}
		}
		faces := func() []triangle { return meshFaces(made) }
		encoded.stand(made, faces, space, material, clear)
		return nil
	case *mesh.Mesh:
		faces := func() []triangle { return slayersFaces(made) }
		encoded.stand(made, faces, space, material, clear)
		return nil
	case *shape.MeshGroup:
		return encoded.group(made, space, material, clear)
	}
	kind := fmt.Sprintf("a shape of kind %T", placed.Shape)
	return unsupported{kind}
}

// sphereStruct is a sphere as the kernel reads it: where it is and how
// big in the world, and the rows that carry a point of the world into
// its own space, where its paint and shine are asked.
var sphereStruct = structure{"Sphere", []field{
	floatField("centre", "radius in w"),
	intField("info", "material, clear"),
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
	faces := make([]triangle, 0, len(made.Faces))
	for _, face := range made.Faces {
		var corners [3]maths.Vec
		for corner, index := range face {
			corners[corner] = made.Vertices[index]
		}
		faces = append(faces, triangle{corners: corners})
	}
	return faces
}

// slayersFaces is slayer's mesh's faces in its own space, each with its
// corners' normals when the mesh is shaded smooth.
func slayersFaces(skin *mesh.Mesh) []triangle {
	faces := make([]triangle, 0, skin.FaceCount())
	for i := range skin.FaceCount() {
		one, two, three := skin.Corners(i)
		normals, smooth := skin.CornerNormals(i)
		corners := threeOf(one, two, three)
		faces = append(faces, triangle{corners: corners,
			normals: normals, smooth: smooth})
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

// mostBends is the most faces shaded smooth a scene may have: a bend's
// place rides in a float, which holds every whole number below this.
const mostBends = 1 << 24

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
	bends := 0
	for i := range grown.triangles {
		if grown.triangles[i].smooth {
			bends++
		}
	}
	if bends > bendLimit {
		return unsupported{fmt.Sprintf("more than %d faces shaded "+
			"smooth", bendLimit)}
	}
	for _, face := range grown.triangles {
		if !finiteFace(face) {
			return unsupported{"a mesh face outside finite float32 coordinates"}
		}
	}
	if err := encoded.geometryCapacity(grown, bends); err != nil {
		return err
	}
	encoded.reserveGeometry(grown, bends)
	// All index/coordinate/byte checks precede either output reservation.
	bends = 0
	for _, face := range grown.triangles {
		bend := -1
		if face.smooth {
			bend = bends
			bends++
		}
		encoded.packedFace(face, bend)
	}
	for _, box := range grown.nodes {
		encoded.packedNode(box)
	}
	encoded.counts.triangles = len(grown.triangles)
	encoded.counts.nodes = len(grown.nodes)
	return nil
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
}}

// distantBeyond is the beyond that marks a lamp as distant
const distantBeyond = -1

// beamZero is defined as no arriving light, or no excluded launch interior.
const beamZero = 0

// lampsOf appends every lamp: a point emitter at its time, or a distant
// lamp, the same way and light at every point.
func (encoded *scene) lampsOf(world *thing.World) {
	encoded.counts.beams = beamZero
	for _, lamp := range world.Lamps {
		if beam, ok := lamp.(*emit.Beam); ok {
			encoded.beamOf(beam)
			encoded.counts.beams++
			continue
		}
		if sun, distant := lamp.(emit.Distant); distant {
			toward, _, light := sun.Light(maths.Vec{}, maths.Vec{})
			way := append(point(toward), distantBeyond)
			encoded.lamps.row(lampStruct, values{
				"at":       way,
				"strength": {light.R, light.G, light.B},
				"toward":   {}, "extent": {}})
			continue
		}
		var lit emit.Lit
		radius := 0.0
		if source, ok := lamp.(emit.Volume); ok {
			lit, radius = source.Point(), source.Radius().InMetres()
		} else {
			lit = lamp.(emit.Lit)
		}
		strength := lit.Strength
		encoded.lamps.row(lampStruct, values{
			"at": append(point(lit.At), lit.Beyond),
			"strength": {strength.R, strength.G, strength.B,
				lit.Soft},
			"toward": append(point(lit.Toward), lit.Width),
			"extent": {radius}})
	}
	encoded.counts.lamps = len(world.Lamps)
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
	})
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
	intField("look", "whether rays are given in x; narrow beam count in y"),
	intField("lighting", "source volume during smoke preparation"),
}}

// camera makes the header: the eye's place and axes, the lens, and how
// many of everything there are.
func (encoded *scene) camera(eye maths.Subspace, lens film.Pinhole,
	width, height int) {
	wide, pixelShape := lens.Wide, lens.Pixel
	if wide == 0 {
		wide = 60 * maths.Degree
	}
	if pixelShape == 0 {
		pixelShape = 1
	}
	axes, counts := eye.Orientation, encoded.counts
	aspect := float64(width) * pixelShape / float64(height)
	encoded.header.row(headerStruct, values{
		"origin": point(eye.Out(maths.Vec{})),
		"right":  point(axes.Out(maths.XAxis)),
		"up":     point(axes.Out(maths.YAxis)),
		"back":   point(axes.Out(maths.ZAxis)),
		"lens": {math.Tan(wide / 2), aspect, float64(width),
			float64(height)},
		"counts": wholes(counts.spheres, counts.triangles, counts.nodes,
			counts.lamps),
		"more": wholes(counts.volumes, strides, shades,
			counts.trees),
		"tuning":   {lift, faint, evenly, encoded.spread},
		"tops":     wholes(counts.instances, counts.tops),
		"look":     wholes(flag(encoded.looked), encoded.counts.beams),
		"lighting": wholes(beamZero)})
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
