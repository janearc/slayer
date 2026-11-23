package session

// Ground at a latitude and longitude on a mapped sphere. A set that places
// its terrain by lat and lon gets a window laid on the sphere about that
// place, its heights read from the map, and below the map's own detail
// made by package titan. This file says how the window is cut and how the
// run says how much of it was made.

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/titan"
)

// the numbers a ground on a sphere is cut with
const (
	// patchShare is the most a window may span, as a share of the sphere's
	// radius. picked: a hundredth, about 60 km on Venus. The window is a
	// flat patch laid on the sphere, and past a hundredth of the radius the
	// patch and the sphere part by tens of metres; a set filmed from the
	// ground has no use for more, and no one has drawn one wider.
	patchShare = 0.01
	// windowMiddle is the share of the window's width and depth at which the
	// place asked for stands. defined: the middle is half way.
	windowMiddle = 0.5
	// groupSize is how many digits a comma groups in a written count.
	// defined: thousands.
	groupSize = 3
	// bigMetres is the length from which a length is written to the whole
	// metre. picked: below ten metres two places are what a person reads.
	bigMetres = 10.0
	// twoPlaces and fourPlaces round the lengths under bigMetres. picked:
	// a length of a tenth of a metre or more keeps two places, and a
	// smaller one four.
	twoPlaces, fourPlaces = 100.0, 10000.0
	// tenthMetre is the length under which four places are kept. picked:
	// see twoPlaces.
	tenthMetre = 0.1
	// floatBytes and flagBytes are what one height and one made flag take
	// in the cache. defined: a float64 is eight bytes and a flag one.
	floatBytes, flagBytes = 8, 1
	// headBytes is what the cache's own header takes: the spacing and the
	// window's length, a number each. defined.
	headBytes = 2 * floatBytes
)

// groundPlace is where on a sphere a set places its ground, how big a window
// it asks for there, in metres, and whether titan makes the ground finer than
// the map from a seed. source is the map's key in sources.yaml. slope and
// longest are the relief titan is asked to give the made ground, where
// relieved says the set asked one.
type groundPlace struct {
	latitude, longitude float64
	across, down        float64
	detailed, relieved  bool
	seed                uint64
	source              string
	slope, longest      float64
}

// placedGround is a window laid on a sphere, and what titan made of it: for
// each point, in the window's order, how many of its metres were made, up or
// down, and whether the point is below the map's detail and generated. Both
// are empty where the set asked for the map alone. spacing is the map's
// samples' distance apart there, in metres.
type placedGround struct {
	window    terrain.Window
	made      []float64
	generated []bool
	spacing   float64
}

// laidOn is the latitude and longitude of a place east and north of the
// window's middle, in metres.
//
// The window is a flat patch laid on the sphere. A distance north is that
// distance over the radius of change in latitude, in radians. A distance
// east is that over the radius times the cosine of the place's latitude, as
// change in longitude. That is good while the window is small beside the
// radius, and refuseWide holds it so.
func (place groundPlace) laidOn(radius, east, north float64) (latitude,
	longitude float64) {
	latitude = place.latitude + north/radius/maths.Degree
	longitude = place.longitude + east/(radius*
		math.Cos(place.latitude*maths.Degree))/maths.Degree
	return latitude, longitude
}

// refuseWide refuses a window too wide for a flat patch to be laid on a
// sphere of the radius, in metres.
func (place groundPlace) refuseWide(radius float64) error {
	most := radius * patchShare
	if math.Max(place.across, place.down) > most {
		return fmt.Errorf("a window %g by %g m is too wide to lay flat on a "+
			"sphere of radius %g m: the most is %g m, picked as a hundredth "+
			"of the radius", place.across, place.down, radius, most)
	}
	return nil
}

