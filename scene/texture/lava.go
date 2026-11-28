package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Lava is the light of a molten ball: a cooling crust of dark, rigid,
// polygonal plates of many sizes drifting on magma, which shows bright in
// thin seams between them where it wells up, as it does in the Hawaiian
// lava lakes and at Io's Loki Patera. The seam's glow falls off quickly
// into the plates on either side. Glow is the light of the open magma,
// the ball's own temperature's colour; a plate's crust is a dull red,
// dimmer where it is thick, and brighter beside a seam where it is thin.
// The light is scaled, in each colour, so that its mean over the ball is
// Glow: a ball drawn with this texture gives the light a ball of its
// temperature and radius gives, only in plates and seams, and its
// luminosity is not changed by the crust.
//
// Radius is the ball's, in the units of the points the texture is asked
// at. A ball far enough off that a pixel covers more than a few plates
// is its mean light, Glow, with no plates. It is a model of a crust, not
// a simulation of one: the plates are the cells of cellular noise (see
// noise.Cellular), bent a little so their sides are not straight, and
// made smaller in one place and larger in another by a smooth change of
// the lattice's scale, so that plate sizes run into one another.
type Lava struct {
	Seed   uint64
	Radius float64
	Glow   maths.Light
	mean   maths.Light
}

// lavaSamples is how many directions the mean of the light is taken over:
// a spiral of points over the sphere. Picked: the mean is steady to a
// share of a per cent from sixteen thousand, which the seams, a few
// thousandths of a ball wide, take.
const lavaSamples = 16384

// The look of the crust, each a number picked, in picked-numbers.yaml.
var (
	lavaFrequency     = pickedNumber("lava-frequency")
	lavaSizeSpread    = pickedNumber("lava-size-spread")
	lavaSizeFrequency = pickedNumber("lava-size-frequency")
	lavaWarp          = pickedNumber("lava-warp")
	lavaWarpFrequency = pickedNumber("lava-warp-frequency")
	lavaSeamWidth     = pickedNumber("lava-seam-width")
	lavaSeamSpread    = pickedNumber("lava-seam-spread")
	lavaHaloWidth     = pickedNumber("lava-halo-width")
	lavaHaloBoost     = pickedNumber("lava-halo-boost")
	lavaCrustLight    = pickedNumber("lava-crust-light")
	lavaPlateSpread   = pickedNumber("lava-plate-spread")
	lavaBlurFrom      = pickedNumber("lava-blur-from")
	lavaBlurTo        = pickedNumber("lava-blur-to")
	lavaCrustTint     = maths.RGB(1, pickedNumber("lava-crust-green"),
		pickedNumber("lava-crust-blue"))
)

// The offsets from a lava ball's seed of the fields it is made of, so each
// is its own. Picked: any different numbers do.
const (
	lavaLatticeStream = 1
	lavaSizeStream    = 20
	lavaWarpStream    = 10
)

// lavaMostStretch is how many times more the plates' lattice may stretch a
// step of the ball one way than the other, at most, before a plate is a
// streak and not a polygon. Picked 2026-11-27 by eye.
const lavaMostStretch = 3.0

// NewLava is the lava of a ball of the radius, its open magma giving
// the glow. It takes the mean of its own light, so that mean is the glow.
func NewLava(seed uint64, radius float64, glow maths.Light) Lava {
	lava := Lava{Seed: seed, Radius: radius, Glow: glow,
		mean: maths.Light{R: 1, G: 1, B: 1}}
	var sum maths.Light
	golden := math.Pi * (math.Sqrt(5) - 1)
	for index := range lavaSamples {
		height := 1 - 2*(float64(index)+0.5)/lavaSamples
		flat := math.Sqrt(1 - height*height)
		angle := golden * float64(index)
		sum = sum.Add(lava.raw(maths.V(flat*math.Cos(angle), height,
			flat*math.Sin(angle))))
	}
	lava.mean = sum.Scale(1.0 / lavaSamples)
	return lava
}

