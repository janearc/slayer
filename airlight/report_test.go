package airlight

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/airs"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/sources"
)

// What the reports are worked at and set beside.
const (
	// luminousEfficacy is lumens a watt at 555 nm, by the candela's
	// definition: lux are it times the irradiance weighted by V.
	luminousEfficacy = 683 // defined: see above
	// setZenith is the sun's zenith in the Venera 13 set, whose sun
	// stands 55 degrees up, in degrees.
	setZenith = 35 // defined: ninety less the set's fifty-five
	// veneraNine is the Venera 9 site's sun zenith, about, in degrees,
	// by Tomasko's report as the design names it.
	veneraNine = 33 // defined: as measured, read in the brief
	// veneraNineLux is the light measured at the Venera 9 site.
	veneraNineLux = 14000 // defined: as measured, read in the design
	// groundShare is the share of the sunlight on Venus its ground
	// absorbs, as measured, over the whole solar spectrum.
	groundShare = 0.025 // defined: as measured, read in the brief
	// countsWanted is the mean count a downward bin should hold.
	countsWanted = 100 // defined: the brief's
	// percent is a share in hundredths.
	percent = 100 // defined: a hundred
)

// reportedZeniths are the sun's zeniths the reports are made at, in
// degrees: the brief's three and Venera 9's.
var reportedZeniths = []float64{
	30, veneraNine, 60, 66, // defined: the brief's, and Venera 9's
}

// luminous is a spectrum's power weighted by the CIE 1931 observer's
// V, its y row, as libtheme's swatch.Monochrome gives it.
func luminous(light moonlight.Spectrum) float64 {
	var sum float64
	for nanometres, power := range light {
		_, eye, _ := swatch.Monochrome(nanometres).XYZ()
		sum += power * eye
	}
	return sum
}

// chromaticity is a spectrum's place on the horseshoe, through the same
// route as moonlight's Picture.
func chromaticity(light moonlight.Spectrum) swatch.XY {
	lines := make(map[float64]float64, len(light))
	for nanometres, power := range light {
		lines[float64(nanometres)] = power
	}
	return observer.Swatch(bands.Lines(lines)).XY()
}

// lookSpectrum is the radiance along a way with the sun at a zenith and
// round from north at nought, by wavelength, as the sky reads it.
func lookSpectrum(fields *Fields, zenith float64,
	look maths.Vec) moonlight.Spectrum {
	view := axisOf((look.Y+one)/two, fields.views)
	round := axisOf(folded(roundOf(look))/math.Pi, fields.azimuths)
	return fields.seen(func(wave int) float64 {
		slice := fields.slice(wave, zenith)
		at := func(row, column int) float64 {
			return slice[row*fields.azimuths+column]
		}
		near := at(view.low, round.low)*(one-round.share) +
			at(view.low, round.high)*round.share
		far := at(view.high, round.low)*(one-round.share) +
			at(view.high, round.high)*round.share
		return near*(one-view.share) + far*view.share
	})
}

// reportAt logs the light at the ground with the sun at a zenith, in
// degrees: what lands on it, in watts and lux; the share of the sun at
// the top that reaches it and is absorbed by it; and the sky's colour.
func reportAt(t *testing.T, plan Plan, fields *Fields, degrees float64) {
	zenith := degrees * degree
	landing := fields.seen(func(wave int) float64 {
		return fields.landing(wave, zenith)
	})
	absorbed := fields.seen(func(wave int) float64 {
		return (one - plan.columns[wave].GroundAlbedo) *
			fields.landing(wave, zenith)
	})
	top := fields.seen(func(int) float64 { return math.Cos(zenith) })
	lux := luminousEfficacy * luminous(landing)
	t.Logf("sun %g degrees from the zenith: %.4g W m^-2 from %d to %d nm "+
		"lands on the ground, %.4g lux; %.3g%% of the sun at the top "+
		"reaches it and %.3g%% is absorbed there", degrees, landing.Total(),
		firstNanometre, lastNanometre, lux,
		percent*landing.Total()/top.Total(),
		percent*absorbed.Total()/top.Total())
	if degrees == veneraNine {
		t.Logf("  that is %.3g times the %d lux measured at Venera 9",
			lux/veneraNineLux, veneraNineLux)
	}
	reportSky(t, fields, zenith)
}

