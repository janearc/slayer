package weather

import (
	"image"
	"image/color"
	"math"
)

// Spun is the vorticity across the grid as a picture, latitude and
// longitude straight across: red where the air turns the planet's way
// relative to its spin's local share, blue the other, white still, as
// strong as the turning is against the planet's own spin.
func (made *Weather) Spun() *image.RGBA {
	grid := made.grid
	picture := image.NewRGBA(image.Rect(0, 0, grid.across, grid.rows))
	for j := range grid.rows {
		for i := range grid.across {
			share := made.vorticity[grid.at(i, j)] / (2 * made.spin) * 5
			share = math.Max(-1, math.Min(1, share))
			picture.Set(i, grid.rows-1-j, diverging(share))
		}
	}
	return picture
}

// Bands is the bands' air across the grid as a picture: each cell shaded
// by the latitude its air came from, in stripes a band wide, so how the
// flow has torn and folded them shows.
func (made *Weather) Bands(every float64) *image.RGBA {
	grid := made.grid
	picture := image.NewRGBA(image.Rect(0, 0, grid.across, grid.rows))
	for j := range grid.rows {
		for i := range grid.across {
			stripe := math.Sin(math.Pi * made.origin[grid.at(i, j)] / every)
			shade := uint8(128 + 120*stripe)
			picture.Set(i, grid.rows-1-j, color.RGBA{shade,
				uint8(0.8 * float64(shade)), uint8(0.55 * float64(shade)), 255})
		}
	}
	return picture
}

// diverging is a share from minus one to one as blue through white to
// red.
func diverging(share float64) color.RGBA {
	fade := uint8(255 * (1 - math.Abs(share)))
	if share > 0 {
		return color.RGBA{255, fade, fade, 255}
	}
	return color.RGBA{fade, fade, 255, 255}
}

// Zonal is the mean wind east round each circle of latitude, metres a
// second, south to north: where jets have formed, it runs east and
// west by turns.
func (made *Weather) Zonal() []float64 {
	grid := made.grid
	psi := grid.walled(grid.streamOf(made.vorticity))
	means := make([]float64, grid.rows)
	for j := range grid.rows {
		for i := range grid.across {
			means[j] -= (psi(i, j+1) - psi(i, j-1)) /
				(2 * grid.rise() * grid.radius)
		}
		means[j] /= float64(grid.across)
	}
	return means
}

// At is the weather at a latitude and longitude in radians, longitude zero at
// the grid's first cell: the latitude its air came from, in radians; its
// vorticity against the planet's, over twice its spin; and its age, the seconds
// it has cooked at the top.
//
// age is never less than zero, though carrying it across a sharp edge leaves
// ripples that dip there. between cells the weather is read across the four
// about it.
//
// past the walls it fades toward its edge row's mean by the pole, spinning
// none, so no longitude is left there to fan out into spokes.
func (made *Weather) At(latitude, longitude float64) (origin, spin,
	age float64) {
	grid := made.grid
	down := (latitude+grid.limit)/grid.rise() - 0.5
	down = math.Max(0, math.Min(float64(grid.rows-1), down))
	round := longitude / grid.run()
	j, i := int(math.Floor(down)), int(math.Floor(round))
	north, east := down-float64(j), round-float64(i)
	upper := min(j+1, grid.rows-1)
	read := func(field []float64) float64 {
		south := field[grid.at(i, j)]*(1-east) + field[grid.at(i+1, j)]*east
		above := field[grid.at(i, upper)]*(1-east) +
			field[grid.at(i+1, upper)]*east
		return south*(1-north) + above*north
	}
	origin, spin, age = read(made.origin),
		read(made.vorticity)/(2*made.spin), math.Max(0, read(made.age))
	edge := grid.limit - grid.rise()/2
	if past := math.Abs(latitude) - edge; past > 0 {
		row := 0
		if latitude > 0 {
			row = grid.rows - 1
		}
		gone := math.Min(1, past/(math.Pi/2-edge))
		origin += gone * (made.rowMean(made.origin, row) - origin)
		spin -= gone * spin
		age += gone * (math.Max(0, made.rowMean(made.age, row)) - age)
	}
	return origin, spin, age
}

// rowMean is a field's mean round one row.
func (made *Weather) rowMean(field []float64, row int) float64 {
	sum := 0.0
	for i := range made.grid.across {
		sum += field[made.grid.at(i, row)]
	}
	return sum / float64(made.grid.across)
}

// tholin is the age picture's ramp, fresh to fully cooked: white,
// cream, yellow, orange, brown. a picture of the age, not the planet's
// colour, which its haze's optics make.
var tholin = [][3]float64{{255, 255, 255}, {250, 238, 205},
	{240, 205, 120}, {215, 140, 60}, {130, 75, 40}}

// Aged is the air's age across the grid as a picture, by how far it has
// cooked over so many seconds, on the tholin ramp.
func (made *Weather) Aged(cooking float64) *image.RGBA {
	grid := made.grid
	picture := image.NewRGBA(image.Rect(0, 0, grid.across, grid.rows))
	for j := range grid.rows {
		for i := range grid.across {
			age := math.Max(0, made.age[grid.at(i, j)])
			cooked := 1 - math.Exp(-age/cooking)
			along := cooked * float64(len(tholin)-1)
			k := min(int(along), len(tholin)-2)
			t := along - float64(k)
			var shade [3]uint8
			for c := range shade {
				shade[c] = uint8(tholin[k][c] + t*(tholin[k+1][c]-tholin[k][c]))
			}
			picture.Set(i, grid.rows-1-j, color.RGBA{shade[0], shade[1],
				shade[2], 255})
		}
	}
	return picture
}

// Zonality is the share of the weather's kinetic energy that its zonal
// jets hold, the mean east wind round each circle, against all its
// winds', each cell weighed by its area.
//
// jupiter's is 0.93 (galperin and others 2014, from cassini's winds of
// december 2000). a flow whose eddies hold most of its energy tears its
// bands to noise.
func (made *Weather) Zonality() float64 {
	grid := made.grid
	psi := grid.walled(grid.streamOf(made.vorticity))
	zonal := made.Zonal()
	held, all := 0.0, 0.0
	for j := range grid.rows {
		cosine := math.Cos(grid.latitude(j))
		held += zonal[j] * zonal[j] * cosine * float64(grid.across)
		for i := range grid.across {
			east := -(psi(i, j+1) - psi(i, j-1)) /
				(2 * grid.rise() * grid.radius)
			north := (psi(i+1, j) - psi(i-1, j)) /
				(2 * grid.run() * cosine * grid.radius)
			all += (east*east + north*north) * cosine
		}
	}
	if all == 0 {
		return 0
	}
	return held / all
}

// Clock is how many seconds the weather has run.
func (made *Weather) Clock() float64 { return made.clock }

// Fastest is the fastest wind on the grid, metres a second.
func (made *Weather) Fastest() float64 { return made.fastest() }
