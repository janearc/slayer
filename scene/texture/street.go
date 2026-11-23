package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Street surfaces: the reflectance of asphalt, of paving slabs and of an
// iron grate, as a street's surfaces have it by day. Each is a colour
// and nothing more: how a wet street mirrors is the material's gloss,
// and what these do is darken as water soaks in, as a porous surface
// does, and keep from shimmering where the eye is far, by fading each
// fine feature to its mean once a pixel is as wide as the feature.

// the width of a pixel, as a share of a feature's width, at which the
// feature is seen whole, and at which it is no longer seen at all; each
// picked and held in picked-numbers.yaml
var (
	// resolvedFull is the share at which a feature is drawn whole.
	resolvedFull = pickedNumber("resolved-full")
	// resolvedNone is the share at which a feature is an even grey.
	resolvedNone = pickedNumber("resolved-none")
)

// resolved is how much of a feature a pixel footprint wide can see: all
// of it while the pixel is narrow, none once the pixel is as wide as the
// feature, and a smooth fade between. A footprint of nought is a lone
// ray, which sees the feature whole.
func resolved(feature, footprint float64) float64 {
	if !(footprint > nought) || !(feature > nought) {
		return unity
	}
	width := footprint / feature
	return unity - smoothed((width-resolvedFull)/(resolvedNone-resolvedFull))
}

// The streams of the asphalt's hash, each defined as a distinct offset
// from the seed, so that the chippings and the damp are drawn alone.
const (
	// chipStream is defined: the offset of the stream the chippings are
	// drawn from.
	chipStream = 1
	// dampStream is defined: the offset of the stream the damp patches
	// are drawn from.
	dampStream = 2
)

// the numbers Asphalt is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// asphaltGrain is how wide the mottling of the binder is, in metres.
	asphaltGrain = pickedNumber("asphalt-grain")
	// asphaltContrast is how far the mottling takes the colour from its
	// mean, as a share of it.
	asphaltContrast = pickedNumber("asphalt-contrast")
	// asphaltChip is how wide a bare chipping is, in metres.
	asphaltChip = pickedNumber("asphalt-chip")
	// asphaltChipShare is the share of the surface in bare chippings.
	asphaltChipShare = pickedNumber("asphalt-chip-share")
	// asphaltChipLift is how far a bare chipping's colour replaces the
	// binder's where one lies, as a share.
	asphaltChipLift = pickedNumber("asphalt-chip-lift")
	// asphaltPatch is how wide a damp patch is, in metres.
	asphaltPatch = pickedNumber("asphalt-patch")
	// asphaltPatchOctaves is how many scales of noise a patch's edge and
	// the binder's mottling have.
	asphaltPatchOctaves = int(pickedNumber("asphalt-patch-octaves"))
	// asphaltSoft is how far, in the noise's own units, a patch's edge
	// fades from dry to damp.
	asphaltSoft = pickedNumber("asphalt-soft")
	// asphaltRange is about how far either side of nought the patch noise
	// runs.
	asphaltRange = pickedNumber("asphalt-range")
	// asphaltDarkening is how much of its light a wet road loses to the
	// water in it, as a share.
	asphaltDarkening = pickedNumber("asphalt-darkening")
	// asphaltSoaked is the wetness at which every patch is damp and the
	// road is one darkness.
	asphaltSoaked = pickedNumber("asphalt-soaked")
	// asphaltTrackWidth is how wide a wheel track is, in metres.
	asphaltTrackWidth = pickedNumber("asphalt-track-width")
	// asphaltTrackBias is how much sooner a wheel track is damp than the
	// road between.
	asphaltTrackBias = pickedNumber("asphalt-track-bias")
	// asphaltDepth is how far up the noise is read, so that a road flat
	// at nought never lies on a lattice plane of it.
	asphaltDepth = pickedNumber("asphalt-depth")
)

// Asphalt is a road surface: the colour of its binder, mottled, with a
// scatter of bare chippings, and, by Wet, damp in patches that join into
// one darkness as the road soaks.
//
// Dry is the colour of the dry binder
// and Chipping of a bare stone; Wet is from nought, dry, to one, soaked
// through, which is the darkness a water film over it sits on. Tracks is
// the distance of the two wheel tracks from the road's middle, which
// wet first; nought is no tracks. Seed picks which patches.
//
// A footprint above nought fades the mottling, the chippings and the
// patches to their means as they grow narrower than a pixel.
type Asphalt struct {
	Dry, Chipping maths.Light
	Wet           float64
	Tracks        float64
	Seed          uint64
}