// size is how many times larger than the coarsest the plates are at a
// direction, from a smooth noise over the ball: the lattice is read at the
// direction's frequency times the reciprocal of this, so plates are
// smaller where it is small. The noise is gentle enough that the lattice
// is stretched and never folded over itself.
func (lava Lava) size(direction maths.Vec) float64 {
	spot := direction.Unit().Scale(lavaSizeFrequency)
	return math.Exp(-lavaSizeSpread * noise.Gradient(lava.Seed+lavaSizeStream,
		spot.X, spot.Y, spot.Z))
}

// bent is a direction as a point in the plates' lattice, its frequency
// over the unit direction, moved a little by smooth noise so the plates'
// sides are not straight, and made finer or coarser by size.
func (lava Lava) bent(direction maths.Vec) maths.Vec {
	at := direction.Unit().Scale(lavaFrequency)
	warp := at.Scale(lavaWarpFrequency)
	bent := at.Add(maths.V(
		noise.Gradient(lava.Seed+lavaWarpStream, warp.X, warp.Y, warp.Z),
		noise.Gradient(lava.Seed+lavaWarpStream+1, warp.X, warp.Y, warp.Z),
		noise.Gradient(lava.Seed+lavaWarpStream+2, warp.X, warp.Y,
			warp.Z)).Scale(lavaWarp))
	return bent.Scale(1 / lava.size(direction))
}

// Seam is what the crust is at a direction: how much the open magma shows,
// from nought to one, which is one on a seam's line and falls off quickly
// into the plates; the halo of the thinner, hotter crust beside it, wider
// and from nought to one; the tone of the plate's thickness, from nought to
// one, thin to thick; how wide the seam is, in coarse plates; and how large
// the plates are there, in coarse plates.
type Seam struct {
	Core, Halo, Thickness, Width, Size float64
}

// seam is the crust at a direction. Each seam is as wide as the plate it
// borders says, between 1 - lavaSeamSpread and 1 + lavaSeamSpread of
// lavaSeamWidth, and its distance is the lattice's divided by the scale, so
// that a seam is as wide where plates are small as where they are large.
func (lava Lava) seam(direction maths.Vec) Seam {
	at := lava.bent(direction)
	size := lava.size(direction)
	cell := noise.Cellular(lava.Seed+lavaLatticeStream, at.X, at.Y, at.Z)
	width := lavaSeamWidth * (1 - lavaSeamSpread +
		2*lavaSeamSpread*cell.Identity)
	border := cell.Border * size
	return Seam{Core: math.Exp(-border / width),
		Halo:      math.Exp(-border / (lavaHaloWidth * width / lavaSeamWidth)),
		Thickness: cell.Identity, Width: width, Size: size}
}

// raw is the light at a direction before it is scaled to its mean: a
// plate's crust, a dull red that is dimmer where the plate is thick, and
// brighter and hotter in a halo beside a seam where it is thin, and the
// open magma in the seam, a glow that falls off exponentially into the
// plate from the seam's line.
func (lava Lava) raw(direction maths.Vec) maths.Light {
	found := lava.seam(direction)
	tone := 1 + lavaPlateSpread*(1-2*found.Thickness)
	crusted := lava.Glow.Times(lavaCrustTint).Scale(lavaCrustLight * tone *
		(1 + lavaHaloBoost*found.Halo))
	return crusted.Scale(1 - found.Core).Add(lava.Glow.Scale(found.Core))
}

// At is the light at a point of the ball, in plates, its mean over the
// ball the Glow.
func (lava Lava) At(point maths.Vec) maths.Light {
	raw := lava.raw(point)
	return lava.Glow.Times(maths.Light{R: share(raw.R, lava.mean.R),
		G: share(raw.G, lava.mean.G), B: share(raw.B, lava.mean.B)})
}

// share is a light's channel over its mean, so the mean is one, or nought
// where the mean is nought.
func share(channel, mean float64) float64 {
	if !(mean > nought) {
		return nought
	}
	return channel / mean
}

// Within is the light at a point for a pixel footprint wide: the plates
// where the pixel is smaller than a few of them, their mean where it is
// larger, and between them blended, so a far ball is not speckled.
func (lava Lava) Within(point maths.Vec, footprint float64) maths.Light {
	plates := footprint / lava.Radius * lavaFrequency / lava.size(point)
	blend := maths.Smooth(lavaBlurFrom, lavaBlurTo, plates)
	return lava.At(point).Scale(1 - blend).Add(lava.Glow.Scale(blend))
}
