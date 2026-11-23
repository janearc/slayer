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

// The streams of the giant's hash, each defined as a distinct offset from
// the seed, so that each draw of a street, a roil and a mottle is alone.
const (
	// streetLonStream is defined: the offset of the stream a street's
	// longitudes are drawn from.
	streetLonStream = 1
	// streetLatStream is defined: the offset of the stream a street's
	// latitudes are drawn from.
	streetLatStream = 2
	// streetSizeStream is defined: the offset of the stream a street's
	// sizes are drawn from.
	streetSizeStream = 3
	// streetStretchStream is defined: the offset of the stream a street's
	// stretches are drawn from.
	streetStretchStream = 4
	// fineStream is defined: the offset of the stream the fine roil is
	// drawn from.
	fineStream = 7
	// mottleStream is defined: the offset of the stream the mottle is
	// drawn from.
	mottleStream = 9
)

// the numbers Banded is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// bandedGrain is the roil's feature size, as a share of the body, when
	// none is given.
	bandedGrain = pickedNumber("banded-grain")
	// bandedAxisParallel is how near to parallel with north, as the cosine
	// of the angle, the x axis may be and still serve to square the
	// longitude.
	bandedAxisParallel = pickedNumber("banded-axis-parallel")
	// bandedJetReach is how far, in radians, a jet slides the roil round
	// the body.
	bandedJetReach = pickedNumber("banded-jet-reach")
	// bandedRoilOctaves is how many scales of noise the roil has.
	bandedRoilOctaves = int(pickedNumber("banded-roil-octaves"))
	// bandedFineScale is how many times finer the fine roil is than the
	// roil.
	bandedFineScale = pickedNumber("banded-fine-scale")
	// bandedFineOctaves is how many scales of noise the fine roil has.
	bandedFineOctaves = int(pickedNumber("banded-fine-octaves"))
	// bandedMottleScale is how big the mottle's features are, against the
	// roil's.
	bandedMottleScale = pickedNumber("banded-mottle-scale")
	// bandedMottleOctaves is how many scales of noise the mottle has.
	bandedMottleOctaves = int(pickedNumber("banded-mottle-octaves"))
	// spotStretchFloor is the least a spot's stretch is read as, so that
	// no spot is infinitely tall.
	spotStretchFloor = pickedNumber("spot-stretch-floor")
	// streetLonJitter is how far, as a share of the gap between storms, a
	// storm is placed off its even place along a street.
	streetLonJitter = pickedNumber("street-lon-jitter")
	// streetSizeFloor is the least a storm of a street is, as a share of
	// the street's size.
	streetSizeFloor = pickedNumber("street-size-floor")
	// streetSizeSpread is how far a storm of a street's size is drawn
	// above its floor.
	streetSizeSpread = pickedNumber("street-size-spread")
	// streetStretchFloor is the least a storm of a street is stretched.
	streetStretchFloor = pickedNumber("street-stretch-floor")
)

// Street is a row of storms along a latitude: Count spots of Size, each
// painted Paint and swirled by Turn, spaced round the body by hash.
type Street struct {
	Lat        float64
	Count      int
	Size, Turn float64
	Paint      maths.Light
	Seed       uint64
}

// spot is the street's storm at a place along it.
func (street Street) spot(place int) Spot {
	index := int64(place)
	jitter := noise.Hash(street.Seed+streetLonStream, index) * streetLonJitter
	lon := (float64(place) + jitter) / float64(street.Count) * pair * math.Pi
	size := streetSizeFloor +
		streetSizeSpread*noise.Hash(street.Seed+streetSizeStream, index)
	return Spot{
		Lat: street.Lat +
			(noise.Hash(street.Seed+streetLatStream, index)-half)*street.Size,
		Lon:  lon,
		Size: street.Size * size,
		Stretch: streetStretchFloor +
			noise.Hash(street.Seed+streetStretchStream, index),
		Turn:  street.Turn,
		Paint: street.Paint,
	}
}

