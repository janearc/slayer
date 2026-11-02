//go:build darwin

package gpu

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// reading is one number the kernel reads out of a buffer the go side
// wrote: the kernel's name for it, and what the go side put there.
type reading struct {
	name string
	want float64
}

var (
	xyz  = []string{"x", "y", "z"}
	xyzw = []string{"x", "y", "z", "w"}
)

// fields is a struct's fields, named as the kernel names them, each
// with the number the go side wrote in it
func fields(name string, names []string, wants ...float64) []reading {
	found := make([]reading, 0, len(wants))
	for index, want := range wants {
		found = append(found, reading{name + "." + names[index], want})
	}
	return found
}

// layoutKernel reads every buffer at its second entry, so that a struct
// of the wrong size reads as the wrong number, and gives back what the
// table asks of it, in order
func layoutKernel(table []reading) string {
	var source strings.Builder
	source.WriteString(`
kernel void layout(constant Header *header [[buffer(0)]],
                   device const Sphere *spheres [[buffer(1)]],
                   device const Triangle *triangles [[buffer(2)]],
                   device const Node *nodes [[buffer(3)]],
                   device const Material *materials [[buffer(4)]],
                   device const Lamp *lamps [[buffer(5)]],
                   device const Volume *volumes [[buffer(6)]],
                   device const float *baked [[buffer(7)]],
                   device const Tree *trees [[buffer(8)]],
                   device const Instance *instances [[buffer(9)]],
                   device const Node *tops [[buffer(10)]],
                   device const float4 *sky [[buffer(11)]],
                   device const Bend *bends [[buffer(12)]],
                   device const float *rays [[buffer(13)]],
                   device const float *boxes [[buffer(14)]],
                   device float *out [[buffer(15)]],
                   constant uint2 &origin [[buffer(16)]],
                   uint2 at [[thread_position_in_grid]]) {
	Sphere sphere = spheres[1];
	Triangle face = triangles[1];
	Bend bend = bends[1];
	Node box = nodes[1];
	Material made = materials[1];
	Lamp lamp = lamps[1];
	Volume volume = volumes[1];
	Tree conifer = trees[1];
	Instance placed = instances[1];
	Node top = tops[1];
`)
	for index, one := range table {
		fmt.Fprintf(&source, "\tout[%d] = %s;\n", index, one.name)
	}
	source.WriteString("}\n")
	return source.String()
}

// a few of each are written, each field a number of its own, and each
// number exact in a float. the second of each is what is read back.
// every kind has a count of its own, so a header that swaps two counts
// reads wrong.
const (
	someSpheres   = 3
	someTriangles = 4
	someNodes     = 5
	someLamps     = 6
	someVolumes   = 2
	someTrees     = 7
	someStandings = 8
	someTops      = 9
)

// writeThings adds materials and spheres, and says what the second of
// each holds
func writeThings(t *testing.T, encoded *scene) []reading {
	t.Helper()
	for index := range someSpheres {
		k := float64(index)
		colour := maths.RGB(.25+k, .5+k, .75+k)
		made := &thing.Material{Name: "m", Colour: colour,
			Glow: maths.RGB(1.5+k, 2.5+k, 3.5+k), Gloss: 0.5 + k/4,
			Index: 1.25 + k}
		if index == 1 {
			made.Shine = shine{}
		}
		encoded.material(made)
		at := maths.V(1+k, 2+k, 3+k)
		space := maths.Subspace{Origin: at, Scale: 2,
			Orientation: maths.About(maths.V(0, 1, 1).Unit(), 0.3+k)}
		encoded.sphere(space, 4+k, index+5, index == 1)
	}
	table := fields("made.colour", xyz, 1.25, 1.5, 1.75)
	table = append(table, fields("made.glow", xyz, 2.5, 3.5, 4.5)...)
	table = append(table, fields("made.mirror", xyz, 0.75, 2.25)...)
	table = append(table, fields("made.info", xyz, 0, 1)...)
	table = append(table, fields("sphere.centre", xyzw, 2, 3, 4, 10)...)
	table = append(table, fields("sphere.info", xyzw, 6, 1)...)
	turned := maths.About(maths.V(0, 1, 1).Unit(), 1.3)
	origin := maths.V(2, 3, 4)
	for axis, way := range axes() {
		row := turned.Out(way).Unit().Scale(0.5)
		table = append(table, fields("sphere.into_"+xyzName(axis), xyzw,
			append(point(row), -row.Dot(origin))...)...)
	}
	return table
}

