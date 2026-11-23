package over

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Fineness is how finely ground is wanted at a place: the metres between
// corners an eye resolves there, or plus infinity where it sees nothing.
// A place is in the map's frame: x east and z south from its first
// sample, y over the floor.
type Fineness interface {
	Wanted(point maths.Vec) float64
}

// How a cell is cut where the ground is laid as an eye sees it.
const (
	// mostLevel is the most times a cell is halved each way: into 4096
	// parts, a millimetre of a four-metre cell. Picked. A shared place's
	// key counts in parts of the finest cut, so a key's column is the
	// map's columns times 4096, well inside an int.
	mostLevel = 12
	// mostSeenStand is the steepest the rules' stand may be where ground
	// is laid as an eye sees it: under it, the roughness an eye leaves out
	// is under one of its pixels (see drawn). Worked from drawn's bound:
	// (4/3)(8/2 pi) stand is under one while stand is under 0.589; 0.5 is
	// picked under that.
	mostSeenStand = 0.5
	// mostSeenDeparture is the most a made corner may move, as a share of
	// the survey's spacing, where ground is laid as an eye sees it: a cell
	// left whole stands within that of the made ground, and the eye there
	// sees no finer than the spacing. Defined.
	mostSeenDeparture = 1.0
	// facesPerSquare is the faces of each square a cell is cut into.
	// Defined: a square is two triangles.
	facesPerSquare = 2
	// middleShare is how far across a cell its middle is. Defined.
	middleShare = 0.5
)

// Seen is the whole map as a mesh, in Lay's frame, with every cell cut in
// halves each way as often as its longest edge needs to be no longer than
// the spacing the eye wants anywhere on it, and no finer than the rules'.
// Where the made faces would be more than the rules allow, the spacing the
// eye wants is doubled everywhere until they fit. The margin is not used.
func Seen(from Map, floor float64, eye Fineness, rules Rules) (*mesh.Mesh,
	Receipt, error) {
	read := survey(from, floor)
	if err := checkSeen(read, eye, rules); err != nil {
		return nil, Receipt{}, err
	}
	rules.Margin = 0
	whole := Stretch{Across: read.across - 1, Down: read.down - 1}
	levels, doubled := read.levels(eye, rules)
	detail := newGround(read, whole, rules, rules.Spacing).
		seenBy(eye, math.Pow(2, float64(doubled)))
	return read.laySeen(levels, detail).assemble(read, doubled)
}

// checkSeen is what is wrong with a map, an eye and the rules, if anything.
func checkSeen(read *shape, eye Fineness, rules Rules) error {
	switch {
	case read.across < 2 || read.down < 2:
		return fmt.Errorf("over: a map %d by %d samples has no cells",
			read.across, read.down)
	case eye == nil:
		return fmt.Errorf("over: ground laid as an eye sees it needs an eye")
	case !(rules.Spacing > 0) || rules.Faces < 1:
		return fmt.Errorf("over: spacing %g m and %d faces say nothing",
			rules.Spacing, rules.Faces)
	case rules.Roughness.Stand > mostSeenStand ||
		rules.Departure > mostSeenDeparture:
		return fmt.Errorf("over: a stand of %g over %g, or a departure of "+
			"%g over %g, would let what an eye leaves out stand more than "+
			"a pixel from what another eye sees", rules.Roughness.Stand,
			mostSeenStand, rules.Departure, mostSeenDeparture)
	}
	return checkRules(rules)
}

// levels is how many times each cell is halved each way, row after row,
// and how many times the eye's spacing was doubled for the made faces to
// fit the rules'. Each doubling takes one from the level the eye wanted,
// and the cut is kept to mostLevel after. Halving no cell makes no faces,
// so it ends.
func (read *shape) levels(eye Fineness, rules Rules) ([]int, int) {
	columns, rows := read.across-1, read.down-1
	levels := make([]int, columns*rows)
	for row := range rows {
		for column := range columns {
			levels[row*columns+column] = read.level(column, row, eye,
				rules.Spacing)
		}
	}
	for doubled := 0; ; doubled++ {
		if madeFaces(levels) <= rules.Faces {
			for i := range levels {
				levels[i] = min(mostLevel, levels[i])
			}
			return levels, doubled
		}
		for i := range levels {
			levels[i] = max(0, levels[i]-1)
		}
	}
}

