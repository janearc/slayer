package planet

import (
	"math"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// A GridFlow is a flow that keeps its weather on a grid of latitude and
// longitude, so a renderer that cannot call At can read the grid itself.
//
// Cells is the grid's size, around and from wall to wall. Wall is the
// latitude, in radians, its last rows stand inside of. Fields are its three
// numbers a cell, row by row from the south, each row from longitude zero:
// the latitude the air came from in radians, its spin over twice the
// planet's, and its age in seconds.
type GridFlow interface {
	Flow
	Cells() (across, rows int)
	Wall() float64
	Fields() (origin, spin, age []float64)
}

// Laid is a face laid out as numbers and lists of numbers, for a renderer
// that paints it without calling Within. Everything that depends on the
// face and not on the point is worked out here, in float64.
type Laid struct {
	numbers map[string]float64
	lists   map[string][]float64
}

// Number is one of the face's numbers, by its name in LaidNumbers.
func (laid *Laid) Number(name string) float64 { return laid.numbers[name] }

// List is one of the face's lists, by its name in LaidLists.
func (laid *Laid) List(name string) []float64 { return laid.lists[name] }

// laidNumbers are the names of a laid face's numbers, in the order a
// renderer writes them.
var laidNumbers = []string{
	"eye_x", "eye_y", "eye_z", "star_x", "star_y", "star_z",
	"pole_x", "pole_y", "pole_z", "east_x", "east_y", "east_z",
	"front_x", "front_y", "front_z",
	"plane_x_x", "plane_x_y", "plane_x_z",
	"plane_z_x", "plane_z_y", "plane_z_z",
	"deck", "top", "hottest", "airband", "calm", "cooking", "edge",
	"churned", "churning", "capped", "cleared",
	"zones", "bands", "columns", "lobes", "rungs", "ladders", "glows",
	"across", "rows", "wall",
	"south_origin", "north_origin", "south_age", "north_age",
	"lifting", "marked", "marking", "fading", "walled",
	"front_step", "most_front", "grazing",
}

// laidLists are the names of a laid face's lists.
var laidLists = []string{"norths", "shades", "lobes", "starts", "leaves",
	"flow"}

// LaidNumbers are the names of a laid face's numbers, in order.
func LaidNumbers() []string { return append([]string(nil), laidNumbers...) }

// LaidLists are the names of a laid face's lists, in order.
func LaidLists() []string { return append([]string(nil), laidLists...) }

// The widths of a laid face's rows.
const (
	// ShadeWidth is defined as a band's numbers, counted: its starlight,
	// the red, green and blue a unit of its light is, its airglow, and its
	// heat's scale and the temperature its heat falls by e at.
	ShadeWidth = 7
	// LobeWidth is defined as a lobe's numbers, counted: rest, near and
	// twice.
	LobeWidth = 3
	// StartWidth is defined as a stack's place among the leaves, counted:
	// its first, and how many.
	StartWidth = 2
	// LeafWidth is defined as a layer's numbers, counted: its depth, the
	// share it scatters, its gas's and haze's scattering and their sum, and
	// its lobe's place.
	LeafWidth = 6
	// FlowWidth is defined as a cell's numbers, counted: origin, spin and
	// age.
	FlowWidth = 3
)

// What a laid face and its chart count in, each defined.
const (
	// nothing is defined as no amount.
	nothing = 0
	// whole is defined as all of a share: one.
	whole = 1
	// double is defined as two of a thing.
	double = 2
	// circle is defined as the radians in a full turn: twice pi.
	circle = 2 * math.Pi
	// rightAngle is defined as the radians in a quarter turn: half of pi.
	rightAngle = math.Pi / double
)

// positive is whether an amount is more than nothing; not a number is not.
func positive(amount float64) bool { return amount > nothing }

// nanometre is a nanometre in metres: defined.
const nanometre = 1e-9

// gridHolds is whether a grid flow's fields are the size its cells say: at
// least one cell round and one row, and each of the three fields a number a
// cell. A grid that is not cannot be laid out.
func gridHolds(grid GridFlow) bool {
	across, rows := grid.Cells()
	origin, spin, age := grid.Fields()
	cells := across * rows
	return across > nothing && rows > nothing && len(origin) == cells &&
		len(spin) == cells && len(age) == cells
}

// Laid is the face laid out, and whether it can be: a face whose air is
// read from a grid that holds the numbers its cells say, and whose colours
// it sums itself. Any other face is painted by Within alone.
func (face *Face) Laid() (*Laid, bool) {
	grid, gridded := face.flow.(GridFlow)
	if !gridded || !face.quick || !gridHolds(grid) {
		return nil, false
	}
	laid := &Laid{numbers: map[string]float64{},
		lists: map[string][]float64{}}
	face.layWays(laid)
	face.layAir(laid)
	face.layShades(laid)
	face.layStacks(laid)
	layFlow(laid, grid)
	return laid, true
}

// Rings are the rings whose shadow falls on the face.
func (face *Face) Rings() []Ring { return face.rings }

// three puts a vector into a laid face's numbers under a name.
func (laid *Laid) three(name string, way maths.Vec) {
	laid.numbers[name+"_x"] = way.X
	laid.numbers[name+"_y"] = way.Y
	laid.numbers[name+"_z"] = way.Z
}

// layWays lays the face's eye, star and frame, and the plane its rings'
// shadow is read in.
func (face *Face) layWays(laid *Laid) {
	laid.three("eye", face.eye)
	laid.three("star", face.star)
	laid.three("pole", face.ways.pole)
	laid.three("east", face.ways.east)
	laid.three("front", face.ways.front)
	plane := maths.Upright(face.pole)
	laid.three("plane_x", plane.X)
	laid.three("plane_z", plane.Z)
}

// layAir lays the numbers of the face's air: its deck, heat, airglow,
// cooking and columns, and the numbers its flow is read by.
func (face *Face) layAir(laid *Laid) {
	set := laid.numbers
	set["deck"], set["top"], set["hottest"] = face.deck, face.top,
		face.hottest
	set["airband"], set["calm"] = face.airband, face.calm
	set["cooking"], set["edge"] = face.cooking, face.storms.edge
	set["churned"], set["churning"] = float64(face.churned), face.churning
	set["capped"], set["cleared"] = float64(face.capped),
		float64(face.cleared)
	set["zones"] = float64(len(face.norths))
	set["columns"] = float64(len(face.stacks))
	set["ladders"] = float64(len(face.ladders))
	if len(face.ladders) > nothing {
		set["rungs"] = float64(rungs)
	}
	if positive(face.airglow) {
		set["glows"] = whole
	}
	set["lifting"], set["marked"], set["marking"] = lifting, marked, marking
	set["fading"], set["walled"] = fading, walled
	set["front_step"], set["most_front"] = frontStep, mostFront
	set["grazing"] = grazing
	laid.lists["norths"] = append([]float64(nil), face.norths...)
}

// layShades lays each band's numbers, with the exposure, the observer's
// unit and the turn from the observer's sums to red, green and blue all
// folded in, so the painter adds bands straight into a colour.
func (face *Face) layShades(laid *Laid) {
	xr, xg, xb := srgb.Light(swatch.FromXYZ(whole, nothing, nothing))
	yr, yg, yb := srgb.Light(swatch.FromXYZ(nothing, whole, nothing))
	zr, zg, zb := srgb.Light(swatch.FromXYZ(nothing, nothing, whole))
	var rows []float64
	for _, each := range face.shades {
		x, y, z := each.wx*face.unit, each.wy*face.unit, each.wz*face.unit
		rows = append(rows, each.power*face.exposure,
			x*xr+y*yr+z*zr, x*xg+y*yg+z*zg, x*xb+y*yb+z*zb,
			each.held*face.airglow*face.exposure,
			each.warm.scale*each.warm.width*nanometre*face.exposure,
			planck*light/each.warm.cool)
	}
	laid.lists["shades"] = rows
	laid.numbers["bands"] = float64(len(face.shades))
	var lobes []float64
	for _, each := range face.lobes {
		lobes = append(lobes, each.rest, each.near, each.twice)
	}
	laid.lists["lobes"] = lobes
	laid.numbers["lobes"] = float64(len(face.lobes))
}

// stackLayer is the lists layStacks fills: where each stack starts among
// the leaves, and the leaves.
type stackLayer struct{ starts, leaves []float64 }

// lay adds each of some stacks to the lists: where it starts, how many
// layers it has, and each layer's numbers.
func (laying *stackLayer) lay(stacks [][]leaf) {
	for _, stack := range stacks {
		first := float64(len(laying.leaves) / LeafWidth)
		count := float64(len(stack))
		laying.starts = append(laying.starts, first, count)
		for _, layer := range stack {
			laying.leaves = append(laying.leaves, layer.tau, layer.albedo,
				layer.gas, layer.haze, layer.together, float64(layer.slot))
		}
	}
}

// layStacks lays every column's stack and every rung's, band by band: the
// leaves in one list, and where each stack starts in it. A column's
// stacks come first, in the columns' order, then each zone's rungs.
func (face *Face) layStacks(laid *Laid) {
	var laying stackLayer
	for _, column := range face.stacks {
		laying.lay(column)
	}
	for _, zone := range face.ladders {
		for _, rung := range zone {
			laying.lay(rung)
		}
	}
	laid.lists["starts"], laid.lists["leaves"] = laying.starts, laying.leaves
}

// layFlow lays the flow's grid, a cell's three numbers together, and the
// means of its edge rows, which it fades to by the poles.
func layFlow(laid *Laid, grid GridFlow) {
	across, rows := grid.Cells()
	origin, spin, age := grid.Fields()
	cells := make([]float64, nothing, len(origin)*FlowWidth)
	for at := range origin {
		cells = append(cells, origin[at], spin[at], age[at])
	}
	laid.lists["flow"] = cells
	set := laid.numbers
	set["across"], set["rows"] = float64(across), float64(rows)
	set["wall"] = grid.Wall()
	last := (rows - whole) * across
	set["south_origin"] = mean(origin[:across])
	set["north_origin"] = mean(origin[last:])
	set["south_age"] = math.Max(nothing, mean(age[:across]))
	set["north_age"] = math.Max(nothing, mean(age[last:]))
}

// mean is the mean of some numbers, summed in order.
func mean(numbers []float64) float64 {
	var sum float64
	for _, each := range numbers {
		sum += each
	}
	return sum / float64(len(numbers))
}
