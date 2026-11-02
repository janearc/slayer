// Package lens models a thin camera lens with focal length, aperture, focus
// distance, and film dimensions. It implements film.Projection to supply rays
// to a tracer.
//
// All refraction occurs at one plane. Points at the focus distance converge on
// the film; nearer and farther points blur according to aperture size. The lens
// treats all wavelengths alike. Chromatic aberration would require a
// wavelength-dependent glass model and is not implemented.
package lens

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/number"
)

// millimetre is a millimetre in metres. It is defined as a thousandth.
const millimetre = 1e-3

// widest is the lowest f-number any lens has. No lens opens wider than
// f over a half, for the sine of an angle cannot pass one. It is
// defined by that.
const widest = 0.5

// half is half of a width or of a share. It is defined as one over two.
const half = 0.5

// pair is a count of two. It is defined as two.
const pair = 2

// fourth is the power of the cosine that light off the axis falls by.
// It is defined by the law for a small aperture.
const fourth = 4

// none is no amount at all. It is defined as zero.
const none = 0

// whole is the whole of a share. It is defined as one.
const whole = 1

// eighth is an eighth of a turn, the angle of the diagonal of the
// concentric map. It is defined as pi over four.
const eighth = math.Pi / 4

// quarter is a quarter of a turn. It is defined as pi over two.
const quarter = math.Pi / 2

// plastic is the plastic number, the real root of x cubed equals x plus
// one. Roberts' r2 sequence is made from it. It spreads points evenly
// over a square, each new one landing in the largest gap left. It is
// defined by that equation.
const plastic = 1.32471795724474602596

var (
	errFrame = errors.New("ninjafilm: a frame is a finite width and " +
		"height above nought, in millimetres")
	errLens = errors.New("ninjafilm: a lens has a finite focal length " +
		"above nought, an f-number of a half or more, and a focus " +
		"further off than its focal length")
)

// Frame is the piece of film one picture is made on.
//
//ninja:vow practice clients name the type between FrameOf and NewThin
type Frame struct {
	width, height float64 // metres
}

// FrameOf is a frame so many millimetres wide and high.
//
//ninja:vow practice slayer's camera sizes its film here; tests only so far
func FrameOf(width, height float64) (made Frame, err error) {
	if !number.Positive(width) || !number.Positive(height) {
		return made, errFrame
	}
	return Frame{width * millimetre, height * millimetre}, nil
}

// Rows is how many rows of pixels a picture so many pixels across the
// frame has, for its pixels to be square.
//
//ninja:vow practice slayer's camera sizes pictures here; tests only so far
func (made Frame) Rows(across int) int {
	rows := float64(across) * made.height / made.width
	return int(math.Round(rows))
}

// Thin is a thin lens in front of a frame.
//
//ninja:vow practice slayer's camera holds one; tests only so far
type Thin struct {
	frame Frame
	// focal and focus are in metres. stop is the f-number. image is the
	// distance behind the middle of the lens where the film stands for
	// the focus to be sharp on it.
	focal, stop, focus, image float64
}

// NewThin is a lens in front of a frame. The focal length is in
// millimetres. The f-number is the stop. The focus is how many metres
// off the lens is focused.
//
// The film stands where the lens law puts the image of the focus. One
// over the focal length is one over the focus plus one over the
// distance of the image.
//
//ninja:vow practice slayer's camera makes its lens here; tests only so far
func NewThin(frame Frame, focal, stop, focus float64) (made Thin,
	err error) {
	focal *= millimetre
	opened := stop >= widest && !math.IsInf(stop, none)
	if !number.Positive(frame.width) || !number.Positive(focal) ||
		!opened || !(focus > focal) || math.IsInf(focus, none) {
		return made, errLens
	}
	image := focal * focus / (focus - focal)
	return Thin{frame, focal, stop, focus, image}, nil
}

// Wide is how much the lens sees across the frame, in radians: twice
// the angle whose tangent is half the frame's width over the image's
// distance.
//
//ninja:vow practice slayer's camera asks its view here; tests only so far
func (glass Thin) Wide() float64 {
	return pair * math.Atan(glass.frame.width/pair/glass.image)
}