// madeFaces is how many faces cells cut to these levels make, each cut no
// finer than mostLevel.
func madeFaces(levels []int) int {
	var faces int
	for _, level := range levels {
		if level > 0 {
			faces += facesPerSquare << (2 * min(mostLevel, level))
		}
	}
	return faces
}

// level is how many times a cell is halved each way for its longest edge
// to be no longer than the finest spacing the eye wants at its corners and
// its middle, kept to the finest given; nought where the survey is as fine.
// It is not kept to mostLevel: levels does that, after any doubling.
func (read *shape) level(column, row int, eye Fineness,
	finest float64) int {
	middle := read.locate((float64(column)+middleShare)*read.step,
		(float64(row)+middleShare)*read.step)
	wanted := eye.Wanted(maths.V((float64(column)+middleShare)*read.step,
		read.surface(middle), (float64(row)+middleShare)*read.step))
	for _, corner := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
		wanted = min(wanted, eye.Wanted(read.sample(column+corner[0],
			row+corner[1])))
	}
	wanted = max(wanted, finest)
	longest := read.longestEdge(column, row)
	if !(wanted < longest) {
		return 0
	}
	return int(math.Ceil(math.Log2(longest / wanted)))
}

// sample is a sample of the map as a point: x east, y over the floor, z
// south.
func (read *shape) sample(column, row int) maths.Vec {
	return maths.V(float64(column)*read.step, read.at(column, row),
		float64(row)*read.step)
}

// longestEdge is the longest of a cell's four edges, metres, up its rises.
func (read *shape) longestEdge(column, row int) float64 {
	var longest float64
	for _, edge := range [4][4]int{{0, 0, 1, 0}, {0, 1, 1, 1}, {0, 0, 0, 1},
		{1, 0, 1, 1}} {
		rise := read.at(column+edge[2], row+edge[3]) -
			read.at(column+edge[0], row+edge[1])
		longest = max(longest, math.Hypot(read.step, rise))
	}
	return longest
}

// seenMesh is the map's mesh as it is laid cell by cell: its points, where
// each stood before detail, the normals the map shades them by, the plain
// cells' faces and the made ones, and how far each made point moved.
type seenMesh struct {
	points, bases, normals []maths.Vec
	plain, made            [][3]int
	// shared is the number of a made place on a cell's edge, by where it
	// is on the finest cut, so that the cells either side share it.
	shared map[[2]int]int
	moves  []float64
	cells  int
	detail ground
}

// laySeen lays every cell at its level: a cell at nought is the map's own
// two triangles, and any other is cut and its detail made.
func (read *shape) laySeen(levels []int, detail ground) *seenMesh {
	points, normals := read.corners()
	laying := &seenMesh{points: points, bases: slices.Clone(points),
		normals: normals, shared: map[[2]int]int{}, detail: detail}
	columns := read.across - 1
	for i, level := range levels {
		column, row := i%columns, i/columns
		if level == 0 {
			near := row*read.across + column
			far := near + read.across
			laying.plain = append(laying.plain, [3]int{near, far + 1,
				near + 1}, [3]int{near, far, far + 1})
			continue
		}
		laying.cell(read, levels, column, row)
	}
	return laying
}

