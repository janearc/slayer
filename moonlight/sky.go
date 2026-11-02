package moonlight

import (
	"math"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// the sky's table: how many heights above the horizon, how many ways
// round from its light's, and every how many nanometres its light is
// summed, picked for time. the sky is smooth, so a coarse table read
// between its points is close; finer is slower, never truer in kind.
const (
	skyRises   = 24
	skyRounds  = 13
	skyEveryNm = 16
	firstNm    = 384
	lastNm     = 768
)

// A Sky is the sky lit by one light far off, the full moon or the sun:
// atmosphere's single scattering of that light, worked out once for a
// table of directions and read between them. below the horizon it is
// the horizon's.
//
// the light may stand below the horizon itself. then the sky is
// twilight, since atmosphere's sky leaves dark the air the earth hides
// from the light.
type Sky struct {
	source maths.Vec
	table  [skyRises][skyRounds]maths.Light
}

// NewSky is the sky seen from a height, lit from a way, unit long, y
// up, by a light as it arrives above the air, at an exposure: what a
// watt a square metre a steradian is in the picture.
func NewSky(height measure.Length, source maths.Vec, light Spectrum,
	exposure float64) (*Sky, error) {
	air, err := atmosphere.DryAir(co2)
	if err != nil {
		return nil, err
	}
	source = source.Unit()
	toward, err := atmosphere.NewWay(source.X, source.Y, source.Z)
	if err != nil {
		return nil, err
	}
	var waves []measure.Length
	for nm := firstNm; nm <= lastNm; nm += skyEveryNm {
		waves = append(waves, measure.Metres(float64(nm)*nanometre))
	}
	sky := &Sky{source: source}
	for rise := range skyRises {
		for round := range skyRounds {
			look := sky.look(rise, round)
			view, err := atmosphere.NewWay(look.X, look.Y, look.Z)
			if err != nil {
				return nil, err
			}
			shares, err := air.Skylight(height, *view, *toward,
				waves)
			if err != nil {
				return nil, err
			}
			seen := Spectrum{}
			for i, share := range shares {
				nm := firstNm + i*skyEveryNm
				seen[nm] = share * light[nm] * skyEveryNm
			}
			sky.table[rise][round] = seen.Picture(exposure)
		}
	}
	return sky, nil
}

// altitudeOf is the height above the horizon of a row of the table:
// closer together low down, where the sky changes most.
func altitudeOf(rise int) float64 {
	share := float64(rise) / (skyRises - 1)
	return share * share * math.Pi / 2
}

// roundOf is the way round from its light's of a column of the table, from zero
// to half a turn: the sky is the same either side of it.
func roundOf(round int) float64 {
	return float64(round) / (skyRounds - 1) * math.Pi
}

// look is the way a point of the table looks: so high, so far round
// from its light's way across the ground.
func (sky *Sky) look(rise, round int) maths.Vec {
	up, about := altitudeOf(rise), roundOf(round)
	ground := math.Atan2(sky.source.X, -sky.source.Z) + about
	return maths.V(math.Sin(ground)*math.Cos(up), math.Sin(up),
		-math.Cos(ground)*math.Cos(up))
}

// Light is the sky's light along a way: the table's, read between its
// four nearest points.
func (sky *Sky) Light(toward maths.Vec) maths.Light {
	toward = toward.Unit()
	up := math.Asin(math.Max(0, math.Min(1, toward.Y)))
	about := math.Abs(math.Remainder(math.Atan2(toward.X, -toward.Z)-
		math.Atan2(sky.source.X, -sky.source.Z), 2*math.Pi))
	row := math.Sqrt(up/(math.Pi/2)) * (skyRises - 1)
	column := about / math.Pi * (skyRounds - 1)
	low, left := int(math.Min(row, skyRises-2)), int(math.Min(column,
		skyRounds-2))
	across, down := column-float64(left), row-float64(low)
	blend := func(a, b maths.Light, share float64) maths.Light {
		return a.Scale(1 - share).Add(b.Scale(share))
	}
	near := blend(sky.table[low][left], sky.table[low][left+1], across)
	far := blend(sky.table[low+1][left], sky.table[low+1][left+1], across)
	return blend(near, far, down)
}

// Skies is the sky lit by several lights at once, the moon and the sun
// below the horizon: air scatters each light on its own, so the sky's
// light along a way is all of theirs added.
type Skies []*Sky

// Light is the skies' light along a way: each one's, added.
func (skies Skies) Light(toward maths.Vec) maths.Light {
	all := maths.Light{}
	for _, sky := range skies {
		all = all.Add(sky.Light(toward))
	}
	return all
}