// spots is the street's storms as spots.
func (street Street) spots() []Spot {
	if street.Count <= nought {
		return nil
	}
	out := make([]Spot, nought, street.Count)
	for place := range street.Count {
		out = append(out, street.spot(place))
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
func (banded Banded) north() maths.Vec {
	if banded.Axis == noAxis {
		return maths.Up
	}
	return banded.Axis.Unit()
}

// grain is the roil's feature size, with zero read as the fallback.
func (banded Banded) grain() float64 {
	if banded.Grain <= nought {
		return bandedGrain
	}
	return banded.Grain
}

// where is a point as a latitude from minus one to one and a longitude
// in radians, in the body's own frame.
func (banded Banded) where(point maths.Vec) (latitude, longitude float64) {
	north := banded.north()
	direction := point.Unit()
	latitude = direction.Dot(north)
	// a pair of axes square to north, for the longitude
	reference := maths.XAxis
	if math.Abs(north.Dot(reference)) > bandedAxisParallel {
		reference = maths.ZAxis
	}
	shadow := north.Scale(north.Dot(reference))
	reference = reference.Sub(shadow).Unit()
	across := north.Cross(reference)
	return latitude, math.Atan2(direction.Dot(across), direction.Dot(reference))
}

// stormed is a place's latitude and longitude after every storm has
// swirled it, and the tint the storms give it and how much of it.
func (banded Banded) stormed(latitude, longitude float64) (float64, float64,
	maths.Light, float64) {
	tint := noPaint
	var tintBy float64
	for _, spot := range banded.Spots {
		latitude, longitude, tint, tintBy = spot.swirl(latitude, longitude,
			tint, tintBy)
	}
	for _, street := range banded.Streets {
		for place := range street.Count {
			latitude, longitude, tint, tintBy = street.spot(place).swirl(
				latitude, longitude, tint, tintBy)
		}
	}
	return latitude, longitude, tint, tintBy
}

// At is the gas's colour at the point.
func (banded Banded) At(point maths.Vec) maths.Light {
	if len(banded.Paints) == nought {
		return noPaint
	}
	latitude, longitude := banded.where(point)
	// the storms swirl the place being asked about
	latitude, longitude, tint, tintBy := banded.stormed(latitude, longitude)
	// the jets slide the roil round the body by latitude, the opposite
	// way in neighbouring bands, so the turbulence shears between them
	var jet float64
	if banded.Jets > nought {
		jet = bandedJetReach * math.Sin(float64(banded.Jets)*math.Pi*latitude)
	}
	// the roil pushes the latitude by noise, stretched along the band,
	// at two scales: the waves along a band's edge and the turbulence
	// inside it
	shear := math.Max(banded.Shear, unity)
	grain := banded.grain()
	x := math.Cos(longitude+jet) / (grain * shear)
	z := math.Sin(longitude+jet) / (grain * shear)
	if banded.Roil > nought {
		latitude += banded.Roil * noise.Fractal(banded.Seed, x, latitude/grain,
			z, bandedRoilOctaves)
	}
	if banded.Fine > nought {
		latitude += banded.Fine * noise.Fractal(banded.Seed+fineStream,
			x*bandedFineScale, latitude/grain*bandedFineScale,
			z*bandedFineScale, bandedFineOctaves)
	}
	band := banded.band(latitude)
	if banded.Mottle > nought {
		mottle := noise.Fractal(banded.Seed+mottleStream, x*bandedMottleScale,
			latitude/grain, z*bandedMottleScale, bandedMottleOctaves)
		band = band.Scale(unity + banded.Mottle*mottle)
	}
	if tintBy > nought {
		band = band.Scale(unity - tintBy).Add(tint.Scale(tintBy))
	}
	return band
}

// band is the paint at a latitude: the two nearest paints, blended. a
// latitude that is no number, from a point that is none, is no number of
// light, the same on every machine.
func (banded Banded) band(latitude float64) maths.Light {
	if math.IsNaN(latitude) {
		return noNumberLight()
	}
	count := len(banded.Paints)
	if count == unity {
		return banded.Paints[firstPlace]
	}
	along := (clamped(latitude) + unity) / pair * float64(count-unity)
	place := int(math.Floor(along))
	if place >= count-unity {
		return banded.Paints[count-unity]
	}
	fraction := along - float64(place)
	return banded.Paints[place].Scale(unity - fraction).
		Add(banded.Paints[place+unity].Scale(fraction))
}

// swirl turns a latitude and longitude round the spot if they are near
// it, and says how much of the spot's paint belongs there.
func (spot Spot) swirl(latitude, longitude float64, tint maths.Light,
	tintBy float64) (float64, float64, maths.Light, float64) {
	if spot.Size <= nought {
		return latitude, longitude, tint, tintBy
	}
	stretch := math.Max(spot.Stretch, spotStretchFloor)
	// the offset from the spot's heart, with the longitude brought to
	// the same side and squashed by the stretch
	dlon := math.Remainder(longitude-spot.Lon, fullTurn)
	heartAngle := math.Asin(clamped(spot.Lat))
	cosLat := math.Cos(heartAngle)
	across, up := dlon*cosLat/stretch, latitude-spot.Lat
	reach := math.Hypot(across, up) / spot.Size
	if reach >= unity {
		return latitude, longitude, tint, tintBy
	}
	in := unity - maths.Smooth(nought, unity, reach)
	angle := spot.Turn * in
	cosine, sine := math.Cos(angle), math.Sin(angle)
	across, up = across*cosine-up*sine, across*sine+up*cosine
	latitude = spot.Lat + up
	longitude = spot.Lon + across*stretch/cosLat
	tint = tint.Scale(tintBy * (unity - in)).Add(spot.Paint.Scale(in))
	tintBy = tintBy + in*(unity-tintBy)
	if tintBy > nought {
		tint = tint.Scale(unity / tintBy)
	}
	return latitude, longitude, tint, tintBy
}
