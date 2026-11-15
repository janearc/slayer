package smoke

import (
	"compress/gzip"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/sky"
)

// The sky behind Webb in smoke fifty: the real stars, where they are
// when Webb looks at its star at its moment.
const (
	// starsFolder names the folder the Bright Star Catalogue is kept in:
	// the file ybsc5.gz, as celestial-data's bsc5 adapter fetches it.
	// Unset, the smoke has no stars and a black sky.
	starsFolder = "SLAYER_STARS"
	starsFile   = "ybsc5.gz"

	// measured: Gaia DR3 4081136684118965760's right ascension and
	// declination, degrees, ICRS at Gaia's epoch 2016. Its proper motion
	// is under an arcsecond in a century and is left out.
	noahRound = 287.50086
	noahUp    = -22.23481

	starZero = 3.0  // picked: the light of a star of magnitude nought in this picture
	starWide = 0.08 // picked: a star's radius, degrees: about two pixels of this lens
)

// webbMoment is when Webb looks: 20:53 UTC on 2027-05-22, the moment the
// star stands at the edge of where Webb may point, as worked from the
// same formula for the sun that SunAmongStars is. A test holds the sun
// to standing 135 degrees from the star then, to half a degree.
var webbMoment = time.Date(2027, time.May, 22, 20, 53, 0, 0, time.UTC)

// among is the unit vector toward a right ascension and a declination,
// in radians: x toward right ascension nought on the equator, z toward
// the north pole.
func among(round, up float64) maths.Vec {
	return maths.V(math.Cos(up)*math.Cos(round), math.Cos(up)*math.Sin(round),
		math.Sin(up))
}

// webbAxes is Webb's own axes among the stars at a moment, and how far
// the sun then stands from its star, in degrees. Its mirror looks along
// its own -z, so -z is the way to the star. Its shield's sun side is
// its own -y, and the sun lies in the plane of the line of sight and
// that side, so -y is the part of the way to the sun that is square to
// the line of sight. x is square to both.
//
// The stars' places are of equinox J2000 and the sun's of the date,
// 27 years on: the equator has turned about 0.4 degree between, which
// is left out, and turns Webb about its line of sight by less than that.
func webbAxes(when time.Time) (x, y, z maths.Vec, fromSun float64) {
	star := among(noahRound*maths.Degree, noahUp*maths.Degree)
	round, up, _ := oceantex.SunAmongStars(when)
	sun := among(round, up)
	fromSun = math.Acos(star.Dot(sun)) / maths.Degree
	square := sun.Sub(star.Scale(star.Dot(sun))).Unit()
	z, y = star.Scale(-unity), square.Scale(-unity)
	return y.Cross(z), y, z, fromSun
}

// webbSky is the real stars behind Webb, turned into Webb's own frame,
// and whether there are any: none when SLAYER_STARS is unset. When it is
// set, a catalogue that cannot be opened, unpacked or read stops the
// picture, with its path and the reason.
func webbSky() (sky.Sky, bool) {
	dir := os.Getenv(starsFolder)
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, starsFile)
	packed, err := os.Open(path)
	if err != nil {
		placed(fmt.Errorf("smoke: %s is set, and %s cannot be opened: %w",
			starsFolder, path, err))
	}
	defer packed.Close()
	lines, err := gzip.NewReader(packed)
	if err != nil {
		placed(fmt.Errorf("smoke: %s is set, and %s is not gzip: %w",
			starsFolder, path, err))
	}
	stars, err := sky.BrightStars(lines, maths.Grey(starZero))
	if err != nil {
		placed(fmt.Errorf("smoke: %s is set, and %s: %w", starsFolder, path, err))
	}
	x, y, z, _ := webbAxes(webbMoment)
	for at, each := range stars {
		stars[at].Toward = maths.V(each.Toward.Dot(x), each.Toward.Dot(y),
			each.Toward.Dot(z))
	}
	return sure(sky.NewCatalogue(stars, starWide*maths.Degree, maths.Light{})), true
}
