package rig

import "math"

// FigureWays is the ways a head's beams go in a figure, about a heading
// and a tilt in degrees, a spread wide, and how bright each beam is for
// it, and whether a split pulse halves it:
//
//   - beam: one, all the head's light in it
//   - pair: two, crossing the lake from the two heads
//   - fan: the cue's beams across the spread; fan64, sixty-four, eight
//     across by eight up and down, a pyramid of beams
//   - scan: one beam swept across the spread faster than a frame, drawn
//     as many faint ones, a sheet of light
//   - cone: one beam swept round a circle, a hollow cone, turning
//   - wave: sixteen across the spread, their tilts a wave travelling
//     along the fan with the figure's bars
//
// a figure it does not know is the fan, as lightshow has always had it.
func FigureWays(figure string, heading, tilt, spread float64, beams int,
	phase float64) ([]Point, float64, bool) {
	across := func(n int, wide float64, lift func(f float64) float64) []Point {
		ways := make([]Point, n)
		for b := range ways {
			f := 0.0
			if n > 1 {
				f = float64(b)/float64(n-1) - half
			}
			ways[b] = Toward(heading+f*wide, tilt+lift(f))
		}
		return ways
	}
	level := func(float64) float64 { return 0 }
	switch figure {
	case "beam":
		return across(1, 0, level), beamWeight, false
	case "pair":
		return []Point{Toward(heading-spread/4, tilt),
			Toward(heading+spread/4, tilt)}, pairWeight, false
	case "fan64":
		ways := make([]Point, 0, pyramidSide*pyramidSide)
		for row := range pyramidSide {
			up := pyramidRise * (float64(row)/(pyramidSide-1) - half)
			ways = append(ways, across(pyramidSide, spread*pyramidWide,
				func(float64) float64 { return up })...)
		}
		return ways, pyramidWeight, true
	case "scan":
		return across(scanSweeps, spread*scanWide, level),
			sheetLight / scanSweeps, false
	case "cone":
		return coneWays(heading, tilt, phase), sheetLight / coneRound, false
	case "wave":
		return across(waveBeams, spread, func(f float64) float64 {
			return waveRise * math.Sin(2*math.Pi*(waveRun*f+phase))
		}), waveWeight, false
	}
	return across(beams, spread, level), 1, true
}

// coneWays is one beam swept round a hollow cone about a heading and a
// tilt, turned by a phase.
func coneWays(heading, tilt, phase float64) []Point {
	axis := Toward(heading, tilt)
	u := axis.Cross(Point{0, 0, 1}).Unit()
	v := u.Cross(axis)
	open := coneHalf * math.Pi / halfTurn
	ways := make([]Point, coneRound)
	for k := range ways {
		a := 2*math.Pi*float64(k)/coneRound + 2*math.Pi*phase
		ring := u.Scale(math.Cos(a)).Add(v.Scale(math.Sin(a)))
		ways[k] = axis.Scale(math.Cos(open)).Add(ring.Scale(math.Sin(open)))
	}
	return ways
}

// Sweep is the mirrors' swing, tilt and spread in degrees: on the
// clock's seconds, or on a cued sweep's share of its bars when one is
// given (nought or more).
func Sweep(seconds, sweep float64) (swing, tilt, spread float64) {
	swing = swingMost * math.Sin(seconds*swingRate)
	tilt = tiltMiddle + tiltNod*math.Sin(seconds*tiltRate+tiltLead)
	spread = spreadMiddle + spreadOpen*math.Sin(seconds*spreadRate)
	if sweep >= 0 {
		turn := 2 * math.Pi * sweep
		swing = swingMost * math.Sin(turn)
		tilt = tiltMiddle + tiltNod*math.Sin(2*turn+tiltLead)
		spread = spreadMiddle + spreadOpen*math.Sin(3*turn)
	}
	return swing, tilt, spread
}
