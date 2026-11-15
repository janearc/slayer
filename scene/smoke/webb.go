package smoke

import (
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Webb and the Larson. Lengths are metres, in Webb's own frame: its
// mirror looks along -z, and its sunshield lies under the mirror with
// the sun's side toward -y.
const (
	fiftyNo = 50 // defined: the number of Webb lit by the Larson

	madeSeed   = 5       // picked: the seed the catalogue makes its specimens from
	madeDetail = "quick" // picked: the manufactory's lightest whole build

	// published: Webb may point 85 to 135 degrees from the sun, a limit
	// of its design and not a measurement, and its sky is darkest at 135
	// (STScI, JWST user documentation, "JWST Field of Regard"; Leinert
	// et al. 1998, table 16).
	//
	// The star it looks at here, Gaia DR3
	// 4081136684118965760, stands at that edge at 20:53 UTC on
	// 2027-05-22: worked by hand from the almanac's low-precision sun,
	// and held since by a test in webb_sky_test.go, which finds the sun
	// 135 degrees from the star at that minute, to half a degree.
	sunFromSight = 135.0

	// measured: sunlight at the earth's distance, in watts a square
	// metre; the figure moonlight.go works from. L2 is a hundredth
	// further from the sun, which is not drawn.
	sunlight = 1361.0

	// picked: what a watt a steradian is in this picture's light, set by
	// eye so the Larson's lamps show; the landed Larson's night picture
	// wanted a number of the same size. It makes the sun millions of
	// times too bright for the film: what the sun touches burns out,
	// which is why the Larson keeps to the shade.
	lampExposure = 15000.0

	// The Larson hovers in Webb's shade. The shade runs up and forward
	// from the shield, away from the sun, so a place as far in front of
	// the shield's middle as it is above it is in the middle of the shade.
	larsonX   = 0.0   // picked: over the shield's middle line,
	larsonY   = 12.0  // picked: above the mirror,
	larsonZ   = -12.0 // picked: and as far in front as it is above: in the shade
	mirrorY   = 6.0   // picked: about the middle of the primary in the model, where the Larson's belly lamp is aimed,
	mirrorZ   = -3.3  // picked: as above
	webbEyeX  = -10.0 // picked: the eye stands a little to one side,
	webbEyeY  = 4.0   // picked: low, under the line from the Larson's rim to the mirror,
	webbEyeZ  = -30.0 // picked: and well out past the shield's nose
	webbLookX = 0.0   // picked: it looks between the mirror and the Larson
	webbLookY = 8.0   // picked: as above
	webbLookZ = -6.0  // picked: as above
	webbWide  = 45.0  // picked: the lens's width, in degrees
	webbTop   = 8.0   // picked: the brightest light the film holds before it is white
	webbKnee  = 0.7   // picked: where the film's shoulder starts
)

// atRest is the time a still picture is taken at.
const atRest = time.Duration(nought)

// stood makes one of the manufactory's kinds and stands it in a world,
// with whatever lights it carries lit.
func stood(w *thing.World, name string, at maths.Subspace) {
	row := sure(manufacture.KindNamed(name))
	made := sure(manufacture.MakeWithComplexity(row,
		manufacture.Sow(madeSeed), madeDetail))
	for _, part := range sure(made.Things(at)) {
		w.Add(part)
	}
	lights := sure(manufacture.PlaceLights(made.Descriptions(), at))
	for _, each := range lights {
		lamp := sure(emit.Of(each.Light(), each.At(), each.Toward(),
			lampExposure))
		lamp.Beyond = each.Beyond().InMetres()
		w.Lamp(lamp.When(atRest))
	}
}

// bellyToward is the turn that points a thing's underside from where it
// stands at a place: its own up, +y, is turned to point away from the
// place, about the line square to both. The place is never straight
// above or below here, so there is such a line.
func bellyToward(from, place maths.Vec) maths.Orientation {
	up := from.Sub(place).Unit()
	return maths.About(maths.YAxis.Cross(up).Unit(), math.Acos(up.Y))
}

// fifty is the fiftieth milestone: the James Webb Space Telescope where
// it would be, and the MCRN Larson come to look at it.
//
// Webb is in its own frame with the sun at the edge of where it may
// point: 135 degrees from the mirror's line of sight, on the shield's
// side. That is how it stands at 20:53 UTC on 2027-05-22 with its
// mirror on Gaia DR3 4081136684118965760. The sun lights the shield's
// far side and nothing on this one: the telescope is in its own shade.
//
// The Larson hovers over the mirror in Webb's shade, tipped so its belly
// lamp is on the gold, with its cabin lamp and its twelve portholes
// lit. What shows of Webb's cold side is by the Larson's light: its
// struts, its secondary, the tower it stands on and the instrument
// module behind the mirror.
//
// The mirror, which is gold from
// gold's optical constants, shows black from where the eye stands: it
// is a mirror, and what it mirrors there is empty space.
//
// Behind them are the real stars, when SLAYER_STARS names the folder
// the Bright Star Catalogue is in: its nine thousand, each where it is
// with Webb turned to its star at its moment, all white. Webb's own
// star, at magnitude 13, is far too faint to be among them. Unset, the
// sky is black.
//
// Not drawn: where Webb is on its orbit around L2; and the two
// hundredths by which sunlight is weaker there than at the earth, L2
// being a hundredth further out.
//
// Not right: the Larson is not wholly in Webb's shade, and cannot be.
// It is 12 m across, and the widest circle the kite holds, 14.162 by
// 21.197 m, is 11.78. Tipped 55 degrees to put its belly lamp on the
// gold, its rim is 0.93 m out of the shade and burns in the sun, and
// three of the mirror's segments mirror it; level it would be 0.11 m
// out.
//
// Worked for a flat kite with the Larson over its middle and the
// shade leaning 45 degrees. The lamps' exposure was found by eye, and why a
// lamp of 1500 watts wants so large a one is not understood; and the
// smoke check calls the picture not honest, worst 0.19.
func fifty() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	stood(w, "webb", maths.Local())
	larson := maths.V(larsonX, larsonY, larsonZ)
	mirror := maths.V(nought, mirrorY, mirrorZ)
	stood(w, "mcrn-larson", maths.NewSubspace(larson,
		bellyToward(larson, mirror), unity))
	edge := sunFromSight * maths.Degree
	sun := maths.V(nought, -math.Sin(edge), -math.Cos(edge))
	w.Lamp(sure(emit.NewDistant(sun, maths.Grey(sunlight*lampExposure))))
	if stars, some := webbSky(); some {
		w.Sky = stars
	}
	eye := maths.V(webbEyeX, webbEyeY, webbEyeZ)
	return w, maths.Subspace{Origin: eye,
		Orientation: maths.LookAt(eye, maths.V(webbLookX, webbLookY, webbLookZ))}
}