// onSphere is the window cut from a raster of a sphere, by the place and the
// window's counts and step in cut. The raster is read by its three
// methods, so a test can stand in for it.
func (cut groundCut) onSphere(raster titan.Sphere) (placedGround, error) {
	place := cut.place
	radius, ok := raster.EquirectangularSphere()
	if !ok {
		return placedGround{}, fmt.Errorf("%s is not an equirectangular map "+
			"of a sphere, and ground placed by lat and lon needs one",
			place.source)
	}
	if err := place.refuseWide(radius); err != nil {
		return placedGround{}, err
	}
	eastward, southward, err := raster.GroundSpacing(place.latitude)
	if err != nil {
		return placedGround{}, err
	}
	read, err := place.reader(raster, radius, cut.step)
	if err != nil {
		return placedGround{}, err
	}
	found := placedGround{spacing: math.Max(eastward, southward)}
	found.window, err = terrain.BuildWindow(cut.west, cut.north, cut.across,
		cut.down, cut.step, func(column, row int) (float64, error) {
			east := float64(column)*cut.step - place.across*windowMiddle
			north := place.down - float64(row)*cut.step -
				place.down*windowMiddle
			latitude, longitude := place.laidOn(radius, east, north)
			return found.keep(read(latitude, longitude))
		})
	return found, err
}

// keep notes how much of a height was made, if titan was asked, and gives
// the height on.
func (found *placedGround) keep(height, made float64, generated bool,
	err error) (float64, error) {
	if err == nil && !math.IsNaN(made) {
		found.made = append(found.made, made)
		found.generated = append(found.generated, generated)
	}
	return height, err
}

// reader is how the place's heights are read: from the map alone, bilinear
// between its samples, or from titan on the map, at the step. A point is
// read by its latitude and longitude, and gives its height, how many of its
// metres were made, whether it is generated, and a refusal that names the
// place. Without titan, made is not a number.
func (place groundPlace) reader(raster titan.Sphere, radius,
	step float64) (func(latitude, longitude float64) (float64, float64, bool,
	error), error) {
	if !place.detailed {
		return func(latitude, longitude float64) (float64, float64, bool,
			error) {
			height, known, err := raster.EquirectangularElevation(longitude,
				latitude)
			if err == nil && !known {
				err = fmt.Errorf("the map has no height")
			}
			return height, math.NaN(), false,
				placeError(latitude, longitude, err)
		}, nil
	}
	provider, err := titan.NewSphereMap(raster, place.source)
	if err != nil {
		return nil, err
	}
	ground, err := place.titanGround(provider, radius)
	if err != nil {
		return nil, err
	}
	return func(latitude, longitude float64) (float64, float64, bool, error) {
		at, err := mesh.NewPoint(longitude, latitude)
		if err != nil {
			return 0, 0, false, placeError(latitude, longitude, err)
		}
		sample, err := ground.At(at, step)
		if err != nil {
			return 0, 0, false, placeError(latitude, longitude, err)
		}
		return sample.Height(), sample.Made(),
			sample.Kind() == titan.Generated, nil
	}, nil
}

// titanGround is titan's ground on the map, from the place's seed, with
// the relief the place asks where it asks one.
func (place groundPlace) titanGround(provider titan.Provider,
	radius float64) (*titan.Ground, error) {
	ground, err := titan.New(provider, radius, place.seed)
	if err != nil || !place.relieved {
		return ground, err
	}
	return ground, ground.Relief(place.slope, place.longest)
}

// placeError is an error with the latitude and longitude it happened at, or
// none.
func placeError(latitude, longitude float64, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("at latitude %.5f, longitude %.5f: %w", latitude,
		longitude, err)
}

// bytes is the placed ground as the cache keeps it: the map's spacing, the
// window's length and the window, then every made height and every flag.
func (found placedGround) bytes() []byte {
	window := found.window.Bytes()
	out := binary.LittleEndian.AppendUint64(nil, math.Float64bits(found.spacing))
	out = binary.LittleEndian.AppendUint64(out, uint64(len(window)))
	out = append(out, window...)
	for _, made := range found.made {
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(made))
	}
	for _, generated := range found.generated {
		out = append(out, boolByte(generated))
	}
	return out
}

