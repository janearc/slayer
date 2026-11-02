package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Banded is the face of a gas giant: colour by latitude from a run of
// paints, the bands roiled at their edges by noise sheared along them,
// and spots, storms that swirl what is round them.
//
// It is a function of position in the body's own space, so it paints a
// volume as well as a surface: ask it anywhere inside the giant and it
// says what colour the gas is there.
type Banded struct {
	// Paints are the bands from the south pole to the north, evenly
	// spaced, blended between.
	Paints []maths.Light
	// Axis is the body's north; zero means up.
	Axis maths.Vec
	// Roil is how far, in the sine of the latitude, from minus one to
	// one, the noise pushes a band's edge. Nought is clean stripes.
	Roil float64
	// Shear is how much the roil is stretched along the bands, so the
	// turbulence streaks rather than blobs. One is no stretch.
	Shear float64
	// Grain is how big the roil's features are, as a share of the body.
	// Zero means a tenth.
	Grain float64
	// Fine is a second, finer roil: how far it pushes the latitude, at
	// a third of the grain, for the turbulence inside a band. Nought is
	// none.
	Fine float64
	// Jets is how many jet streams run round the body, alternating in
	// direction from one band to the next, which shears the roil the
	// opposite way in neighbouring bands. Nought is none.
	Jets int
	// Mottle is how much a band's paint varies along its length, from zero
	// to one, so a band is not one colour all the way round.
	Mottle float64
	// Spots are the storms.
	Spots []Spot
	// Streets are rows of small storms, each a run of spots at one
	// latitude.
	Streets []Street
	Seed    uint64
}

// Street is a row of storms along a latitude: Count spots of Size, each
// painted Paint and swirled by Turn, spaced round the body by hash.
type Street struct {
	Lat        float64
	Count      int
	Size, Turn float64
	Paint      maths.Light
	Seed       uint64
}

// spots is the street's storms as spots.
func (s Street) spots() []Spot {
	if s.Count <= 0 {
		return nil
	}
	out := make([]Spot, 0, s.Count)
	for i := range s.Count {
		k := int64(i)
		lon := (float64(i) + noise.Hash(s.Seed+1, k)*0.6) /
			float64(max(1, s.Count)) * 2 * math.Pi
		out = append(out, Spot{
			Lat:     s.Lat + (noise.Hash(s.Seed+2, k)-0.5)*s.Size,
			Lon:     lon,
			Size:    s.Size * (0.7 + 0.6*noise.Hash(s.Seed+3, k)),
			Stretch: 1.3 + noise.Hash(s.Seed+4, k),
			Turn:    s.Turn,
			Paint:   s.Paint,
		})
	}
	return out
}

// Spot is a storm on a gas giant: an oval at a latitude and a longitude
// that turns the gas round it, with a paint of its own at its heart.
type Spot struct {
	// Lat is the sine of the latitude, minus one at the south pole to one
	// at the north, so spots evenly spaced in it are not in degrees; Lon
	// is in radians round the axis.
	Lat, Lon float64
	// Size is the spot's radius in latitude units, and Stretch how many
	// times wider than tall it is; one is round.
	Size, Stretch float64
	// Turn is how far, in radians, the gas at the heart is swirled.
	Turn float64
	// Paint is the colour at the heart, blending out to the band's.
	Paint maths.Light
}

// north is the axis, with zero read as up.
func (b Banded) north() maths.Vec {
	if b.Axis == (maths.Vec{}) {
		return maths.Up
	}
	return b.Axis.Unit()
}

// grain is the roil's feature size, with zero read as a tenth.
func (b Banded) grain() float64 {
	if b.Grain <= 0 {
		return 0.1
	}
	return b.Grain
}

// where is a point as a latitude from minus one to one and a longitude
// in radians, in the body's own frame.
func (b Banded) where(p maths.Vec) (lat, lon float64) {
	n := b.north()
	d := p.Unit()
	lat = d.Dot(n)
	// a pair of axes square to north, for the longitude
	e := maths.XAxis
	if math.Abs(n.Dot(e)) > 0.9 {
		e = maths.ZAxis
	}
	e = e.Sub(n.Scale(n.Dot(e))).Unit()
	f := n.Cross(e)
	return lat, math.Atan2(d.Dot(f), d.Dot(e))
}