// cell lays one cell of the map cut in halves each way as often as its
// level says, its edges laid on any coarser neighbour's, and adds its
// places and faces to the mesh.
func (laying *seenMesh) cell(read *shape, levels []int, column, row int) {
	level := levels[row*(read.across-1)+column]
	parts := 1 << level
	made := grid{stretch: Stretch{Column: column, Row: row, Across: 1,
		Down: 1}, eastward: newAxis([]int{parts}),
		southward: newAxis([]int{parts})}
	made.cut()
	placed := made.layWith(read, laying.detail)
	placed.meet(read, levels, level)
	ids := laying.identify(read, placed, level)
	for _, triangle := range made.faces {
		laying.made = append(laying.made, [3]int{ids[triangle[0]],
			ids[triangle[1]], ids[triangle[2]]})
	}
	laying.cells++
}

// side is one edge of a cell: the neighbour across it, as a step across
// and down, and its places in the cell's grid as the first and the step
// from one to the next, in a cell of so many parts each way.
type side struct {
	across, down, first, next int
}

// sides are a cell's four edges, north, south, west and east, in a cell
// cut into so many parts each way.
func sides(parts int) [4]side {
	wide := parts + 1
	return [4]side{{0, -1, 0, 1}, {0, 1, parts * wide, 1},
		{-1, 0, 0, wide}, {1, 0, parts, wide}}
}

// meet lays each edge of a cell whose neighbour is cut coarser on that
// neighbour's edge, so the two meet without a gap. A neighbour off the map
// has no edge to meet.
func (placed laid) meet(read *shape, levels []int, level int) {
	columns, rows := read.across-1, read.down-1
	for _, edge := range sides(1 << level) {
		column := placed.grid.stretch.Column + edge.across
		row := placed.grid.stretch.Row + edge.down
		if column < 0 || row < 0 || column >= columns || row >= rows {
			continue
		}
		if coarser := level - levels[row*columns+column]; coarser > 0 {
			placed.along(edge, 1<<level, 1<<coarser)
		}
	}
}

// along lays the places of one edge that a neighbour cut so many times
// coarser does not have on the straight line between the two of its places
// either side, which this cell has too.
func (placed laid) along(edge side, parts, ratio int) {
	for k := range parts + 1 {
		if k%ratio == 0 {
			continue
		}
		low := k / ratio * ratio
		share := float64(k-low) / float64(ratio)
		one := placed.corners[edge.first+low*edge.next]
		other := placed.corners[edge.first+(low+ratio)*edge.next]
		placed.corners[edge.first+k*edge.next] = corner{
			base:   between(one.base, other.base, share),
			point:  between(one.point, other.point, share),
			normal: between(one.normal, other.normal, share).Unit(),
			moved:  one.moved + share*(other.moved-one.moved)}
	}
}

// between is the point a share of the way from one point to another.
func between(one, other maths.Vec, share float64) maths.Vec {
	return one.Add(other.Sub(one).Scale(share))
}

// identify is the mesh's number for each place of a laid cell: a sample of
// the map is the map's own corner, a place on the cell's edge is shared
// with the cell across it, and any other is added.
func (laying *seenMesh) identify(read *shape, placed laid, level int) []int {
	parts := 1 << level
	wide := parts + 1
	ids := make([]int, len(placed.corners))
	for place, each := range placed.corners {
		across, down := place%wide, place/wide
		rim := across == 0 || across == parts
		top := down == 0 || down == parts
		switch {
		case !rim && !top:
			ids[place] = laying.add(each)
		case rim && top:
			ids[place] = (placed.grid.stretch.Row+down/parts)*read.across +
				placed.grid.stretch.Column + across/parts
		default:
			ids[place] = laying.edgePlace(placed.grid.stretch, each,
				[2]int{across, down}, level)
		}
	}
	return ids
}

// edgePlace is the number of a place on a cell's edge, at so many parts
// across and down a cell of the given level: the one already added by the
// cell across the edge, or this one, added now.
func (laying *seenMesh) edgePlace(cell Stretch, each corner, at [2]int,
	level int) int {
	parts, finer := 1<<level, 1<<(mostLevel-level)
	key := [2]int{(cell.Column*parts + at[0]) * finer,
		(cell.Row*parts + at[1]) * finer}
	if id, found := laying.shared[key]; found {
		return id
	}
	id := laying.add(each)
	laying.shared[key] = id
	return id
}

