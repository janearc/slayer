package manufacture

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// nitrogenGreen is nitrogen dioxide's absorption at the green channel, in
// inverse metres for each part in a thousand million: the revised IMPROVE
// equation's "0.33 x [NO2 (ppb)]", in inverse megametres (Pitchford and
// others 2007, as the IMPROVE algorithm page at vista.cira.colostate.edu
// gives it; the page names no wavelength). Read.
const nitrogenGreen = 0.33e-6

// nitrogenBlue and nitrogenRed are the gas's absorption at the blue and
// red channels, 465 and 610 nm, as times its green's. Picked from the
// recalled shape of its visible band, strongest near 400 nm and falling
// into the red; not read, to be pinned to Vandaele and others 1998.
const nitrogenBlue, nitrogenRed = 3.0, 0.45

// errAtmosphere is the refusal of an atmosphere asked its box before it is
// told the ground it lies over.
var errAtmosphere = errors.New("manufacture: an atmosphere is asked its " +
	"box over the box of a ground")

// Atmosphere is a layer of the air a set prescribes: a cloud deck between
// two heights, a level deck of mist from a base, or mist over the ground,
// the mist's weather, over the terrain and as far beyond it as the layer
// says, thickest over a thing where it names one. It is an object of the
// manufactory that makes no mesh: it adds a volume of the mist's particles.
type Atmosphere struct {
	// Name is the layer's weather, which its material is called.
	Name string
	// Weather is the mist's: its extinction, depth, coverage, seed, wind
	// and how it scatters.
	Weather medium.GroveWeather
	// Albedo is how much of the light the particles keep.
	Albedo float64
	// Over is the box of the ground the layer lies over, and Beyond how
	// far past it, east, west, north and south, the layer reaches.
	Over   maths.Box
	Beyond float64
	// Cloud, with CloudBase and CloudTop, says the layer is a cloud deck
	// between two heights. Level, with Base, says it is a level deck of
	// mist from a height. Neither says it lies over the ground.
	Cloud, Level        bool
	CloudBase, CloudTop float64
	Base                float64
	// Floor is the ground's height at a scene east and south, which a
	// layer over the ground or a deck is held above.
	Floor func(x, z float64) float64
	// Toward names the thing the layer is thickest over, or is empty, with
	// the reach it thins over and the least share it thins to.
	Toward       string
	Reach, Least float64
	// Angstrom is the particles' Angstrom exponent where Angstromed says
	// they have one, and Nitrogen the parts in a thousand million of
	// nitrogen dioxide mixed through them.
	Angstrom   float64
	Angstromed bool
	Nitrogen   float64
}

// Atmosphere is an object.
var _ object = Atmosphere{}

// Where finds the place a named thing stands, if it has been placed.
type Where func(name string) (maths.Vec, bool)

// bounds is the box the layer's field fills: the ground's, widened by
// Beyond east, west, north and south; a cloud deck is held between its
// heights, a level deck between its base and the mist's depth over it, and
// a layer over the ground stands the depth over the ground's highest.
func (atmosphere Atmosphere) bounds() maths.Box {
	box := atmosphere.Over
	beyond := atmosphere.Beyond
	box.Min.X, box.Max.X = box.Min.X-beyond, box.Max.X+beyond
	box.Min.Z, box.Max.Z = box.Min.Z-beyond, box.Max.Z+beyond
	return box
}

// held is the field's box in height, as the kind of layer says.
func (atmosphere Atmosphere) held(box maths.Box) maths.Box {
	switch {
	case atmosphere.Level:
		box.Min.Y = atmosphere.Base
		box.Max.Y = atmosphere.Base + atmosphere.Weather.Depth
	case !atmosphere.Cloud:
		box.Max.Y += atmosphere.Weather.Depth
	}
	return box
}

// extent is the box of the atmosphere, from its numbers alone.
func (atmosphere Atmosphere) extent(from seed) (maths.Box, error) {
	return atmosphere.Extent()
}