// writeLamps adds lamps, and says what the second holds
func writeLamps(encoded *scene) []reading {
	var lamps []thing.Lamp
	for index := range someLamps {
		k := float64(index)
		lamps = append(lamps, emit.Lit{At: maths.V(7+k, 8+k, 9+k),
			Beyond: 0.5 + k, Strength: maths.RGB(10+k, 11+k, 12+k),
			Toward: maths.V(20+k, 21+k, 22+k), Width: 0.25 + k,
			Soft: 0.125 + k})
	}
	encoded.lampsOf(&thing.World{Lamps: lamps})
	table := fields("lamp.at", xyzw, 8, 9, 10, 1.5)
	table = append(table, fields("lamp.strength", xyzw, 11, 12, 13,
		1.125)...)
	return append(table, fields("lamp.toward", xyzw, 21, 22, 23,
		1.25)...)
}

// writeTree adds triangles and boxes, and says what the second of each
// holds
func writeTree(encoded *scene) []reading {
	var faces []triangle
	var boxes []node
	for index := range someTriangles {
		k := float64(index)
		corners := [3]maths.Vec{maths.V(k, 1, 2), maths.V(3, 4+k, 5),
			maths.V(6, 7, 8+k)}
		normals := [3]maths.Vec{maths.V(9+k, 10, 11),
			maths.V(12, 13+k, 14), maths.V(15, 16, 17+k)}
		faces = append(faces, triangle{corners: corners,
			normals: normals, smooth: index > 0})
	}
	for index := range someNodes {
		k := float64(index)
		boxes = append(boxes, node{first: 20 + index, count: 30 + index,
			bounds: box{maths.V(-1-k, -2, -3), maths.V(1, 2, 3+k)}})
	}
	if err := encoded.tree(tree{triangles: faces, nodes: boxes}); err != nil {
		panic(err)
	}
	// the first face is flat, so the second face's bend is the first
	// one written, and the second bend is the third face's
	corners := faces[1].corners
	table := fields("face.corner_a", xyzw,
		append(point(corners[0]), 0)...)
	for index, name := range []string{"face.corner_b", "face.corner_c"} {
		more := fields(name, xyz, point(corners[index+1])...)
		table = append(table, more...)
	}
	normals := faces[2].normals
	for index, name := range []string{"bend.normal_a", "bend.normal_b",
		"bend.normal_c"} {
		table = append(table, fields(name, xyz,
			point(normals[index])...)...)
	}
	table = append(table, fields("box.low", xyz, -2, -2, -3)...)
	table = append(table, fields("box.high", xyz, 1, 2, 4)...)
	return append(table, fields("box.info", xyzw, 21, 31)...)
}

// writeVolumes adds two wispy volumes under a turned space, and says
// what the second holds
func writeVolumes(encoded *scene) []reading {
	turned := maths.About(maths.V(1, 1, 0).Unit(), 0.7)
	for index := range someVolumes {
		k := float64(index)
		space := maths.Subspace{Origin: maths.V(1+k, 2+k, 3+k),
			Orientation: turned, Scale: 2 + k}
		encoded.volume(space, haze{
			puff:  medium.Puff{Radius: 7 + k, Thick: 8 + k},
			wispy: true, seed: 0xB0000000A + uint64(index)<<32,
			scale: 9 + k, amount: 0.5}, index)
	}
	cells := float64(bakeSide * bakeSide * bakeSide)
	table := fields("volume.origin", xyzw, 2, 3, 4, 3)
	along := []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis}
	for axis, way := range along {
		name := "volume." + string("xyz"[axis])
		more := fields(name, xyz, point(turned.Out(way))...)
		table = append(table, more...)
	}
	table = append(table, fields("volume.puff", xyzw, 8, 9, 10, 0.5)...)
	table = append(table, fields("volume.info", xyzw, 10, 12, 1, 1)...)
	return append(table, fields("volume.baked", xyz, cells,
		float64(bakeSide))...)
}

