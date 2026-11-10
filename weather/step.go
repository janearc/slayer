package weather

import (
	"math"

	"github.com/janearc/slayer/maths/noise"
)

// courant is the share of a cell the fastest wind may cross in a step,
// and longest the longest step, in seconds, when the air is near still:
// picked.
const (
	courant = 0.4
	longest = 3600.0
)

// filtered is the latitude, in radians, past which the polar filter
// takes off waves round a circle finer than a cell there can hold, as
// the cells crowd toward the walls: picked.
var filtered = 50 * math.Pi / 180

// Run carries the weather on, a step at a time, until so many seconds have
// passed since it began, and is how many steps it took.
//
// its last step is not cut short to land on the moment: each step's length is
// the weather's own say, so a run broken anywhere and taken up again takes the
// steps a run straight through takes, bit for bit.
func (made *Weather) Run(until float64) int {
	taken := 0
	add := func(k int, spin, core float64) {
		made.vorticity[k] += spin
		made.age[k] *= 1 - core
	}
	for made.clock < until {
		step := made.stepFor()
		made.step(step)
		made.stir(step, add)
		made.clock += step
		taken++
	}
	return taken
}

// stepFor is how long a step may be: so the fastest wind crosses no more
// than courant of the narrowest cell.
func (made *Weather) stepFor() float64 {
	fastest := made.fastest()
	if fastest == 0 {
		return longest
	}
	return math.Min(longest, courant*made.cell()/fastest)
}

// fastest is the fastest wind on the grid, metres a second.
func (made *Weather) fastest() float64 {
	grid := made.grid
	return made.fastestIn(grid.walled(grid.streamOf(made.vorticity)),
		1/grid.radius)
}

// fastestIn is the fastest wind a stream function gives, from its
// slopes, scaled to metres a second: east from its slope across the
// latitudes, north from its slope round the circle.
func (made *Weather) fastestIn(psi reading, scale float64) float64 {
	grid := made.grid
	most := 0.0
	for j := range grid.rows {
		cosine := math.Cos(grid.latitude(j))
		for i := range grid.across {
			east := (psi(i, j+1) - psi(i, j-1)) / (2 * grid.rise())
			north := (psi(i+1, j) - psi(i-1, j)) / (2 * grid.run() * cosine)
			most = math.Max(most, math.Hypot(east, north)*scale)
		}
	}
	return most
}

// a state is the weather's three fields, in the order changing takes
// them: vorticity, the bands' air, and its age.
type state [3][]float64

// step carries the weather's fields on by a step of so many seconds, by
// the strong-stability-keeping third-order runge-kutta of shu and osher,
// then filters the waves the cells near the walls cannot hold.
func (made *Weather) step(seconds float64) {
	start := state{made.vorticity, made.origin, made.age}
	ahead := func(now state, weigh, keep float64) state {
		change := made.changing(now[0], now[1], now[2])
		rates := state{change.vorticity, change.origin, change.age}
		var next state
		for f := range next {
			next[f] = make([]float64, len(now[f]))
			for k := range now[f] {
				next[f][k] = keep*start[f][k] +
					weigh*(now[f][k]+seconds*rates[f][k])
			}
		}
		return next
	}
	end := ahead(ahead(ahead(start, 1, 0), 0.25, 0.75), 2.0/3, 1.0/3)
	made.vorticity, made.origin = made.filter(end[0]), made.filter(end[1])
	made.age = made.filter(end[2])
}

// filter takes off, from each row past filtered, the waves round its
// circle finer than its cells can hold: those of more turns than the
// row's share of the cosine at filtered.
func (made *Weather) filter(field []float64) []float64 {
	grid := made.grid
	made.rowsAtOnce(func(j int) {
		cosine := math.Cos(grid.latitude(j))
		if cosine >= math.Cos(filtered) {
			return
		}
		most := float64(grid.across/2) * cosine / math.Cos(filtered)
		row := make([]complex128, grid.across)
		for i := range grid.across {
			row[i] = complex(field[grid.at(i, j)], 0)
		}
		fft(row, -1)
		for k := range row {
			if math.Abs(float64(wavenumber(k, grid.across))) > most {
				row[k] = 0
			}
		}
		fft(row, 1)
		for i := range grid.across {
			field[grid.at(i, j)] = real(row[i]) / float64(grid.across)
		}
	})
	return field
}

// stir breaks out the storms of a step of so many seconds: as many as the
// rate says over the sphere's area between the walls, each at a place
// drawn evenly over that area, spinning one way or the other at random.
//
// each is a vortex with a ring of the other spin about it, so it adds no
// spin to the sphere as a whole, only turbulence; and each lifts fresh air
// from below in its core, so the air there is new.
func (made *Weather) stir(seconds float64, add stamp) {
	grid := made.grid
	area := 4 * math.Pi * grid.radius * grid.radius * math.Sin(grid.limit)
	expected := made.stirring.rate * area * seconds
	count := int(expected)
	if made.draw() < expected-float64(count) {
		count++
	}
	for range count {
		latitude := math.Asin((2*made.draw() - 1) * math.Sin(grid.limit))
		longitude := 2 * math.Pi * made.draw()
		spin := made.stirring.strength
		if made.draw() < 0.5 {
			spin = -spin
		}
		made.storm(latitude, longitude, spin, add)
	}
}

// draw is the next number from zero to one from the weather's seed.
func (made *Weather) draw() float64 {
	made.drawn++
	return noise.Hash(made.seed, made.drawn)
}

// a stamp adds a storm to a cell: so much spin, and its core, from zero to one,
// the share of the cell's air it makes new.
type stamp func(k int, spin, core float64)

// storm adds one storm's vortex at a latitude and longitude in radians, cell by
// cell by add: a mexican hat of spin, (1 - r^2/s^2) * e^(-r^2/s^2), of its
// strength at r from its middle for a storm s wide. Its spin sums to zero;
// its core is e^(-r^2/s^2).
func (made *Weather) storm(latitude, longitude, spin float64, add stamp) {
	grid := made.grid
	wide := made.stirring.radius / grid.radius
	rows := int(math.Ceil(3*wide/grid.rise())) + 1
	middle := int(math.Floor((latitude + grid.limit) / grid.rise()))
	for j := middle - rows; j <= middle+rows; j++ {
		if j < 0 || j >= grid.rows {
			continue
		}
		cosine := math.Cos(grid.latitude(j))
		cells := int(math.Ceil(3*wide/(grid.run()*cosine))) + 1
		centre := int(math.Floor(longitude / grid.run()))
		for i := centre - cells; i <= centre+cells; i++ {
			north := (grid.latitude(j) - latitude) / wide
			east := (float64(i)*grid.run() - longitude) * cosine / wide
			apart := north*north + east*east
			core := math.Exp(-apart)
			add(grid.at(i, j), spin*(1-apart)*core, renewing*core)
		}
	}
}
