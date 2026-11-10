package weather

import (
	"image/color"
	"math"
	"testing"
)

func TestWeatherMapsKeepSouthAtTheBottom(t *testing.T) {
	made := small(t, 7)
	grid := made.grid
	for j := range grid.rows {
		for i := range grid.across {
			n := grid.at(i, j)
			made.origin[n] = 0
			made.age[n] = 0
			if j == 0 {
				made.vorticity[n] = made.spin
				made.age[n] = 1e9
			}
			if j == grid.rows-1 {
				made.vorticity[n] = -made.spin
			}
		}
	}
	spun, bands, aged := made.Spun(), made.Bands(0.1), made.Aged(1)
	if spun.RGBAAt(0, grid.rows-1) != (color.RGBA{255, 0, 0, 255}) || spun.RGBAAt(0, 0) != (color.RGBA{0, 0, 255, 255}) {
		t.Fatal("vorticity map reversed")
	}
	if spun.RGBAAt(0, 1) != (color.RGBA{255, 255, 255, 255}) {
		t.Fatal("still air is not white")
	}
	if bands.RGBAAt(0, 0) != (color.RGBA{128, 102, 70, 255}) {
		t.Fatal("equatorial bands changed")
	}
	if aged.RGBAAt(0, 0) != (color.RGBA{255, 255, 255, 255}) || aged.RGBAAt(0, grid.rows-1) != (color.RGBA{130, 75, 40, 255}) {
		t.Fatal("age map endpoint or latitude reversed")
	}
	if made.Clock() != 0 || made.Fastest() <= 0 {
		t.Fatal("clock or wind ignored")
	}
}

func TestWeatherInterpolationWrapsAndPolesForgetLongitude(t *testing.T) {
	made := small(t, 5)
	g := made.grid
	for j := range g.rows {
		for i := range g.across {
			n := g.at(i, j)
			made.origin[n] = float64(i)
			made.vorticity[n] = made.spin * float64(i)
			made.age[n] = -float64(i)
		}
	}
	latitude := g.latitude(5)
	origin, spin, age := made.At(latitude, 1.5*g.run())
	if math.Abs(origin-1.5) > 1e-12 || math.Abs(spin-.75) > 1e-12 || age != 0 {
		t.Fatalf("midpoint %g %g %g", origin, spin, age)
	}
	a, b, c := made.At(latitude, -.5*g.run())
	d, e, f := made.At(latitude, (float64(g.across)-.5)*g.run())
	if math.Abs(a-d) > 1e-12 || math.Abs(b-e) > 1e-12 || c != f {
		t.Fatal("longitude seam")
	}
	for _, pole := range []float64{-math.Pi / 2, math.Pi / 2} {
		origin, spin, age := made.At(pole, 0)
		other, _, _ := made.At(pole, math.Pi)
		if math.Abs(origin-15.5) > 1e-12 || origin != other || spin != 0 || age != 0 {
			t.Fatalf("pole %g %g %g %g", origin, other, spin, age)
		}
	}
}

func TestStillWeatherHasNoZonalEnergyAndJetsHoldTheirEnergy(t *testing.T) {
	made := small(t, 4)
	if made.Zonality() != 0 {
		t.Fatal("energy in still air")
	}
	for _, v := range made.Zonal() {
		if v != 0 {
			t.Fatal("wind in still air")
		}
	}
	for j := range made.grid.rows {
		for i := range made.grid.across {
			made.vorticity[made.grid.at(i, j)] = 1e-5 * math.Sin(float64(j))
		}
	}
	if got := made.Zonality(); math.Abs(got-1) > 1e-12 {
		t.Fatalf("pure zonal flow holds %g", got)
	}
}

func TestWeatherConstructorsRefuseInvalidDomains(t *testing.T) {
	for _, test := range [][4]float64{{7, 4, 1, 1}, {8, 3, 1, 1}, {8, 4, 0, 1}, {8, 4, math.Pi / 2, 1}, {8, 4, 1, 0}} {
		if _, err := NewGrid(int(test[0]), int(test[1]), test[2], test[3]); err == nil {
			t.Fatalf("grid %v", test)
		}
	}
	for _, test := range [][3]float64{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}} {
		if _, err := NewStirring(test[0], test[1], test[2]); err == nil {
			t.Fatal("invalid stirring")
		}
		made := small(t, 1)
		if _, err := New(made.grid, test[0], test[1], test[2], made.stirring, 1); err == nil {
			t.Fatal("invalid weather time")
		}
	}
}