// compile-time check that Asphalt paints by footprint.
var _ Footprinted = Asphalt{}

// At is the road at the point as a lone ray sees it.
func (asphalt Asphalt) At(point maths.Vec) maths.Light {
	return asphalt.Within(point, nought)
}

// Within is the road at the point as a pixel a footprint wide sees it: the
// binder, its mottling and its chippings, each only as far as the pixel
// can tell it, darkened by the damp.
func (asphalt Asphalt) Within(point maths.Vec, footprint float64) maths.Light {
	mottled := unity + asphaltContrast*asphalt.mottle(point)*
		resolved(asphaltGrain, footprint)
	chips := asphalt.chipped(point, footprint) * asphaltChipLift
	colour := asphalt.Dry.Scale(mottled * (unity - chips)).
		Add(asphalt.Chipping.Scale(chips))
	return colour.Scale(unity - asphaltDarkening*asphalt.damp(point, footprint))
}

// mottle is the binder's mottling at the point, from about minus one to
// one.
func (asphalt Asphalt) mottle(point maths.Vec) float64 {
	return noise.Fractal(asphalt.Seed, point.X/asphaltGrain, asphaltDepth,
		point.Z/asphaltGrain, asphaltPatchOctaves)
}

// chipCell is which chipping cell a coordinate is in.
func chipCell(coordinate float64) int64 {
	return int64(math.Floor(coordinate / asphaltChip))
}

// chipped is how much of a bare chipping lies at the point, as a share of
// what a chipping adds: nought or one for a lone ray, and the mean share
// for a pixel too wide to see one.
func (asphalt Asphalt) chipped(point maths.Vec, footprint float64) float64 {
	var bare float64
	hash := noise.Hash2(asphalt.Seed+chipStream, chipCell(point.X),
		chipCell(point.Z))
	if hash > unity-asphaltChipShare {
		bare = unity
	}
	seen := resolved(asphaltChip, footprint)
	return asphaltChipShare + (bare-asphaltChipShare)*seen
}

// damp is how damp the road is at the point, from nought to one: the share
// of the road wet by Wet, laid in patches that begin in the wheel tracks,
// and the share itself where a pixel is wider than a patch.
func (asphalt Asphalt) damp(point maths.Vec, footprint float64) float64 {
	share := smoothed(asphalt.Wet / asphaltSoaked)
	if share <= nought {
		return nought
	}
	if share >= unity {
		return unity
	}
	wave := noise.Fractal(asphalt.Seed+dampStream, point.X/asphaltPatch,
		asphaltDepth, point.Z/asphaltPatch, asphaltPatchOctaves) +
		asphalt.track(point.X)
	level := asphaltRange * (unity - pair*share)
	patch := smoothed((wave - level) / asphaltSoft)
	seen := resolved(asphaltPatch, footprint)
	return share + (patch-share)*seen
}

// track is how much sooner the road is damp at across than between its
// wheel tracks: nought away from them, and asphaltTrackBias on one.
func (asphalt Asphalt) track(across float64) float64 {
	if !(asphalt.Tracks > nought) {
		return nought
	}
	near := math.Min(math.Abs(across-asphalt.Tracks),
		math.Abs(across+asphalt.Tracks))
	return asphaltTrackBias * (unity - smoothed(near/asphaltTrackWidth))
}

// the numbers Slabs is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// slabsDarkening is how much of its light a wet slab loses, as a
	// share.
	slabsDarkening = pickedNumber("slabs-darkening")
	// slabsTone is how far one slab's colour strays from the next's, as
	// a share of the colour.
	slabsTone = pickedNumber("slabs-tone")
	// slabsJointLift is the share of a slab's colour a joint has, from
	// mortar and dirt.
	slabsJointLift = pickedNumber("slabs-joint-lift")
	// slabsSharp is the narrowest footprint a joint is filtered by, in
	// metres, so that a lone ray finds a joint without dividing by nought.
	slabsSharp = pickedNumber("slabs-sharp")
)

