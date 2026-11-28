package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// degreesPerRadian turns a logged angle into degrees.
const degreesPerRadian = 180 / math.Pi

// weatherSeeds are the skies the weather tests average over.
var weatherSeeds = []uint64{1, 2, 3, 4, 5}

// onSphere is the unit direction at a latitude and longitude in radians.
func onSphere(latitude, longitude float64) maths.Vec {
	flat := math.Cos(latitude)
	return maths.V(flat*math.Cos(longitude), math.Sin(latitude),
		flat*math.Sin(longitude))
}

// cloudedShare is the share of the longitudes round a latitude's circle
// that are clouded in the middle of the layer, over every test sky.
func cloudedShare(t *testing.T, latitude float64) float64 {
	t.Helper()
	const longitudes = 360
	var clouded, asked int
	for _, seed := range weatherSeeds {
		layer := cloudsLayer(t)
		layer.Seed = seed
		middle := layer.Radius + layer.Low + (layer.High-layer.Low)/10
		for step := range longitudes {
			at := onSphere(latitude, 2*math.Pi*float64(step)/longitudes)
			asked++
			if layer.Density(at.Scale(middle)) > cloudVisible*layer.Thick {
				clouded++
			}
		}
	}
	return float64(clouded) / float64(asked)
}

// The equator is cloudier than the dry belts, and a storm track cloudier
// than the dry belts: the subtropics are never the cloudiest latitude.
func TestCloudBeltsByLatitude(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	equator := cloudedShare(t, 0)
	dry := cloudedShare(t, weather.DryAt)
	track := cloudedShare(t, weather.TrackAt)
	if equator <= dry {
		t.Errorf("equator %.2f clouded, no more than the dry belt's %.2f",
			equator, dry)
	}
	if track <= dry {
		t.Errorf("storm track %.2f clouded, no more than the dry belt's %.2f",
			track, dry)
	}
	if south := cloudedShare(t, -weather.DryAt); south > equator {
		t.Errorf("southern dry belt %.2f cloudier than the equator %.2f",
			south, equator)
	}
}

// Belts add cover at the equator and the tracks and take it from the dry
// belts, the same in either hemisphere.
func TestBeltsSignAndSymmetry(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	if got := weather.Belts(0); got <= 0 {
		t.Errorf("equator adds %v, want cover", got)
	}
	if got := weather.Belts(weather.DryAt); got >= 0 {
		t.Errorf("dry belt adds %v, want less cover", got)
	}
	if got := weather.Belts(weather.TrackAt); got <= 0 {
		t.Errorf("storm track adds %v, want cover", got)
	}
	if north, south := weather.Belts(0.7), weather.Belts(-0.7); north != south {
		t.Errorf("belts at 0.7 are %v north and %v south", north, south)
	}
}

// A storm turns the sky counter-clockwise in the north and clockwise in
// the south: content read at q is shown at p, so q to p goes round the
// storm's centre one way. Never the other way for a hemisphere.
func TestWhorlsTurnTheirHemisphereWay(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	for index, storm := range weather.Storms {
		// a point beside the heart, near the whorl's edge, where the turn
		// is under half a turn and so reads as one way and not the other
		beside := storm.Centre.Add(
			maths.YAxis.Cross(storm.Centre).Unit().Scale(
				weather.Size * 1.2)).Unit()
		read, _ := weather.Swirl(beside)
		round := read.Cross(beside).Dot(storm.Centre)
		if storm.Sign > 0 && round <= 0 {
			t.Errorf("north storm %d turns clockwise (%v)", index, round)
		}
		if storm.Sign < 0 && round >= 0 {
			t.Errorf("south storm %d turns counter-clockwise (%v)", index,
				round)
		}
		if (storm.Centre.Y > 0) != (storm.Sign > 0) {
			t.Errorf("storm %d at height %v turns by sign %v", index,
				storm.Centre.Y, storm.Sign)
		}
	}
}