// At is the gas's colour at p.
func (b Banded) At(p maths.Vec) maths.Light {
	if len(b.Paints) == 0 {
		return maths.Light{}
	}
	lat, lon := b.where(p)
	// the storms swirl the place being asked about
	tint, tintBy := maths.Light{}, 0.0
	for _, s := range b.Spots {
		lat, lon, tint, tintBy = s.swirl(lat, lon, tint, tintBy)
	}
	for _, st := range b.Streets {
		for _, s := range st.spots() {
			lat, lon, tint, tintBy = s.swirl(lat, lon, tint, tintBy)
		}
	}
	// the jets slide the roil round the body by latitude, the opposite
	// way in neighbouring bands, so the turbulence shears between them
	jet := 0.0
	if b.Jets > 0 {
		jet = 0.35 * math.Sin(float64(b.Jets)*math.Pi*lat)
	}
	// the roil pushes the latitude by noise, stretched along the band,
	// at two scales: the waves along a band's edge and the turbulence
	// inside it
	shear := math.Max(b.Shear, 1)
	g := b.grain()
	x := math.Cos(lon+jet) / (g * shear)
	z := math.Sin(lon+jet) / (g * shear)
	if b.Roil > 0 {
		lat += b.Roil * noise.Fractal(b.Seed, x, lat/g, z, 4)
	}
	if b.Fine > 0 {
		f := 3.0
		lat += b.Fine * noise.Fractal(b.Seed+7, x*f, lat/g*f, z*f, 5)
	}
	band := b.band(lat)
	if b.Mottle > 0 {
		m := noise.Fractal(b.Seed+9, x*0.5, lat/g, z*0.5, 4)
		band = band.Scale(1 + b.Mottle*m)
	}
	if tintBy > 0 {
		band = band.Scale(1 - tintBy).Add(tint.Scale(tintBy))
	}
	return band
}

// band is the paint at a latitude: the two nearest paints, blended. a
// latitude that is no number, from a point that is none, is no number of
// light, the same on every machine.
func (b Banded) band(lat float64) maths.Light {
	if math.IsNaN(lat) {
		return maths.Grey(math.NaN())
	}
	n := len(b.Paints)
	if n == 1 {
		return b.Paints[0]
	}
	t := (math.Max(-1, math.Min(1, lat)) + 1) / 2 * float64(n-1)
	i := int(math.Floor(t))
	if i >= n-1 {
		return b.Paints[n-1]
	}
	f := t - float64(i)
	return b.Paints[i].Scale(1 - f).Add(b.Paints[i+1].Scale(f))
}

// swirl turns a latitude and longitude round the spot if they are near
// it, and says how much of the spot's paint belongs there.
func (s Spot) swirl(lat, lon float64, tint maths.Light,
	tintBy float64) (float64, float64, maths.Light, float64) {
	if s.Size <= 0 {
		return lat, lon, tint, tintBy
	}
	stretch := math.Max(s.Stretch, 1e-3)
	// the offset from the spot's heart, with the longitude brought to
	// the same side and squashed by the stretch
	dlon := math.Remainder(lon-s.Lon, 2*math.Pi)
	cosLat := math.Cos(math.Asin(math.Max(-1, math.Min(1, s.Lat))))
	u, v := dlon*cosLat/stretch, lat-s.Lat
	r := math.Hypot(u, v) / s.Size
	if r >= 1 {
		return lat, lon, tint, tintBy
	}
	in := 1 - maths.Smooth(0, 1, r)
	a := s.Turn * in
	c, sn := math.Cos(a), math.Sin(a)
	u, v = u*c-v*sn, u*sn+v*c
	lat = s.Lat + v
	lon = s.Lon + u*stretch/cosLat
	tint = tint.Scale(tintBy).Add(s.Paint.Scale(in))
	tintBy = tintBy + in*(1-tintBy)
	if tintBy > 0 {
		tint = tint.Scale(1 / tintBy)
	}
	return lat, lon, tint, tintBy
}
