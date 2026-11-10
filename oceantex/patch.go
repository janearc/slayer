package oceantex

import (
	"errors"
	"math"
	"math/cmplx"
	"math/rand/v2"
)

// Patch is one square of sea, periodic, as the random amplitudes of its
// waves. It can be asked for the surface at any time.
type Patch struct {
	n      int
	extent float64
	sea    Sea
	// shortest and longest are the band of wave numbers the patch
	// carries, radians a metre: from shortest up to, not including,
	// longest. A patch on its own carries everything its grid can.
	shortest, longest float64
	start             []complex128 // amplitude of each wave at time zero
}

// Field is the surface of a patch at one moment, row by row, N by N.
type Field struct {
	N      int
	Extent float64   // metres along a side
	Height []float64 // metres above the mean
	SlopeX []float64 // rise per metre along x
	SlopeY []float64 // rise per metre along y
	// PushX and PushY are how far the water at each point is carried
	// sideways by the waves through it, in metres: the displacement of
	// Tessendorf's choppy waves, which sharpens crests and flattens
	// troughs. StretchXX, StretchXY and StretchYY are its gradient.
	PushX, PushY                    []float64
	StretchXX, StretchXY, StretchYY []float64
}

// wave is the signed wave number of a grid index: 0, 1, ... n/2-1, then
// -n/2 ... -1, the order the FFT keeps them in.
func wave(index, n int) int {
	if index < n/2 {
		return index
	}
	return index - n
}

// NewPatch draws the waves of a sea on an n by n grid, extent metres along
// a side. n must be a power of two, at least 2. The same seed gives the
// same sea.
//
// This is the Fourier synthesis of [MWM87]. Each wave vector on the grid
// gets a complex Gaussian amplitude whose mean square is half the spectrum
// there times the cell's area in wavenumber space, half because a real
// surface needs each wave and its mirror image [Ho15, eqs. 46-47].
func NewPatch(sea Sea, n int, extent float64, seed uint64) (*Patch, error) {
	return newBand(sea, n, extent, seed, 0, math.Inf(1))
}

// newBand is NewPatch keeping only the waves whose wave number is at
// least shortest and under longest.
func newBand(
	sea Sea, n int, extent float64, seed uint64, shortest, longest float64,
) (*Patch, error) {
	if n < 2 || n&(n-1) != 0 {
		return nil, errors.New("oceantex: grid size must be a power of two")
	}
	if !(extent > 0) || !finite(extent) {
		return nil, errors.New("oceantex: extent must be positive")
	}
	limits, err := readLimits()
	if err != nil {
		return nil, err
	}
	if n > limits.WaveCells/n {
		return nil, errors.New("oceantex: wave grid exceeds cell budget")
	}
	if !finite(sea.Wind) || !finite(sea.Fetch) || !finite(sea.Toward) ||
		!finite(sea.Depth) || sea.Wind < 0 || sea.Fetch < 0 || sea.Depth < 0 {
		return nil, errors.New("oceantex: invalid sea state")
	}

	source := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	cell := 2 * math.Pi / extent
	p := &Patch{n: n, extent: extent, sea: sea,
		shortest: shortest, longest: longest,
		start: make([]complex128, n*n)}
	for iy := 0; iy < n; iy++ {
		for ix := 0; ix < n; ix++ {
			re, im := source.NormFloat64(), source.NormFloat64()
			kx, ky, carried := p.vector(ix, iy)
			if !carried {
				continue
			}
			density := sea.Wavenumber(kx, ky)
			size := math.Sqrt(density * cell * cell / 2)
			p.start[iy*n+ix] = complex(re, im) * complex(size/math.Sqrt2, 0)
		}
	}
	return p, nil
}