// add is a made place added to the mesh, and its number.
func (laying *seenMesh) add(each corner) int {
	laying.points = append(laying.points, each.point)
	laying.bases = append(laying.bases, each.base)
	laying.normals = append(laying.normals, each.normal)
	laying.moves = append(laying.moves, math.Abs(each.moved))
	return len(laying.points) - 1
}

// assemble is the mesh, shaded by the normals the map shades by changed
// by how the faces lean after detail, and the receipt of what was laid.
func (laying *seenMesh) assemble(read *shape, doubled int) (*mesh.Mesh,
	Receipt, error) {
	faces := append(slices.Clip(laying.plain), laying.made...)
	built, err := mesh.New(laying.points, faces, laying.shading(faces),
		true)
	if err != nil {
		return nil, Receipt{}, err
	}
	surveyed := read.across * read.down
	receipt := Receipt{Surveyed: surveyed,
		Made: len(laying.points) - surveyed, Faces: len(laying.made),
		Bound: laying.detail.bound, Moved: read.movedIn(built),
		Cells: laying.cells, Doubled: doubled,
		Scaling: laying.detail.scaling, Sag: laying.sag(read)}
	receipt.Finest, receipt.Coarsest = laying.edges()
	laying.spread(&receipt)
	return built, receipt, nil
}

// shading is each point's normal: the map's, turned by how the faces round
// it lean after detail against how they leaned before. A point no made
// face touches shades as the map does.
func (laying *seenMesh) shading(faces [][3]int) []maths.Vec {
	before := gatherOver(laying.bases, faces)
	after := gatherOver(laying.points, faces)
	shading := slices.Clone(laying.normals)
	for id, normal := range laying.normals {
		if turned := after[id].Sub(before[id]); turned.Len() > 0 {
			shading[id] = normal.Add(turned).Unit()
		}
	}
	return shading
}

// gatherOver is each point's normal from the faces round it, taken one
// long, or nothing where no face is round it.
func gatherOver(points []maths.Vec, faces [][3]int) []maths.Vec {
	sums := make([]maths.Vec, len(points))
	for _, triangle := range faces {
		first, second := points[triangle[0]], points[triangle[1]]
		across := second.Sub(first).Cross(points[triangle[2]].Sub(first))
		for _, id := range triangle {
			sums[id] = sums[id].Add(across)
		}
	}
	for id, sum := range sums {
		if sum.Len() > 0 {
			sums[id] = sum.Unit()
		}
	}
	return sums
}

// edges are the shortest and longest edges of the made faces, after
// detail, or nought where none is made.
func (laying *seenMesh) edges() (finest, coarsest float64) {
	if len(laying.made) == 0 {
		return 0, 0
	}
	finest = math.Inf(1)
	for _, triangle := range laying.made {
		for i, id := range triangle {
			length := laying.points[triangle[(i+1)%len(triangle)]].
				Sub(laying.points[id]).Len()
			finest, coarsest = min(finest, length), max(coarsest, length)
		}
	}
	return finest, coarsest
}

// sag is the most a made face's middle stands, vertically, from the map's
// own surface there, before detail: the arithmetic's rounding, as in Lay.
func (laying *seenMesh) sag(read *shape) float64 {
	var most float64
	for _, triangle := range laying.made {
		var sum maths.Vec
		for _, id := range triangle {
			sum = sum.Add(laying.bases[id])
		}
		middle := sum.Scale(1 / float64(len(triangle)))
		stand := middle.Y - read.surface(read.locate(middle.X, middle.Z))
		most = max(most, math.Abs(stand))
	}
	return most
}

// spread fills in how far the made points moved, as Lay's receipt does,
// where any was made.
func (laying *seenMesh) spread(receipt *Receipt) {
	spreadOf(laying.moves, receipt)
}