// reportSky logs the sky's colour at the zenith and just above the
// horizon toward and away from the sun, at exposure one and as
// chromaticity.
func reportSky(t *testing.T, fields *Fields, zenith float64) {
	low := math.Acos(centre(fields.views/two, fields.views)*two - one)
	for _, look := range []struct {
		name string
		way  maths.Vec
	}{{"zenith", straightUp},
		{"lowest bin above the horizon, toward the sun", wayAt(low, none)},
		{"lowest bin above the horizon, away", wayAt(low, math.Pi)}} {
		light := lookSpectrum(fields, zenith, look.way)
		place := chromaticity(light)
		t.Logf("  the sky at the %s: %+v at exposure one, x %.4f y %.4f",
			look.name, light.Picture(one), place.X, place.Y)
	}
}

// reportCounts logs the mean count of the downward bins with the sun at
// a zenith, in degrees, read between the zenith bins as the sky reads
// them: the least and the most over the wavelengths.
func reportCounts(t *testing.T, fields *Fields, degrees float64) {
	at := axisOf(degrees*degree/math.Pi, fields.zeniths)
	least, most := math.Inf(one), math.Inf(-one)
	for wave := range fields.wavelengths {
		var sum, bins float64
		for view := fields.views / two; view < fields.views; view++ {
			for round := range fields.azimuths {
				low := fields.counts[wave][fields.index(at.low, view, round)]
				high := fields.counts[wave][fields.index(at.high, view,
					round)]
				sum += low*(one-at.share) + high*at.share
				bins++
			}
		}
		least, most = min(least, sum/bins), max(most, sum/bins)
	}
	t.Logf("downward bins with the sun %g degrees from the zenith: a mean "+
		"of %.4g to %.4g counts over the wavelengths, against %d wanted",
		degrees, least, most, countsWanted)
}

// shelfEnding is the ending of the address the sun's spectrum came
// from, which the shelf keeps on its file's name after the key.
const shelfEnding = ".csv" // defined: see above

// shelfSun is the sun's spectrum read from the shelf SLAYER_SHELF names,
// or the test skipped where none is named, as the tree's other tests
// that read the shelf are.
func shelfSun(t *testing.T) airs.Sun {
	t.Helper()
	shelf := os.Getenv(sources.Placed)
	if shelf == "" {
		t.Skip("no shelf named in SLAYER_SHELF")
	}
	file, err := os.Open(filepath.Join(shelf, airs.SunSource+shelfEnding))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sun, err := airs.ReadSun(file)
	if err != nil {
		t.Fatal(err)
	}
	return sun
}

// solvedAt is the fields of a named air under the shelf's sun, from a
// ground so many metres up, and says what they were worked out from and
// how long they took.
func solvedAt(t *testing.T, name string, ground float64) (Plan, *Fields) {
	t.Helper()
	known, err := Named(name, shelfSun(t))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := NewPlan(known, ground)
	if err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	fields, err := plan.Solve()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s, from %g m, worked out in %v", plan.Said(), ground,
		time.Since(began).Round(time.Second))
	return plan, fields
}

// Venus's air at its mean radius, reported: what lands on the ground at
// four sun zeniths beside what was measured, the sky's colour, the noise
// behind the bins, and the ground's share of the sunlight on the planet.
func TestVenusAtTheGroundIsReported(t *testing.T) {
	if testing.Short() {
		t.Skip("the Venus walk takes minutes")
	}
	plan, fields := solvedAt(t, venusName, none)
	for _, degrees := range reportedZeniths {
		reportAt(t, plan, fields, degrees)
	}
	reportCounts(t, fields, setZenith)
	planet := fields.seen(func(wave int) float64 {
		return fields.absorbed[wave]
	}).Total() / fields.seen(func(int) float64 { return one }).Total()
	t.Logf("the ground absorbs %.3g%% of the sunlight on the whole planet "+
		"from %d to %d nm; measured, about %.3g%% over the whole spectrum",
		percent*planet, firstNanometre, lastNanometre, percent*groundShare)
}

