package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// bandPoint is the point on the unit sphere at a latitude and a
// longitude about the default axis, up: its Y is the latitude, and its
// X and Z are the longitude toward -Z.
func bandPoint(lat, lon float64) maths.Vec {
	r := math.Sqrt(1 - lat*lat)
	return maths.V(r*math.Cos(lon), lat, -r*math.Sin(lon))
}

// bandAt is the colour of the paints at a latitude, blended by hand: the
// latitude from -1 to 1 is a place along the paints, and the colour is
// the two either side of it by how far along.
func bandAt(paints []maths.Light, lat float64) maths.Light {
	lat = math.Max(-1, math.Min(1, lat))
	place := (lat + 1) / 2 * float64(len(paints)-1)
	i := min(int(place), len(paints)-2)
	f := place - float64(i)
	return paints[i].Scale(1 - f).Add(paints[i+1].Scale(f))
}

// roiled is the colour the roil, the fine roil, the jets and the mottle
// make of a band at a latitude and longitude, worked from the noise.
//
// the roil and the fine roil each push the latitude by noise asked at
// the longitude's cosine and sine over the grain (a tenth if none) and
// the shear (one at least). the jets turn the longitude by 0.35 times
// the sine of their count times pi times the latitude.
//
// the mottle scales the colour by one plus its share of a noise at half
// the longitude.
func roiled(b Banded, lat, lon float64) maths.Light {
	g := b.Grain
	if g <= 0 {
		g = 0.1
	}
	shear := math.Max(b.Shear, 1)
	jet := 0.0
	if b.Jets > 0 {
		jet = 0.35 * math.Sin(float64(b.Jets)*math.Pi*lat)
	}
	x, z := math.Cos(lon+jet)/(g*shear), math.Sin(lon+jet)/(g*shear)
	if b.Roil > 0 {
		lat += b.Roil * noise.Fractal(b.Seed, x, lat/g, z, 4)
	}
	if b.Fine > 0 {
		lat += b.Fine * noise.Fractal(b.Seed+7, 3*x, 3*(lat/g), 3*z, 5)
	}
	out := bandAt(b.Paints, lat)
	if b.Mottle > 0 {
		out = out.Scale(1 + b.Mottle*noise.Fractal(b.Seed+9, 0.5*x, lat/g, 0.5*z, 4))
	}
	return out
}

var roilPoints = []struct{ lat, lon float64 }{
	{0.3, math.Pi / 3}, {-0.55, 2.2}, {0.8, -1.0},
}

// holdsRoil says the banded giant is what the noise makes of it at each
// of the points, and that the feature changed something at one at least,
// so the test is not holding the plain bands.
func holdsRoil(t *testing.T, why string, b, plain Banded) {
	t.Helper()
	moved := 0.0
	for _, p := range roilPoints {
		got, want := b.At(bandPoint(p.lat, p.lon)), roiled(b, p.lat, p.lon)
		if !near(got, want) {
			t.Errorf("%s at %v: %v, want %v", why, p, got, want)
		}
		moved = math.Max(moved, math.Abs(want.R-plain.At(bandPoint(p.lat, p.lon)).R))
		moved = math.Max(moved, math.Abs(want.G-plain.At(bandPoint(p.lat, p.lon)).G))
	}
	if moved < 1e-3 {
		t.Errorf("%s: the feature moved the colour by only %v", why, moved)
	}
}

func roilBands() Banded {
	return Banded{Paints: []maths.Light{red3, green3, blue3}, Seed: 4}
}

// The roil pushes the latitude by Roil times the noise at the point, and every
// part of the colour follows, not red alone: the grain and the shear set the
// noise's scale, a grain of zero is a tenth and a shear below one is one.
func TestBandedRoilIsTheNoiseOverTheGrainAndShear(t *testing.T) {
	for _, c := range []struct{ grain, shear float64 }{
		{0, 0}, {0.25, 0}, {1, 0}, {0.25, 0.5}, {0.25, 3}, {0, 2},
	} {
		b := roilBands()
		b.Roil, b.Grain, b.Shear = 0.3, c.grain, c.shear
		holdsRoil(t, "roil", b, roilBands())
	}
}

// The fine roil is a second push at three times the scale and five
// octaves, from the latitude the roil left.
func TestBandedFineRoilIsThreeTimesTheScale(t *testing.T) {
	b := roilBands()
	b.Fine = 0.25
	holdsRoil(t, "fine alone", b, roilBands())
	b.Roil, b.Grain = 0.15, 0.2
	holdsRoil(t, "fine after the roil", b, roilBands())
}

// The mottle scales the colour of the band by one plus its share of a
// noise at the latitude the roil and the fine roil left.
func TestBandedMottleScalesTheBand(t *testing.T) {
	b := roilBands()
	b.Mottle = 0.4
	holdsRoil(t, "mottle alone", b, roilBands())
	b.Roil, b.Fine, b.Grain = 0.1, 0.05, 0.2
	holdsRoil(t, "mottle after both", b, roilBands())
}

// The jets turn the longitude the roil is asked at by 0.35 times the
// sine of the count times pi times the latitude, a different way in each
// band. a count of one is a jet, and so is more.
func TestBandedJetsTurnTheLongitudeByBand(t *testing.T) {
	for _, jets := range []int{1, 3, 6} {
		b := roilBands()
		b.Roil, b.Jets = 0.3, jets
		plain := roilBands()
		plain.Roil = 0.3
		holdsRoil(t, "jets", b, plain)
	}
}

// All together, the same.
func TestBandedRoilFineJetsAndMottleTogether(t *testing.T) {
	b := roilBands()
	b.Roil, b.Fine, b.Jets, b.Mottle, b.Grain, b.Shear = 0.2, 0.05, 2, 0.3, 0.15, 2
	holdsRoil(t, "everything", b, roilBands())
}