// vector is the wave vector at a grid index, and whether the patch
// carries a wave there. The row with no mirror image stays flat.
func (p *Patch) vector(ix, iy int) (kx, ky float64, carried bool) {
	cell := 2 * math.Pi / p.extent
	wx, wy := wave(ix, p.n), wave(iy, p.n)
	kx, ky = float64(wx)*cell, float64(wy)*cell
	k := math.Hypot(kx, ky)
	carried = wx != -p.n/2 && wy != -p.n/2 &&
		k >= p.shortest && k < p.longest
	return kx, ky, carried
}

// Resolved is the slope variance the grid carries, expected over every sea
// with this wind: along the wind and across it. It is the spectrum times
// the squared wave number, summed over the grid.
func (p *Patch) Resolved() (along, across float64) {
	cell := 2 * math.Pi / p.extent
	sin, cos := math.Sincos(p.sea.Toward)
	for iy := 0; iy < p.n; iy++ {
		for ix := 0; ix < p.n; ix++ {
			kx, ky, carried := p.vector(ix, iy)
			if !carried {
				continue
			}
			power := p.sea.Wavenumber(kx, ky) * cell * cell
			ka, kc := kx*cos+ky*sin, -kx*sin+ky*cos
			along += ka * ka * power
			across += kc * kc * power
		}
	}
	return along, across
}

// At is the surface at a time in seconds. Each wave turns at the
// frequency the dispersion relation gives it.
func (p *Patch) At(seconds float64) *Field {
	n := p.n
	grids := make([][]complex128, 8)
	for i := range grids {
		grids[i] = make([]complex128, n*n)
	}
	for iy := 0; iy < n; iy++ {
		for ix := 0; ix < n; ix++ {
			kx, ky, _ := p.vector(ix, iy)
			k := math.Hypot(kx, ky)
			if k == 0 {
				continue
			}
			omega, _ := p.sea.turning(k)
			turn := cmplx.Rect(1, -omega*seconds)
			mirror := ((n-iy)%n)*n + (n-ix)%n
			h := p.start[iy*n+ix]*turn + cmplx.Conj(p.start[mirror]*turn)
			at := iy*n + ix
			grids[0][at] = h
			grids[1][at] = complex(0, kx) * h
			grids[2][at] = complex(0, ky) * h
			// the push is i (k/|k|) h, and its gradient is i k times that.
			// The sign is the one that makes a single wave Gerstner's:
			// height A cos, push -A sin, water gathered at the crests.
			pushX := complex(0, kx/k) * h
			pushY := complex(0, ky/k) * h
			grids[3][at] = pushX
			grids[4][at] = pushY
			grids[5][at] = complex(0, kx) * pushX
			grids[6][at] = complex(0, ky) * pushX
			grids[7][at] = complex(0, ky) * pushY
		}
	}
	out := make([][]float64, len(grids))
	for i, grid := range grids {
		fft2(grid, n, true)
		out[i] = make([]float64, n*n)
		for j, v := range grid {
			out[i][j] = real(v)
		}
	}
	return &Field{N: n, Extent: p.extent,
		Height: out[0], SlopeX: out[1], SlopeY: out[2],
		PushX: out[3], PushY: out[4],
		StretchXX: out[5], StretchXY: out[6], StretchYY: out[7]}
}

// sample is a grid's value at a place in metres, on the straight lines
// between its four nearest samples. The patch repeats in every direction.
func (f *Field) sample(grid []float64, x, y float64) float64 {
	n := f.N
	gx := x / f.Extent * float64(n)
	gy := y / f.Extent * float64(n)
	fx, fy := math.Floor(gx), math.Floor(gy)
	tx, ty := gx-fx, gy-fy
	x0 := ((int(fx) % n) + n) % n
	y0 := ((int(fy) % n) + n) % n
	x1, y1 := (x0+1)%n, (y0+1)%n
	top := grid[y0*n+x0]*(1-tx) + grid[y0*n+x1]*tx
	bottom := grid[y1*n+x0]*(1-tx) + grid[y1*n+x1]*tx
	return top*(1-ty) + bottom*ty
}
