package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// standardGravity is the standard acceleration of gravity, m s^-2, as the
// plume package takes it for a column's shape. Defined.
const standardGravity = 9.80665

// plumeStretches is how many stretches a plume is cut into down the wind,
// each its own volume, so that a march across the thin plume near the
// mouth is not spread over the width of the wide one far off. Two, since
// every volume in the world is tried by every shadow ray of every step of
// a march, and a plume of eight stretches a mouth outlasted a 40 minute
// draw of 55 mouths. Picked.
const plumeStretches = 2

// plumeStretchPower is how the stretches lengthen down the wind: the k-th
// of n ends at the reach times k over n to this power, so the first, an
// eighth of the reach, is short where the plume is narrow. Picked.
const plumeStretchPower = 3

// Plume is one mouth's plume of particles, carried down the wind: an object
// of the manufactory that makes no mesh, only stretches of a medium, each
// a volume of the particles' optics.
type Plume struct {
	// Name is the plume's, which its volumes' material wears.
	Name string
	// At is where the mouth stands and Turn how the plume is turned down
	// the wind, in the world's frame.
	At   maths.Vec
	Turn maths.Orientation
	// Wind is the wind's speed, metres a second, and Reach how far down it
	// the plume is laid, metres.
	Wind, Reach float64
	// Mouth is the mouth's radius, metres, Exit the gas's speed there,
	// metres a second, Hot its heat and Air the air's, kelvin.
	Mouth, Exit, Hot, Air float64
	// Lid is the height, over the world's floor, of the top of the air
	// that mixes, where Lidded says there is one.
	Lid    float64
	Lidded bool
	// Rate is how fast the particles leave, grams a second, Extinction
	// their mass extinction, square metres a gram, Asymmetry and Exponent
	// how they scatter, and Albedo how much of the light they keep.
	Rate, Extinction, Asymmetry, Exponent, Albedo float64
}

// Plume is an object.
var _ object = Plume{}

// stack is what leaves the mouth: Briggs's buoyancy flux, g w r squared
// times the share the gas is lighter than the air, (T - Ta) / T for gases
// alike but for heat, and the lid over the mouth.
func (plume Plume) stack() (medium.Stack, error) {
	flux := standardGravity * plume.Exit * plume.Mouth * plume.Mouth *
		(plume.Hot - plume.Air) / plume.Hot
	lid := 0.0
	if plume.Lidded && plume.Lid > plume.At.Y {
		lid = plume.Lid - plume.At.Y
	}
	stack, err := medium.NewStack(plume.Mouth, flux, plume.Wind,
		plume.Rate, plume.Extinction, lid)
	if err != nil {
		return medium.Stack{}, err
	}
	return stack.Scattering(plume.Asymmetry, plume.Exponent)
}

// stretches are the plume's stretches, from the mouth down the wind, each
// ending further out than the last by the stretch power.
func (plume Plume) stretches() ([]*medium.Bent, error) {
	stack, err := plume.stack()
	if err != nil {
		return nil, err
	}
	var cut []*medium.Bent
	from := 0.0
	for stretch := 1; stretch <= plumeStretches; stretch++ {
		share := float64(stretch) / plumeStretches
		upto := plume.Reach * math.Pow(share, plumeStretchPower)
		bent, err := medium.NewBent(stack, from, upto)
		if err != nil {
			return nil, err
		}
		cut = append(cut, bent)
		from = upto
	}
	return cut, nil
}

// space is where the plume stands: at its mouth, turned down the wind.
func (plume Plume) space() maths.Subspace {
	return maths.NewSubspace(plume.At, plume.Turn, 1)
}

// extent is the box of the plume, from its numbers alone.
func (plume Plume) extent(from seed) (maths.Box, error) {
	return plume.Extent()
}

// Extent is the box that holds every stretch of the plume, in the world's
// frame: from the mouth down the wind for the whole reach, as wide and as
// high as the stretches' sections are a few spreads out, held under the
// lid. It makes no volume.
func (plume Plume) Extent() (maths.Box, error) {
	cut, err := plume.stretches()
	if err != nil {
		return maths.Box{}, err
	}
	box := cut[0].Bounds()
	for _, bent := range cut[1:] {
		box = enclosing(box, bent.Bounds())
	}
	return transformed(box, plume.space()), nil
}

// seen is how an eye sees the plume's box.
func (plume Plume) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenObject(plume, at, by)
}

// Staged adds the plume to a world as it has always been added: each
// stretch a volume in the particles' diffuse material, in order from the
// mouth, after the volumes there are.
func (plume Plume) Staged(world *thing.World) error {
	cut, err := plume.stretches()
	if err != nil {
		return err
	}
	material := thing.Diffuse(plume.Name, maths.Grey(plume.Albedo))
	placed := thing.Still(plume.space())
	for _, bent := range cut {
		volume := thing.NewVolume(bent, material)
		volume.Motion = placed
		world.AddVolume(volume)
	}
	return nil
}