// writeConifers adds conifers under a turned space, and says what the
// second holds
func writeConifers(encoded *scene) []reading {
	turned := maths.About(maths.V(0, 1, 1).Unit(), 0.3)
	for index := range someTrees {
		k := float64(index)
		place := maths.Subspace{Origin: maths.V(1+k, 2+k, 3+k),
			Orientation: turned, Scale: 2 + k}
		params := manufacture.DefaultConiferParams()
		params.Height, params.BaseRadius = 4+k, 5+k
		params.TaperRate, params.DepthScale = 0.5+k, 0.25+k
		params.BranchStart = 1.5 + k
		params.Depth, params.Sides = index%mostConiferDepth, 8+index
		encoded.conifer(grown{place: place, params: params}, 9+index)
	}
	table := fields("conifer.origin", xyzw, 2, 3, 4, 3)
	along := []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis}
	for axis, way := range along {
		name := "conifer." + string("xyz"[axis])
		more := fields(name, xyz, point(turned.Out(way))...)
		table = append(table, more...)
	}
	table = append(table, fields("conifer.shape", xyzw, 5, 6, 1.5,
		1.25)...)
	table = append(table, fields("conifer.lean", xyz[:1], 2.5)...)
	return append(table, fields("conifer.info", xyz, 1, 9, 10)...)
}

// writeStandings adds meshes standing under a turned space, and boxes
// of their tree, and says what the second of each holds
func writeStandings(encoded *scene) []reading {
	turned := maths.About(maths.V(1, 0, 1).Unit(), 0.4)
	for index := range someStandings {
		k := float64(index)
		space := maths.Subspace{Origin: maths.V(1+k, 2+k, 3+k),
			Orientation: turned, Scale: 2 + k}
		encoded.instance(standing{root: 40 + index, space: space,
			material: 50 + index, clear: index == 1})
	}
	for index := range someTops {
		k := float64(index)
		encoded.tops.row(nodeStruct, values{
			"low": {-4 - k, -5, -6}, "high": {4, 5, 6 + k},
			"info": wholes(60+index, 70+index)})
	}
	encoded.counts.instances, encoded.counts.tops = someStandings,
		someTops
	origin := maths.V(2, 3, 4)
	var table []reading
	for axis, way := range axes() {
		name := xyzName(axis)
		out := turned.Out(way).Unit()
		row := out.Scale(1.0 / 3)
		table = append(table, fields("placed.into_"+name, xyzw,
			append(point(row), -row.Dot(origin))...)...)
		table = append(table, fields("placed.out_"+name, xyz,
			point(out)...)...)
	}
	table = append(table, fields("placed.info", xyz, 41, 51, 1)...)
	table = append(table, fields("top.low", xyz, -5, -5, -6)...)
	table = append(table, fields("top.high", xyz, 4, 5, 7)...)
	return append(table, fields("top.info", xyzw, 61, 71)...)
}

// writeHeader sets the camera, and says what the header holds: the eye,
// the lens, and how many of everything there are
func writeHeader(encoded *scene) []reading {
	eye := maths.Subspace{Origin: maths.V(1, 2, 3),
		Orientation: maths.Identity}
	encoded.camera(eye, film.Pinhole{Wide: 1, Pixel: 1}, 640, 480)
	table := fields("header->origin", xyz, 1, 2, 3)
	table = append(table, fields("header->right", xyz, 1, 0, 0)...)
	table = append(table, fields("header->up", xyz, 0, 1, 0)...)
	table = append(table, fields("header->back", xyz, 0, 0, 1)...)
	table = append(table, fields("header->lens", xyzw,
		math.Tan(0.5), 640.0/480, 640, 480)...)
	table = append(table, fields("header->counts", xyzw, someSpheres,
		someTriangles, someNodes, someLamps)...)
	table = append(table, fields("header->more", xyzw, someVolumes,
		strides, shades, someTrees)...)
	table = append(table, fields("header->tuning", xyz, lift, faint,
		evenly)...)
	table = append(table, fields("header->tops", xyzw, someStandings,
		someTops)...)
	return append(table, fields("header->look", xyzw, 0)...)
}

// every field the go side writes is where the kernel reads it, and every
// struct is the size the kernel strides by. the two are written apart,
// by position, and nothing else holds them together.
func TestTheKernelReadsWhatTheGoSideWrote(t *testing.T) {
	encoded := newScene()
	table := writeThings(t, encoded)
	table = append(table, writeLamps(encoded)...)
	table = append(table, writeTree(encoded)...)
	table = append(table, writeVolumes(encoded)...)
	table = append(table, writeConifers(encoded)...)
	table = append(table, writeStandings(encoded)...)
	table = append(table, writeHeader(encoded)...)
	text := source() + layoutKernel(table)
	kernel, err := metal.Open(text, "layout")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	got, err := kernel.Grid(1, 1, 1, len(table), encoded.inputs())
	if err != nil {
		t.Fatal(err)
	}
	for index, one := range table {
		if got[index] != float32(one.want) {
			t.Errorf("%s: kernel reads %v, go wrote %v", one.name,
				got[index], float32(one.want))
		}
	}
}
