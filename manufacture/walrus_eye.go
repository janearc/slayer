package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The eye is small and set high on the side of the head, in a hollow over
// the orbit, with an upper and a lower lid of thick skin round it. The
// globe shows as a wet, dark surface. Sizes chosen.
const (
	// walrusLidGap is how far above and below the eye's centre each lid
	// lies along the hide, in metres.
	walrusLidGap = .021
	// walrusCorneaIndex is the refractive index given the wet surface of
	// the eye: that of water, for the tear film over it. Chosen.
	walrusCorneaIndex = 1.333
)

// walrusLids is the upper and lower lid of each eye as masses on the
// outline's surface: each a short roll of skin lying fore and aft, above
// and below the eye. It refuses an outline the eyes do not lie on.
func walrusLids(outline *walrusOutline) ([]walrusMass, error) {
	var masses []walrusMass
	for _, side := range []float64{-1, 1} {
		eye, err := walrusEyeOn(outline, side)
		if err != nil {
			return nil, err
		}
		for _, lid := range []float64{-1, 1} {
			centre := eye.Add(maths.V(0, lid*walrusLidGap, 0))
			masses = append(masses,
				walrusBlob(centre, maths.V(.034, .009, .014), .9))
		}
	}
	return masses, nil
}

// walrusCorneaMaterial is the eye's surface: dark, wet and glossy, with
// the tear film's index.
func walrusCorneaMaterial() *thing.Material {
	return &thing.Material{Name: "walrus eye", Colour: maths.RGB(.012, .009, .008),
		Gloss: .95, Satin: .1, Index: walrusCorneaIndex}
}