// What Earth's report is worked at.
const (
	// earthCounted is the sun's zenith the counts are reported at, in
	// degrees: the brief's thirty.
	earthCounted = 30 // defined: the brief's
	// lowestRows is how many view bins up from the horizon the report
	// gives the light of, to show how fast it changes there.
	lowestRows = 4 // picked: the lowest few
)

// earthZeniths are the sun's zeniths Earth's report is made at, in
// degrees: the brief's two.
var earthZeniths = []float64{30, 60} // defined: the brief's

// rowSpectrum is the light by wavelength in one view bin, with the sun
// at a zenith, in radians, the mean over every azimuth round from it.
func rowSpectrum(fields *Fields, zenith float64,
	row int) moonlight.Spectrum {
	return fields.seen(func(wave int) float64 {
		slice := fields.slice(wave, zenith)
		var sum float64
		for round := range fields.azimuths {
			sum += slice[row*fields.azimuths+round]
		}
		return sum / float64(fields.azimuths)
	})
}

// reportLanding logs the light on level ground with the sun at a zenith,
// in degrees, from the sun and from the sky, and the sky's share.
func reportLanding(t *testing.T, fields *Fields, degrees float64) {
	zenith := degrees * degree
	direct := fields.seen(func(wave int) float64 {
		return fields.direct(wave, zenith)
	}).Total()
	diffuse := fields.seen(func(wave int) float64 {
		return fields.diffuse(wave, zenith)
	}).Total()
	t.Logf("sun %g degrees from the zenith: on level ground, %d to %d nm, "+
		"the sun gives %.4g W m^-2 and the sky %.4g W m^-2; the sky's "+
		"share is %.3g%%", degrees, firstNanometre, lastNanometre, direct,
		diffuse, percent*diffuse/(direct+diffuse))
}

// reportZenithAndHorizon logs the sky's colour and brightness at the
// zenith and at the horizon, the lowest bin above it, the mean round
// every azimuth; then the light of the lowest few bins.
func reportZenithAndHorizon(t *testing.T, fields *Fields, degrees float64) {
	zenith := degrees * degree
	middle := fields.views / two
	top := lookSpectrum(fields, zenith, straightUp)
	low := rowSpectrum(fields, zenith, middle)
	upper, lower := chromaticity(top), chromaticity(low)
	t.Logf("  zenith x %.4f y %.4f; horizon x %.4f y %.4f; the horizon's "+
		"luminance is %.3g times the zenith's", upper.X, upper.Y, lower.X,
		lower.Y, luminous(low)/luminous(top))
	for row := middle; row < middle+lowestRows; row++ {
		cosine := centre(row, fields.views)*two - one
		t.Logf("  view bin %d, %.2f degrees up: luminance %.4g, the mean "+
			"round the azimuths, at a unit of sun", row,
			math.Asin(cosine)/degree, luminous(rowSpectrum(fields, zenith,
				row)))
	}
}

// Earth's air at sea level, reported: at two sun zeniths, the light on
// level ground from the sun and from the sky, and the sky's colour and
// brightness at the zenith and the horizon; and the counts behind the
// bins with the sun 30 degrees from the zenith.
func TestEarthAtSeaLevelIsReported(t *testing.T) {
	if testing.Short() {
		t.Skip("the Earth walk takes minutes")
	}
	_, fields := solvedAt(t, earthName, none)
	for _, degrees := range earthZeniths {
		reportLanding(t, fields, degrees)
		reportZenithAndHorizon(t, fields, degrees)
		reportSky(t, fields, degrees*degree)
	}
	reportCounts(t, fields, earthCounted)
}