// boolByte is a flag as the byte the cache keeps it as.
func boolByte(flag bool) byte {
	if flag {
		return 1 // defined: a flag set is one
	}
	return 0 // defined: a flag unset is nought
}

// placedFromBytes is the placed ground bytes made, or a refusal of bytes
// that do not say one.
func placedFromBytes(raw []byte) (placedGround, error) {
	if len(raw) < headBytes {
		return placedGround{}, fmt.Errorf("%d bytes is not a ground on a "+
			"sphere", len(raw))
	}
	found := placedGround{spacing: math.Float64frombits(
		binary.LittleEndian.Uint64(raw))}
	length := binary.LittleEndian.Uint64(raw[floatBytes:])
	raw = raw[headBytes:]
	if length > uint64(len(raw)) {
		return placedGround{}, fmt.Errorf("a window of %d bytes in %d", length,
			len(raw))
	}
	window, err := terrain.WindowFromBytes(raw[:length])
	if err != nil {
		return placedGround{}, err
	}
	found.window, raw = window, raw[length:]
	across, down, _ := window.Size()
	points := across * down
	if len(raw) == 0 {
		return found, nil
	}
	if len(raw) != points*(floatBytes+flagBytes) {
		return placedGround{}, fmt.Errorf("%d bytes of made ground for %d "+
			"points", len(raw), points)
	}
	for point := range points {
		found.made = append(found.made, math.Float64frombits(
			binary.LittleEndian.Uint64(raw[point*floatBytes:])))
		found.generated = append(found.generated,
			raw[points*floatBytes+point] != 0)
	}
	return found, nil
}

// sentence is what the run says of the ground: how many points, how far
// apart, from which map and how far apart its samples are, how many of the
// points are made, and how far the made ground rises and falls. The figures
// are the ground's own.
func (found placedGround) sentence(source string) string {
	across, down, step := found.window.Size()
	said := fmt.Sprintf("ground: %s points at %s m from %s, whose samples "+
		"are %s m apart; ", grouped(across*down), lengthText(step), source,
		lengthText(found.spacing))
	if len(found.made) == 0 {
		return said + "read from the map alone, between its samples, so " +
			"none of it is made"
	}
	count, low, high := 0, math.Inf(1), math.Inf(-1)
	for point, made := range found.made {
		if found.generated[point] {
			count++
		}
		low, high = math.Min(low, made), math.Max(high, made)
	}
	return said + madeShare(count, across*down) + "; made relief spans " +
		lengthText(high-low) + " m"
}

// madeShare says how many of the points are below the map and made.
func madeShare(made, points int) string {
	switch made {
	case points:
		return "every point is below the map and made"
	case 0:
		return "no point is below the map, and none is made"
	}
	return fmt.Sprintf("%s of them are below the map and made",
		grouped(made))
}

// grouped is a count with a comma between each three digits.
func grouped(count int) string {
	digits := strconv.Itoa(count)
	var parts []string
	for len(digits) > groupSize {
		parts = append([]string{digits[len(digits)-groupSize:]}, parts...)
		digits = digits[:len(digits)-groupSize]
	}
	return strings.Join(append([]string{digits}, parts...), ",")
}

// lengthText is a length in metres as a person reads it: whole metres from
// ten up, with commas, two places from a tenth, and four below.
func lengthText(metres float64) string {
	switch {
	case metres >= bigMetres:
		return grouped(int(math.Round(metres)))
	case metres >= tenthMetre:
		return strconv.FormatFloat(math.Round(metres*twoPlaces)/twoPlaces,
			'f', -1, 64)
	}
	return strconv.FormatFloat(math.Round(metres*fourPlaces)/fourPlaces,
		'f', -1, 64)
}
