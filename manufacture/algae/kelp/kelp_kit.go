package kelp

import (
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// The meshes every giant kelp shares: a few lengths of stipe by girth,
// a holdfast finger, a gas bladder and some blade shapes. They are made
// once. A plant is placements of these, so the faces a forest stores do
// not grow with the plants in it.

// the shape of the kit
const (
	kelpClasses      = 4    // picked: girths a stipe narrows through
	kelpSides        = 6    // picked: flat sides round a stipe
	kelpBladeRows    = 160  // picked: cells along a blade
	kelpBladeCols    = 10   // picked: cells across a blade
	kelpBladderSides = 8    // picked: flat sides round a gas bladder
	kelpBladderRings = 7    // picked: rings along a gas bladder
	kelpFingerTip    = 0.3  // picked: a finger's tip radius over its foot's
	kelpHalf         = 0.5  // defined: one over two
	kelpEdgePower    = 2    // picked: how fast a ruffle grows toward the edge
	kelpBladderEnd   = 0.08 // picked: a bladder's end radius over its middle's
	kelpTipPower     = 0.7  // picked: how a blade's width swells from its foot
)

// kelpKit is the shared meshes.
type kelpKit struct {
	stipes  []*mesh.Mesh
	finger  *mesh.Mesh
	bladder *mesh.Mesh
	blades  []*mesh.Mesh
}

// middle is the middle of a figure's range.
func (book kelpBook) middle(name string) float64 {
	row := book.Figures[name]
	return (row.Least + row.Most) * kelpHalf
}

// newKelpKit makes each shared mesh.
func newKelpKit(book kelpBook, factory SeedFactory) (*kelpKit, error) {
	kit := &kelpKit{}
	foot, tip := book.middle("stipe-foot-radius"), book.middle("stipe-tip-radius")
	length := book.middle("segment")
	for class := range kelpClasses {
		near := foot + (tip-foot)*float64(class)/kelpClasses
		far := foot + (tip-foot)*float64(class+1)/kelpClasses
		round, err := kelpCone(length, near, far)
		if err != nil {
			return nil, err
		}
		kit.stipes = append(kit.stipes, round)
	}
	var err error
	cone := book.constantOf("holdfast-cone")
	if kit.finger, err = kelpCone(1, cone, cone*kelpFingerTip); err != nil {
		return nil, err
	}
	if kit.bladder, err = kelpBladder(book.constantOf("float-aspect")); err != nil {
		return nil, err
	}
	kit.blades, err = kelpBlades(book, factory)
	return kit, err
}

// kelpCone is a round cone of a length, as a flat mesh.
func kelpCone(length, foot, tip float64) (*mesh.Mesh, error) {
	return geometry.TubeMesh(length, foot, tip, kelpSides)
}

// kelpBladder is a gas bladder one metre long of an aspect, its length
// over its width, as a flat mesh.
func kelpBladder(aspect float64) (*mesh.Mesh, error) {
	rings := make([][]maths.Vec, kelpBladderRings)
	for step := range rings {
		along := float64(step) / float64(kelpBladderRings-1)
		bulge := math.Sqrt(1 - math.Pow(2*along-1, 2))
		radius := math.Max(kelpBladderEnd, bulge) / (2 * aspect)
		rings[step] = geometry.TubeRing(radius, along, kelpBladderSides, false)
	}
	return geometry.FaceMesh(geometry.Loft[[3]maths.Vec](rings), nil)
}

// kelpBladeShape is one blade's proportions, over a length of one.
type kelpBladeShape struct {
	width, thick, ruffle, waves, phase, tip, curve float64
}

// kelpBlades makes the blade shapes the figures allow: each draws its
// width, ruffle and waves from a seed of its own, the same each time.
func kelpBlades(book kelpBook, factory SeedFactory) ([]*mesh.Mesh, error) {
	count := int(book.Figures["variants"].Least)
	length := book.middle("blade-length")
	var blades []*mesh.Mesh
	for variant := range count {
		from := factory(uint64(variant + 1))
		shape := kelpBladeShape{
			width:  book.pick("blade-width", from, 0) / length,
			thick:  book.pick("blade-thickness", from, 0) / length,
			ruffle: book.pick("ruffle-height", from, 0) / length,
			waves:  float64(book.count("ruffle-waves", from, 0)),
			phase:  (2 * math.Pi) * from.Draw(0, 0, variant),
			tip:    book.constantOf("tip-width"),
			curve:  book.pick("blade-curve", from, 0),
		}
		made, err := geometry.FaceMesh(shape.skin(), nil)
		if err != nil {
			return nil, err
		}
		blades = append(blades, made)
	}
	return blades, nil
}

// point is a corner of the blade: a row along it, a column across, and
// whether it is on the upper face or the lower.
func (shape kelpBladeShape) point(row, column int, upper bool) maths.Vec {
	along := float64(row) / kelpBladeRows
	across := 2*float64(column)/kelpBladeCols - 1
	swell := math.Sin(math.Pi * math.Pow(along, kelpTipPower))
	narrow := shape.tip + (1-shape.tip)*swell
	ripple := math.Sin((2*math.Pi)*shape.waves*along + shape.phase)
	height := shape.ruffle * ripple *
		math.Pow(math.Abs(across), kelpEdgePower)
	face := shape.thick * kelpHalf
	if !upper {
		face = -face
	}
	bow := shape.curve * along * along
	return maths.V(across*shape.width*kelpHalf*narrow, along,
		height+face+bow)
}

// skin is the blade as a closed thin solid: an upper face, a lower and
// a rim, wound outward.
func (shape kelpBladeShape) skin() [][3]maths.Vec {
	var skin [][3]maths.Vec
	for row := range kelpBladeRows {
		for column := range kelpBladeCols {
			corner := func(down, over int, upper bool) maths.Vec {
				return shape.point(row+down, column+over, upper)
			}
			skin = geometry.Quad(skin, corner(0, 0, true), corner(0, 1, true),
				corner(1, 1, true), corner(1, 0, true))
			skin = geometry.Quad(skin, corner(0, 0, false), corner(1, 0, false),
				corner(1, 1, false), corner(0, 1, false))
		}
	}
	skin = shape.rimmed(skin)
	if geometry.Volume(skin) < 0 {
		for index := range skin {
			skin[index][1], skin[index][2] = skin[index][2], skin[index][1]
		}
	}
	return skin
}

// rimmed closes the blade's edge: round the perimeter of the grid, in the
// way the upper face runs it, each edge a face the other way.
func (shape kelpBladeShape) rimmed(skin [][3]maths.Vec) [][3]maths.Vec {
	var loop [][2]int
	for column := 0; column < kelpBladeCols; column++ {
		loop = append(loop, [2]int{0, column})
	}
	for row := 0; row < kelpBladeRows; row++ {
		loop = append(loop, [2]int{row, kelpBladeCols})
	}
	for column := kelpBladeCols; column > 0; column-- {
		loop = append(loop, [2]int{kelpBladeRows, column})
	}
	for row := kelpBladeRows; row > 0; row-- {
		loop = append(loop, [2]int{row, 0})
	}
	for at, here := range loop {
		next := loop[(at+1)%len(loop)]
		skin = geometry.Quad(skin, shape.point(next[0], next[1], true),
			shape.point(here[0], here[1], true),
			shape.point(here[0], here[1], false),
			shape.point(next[0], next[1], false))
	}
	return skin
}
