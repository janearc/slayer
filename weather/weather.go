package weather

import (
	"fmt"
	"math"
	"sync"
)

// Weather is a thin layer of air on a turning sphere as it flows: its
// vorticity, and for each cell the latitude its air came from, which
// the face reads to know which band's air it shows. it is the
// barotropic vorticity equation,
//
//	d(zeta)/dt = -J(psi, zeta + f) - zeta/drag - nu * grad^8(zeta) + storms
//	grad^2(psi) = zeta
//
// with the bands' air carried by the same flow and drawn back slowly
// toward the band it was made in.
//
// its age, the seconds the air has spent at the top under the light,
// cooks its haze from white toward brown; it is carried alike and made
// new where the air rises. modelled: one layer, not the depth of a
// giant's air.
type Weather struct {
	grid                   Grid
	spin, drag, relax      float64
	stirring               Stirring
	seed                   uint64
	drawn                  int64
	vorticity, origin, age []float64
	clock, next            float64
}

// fields is how many fields the weather carries: vorticity, the bands'
// air and its age, in that order.
const fields = 3

// fields is the weather's fields, in that order.
func (made *Weather) fields() [fields][]float64 {
	return [fields][]float64{made.vorticity, made.origin, made.age}
}

// upwelling is how fast anticyclonic spin, against twice the planet's,
// wells fresh air up, per unit of spin; fresh, the seconds in which air
// welling up in full is made new; and renewing, the share of the air in
// a storm's core it makes new: picked.
//
// the jets' shear spins the air at a few hundredths of the planet's,
// and it is the zones, anticyclonic between the jets, that well up white
// on jupiter while the belts sink and age, so the upwelling is strong.
//
// a storm core passes each point about once an hour, so storms renew
// little, or they would leave no air old enough to cook.
const (
	upwelling = 20.0
	fresh     = 1e5
	renewing  = 0.002
)

// freshening is how fast the air in a row is made new, a share a second,
// by the row's mean vorticity round its circle: where the jets' shear is
// anticyclonic, the zones, the mean circulation wells fresh air up, and
// where it is cyclonic, the belts, the air sinks and ages.
//
// the mean, not each eddy's own spin: eddies turn both ways everywhere, and
// freshening by them makes no band fresher than another.
func (made *Weather) freshening(vorticity float64, j int) float64 {
	cyclonic := vorticity / (2 * made.spin)
	if made.grid.latitude(j) < 0 {
		cyclonic = -cyclonic
	}
	return math.Max(0, math.Min(1, -cyclonic*upwelling)) / fresh
}

// Stirring is the storms that feed the weather from below: how many
// break out a second over a square metre, how wide each is in metres,
// and how fast each spins at its middle, in radians a second.
type Stirring struct{ rate, radius, strength float64 }

// NewStirring is storms, refused at no rate, size or strength.
func NewStirring(rate, radius, strength float64) (Stirring, error) {
	if !(rate > 0) || !(radius > 0) || !(strength > 0) {
		return Stirring{}, fmt.Errorf("storms at %g a square metre a "+
			"second, %g m wide, spinning %g a second", rate, radius,
			strength)
	}
	return Stirring{rate: rate, radius: radius, strength: strength}, nil
}

// New is still air on a grid: a planet turning once a day of so many
// seconds, friction that slows its winds over so many seconds, its
// bands' air drawn back over so many, stirred by storms from a seed.
func New(grid Grid, day, drag, relax float64, stirring Stirring,
	seed uint64) (*Weather, error) {
	if !(day > 0) || !(drag > 0) || !(relax > 0) {
		return nil, fmt.Errorf("a day of %g s, drag over %g s, the bands "+
			"drawn back over %g s", day, drag, relax)
	}
	made := &Weather{grid: grid, spin: 2 * math.Pi / day, drag: drag,
		relax: relax, stirring: stirring, seed: seed,
		vorticity: make([]float64, grid.cells()),
		origin:    make([]float64, grid.cells()),
		age:       make([]float64, grid.cells())}
	for j := range grid.rows {
		for i := range grid.across {
			made.origin[grid.at(i, j)] = grid.latitude(j)
		}
	}
	return made, nil
}

