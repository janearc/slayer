package weather

import "math"

// jacobian is how a stream function carries a field at a cell: the stream
// function's slope round the circle times the field's across the
// latitudes, less the other way about, over the radius squared and the
// cosine of the latitude.
//
// it is arakawa's (1966), the mean of three ways of writing it, which
// together keep the sum over the sphere of the field and of its square as
// the air keeps them, so the grid's error neither pumps the turbulence up
// nor bleeds it away.
func (grid Grid) jacobian(psi, field reading, i, j int) float64 {
	p := func(di, dj int) float64 { return psi(i+di, j+dj) }
	q := func(di, dj int) float64 { return field(i+di, j+dj) }
	plain := (p(1, 0)-p(-1, 0))*(q(0, 1)-q(0, -1)) -
		(p(0, 1)-p(0, -1))*(q(1, 0)-q(-1, 0))
	carried := p(1, 0)*(q(1, 1)-q(1, -1)) - p(-1, 0)*(q(-1, 1)-q(-1, -1)) -
		p(0, 1)*(q(1, 1)-q(-1, 1)) + p(0, -1)*(q(1, -1)-q(-1, -1))
	carrying := q(0, 1)*(p(1, 1)-p(-1, 1)) - q(0, -1)*(p(1, -1)-p(-1, -1)) -
		q(1, 0)*(p(1, 1)-p(1, -1)) + q(-1, 0)*(p(-1, 1)-p(-1, -1))
	sum := (plain + carried + carrying) / (12 * grid.run() * grid.rise())
	return sum / (grid.radius * grid.radius * math.Cos(grid.latitude(j)))
}
