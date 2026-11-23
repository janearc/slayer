package cities

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// roofForm is the shape of a roof.
type roofForm string

// The forms of roof. Across and along are said of the ridge, against the
// row of houses the roof is on: across runs front to back, and shows its
// gable to the street; along runs the length of the row.
const (
	// gableAcross is two slopes meeting at a ridge that runs front to
	// back, so the triangle of the gable faces the street.
	gableAcross roofForm = "gable-across"
	// gableAlong is two slopes meeting at a ridge that runs along the
	// row, so the gable ends are at the sides.
	gableAlong roofForm = "gable-along"
	// monoUp is one slope, high at the front.
	monoUp roofForm = "mono-up"
	// monoDown is one slope, high at the back and falling to the front.
	monoDown roofForm = "mono-down"
	// roofFlat is a slab, with a parapet round it if it has one.
	roofFlat roofForm = "flat"
	// hip is four slopes: a prism whose top outline is smaller than its
	// bottom.
	hip roofForm = "hip"
)

// roofForms are the forms a roof may have.
func roofForms() []roofForm {
	return []roofForm{gableAcross, gableAlong, monoUp, monoDown, roofFlat, hip}
}

// roof is the roof of one mass, standing over a footprint of a width
// along x and a depth along z, with its own origin at the corner of the
// footprint on the wall's top. Everything about it is given, none drawn.
type roof struct {
	Form roofForm
	// Width and Depth are the footprint the roof covers, in metres.
	Width, Depth float64
	// Rise is how high the ridge, or the high edge, stands over the
	// eaves. A flat roof has none.
	Rise float64
	// Eave is how far the roof overhangs the walls. Nought is none, and
	// a flat roof has none.
	Eave float64
	// Slab is how thick the sloped slabs and the flat slab are.
	Slab float64
	// Parapet is how high a flat roof's parapet stands over its slab,
	// and ParapetWall how thick it is. Nought is no parapet.
	Parapet, ParapetWall float64
	// Inset is how far a hip's top outline is drawn in from its bottom
	// outline on every side.
	Inset float64
	// Cover is what the slopes are made of, and Gable what the triangle
	// of a gable end is made of: the wall's colour.
	Cover, Gable *thing.Material
}

// check refuses a roof that cannot be drawn: a form unknown, a size
// that is not positive, a flat roof with eaves, a sloped one with no
// rise, a hip drawn in past its middle.
func (roof roof) check() error {
	if !slices.Contains(roofForms(), roof.Form) {
		return fmt.Errorf("cities: no roof form %q, only %v", roof.Form,
			roofForms())
	}
	if !(roof.Width > 0 && roof.Depth > 0 && roof.Slab > 0) ||
		roof.Eave < 0 || roof.Cover == nil {
		return fmt.Errorf("cities: a roof needs a footprint, a slab and a "+
			"cover, and no negative eave, not %+v", roof)
	}
	if roof.Form == roofFlat {
		if roof.Eave != 0 {
			return fmt.Errorf("cities: a flat roof has no eaves, and this "+
				"one has %v m", roof.Eave)
		}
		return nil
	}
	if !(roof.Rise > 0) {
		return fmt.Errorf("cities: a %s roof needs a rise, not %v",
			roof.Form, roof.Rise)
	}
	return roof.checkShape()
}

// checkShape is check for what only some forms need: a hip's inset
// must leave a top, and a gable needs its end material.
func (roof roof) checkShape() error {
	least := math.Min(roof.Width, roof.Depth)/2 + roof.Eave
	if roof.Form == hip && !(roof.Inset > 0 && roof.Inset < least) {
		return fmt.Errorf("cities: a hip's inset must be above nought and "+
			"under %v, not %v", least, roof.Inset)
	}
	if (roof.Form == gableAcross || roof.Form == gableAlong) &&
		roof.Gable == nil {
		return fmt.Errorf("cities: a gable needs a material for its end")
	}
	return nil
}

// parts is the roof drawn as solids, standing on its origin.
func (roof roof) parts() ([]placedElement, error) {
	if err := roof.check(); err != nil {
		return nil, err
	}
	switch roof.Form {
	case roofFlat:
		return roof.flat(), nil
	case hip:
		return roof.hip(), nil
	case monoUp, monoDown:
		return roof.mono(), nil
	case gableAlong:
		return roof.along(), nil
	}
	return roof.gable(), nil
}