// A whorl unwinds with distance: far from every storm the sky is not
// turned at all, and the turn is least at the whorl's edge.
func TestWhorlUnwinds(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	// one storm alone: the others have no centre, a quarter turn from
	// everywhere, and are out of reach
	storm := weather.Storms[0]
	weather.Storms = [stormCount]Storm{storm}
	across := maths.YAxis.Cross(storm.Centre).Unit()
	var last = math.Inf(1)
	for _, share := range []float64{1, 1.5, 2, 3} {
		at := storm.Centre.Add(across.Scale(share * weather.Size)).Unit()
		read, _ := weather.Swirl(at)
		moved := math.Acos(math.Min(1, read.Dot(at)))
		if moved >= last {
			t.Errorf("turn %v at %v radii is no less than %v nearer",
				moved, share, last)
		}
		last = moved
	}
	tiny := weather
	tiny.Size = 1e-3
	far := maths.V(0, 1, 0).Scale(-1)
	if read, added := tiny.Swirl(far); read != far || added != 0 {
		t.Errorf("sky far from every storm turned to %v adding %v", read, added)
	}
}

// The storms are in the storm tracks, three to a hemisphere, placed by
// the seed: the same seed the same places, another seed others.
func TestStormsStayInTheirTracks(t *testing.T) {
	layer := cloudsLayer(t)
	weather := layer.Weather()
	north, south := 0, 0
	for index, storm := range weather.Storms {
		latitude := math.Asin(storm.Centre.Y)
		t.Logf("storm %d: latitude %.0f longitude %.0f degrees, sign %v", index,
			latitude*degreesPerRadian,
			math.Atan2(storm.Centre.Z, storm.Centre.X)*degreesPerRadian,
			storm.Sign)
		off := math.Abs(math.Abs(latitude) - weather.TrackAt)
		if off > stormLatitudeSpread+1e-9 {
			t.Errorf("storm %d at latitude %v is %v off the track", index,
				latitude, off)
		}
		if latitude > 0 {
			north++
		} else {
			south++
		}
	}
	if north != stormsPerHemisphere || south != stormsPerHemisphere {
		t.Errorf("%d north and %d south storms, want %d each", north, south,
			stormsPerHemisphere)
	}
	if again := layer.Weather(); again != weather {
		t.Errorf("the same seed placed storms differently")
	}
	layer.Seed++
	if other := layer.Weather(); other.Storms == weather.Storms {
		t.Errorf("another seed placed the same storms")
	}
}

// wispRatio is the most the grain may change along the wind, as a share of
// across it: the stretch of 2.5 gives about a half, 2026-11-26.
const wispRatio = 0.7

// Wisps: along a latitude line the sky changes more slowly than across
// it, in the tropics where no storm turns it. A cloud is a streak, not a
// blob.
func TestWispsRunAlongLatitude(t *testing.T) {
	const step, around = 0.08, 120
	var along, across float64
	for _, seed := range weatherSeeds {
		layer := cloudsLayer(t)
		layer.Seed = seed
		for index := range around {
			longitude := 2 * math.Pi * float64(index) / around
			here, _ := layer.weathered(onSphere(0.1, longitude))
			east, _ := layer.weathered(onSphere(0.1, longitude+step))
			north, _ := layer.weathered(onSphere(0.1+step, longitude))
			along += math.Abs(east - here)
			across += math.Abs(north - here)
		}
	}
	if along >= across*wispRatio {
		t.Errorf("grain changes %v along the wind and %v across it", along,
			across)
	}
}

// The table the kernel reads has the weather's numbers and each storm's
// centre and sign, in the order the kernel reads them.
func TestWeatherTable(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	table := weather.Table()
	if want := weatherNumbers + stormCount*stormNumbers +
		lockedNumbers; len(table) != want {
		t.Fatalf("table has %d numbers, want %d", len(table), want)
	}
	if table[0] != weather.Equator || table[6] != weather.Stretch ||
		table[10] != weather.Core {
		t.Errorf("table leads with %v", table[:weatherNumbers])
	}
	last := weather.Storms[stormCount-1]
	stormsEnd := len(table) - lockedNumbers
	end := table[stormsEnd-stormNumbers : stormsEnd]
	if end[0] != last.Centre.X || end[2] != last.Centre.Z || end[3] != last.Sign {
		t.Errorf("table ends with %v, want the last storm %v", end, last)
	}
}

