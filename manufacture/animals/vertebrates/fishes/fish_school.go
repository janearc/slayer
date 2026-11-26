package fishes

import (
	"math"
	"sort"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// A school: many fish of one species, packed as the row says schools
// pack, about one fish to a cube of body length, each heading near the
// school's heading.
//
// The school is a rounded body of fish three parts
// long to two wide to one high, as the row gives it. Only a few fish are
// made, each its own size, markings and phase of the stroke; every fish
// of the school is one of those, placed, a little larger or smaller. The
// meshes are shared, so the made geometry does not grow with the count;
// what grows is one placement a fish in each shared group.

// How the school is laid out. Picked.
const (
	fishPlacing    = 6   // defined: the seed's place for the school
	fishSchoolSize = .08 // picked: a fish's size, of its made one
	fishSchoolGrow = 1.1 // picked: the school's growth, to fit all
)

// buildSchool makes one ordered set of shared mesh groups per variant.
func (row *Recipe) buildSchool(from branching.Seed) (Result, error) {
	plan := fishPlan{name: row.name, species: row.profile, from: from}
	variants := plan.whole("school-variants")
	places := plan.schoolPlaces(row.options.School)
	made := Result{}
	for v := 0; v < min(variants, len(places)); v++ {
		var mine []maths.Subspace
		for i := v; i < len(places); i += variants {
			mine = append(mine, places[i])
		}
		fish := fishPlan{name: row.name, species: row.profile, from: from.Sprout(growing, v)}
		body, err := buildFish(fish)
		if err != nil {
			return Result{}, err
		}
		parts, err := body.posed(body.stroke(fish.figure("amplitude")), false)
		if err != nil {
			return Result{}, err
		}
		looks := body.looks()
		for _, part := range parts {
			prototype, err := part.meshed(false)
			if err != nil {
				return Result{}, err
			}
			members := make([]shape.MeshMember, len(mine))
			for k, place := range mine {
				members[k] = shape.MeshMember{Mesh: prototype, Space: place}
			}
			group, err := shape.NewMeshGroup(members)
			if err != nil {
				return Result{}, err
			}
			made.Groups = append(made.Groups, Group{Mesh: group, Material: looks[part.look]})
		}
	}
	return made, nil
}

// schoolPlaces is where each of count fish stands in the school, and
// which way it heads: the cells of a lattice a body length and a spacing
// apart that lie nearest the middle of the school's rounded body, each
// fish strayed in its cell and turned a little from the school's heading.
func (plan fishPlan) schoolPlaces(count int) []maths.Subspace {
	row := plan.species.Figures["length"]
	cell := plan.figure("school-spacing") * (row.Least + row.Most) / 2
	cells := plan.schoolCells(count, cell)
	places := make([]maths.Subspace, count)
	for i := range places {
		places[i] = plan.schoolPlace(i, cells[i], cell)
	}
	return places
}

// schoolCells is the middles of the count lattice cells nearest the
// middle of the school, grown until there are as many as fish.
func (plan fishPlan) schoolCells(count int, cell float64) []maths.Vec {
	long, wide := plan.figure("school-long"), plan.figure("school-wide")
	high := plan.figure("school-high")
	volume := float64(count) * cell * cell * cell
	scale := math.Cbrt(volume * 3 / (4 * math.Pi * long * wide * high))
	axes := maths.V(long*scale, high*scale, wide*scale)
	cells := fishLattice(axes, cell)
	for len(cells) < count {
		axes = axes.Scale(fishSchoolGrow)
		cells = fishLattice(axes, cell)
	}
	sort.SliceStable(cells, func(a, b int) bool {
		return fishOutness(cells[a], axes) < fishOutness(cells[b], axes)
	})
	return cells[:count]
}

// fishOutness is how far out a point lies in an ellipsoid of half axes
// axes, squared: one on its surface.
func fishOutness(p, axes maths.Vec) float64 {
	x, y, z := p.X/axes.X, p.Y/axes.Y, p.Z/axes.Z
	return x*x + y*y + z*z
}

// fishLattice is the middles of the cells, cell apart, inside the
// ellipsoid of half axes axes.
func fishLattice(axes maths.Vec, cell float64) []maths.Vec {
	var cells []maths.Vec
	steps := func(half float64) int { return int(math.Ceil(half / cell)) }
	for i := -steps(axes.X); i <= steps(axes.X); i++ {
		for j := -steps(axes.Y); j <= steps(axes.Y); j++ {
			for k := -steps(axes.Z); k <= steps(axes.Z); k++ {
				p := maths.V(float64(i), float64(j), float64(k)).Scale(cell)
				if fishOutness(p, axes) <= 1 {
					cells = append(cells, p)
				}
			}
		}
	}
	return cells
}

// schoolPlace is fish i's place in its cell: strayed by the school's
// scatter, turned from the heading by up to the school's heading figure
// across and half that up and down, and sized a little from its made fish.
func (plan fishPlan) schoolPlace(i int, middle maths.Vec,
	cell float64) maths.Subspace {
	from := plan.from
	stray := plan.figure("school-scatter") * cell
	at := middle.Add(maths.V(from.Swing(fishPlacing, i, 0),
		from.Swing(fishPlacing, i, 1), from.Swing(fishPlacing, i, 2)).
		Scale(stray))
	heading := plan.figure("school-heading") * math.Pi / 180
	yaw := heading * from.Swing(fishPlacing, i, 3)
	pitch := heading / 2 * from.Swing(fishPlacing, i, 4)
	turn := maths.About(maths.YAxis, yaw).Then(maths.About(maths.ZAxis, pitch))
	size := 1 + fishSchoolSize*from.Swing(fishPlacing, i, 5)
	return maths.NewSubspace(at, turn, size)
}
