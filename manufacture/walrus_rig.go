package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// NewWalrusRig adapts the continuous pinniped skin and its joint influences
// into a heavy walrus study. The body and flippers remain a single indexed
// shell; dimensions and face landmarks are visual approximations.
func NewWalrusRig() *SealRig {
	rig := NewSealRig()
	for i := range rig.vertices {
		rig.vertices[i].point = walrusProportions(rig.vertices[i].point)
		rig.rest[i] = rig.vertices[i].point
	}
	for i := range rig.limbs {
		rig.limbs[i].root = walrusProportions(rig.limbs[i].root)
		rig.limbs[i].tip = walrusProportions(rig.limbs[i].tip)
	}
	return rig
}

// walrusProportions reshapes a point of the seal-derived body into the
// walrus's: a broader chest and trunk, and a wider muzzle.
func walrusProportions(p maths.Vec) maths.Vec {
	// Broaden the chest and trunk, and give the muzzle more width without
	// narrowing the connecting neck into the seal's distinct head shape.
	chest := 1 - maths.Smooth(.25, .75, p.X)
	p.Y *= 1.70 - .10*chest
	p.Z *= 1.95 - .25*chest
	return p
}

// Walrus builds an original poseable study; its scale is approximately an
// adult's, while tusks and sensory hairs are independent attached meshes.
func Walrus(rig *SealRig, pose SealRigPose, place maths.Subspace) ([]thing.Thing, error) {
	skin, err := rig.Skin(pose)
	if err != nil {
		return nil, err
	}
	coat := &thing.Material{Name: "walrus skin", Colour: maths.RGB(.30, .185, .14), Gloss: .13, Satin: .59}
	ivory := &thing.Material{Name: "walrus tusk", Colour: maths.RGB(.77, .72, .58), Gloss: .15, Satin: .42}
	eye := &thing.Material{Name: "walrus eye", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	bristle := &thing.Material{Name: "walrus vibrissae", Colour: maths.RGB(.70, .60, .46), Gloss: .08}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, side := range []float64{-1, 1} {
		// The heavy paired upper-lip pads carry the sensory whiskers and
		// overlap the connecting head skin.
		pad := rig.HeadPoint(pose, maths.V(.815, .073, side*.065))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .079}, Material: coat,
			Motion: thing.Still(maths.NewSubspace(pad, maths.Identity, 1).Within(place))})
		// Eyes are small and set in the side of the broad head.
		at := rig.HeadPoint(pose, maths.V(.665, .185, side*.193))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .013}, Material: eye,
			Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		// A tapered, slightly forward curving tooth grows down from each side
		// of the upper jaw. Section rings are posed with the head.
		rings := make([][]maths.Vec, 0, 10)
		for j := 0; j <= 9; j++ {
			t := float64(j) / 9
			centre := maths.V(.806+.040*t*t, .064-.36*t, side*.10)
			radius := .027 * (1 - .93*t)
			ring := make([]maths.Vec, 16)
			for k := range ring {
				a := 2 * math.Pi * float64(k) / float64(len(ring))
				ring[k] = rig.HeadPoint(pose, centre.Add(maths.V(radius*math.Cos(a), 0, radius*math.Sin(a))))
			}
			rings = append(rings, ring)
		}
		tusk, err := sealPiece(rings, ivory, place)
		if err != nil {
			return nil, err
		}
		parts = append(parts, tusk)
		// Sparse, individually swept reference vibrissae; unlike a fur
		// shader these are modelled filaments, and density is illustrative.
		for row := 0; row < 4; row++ {
			for column := 0; column < 10; column++ {
				r := float64(row)
				c := float64(column)
				root := maths.V(.865+.002*c, .095-.018*r, side*(.076+.004*c))
				tip := root.Add(maths.V(.08+.006*c, .04-.025*r+.004*c, side*(.11+.011*c)))
				from, to := rig.HeadPoint(pose, root), rig.HeadPoint(pose, tip)
				axis := to.Sub(from)
				parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .0012, Thick: .0024, Height: axis.Len()},
					Material: bristle, Motion: thing.Still(maths.NewSubspace(from.Add(to).Scale(.5), maths.Upright(axis), 1).Within(place))})
			}
		}
	}
	return parts, nil
}