// Pixel is the angle one pixel of a picture so many pixels across the
// frame sees: its width on the film over the image's distance, as the
// lens's middle sees it.
//
//ninja:vow practice slayer's camera asks a pixel's view; tests only so far
func (glass Thin) Pixel(across int) float64 {
	return glass.frame.width / float64(across) / glass.image
}

// aperture is how far the open aperture reaches from the lens's middle,
// in metres: half the focal length over the f-number.
func (glass Thin) aperture() float64 {
	return glass.focal / (pair * glass.stop)
}

// acrossFilm is how far a pixel's middle lies to the right of the
// middle of the film, as a share of the film's width.
func acrossFilm(x, width int) float64 {
	return (float64(x)+half)/float64(width) - half
}

// upFilm is how far a pixel's middle lies above the middle of the film, as a
// share of the film's height. Row zero is the top.
func upFilm(y, height int) float64 {
	return half - (float64(y)+half)/float64(height)
}

// Ray is a ray through pixel (x, y) of a picture width by height pixels.
// It is in the space of the camera, which looks down minus z with x to
// the right and y up. Row 0 is the top of the picture.
//
// The ray starts at a point on the open aperture. It goes toward the
// point at the focus distance that the middle of the pixel sees sharp.
// That point is through the middle of the lens on the far side of it
// from the film, where the picture is upside down.
//
// Which point of the aperture is a fixed sequence of the pixel. So the
// same pixel always takes the same ray. Pixels beside each other take
// points spread across the opening.
func (glass Thin) Ray(x, y, width, height int, _ maths.Band) maths.Ray {
	across := acrossFilm(x, width) * glass.frame.width
	up := upFilm(y, height) * glass.frame.height
	scale := glass.focus / glass.image
	sharp := maths.V(across*scale, up*scale, -glass.focus)
	from := glass.through(y*width + x)
	return maths.Ray{At: from, Dir: sharp.Sub(from).Unit()}
}

// Gather is the share of a radiance arriving through the lens that
// reaches the film as irradiance behind pixel (x, y) of a picture width
// by height. It is watts a square metre of film for each watt a square
// metre a steradian.
//
// On the axis it is pi times the sine squared of the angle from the
// middle of the film to the edge of the aperture. That is exact for a
// round aperture and a scene of one radiance.
//
// Away from the axis it falls as the fourth power of the cosine of the
// angle from the axis. That is the law for an aperture small beside its
// distance from the film.
//
//ninja:vow practice slayer's camera asks the light a pixel gets; tests only
func (glass Thin) Gather(x, y, width, height int) float64 {
	reach := glass.aperture()
	axis := math.Pi * reach * reach / (reach*reach +
		glass.image*glass.image)
	across := acrossFilm(x, width) * glass.frame.width
	up := upFilm(y, height) * glass.frame.height
	away := across*across + up*up
	cosine := glass.image / math.Sqrt(glass.image*glass.image+away)
	return axis * math.Pow(cosine, fourth)
}

// squarePoint is where the index of the r2 sequence lies along one side
// of the unit square, from minus one to one. Each side steps by its own
// reciprocal power of the plastic number.
func squarePoint(step float64, index int) float64 {
	along := math.Mod(half+step*float64(index), whole)
	return along*pair - whole
}

// concentric is the radius and turn that shirley and chiu's concentric
// map gives a point of the unit square. It keeps points spread evenly.
func concentric(abscissa, ordinate float64) (radius, turn float64) {
	if math.Abs(abscissa) > math.Abs(ordinate) {
		return abscissa, eighth * (ordinate / abscissa)
	}
	return ordinate, quarter - eighth*(abscissa/ordinate)
}

// through is the point on the open aperture an index of the sequence
// takes: the index's point of the r2 sequence on the unit square,
// carried onto the round opening by the concentric map.
func (glass Thin) through(index int) maths.Vec {
	abscissa := squarePoint(whole/plastic, index)
	ordinate := squarePoint(whole/(plastic*plastic), index)
	if abscissa == none && ordinate == none {
		return maths.Vec{}
	}
	radius, turn := concentric(abscissa, ordinate)
	radius *= glass.aperture()
	return maths.V(radius*math.Cos(turn), radius*math.Sin(turn), none)
}

// Thin lenses are projections, as pinholes are.
var _ film.Projection = Thin{}
