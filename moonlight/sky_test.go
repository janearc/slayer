package moonlight

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// the moonlit sky is brighter toward the moon than away from it, bluer
// overhead than its moonlight is, and the same either side of the moon.
func TestTheMoonlitSkyIsTheAirsLight(t *testing.T) {
	moon := maths.V(math.Sin(1.8), math.Sin(0.1), -math.Cos(1.8)).Unit()
	began := time.Now()
	sky, err := NewSky(measure.Metres(1222), moon, Full(), 1e5)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("the table took %v", time.Since(began))
	near := sky.Light(maths.V(moon.X, 0.15, moon.Z))
	away := sky.Light(maths.V(-moon.X, 0.15, -moon.Z))
	if !(near.Luminance() > away.Luminance()) {
		t.Errorf("toward the moon %v, away %v", near, away)
	}
	overhead := sky.Light(maths.V(0, 1, 0))
	lit := Full().Picture(1e5)
	if !(overhead.B/overhead.R > lit.B/lit.R) {
		t.Errorf("overhead %v is no bluer than moonlight %v", overhead, lit)
	}
	left := sky.Light(maths.V(moon.Z, 0.3, -moon.X))
	right := sky.Light(maths.V(-moon.Z, 0.3, moon.X))
	if math.Abs(left.Luminance()-right.Luminance()) >
		1e-9*left.Luminance() {
		t.Errorf("either side of the moon: %v and %v", left, right)
	}
}

// air scatters the moon's light as it scatters the sun's, so a sky lit
// by the full moon is the sky lit by the sun from the same way, times
// the moon's geometric albedo and its radius over its distance,
// squared; and two skies seen at once are the two added.
func TestASkyIsItsLightsOwn(t *testing.T) {
	way := maths.V(math.Sin(1.9), math.Sin(0.02), -math.Cos(1.9)).Unit()
	height := measure.Metres(1100)
	byMoon, err := NewSky(height, way, Full(), 2500)
	if err != nil {
		t.Fatal(err)
	}
	bySun, err := NewSky(height, way, Sunlight(), 2500)
	if err != nil {
		t.Fatal(err)
	}
	moon := Moon()
	size := moon.Radius().Over(moon.Distance())
	share := moon.Albedo() * size * size
	both := Skies{byMoon, bySun}
	for _, look := range []maths.Vec{maths.V(0, 1, 0),
		maths.V(way.X, 0.1, way.Z), maths.V(-way.X, 0.1, -way.Z)} {
		moonlit, sunlit := byMoon.Light(look), bySun.Light(look)
		want := sunlit.Scale(share)
		for _, pair := range [][2]float64{{moonlit.R, want.R},
			{moonlit.G, want.G}, {moonlit.B, want.B}} {
			if !(math.Abs(pair[0]-pair[1]) <= 1e-9*pair[1]) {
				t.Errorf("toward %v: moonlit %v, the sun's times %g %v",
					look, moonlit, share, want)
			}
		}
		if added := moonlit.Add(sunlit); both.Light(look) != added {
			t.Errorf("toward %v: both %v, added %v", look,
				both.Light(look), added)
		}
	}
}

// sunAt is the way to the sun at an azimuth and an altitude, in degrees.
func sunAt(azimuth, altitude float64) maths.Vec {
	round, up := azimuth*math.Pi/180, altitude*math.Pi/180
	return maths.V(math.Sin(round)*math.Cos(up), math.Sin(up),
		-math.Cos(round)*math.Cos(up))
}

// with the sun four degrees below the horizon, the earth's shadow
// stands overhead at the earth's radius over the cosine of four
// degrees, less the radius: about 15.6 km. the air below it is dark.
// with the sun four degrees up, all the column is lit, at the same
// angle to the sun. so dusk overhead is at most the share of the
// column above the shadow, by its pressure, of the day's: sunlight
// reaching the air above the shadow comes down through more air than
// by day, never less. the eye's own way up takes up to a quarter more
// of the low air's light than the high's, and the bound allows that.
// a sun below the horizon taken as one above gives about one.
func TestTwilightIsTheAirAboveTheEarthsShadow(t *testing.T) {
	height := measure.Metres(1100)
	dusk, err := NewSky(height, sunAt(289.65, -4), Sunlight(), 1)
	if err != nil {
		t.Fatal(err)
	}
	day, err := NewSky(height, sunAt(289.65, 4), Sunlight(), 1)
	if err != nil {
		t.Fatal(err)
	}
	const earth = 6371000.0
	shadow := earth/math.Cos(4*math.Pi/180) - earth
	above, err := atmosphere.Standard(measure.Metres(shadow))
	if err != nil {
		t.Fatal(err)
	}
	here, err := atmosphere.Standard(height)
	if err != nil {
		t.Fatal(err)
	}
	share := above.Pressure().InPascals() / here.Pressure().InPascals()
	up := maths.V(0, 1, 0)
	ratio := dusk.Light(up).Luminance() / day.Light(up).Luminance()
	t.Logf("shadow %.0f m, share above it %.3f, dusk over day %.3f",
		shadow, share, ratio)
	if !(ratio > 0 && ratio <= 1.25*share) {
		t.Errorf("dusk over day overhead %g, the air above the shadow %g",
			ratio, share)
	}
}