// rectangle is the outline of the ground between two corners.
func rectangle(left, back, right, front float64) polygon {
	return polygon{Corners: []outlineCorner{{left, back}, {right, back},
		{right, front}, {left, front}}}
}

// flat is a slab, and the four boxes of a parapet round it.
func (roof roof) flat() []placedElement {
	parts := []placedElement{put("roof slab",
		Box{roof.Width, roof.Slab, roof.Depth}, roof.Cover, maths.Vec{})}
	if !(roof.Parapet > 0) {
		return parts
	}
	wall, depth := roof.ParapetWall, roof.Depth-2*roof.ParapetWall
	along := Box{roof.Width, roof.Parapet, wall}
	across := Box{wall, roof.Parapet, depth}
	return append(parts,
		put("parapet", along, roof.Cover, maths.Vec{}),
		put("parapet", along, roof.Cover,
			maths.V(0, 0, roof.Depth-wall)),
		put("parapet", across, roof.Cover, maths.V(0, 0, wall)),
		put("parapet", across, roof.Cover,
			maths.V(roof.Width-wall, 0, wall)))
}

// hip is a prism from the footprint, widened by the eave, to an outline
// drawn in from it.
func (roof roof) hip() []placedElement {
	eave, inset := roof.Eave, roof.Inset
	bottom := rectangle(-eave, -eave, roof.Width+eave, roof.Depth+eave)
	top := rectangle(inset-eave, inset-eave, roof.Width+eave-inset,
		roof.Depth+eave-inset)
	return []placedElement{put("roof", prism{Bottom: bottom, Top: top,
		Height: roof.Rise}, roof.Cover, maths.Vec{})}
}

// mono is a triangle laid on its side, across the depth, as long as the
// footprint and its eaves.
func (roof roof) mono() []placedElement {
	across := roof.Depth + 2*roof.Eave
	section := []outlineCorner{{0, 0}, {across, 0}, {across, roof.Rise}}
	if roof.Form == monoDown {
		section[2] = outlineCorner{0, roof.Rise}
	}
	form := extrusion{Section: section, Length: roof.Width + 2*roof.Eave,
		Ridge: ridgeAlongX}
	return []placedElement{put("roof", form, roof.Cover,
		maths.V(-roof.Eave, 0, -roof.Eave))}
}

// gable is the triangle of the gable end, in the wall's material, run
// the depth of the footprint, with the two sloped slabs laid on it. The
// slabs are longer than the slope by a lap, so that they cross at the
// ridge and leave no gap, and wider than the depth by the eave at each
// end, so that they overhang the gable.
func (roof roof) gable() []placedElement {
	half := roof.Width / 2
	slope := math.Atan2(roof.Rise, half)
	lap := roof.Slab * math.Tan(slope)
	length := (half+roof.Eave)/math.Cos(slope) + lap
	slab := Box{length, roof.Slab, roof.Depth + 2*roof.Eave}
	end := extrusion{Section: []outlineCorner{{0, 0}, {roof.Width, 0},
		{half, roof.Rise}}, Length: roof.Depth, Ridge: ridgeAlongZ}
	left := maths.NewSubspace(maths.V(-roof.Eave,
		-roof.Eave*math.Tan(slope), -roof.Eave),
		maths.Rotation(maths.ZAxis, slope), 0)
	right := maths.NewSubspace(maths.V(half-lap*math.Cos(slope),
		roof.Rise+lap*math.Sin(slope), -roof.Eave),
		maths.Rotation(maths.ZAxis, -slope), 0)
	return []placedElement{
		put("gable end", end, roof.Gable, maths.Vec{}),
		{Element: solid{Name: "roof slope", Form: slab,
			Material: roof.Cover}, Place: left},
		{Element: solid{Name: "roof slope", Form: slab,
			Material: roof.Cover}, Place: right},
	}
}

// along is a gable roof whose ridge runs along x: the gable drawn the
// other way about, and turned a quarter turn to stand on the footprint.
func (roof roof) along() []placedElement {
	turned := roof
	turned.Width, turned.Depth = roof.Depth, roof.Width
	turn := maths.NewSubspace(maths.V(0, 0, roof.Depth),
		maths.Rotation(maths.YAxis, quarterTurn), 0)
	return within(turned.gable(), turn)
}
