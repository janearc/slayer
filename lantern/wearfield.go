package lantern

import "github.com/janearc/slayer/maths/noise"

// metal is how much metal is left at a place on the panel: above zero where
// there is some, below where it has gone. it is the least of four bites, one
// from each edge, and a punch.
func (worn *plate) metal(u, v float64) float64 {
	left := min(v-worn.bite(0, u), (1-v)-worn.bite(1, u),
		u-worn.bite(2, v), (1-u)-worn.bite(3, v))
	punch := 0.5 + 0.5*noise.Fractal(worn.seed+2, u*7, v*7, 0, 3)
	return min(left, (1-0.12*worn.wear)-punch)
}

// bite is how deep an edge is bitten at a place along it: a slow noise
// for the big bites, and a quick one on top that abrades the line.
// edge picks which of the four.
func (worn *plate) bite(edge int, along float64) float64 {
	seed := worn.seed + 3 + uint64(edge)
	slow := 0.5 + 0.5*noise.Fractal(seed, along*5, 0, 0, 3)
	fine := noise.Fractal(seed+10, along*60, 0, 0, 2)
	return worn.wear * (0.03 + 0.3*slow*slow + 0.004*fine)
}

// cut is a triangle of the grid with the gone part taken away: whole
// where it is all metal, nothing where it is all gone, and cut along
// where the metal ends where it crosses
func (worn *plate) cut(one, two, three int) [][3]int {
	corners := [3]int{one, two, three}
	for turn := range 3 {
		first := corners[turn]
		second := corners[(turn+1)%3]
		third := corners[(turn+2)%3]
		kept := [3]bool{worn.kept(first), worn.kept(second),
			worn.kept(third)}
		switch kept {
		case [3]bool{true, true, true}:
			return [][3]int{{first, second, third}}
		case [3]bool{false, false, false}:
			return nil
		case [3]bool{true, false, false}:
			return [][3]int{{first, worn.crossing(first, second),
				worn.crossing(first, third)}}
		case [3]bool{true, true, false}:
			onward := worn.crossing(second, third)
			back := worn.crossing(third, first)
			return [][3]int{{first, second, onward},
				{first, onward, back}}
		}
	}
	return nil
}

// kept is whether there is metal at a point
func (worn *plate) kept(index int) bool {
	at := worn.places[index]
	return worn.metal(at[0], at[1]) > 0
}

// crossing is the point where the metal ends between two points, found
// the same way whichever order they come in, so two triangles that
// share the side share the point
func (worn *plate) crossing(one, other int) int {
	if other < one {
		one, other = other, one
	}
	from, to := worn.places[one], worn.places[other]
	near := worn.metal(from[0], from[1])
	far := worn.metal(to[0], to[1])
	share := near / (near - far)
	return worn.point(from[0]+share*(to[0]-from[0]),
		from[1]+share*(to[1]-from[1]))
}