// The table's tail past the leading numbers is the storms', and the
// numbers the kernel reads after Core are the filaments', clusters', eye
// and arms', in the order the kernel reads them.
func TestWeatherTableOrder(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	table := weather.Table()
	want := []float64{weather.Filament, weather.Across, weather.Cluster,
		weather.Beads, weather.Eye, weather.EyeClear, weather.Arms,
		weather.ArmCover}
	for index, number := range want {
		if table[11+index] != number {
			t.Errorf("table[%d] is %v, want %v", 11+index, table[11+index],
				number)
		}
	}
	first := weather.Storms[0]
	if table[weatherNumbers] != first.Centre.X {
		t.Errorf("the storms do not follow the %d numbers", weatherNumbers)
	}
}

// A storm's eye is clearer than the cover its heart has without one, and
// its arms are banded: round a ring at one radius the cover rises and
// falls, and far past the arms it is nothing.
func TestWhorlsHaveEyesAndArms(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	storm := weather.Storms[0]
	first := maths.YAxis.Cross(storm.Centre).Unit()
	second := storm.Centre.Cross(first)
	at := func(away, azimuth float64) maths.Vec {
		angle := away * weather.Size
		return storm.Centre.Scale(math.Cos(angle)).Add(
			first.Scale(math.Sin(angle) * math.Cos(azimuth))).Add(
			second.Scale(math.Sin(angle) * math.Sin(azimuth)))
	}
	centre := weather.heart(storm, storm.Centre, 0, 1)
	if centre >= weather.Core {
		t.Errorf("the eye adds %v, no clearer than a plain heart's %v", centre,
			weather.Core)
	}
	low, high := math.Inf(1), math.Inf(-1)
	const ring = 36
	for step := range ring {
		away := 0.8
		got := weather.heart(storm, at(away, 2*math.Pi*float64(step)/ring), away,
			math.Exp(-away*away))
		low, high = math.Min(low, got), math.Max(high, got)
	}
	if high-low < weather.ArmCover {
		t.Errorf("a ring of the whorl varies by %v, arms add %v", high-low,
			weather.ArmCover)
	}
	if far := weather.heart(storm, at(4, 1), 4, math.Exp(-16)); math.Abs(far) > 1e-3 {
		t.Errorf("a whorl adds %v four radii out", far)
	}
}

// A southern whorl is the northern one mirrored: the same cover at the
// mirrored place, so it winds the other way, never the same way.
func TestSouthernWhorlMirrorsNorthern(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	north := weather.Storms[0]
	south := Storm{Centre: maths.V(north.Centre.X, -north.Centre.Y,
		north.Centre.Z), Sign: -north.Sign}
	for _, away := range []float64{0.5, 1, 1.5} {
		beside := north.Centre.Add(maths.YAxis.Cross(north.Centre).Unit().Scale(
			away * weather.Size)).Unit()
		mirrored := maths.V(beside.X, -beside.Y, beside.Z)
		pull := math.Exp(-away * away)
		a := weather.heart(north, beside, away, pull)
		b := weather.heart(south, mirrored, away, pull)
		if math.Abs(a-b) > 1e-9 {
			t.Errorf("at %v radii: north %v, mirrored south %v", away, a, b)
		}
	}
}

// The equator's belt is broken into clusters: round it the cover rises and
// falls, and in places the belt is gone, so it is never one clump.
func TestEquatorIsBeaded(t *testing.T) {
	weather := cloudsLayer(t).Weather()
	var gaps, strung int
	const round = 360
	for step := range round {
		at := onSphere(0, 2*math.Pi*float64(step)/round)
		got := weather.Beaded(0, at)
		if got < weather.Belts(0)-weather.Equator*0.5 {
			gaps++
		}
		if got > weather.Belts(0)-weather.Equator*0.5 {
			strung++
		}
	}
	if gaps == 0 || strung == 0 {
		t.Errorf("the equator has %d gaps and %d clusters in %d places", gaps,
			strung, round)
	}
}

// Weather leaves the old rules alone: a clear sky stays clear however the
// belts and storms lie.
func TestWeatherNeverCloudsAClearSky(t *testing.T) {
	layer := cloudsLayer(t)
	layer.Cover = 0
	for _, latitude := range []float64{0, 0.44, 0.87, -0.87} {
		at := onSphere(latitude, 1).Scale(layer.Radius + layer.Low)
		if got := layer.Density(at); got != 0 {
			t.Errorf("clear sky has density %v at latitude %v", got, latitude)
		}
	}
}