// Extent is the box of the layer's field, in the scene's frame: the ground's
// widened by the layer's reach beyond it, and in height as its kind says. It
// makes no field. A cloud deck's is between its two heights.
func (atmosphere Atmosphere) Extent() (maths.Box, error) {
	over := atmosphere.Over
	if !(over.Max.X > over.Min.X && over.Max.Z > over.Min.Z) {
		return maths.Box{}, errAtmosphere
	}
	box := atmosphere.held(atmosphere.bounds())
	if atmosphere.Cloud {
		box.Min.Y, box.Max.Y = atmosphere.CloudBase, atmosphere.CloudTop
	}
	return box, nil
}

// seen is how an eye sees the atmosphere's box.
func (atmosphere Atmosphere) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenObject(atmosphere, at, by)
}

// thinned is a field that can be made thickest over a point: a layer, or
// a cloud deck whose cover is greatest there.
type thinned interface {
	Toward(point maths.Vec, reach, least float64) error
}

// Staged adds the layer to a world as a volume: its field, thickest over
// the thing it names, and the material of the particles and the gas the
// mist says.
func (atmosphere Atmosphere) Staged(world *thing.World, where Where) error {
	field, err := atmosphere.field(where)
	if err != nil {
		return err
	}
	material := thing.Diffuse(atmosphere.Name, maths.Grey(atmosphere.Albedo))
	if made, ok := field.(*medium.Layer); ok {
		field, material, err = atmosphere.coloured(made)
		if err != nil {
			return err
		}
	}
	world.AddVolume(thing.NewVolume(field, material))
	return nil
}

// field is the layer's field: a cloud deck, a level deck, or a layer over
// the ground, over the terrain and as far beyond it as the layer says,
// thickest over its thing where it names one.
func (atmosphere Atmosphere) field(where Where) (medium.Medium, error) {
	bounds := atmosphere.bounds()
	if atmosphere.Cloud {
		deck, err := medium.NewCloud(atmosphere.Weather, bounds,
			atmosphere.CloudBase, atmosphere.CloudTop)
		if err != nil || atmosphere.Toward == "" {
			return deck, err
		}
		return deck, atmosphere.thickest(deck, where)
	}
	var field *medium.Layer
	var err error
	bounds = atmosphere.held(bounds)
	if atmosphere.Level {
		field, err = medium.NewDeck(atmosphere.Weather, bounds,
			atmosphere.Floor, atmosphere.Base)
	} else {
		field, err = medium.NewLayer(atmosphere.Weather, bounds,
			atmosphere.Floor)
	}
	if err != nil || atmosphere.Toward == "" {
		return field, err
	}
	return field, atmosphere.thickest(field, where)
}

// thickest makes the field thickest over the layer's thing.
func (atmosphere Atmosphere) thickest(field thinned, where Where) error {
	at, found := where(atmosphere.Toward)
	if !found {
		return fmt.Errorf("the layer thins toward %q, which has no "+
			"placed origin", atmosphere.Toward)
	}
	return field.Toward(at, atmosphere.Reach, atmosphere.Least)
}

// coloured is the layer and its material as the mist's particles and gas
// make them: a layer whose particles say an Angstrom exponent, or that
// holds nitrogen dioxide, is an aerosol thicker to some colours than
// others, and its albedo by channel is its particles' share of what each
// channel meets. A mist that says neither is grey, as before.
func (atmosphere Atmosphere) coloured(field *medium.Layer) (medium.Medium,
	*thing.Material, error) {
	if !atmosphere.Angstromed && atmosphere.Nitrogen == 0 {
		return field, thing.Diffuse(atmosphere.Name,
			maths.Grey(atmosphere.Albedo)), nil
	}
	absorb := maths.Light{}
	if atmosphere.Nitrogen > 0 {
		extinction := atmosphere.Weather.Extinction
		if !(extinction > 0) {
			return nil, nil, errors.New("a gas is mixed through " +
				"particles, which need an extinction above nought")
		}
		green := atmosphere.Nitrogen * nitrogenGreen / extinction
		absorb = maths.RGB(green*nitrogenRed, green, green*nitrogenBlue)
	}
	aerosol, err := medium.NewAbsorbingAerosol(field, atmosphere.Angstrom,
		absorb)
	if err != nil {
		return nil, nil, err
	}
	return aerosol, thing.Diffuse(atmosphere.Name,
		aerosol.Albedo(atmosphere.Albedo)), nil
}