// coriolis is twice the planet's spin times the sine of a row's
// latitude: the vorticity the turning planet gives the air there.
func (made *Weather) coriolis(j int) float64 {
	return 2 * made.spin * math.Sin(made.grid.latitude(j))
}

// a tendency is how fast the vorticity, the bands' air and its age
// change, cell by cell.
type tendency struct{ vorticity, origin, age []float64 }

// rowsAtOnce runs a piece of work for each row of the grid, the rows
// shared among goroutines.
func (made *Weather) rowsAtOnce(work func(j int)) {
	var group sync.WaitGroup
	for j := range made.grid.rows {
		group.Add(1)
		go func() {
			defer group.Done()
			work(j)
		}()
	}
	group.Wait()
}

// changing is the tendency of a state: its stream function found, then
// each cell's vorticity carried by it with the planet's, slowed by
// friction and smoothed at the grid's scale by grad^8.
//
// its bands' air is carried, smoothed by grad^4, and drawn back toward its
// own band. its age is carried and smoothed alike, a second older each
// second, and made new where the air rises.
func (made *Weather) changing(vorticity, origin, age []float64) tendency {
	grid := made.grid
	psi := grid.walled(grid.streamOf(vorticity))
	vortical := grid.mirrored(vorticity)
	spun := func(i, j int) float64 { return vortical(i, j) + made.coriolis(j) }
	carried, aged := grid.mirrored(origin), grid.mirrored(age)
	thrice := made.laplacians(vorticity, 3)
	once, older := made.laplacians(origin, 1), made.laplacians(age, 1)
	means := grid.means(vorticity)
	viscosity, diffusion := made.smoothed(4), made.smoothed(2)
	change := tendency{make([]float64, grid.cells()),
		make([]float64, grid.cells()), make([]float64, grid.cells())}
	made.rowsAtOnce(func(j int) {
		for i := range grid.across {
			k := grid.at(i, j)
			change.vorticity[k] = -grid.jacobian(psi, spun, i, j) -
				vorticity[k]/made.drag -
				viscosity*grid.laplacian(thrice, i, j)
			change.origin[k] = -grid.jacobian(psi, carried, i, j) -
				(origin[k]-grid.latitude(j))/made.relax -
				diffusion*grid.laplacian(once, i, j)
			change.age[k] = -grid.jacobian(psi, aged, i, j) + 1 -
				age[k]*made.freshening(means[j], j) -
				diffusion*grid.laplacian(older, i, j)
		}
	})
	return change
}

// laplacians is a field's laplacian taken so many times over, read
// mirrored past the walls each time.
func (made *Weather) laplacians(field []float64, times int) reading {
	grid := made.grid
	for range times {
		read := grid.mirrored(field)
		next := make([]float64, grid.cells())
		made.rowsAtOnce(func(j int) {
			for i := range grid.across {
				next[grid.at(i, j)] = grid.laplacian(read, i, j)
			}
		})
		field = next
	}
	return grid.mirrored(field)
}

// smoothing is how long the grid's finest waves take to die away under
// the hyperviscosity and the bands' hyperdiffusion, in seconds: picked,
// so what piles up at the grid's scale is taken off before it turns to
// noise.
//
// grad^8 takes it off sharply there and leaves a storm a few cells wide all
// but alone; grad^4 over twenty thousand seconds left the grid's noise, and
// over one hour killed the storms.
const smoothing = 3600.0

// smoothed is the coefficient of a laplacian taken so many times over,
// metres to twice that power a second, that damps a wave half a cell
// long over smoothing.
func (made *Weather) smoothed(times int) float64 {
	return math.Pow(made.cell()/math.Pi, float64(2*times)) / smoothing
}

// cell is the narrowest a cell is, in metres, after the polar filter:
// a row's height, or a cell's width at the filter's latitude.
func (made *Weather) cell() float64 {
	grid := made.grid
	return grid.radius * math.Min(grid.rise(),
		grid.run()*math.Cos(filtered))
}
