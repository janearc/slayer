package ocular

import "github.com/janearc/slayer/maths"

// Colour selects an illustrative adult human iris palette.
type Colour uint8

const (
	Brown Colour = iota
	Hazel
	Amber
	Green
	Blue
	Gray
)

// HumanIris selects the palette and the individual pattern within it.
type HumanIris struct {
	Colour Colour
	Seed   uint64
}

// Human returns a reference adult eye texture specification in metres.
// The zero choice is brown. An unrecognised colour also uses brown.
// Callers can edit the returned value before passing it to NewSurface.
func Human(choice HumanIris) Spec {
	stroma, inner := humanPigment(choice.Colour)
	return Spec{
		Radius: .00578,
		Pupil:  Pupil{HalfWidth: .33, HalfHeight: .33},
		Palette: Palette{
			Stroma: stroma,
			Inner:  inner,
			Pupil:  maths.RGB(.0035, .0032, .0035),
			Sclera: maths.RGB(.75, .69, .65),
		},
		Morphology: Morphology{
			Fibres: .92, Crypts: .89, Furrows: .8,
			Limbus: .86, InnerPigment: .88, ScleralVessels: .55,
		},
		Seed: choice.Seed,
	}
}

func humanPigment(colour Colour) (stroma, inner maths.Light) {
	switch colour {
	case Hazel:
		return maths.RGB(.159, .191, .116), maths.RGB(.30, .166, .062)
	case Amber:
		return maths.RGB(.28, .17, .066), maths.RGB(.36, .218, .078)
	case Green:
		return maths.RGB(.144, .213, .147), maths.RGB(.236, .212, .106)
	case Blue:
		return maths.RGB(.128, .194, .265), maths.RGB(.224, .214, .171)
	case Gray:
		return maths.RGB(.162, .184, .192), maths.RGB(.186, .182, .161)
	default:
		return maths.RGB(.19, .096, .047), maths.RGB(.283, .142, .065)
	}
}