// Slabs is a pavement of square slabs with a joint between them, each
// slab a little lighter or darker than the next. Size is the slab's side
// and Joint the joint's width, in metres; Wet is from nought to one; Seed
// picks each slab's tone. A footprint wider than the joint thins its
// darkness to the share of the pavement it covers.
type Slabs struct {
	Slab        maths.Light
	Size, Joint float64
	Wet         float64
	Seed        uint64
}

// compile-time check that Slabs paints by footprint.
var _ Footprinted = Slabs{}

// compile-time check that Slabs names its joints, as brick does.
var _ interface{ InJoint(point maths.Vec) bool } = Slabs{}

// At is the pavement at the point as a lone ray sees it.
func (slabs Slabs) At(point maths.Vec) maths.Light {
	return slabs.Within(point, nought)
}

// Within is the pavement at the point as a pixel a footprint wide sees it.
func (slabs Slabs) Within(point maths.Vec, footprint float64) maths.Light {
	column, row := slabs.cell(point)
	hash := noise.Hash2(slabs.Seed, column, row)
	tone := unity + slabsTone*(pair*hash-unity)
	colour := slabs.Slab.Scale(tone * (unity - slabsDarkening*slabs.Wet))
	joint := slabs.jointed(point, footprint)
	return colour.Scale(unity - joint*(unity-slabsJointLift))
}

// cell is which slab the point lies on, by column across and row along.
func (slabs Slabs) cell(point maths.Vec) (int64, int64) {
	return int64(math.Floor(point.X / slabs.Size)),
		int64(math.Floor(point.Z / slabs.Size))
}

// jointed is how much of a pixel a footprint wide at the point is joint,
// from nought to one: the joint's width laid over the pixel's, and, for a
// pixel wider than a slab, the share of the pavement the joints cover.
func (slabs Slabs) jointed(point maths.Vec, footprint float64) float64 {
	reach := math.Max(footprint, slabsSharp)
	edge, reaching := slabs.edge(point), slabs.Joint/pair
	low := math.Max(edge-reach/pair, -reaching)
	high := math.Min(edge+reach/pair, reaching)
	covered := math.Max(nought, high-low) / reach
	mean := unity - math.Pow(unity-slabs.Joint/slabs.Size, pair)
	seen := resolved(slabs.Size, footprint)
	return mean + (covered-mean)*seen
}

// edge is how far the point is from the nearest joint's middle, in
// metres.
func (slabs Slabs) edge(point maths.Vec) float64 {
	across := math.Mod(math.Abs(point.X), slabs.Size)
	along := math.Mod(math.Abs(point.Z), slabs.Size)
	return math.Min(math.Min(across, slabs.Size-across),
		math.Min(along, slabs.Size-along))
}

// InJoint is whether the point lies in a joint, which grime and weather
// lodge in.
func (slabs Slabs) InJoint(point maths.Vec) bool {
	return slabs.edge(point) < slabs.Joint/pair
}

// grateSlot is the share of the grate's pitch that is open slot: the
// bars of a gully grating are about as wide as the gaps between them. It
// is picked and held in picked-numbers.yaml.
var grateSlot = pickedNumber("grate-slot")

// Grate is an iron gully grating seen from above: bars across it with
// slots between that are black, since the pit below is. Pitch is the
// distance from one bar to the next along the street's length, in
// metres; Bar is the colour of the iron and Pit of the dark below.
// Bars begin at nought along, so a grating placed on a multiple of the
// pitch starts on a bar.
type Grate struct {
	Bar, Pit maths.Light
	Pitch    float64
}

// compile-time check that Grate paints by footprint.
var _ Footprinted = Grate{}

// At is the grating at the point as a lone ray sees it.
func (grate Grate) At(point maths.Vec) maths.Light {
	return grate.Within(point, nought)
}

// Within is the grating at the point as a pixel a footprint wide sees it:
// bars and slots, or their mean once a pixel is as wide as the pitch.
func (grate Grate) Within(point maths.Vec, footprint float64) maths.Light {
	phase := math.Mod(math.Abs(point.Z), grate.Pitch) / grate.Pitch
	var slot float64
	if phase > unity-grateSlot {
		slot = unity
	}
	seen := resolved(grate.Pitch, footprint)
	slot = grateSlot + (slot-grateSlot)*seen
	return grate.Bar.Scale(unity - slot).Add(grate.Pit.Scale(slot))
}
