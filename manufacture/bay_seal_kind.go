package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// baySealKind preserves the original bay scene's phase-zero bottling pose,
// materials, corneas and whiskers. It is a static specimen, not a clock.
type baySealKind struct{}

// made is the bay scene's seal as one posed specimen: skin, eyes and
// whiskers. It refuses a seed that was not made.
func (baySealKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	rig := NewSealRig()
	pose := SealRigPose{Yaw: .18, Pitch: -.20, HeadPitch: -.86, Fore: [2]float64{-.26, -.20}}
	skin, err := rig.Skin(pose)
	if err != nil {
		return good{}, err
	}
	cornea, err := SealCornea()
	if err != nil {
		return good{}, err
	}
	coat := &thing.Material{Name: "wet dark harbor seal", Colour: maths.RGB(.068, .063, .055), Texture: baySealWetPaint{spots: HarborSealCoat(407)}, Gloss: .42, Satin: .33, Index: 1.43}
	eyeLook := &thing.Material{Name: "harbor seal eye", Colour: maths.RGB(.010, .008, .007), Gloss: .41, Satin: .28}
	whiskerLook := &thing.Material{Name: "wet muzzle whiskers", Colour: maths.RGB(.44, .37, .28), Gloss: .21, Satin: .4}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(maths.Local())}}
	for _, side := range [...]float64{-1, 1} {
		rest := maths.V(.730, .110, side*.064)
		point := rig.HeadPoint(pose, rest)
		look := rig.HeadPoint(pose, rest.Add(maths.V(.018, 0, side*.006)))
		up := rig.HeadPoint(pose, rest.Add(maths.V(0, .010, 0))).Sub(point)
		eye := maths.NewSubspace(point, maths.LookAtUp(point, look, up), 1)
		parts = append(parts, thing.Thing{Shape: cornea, Material: eyeLook, Motion: thing.Still(eye)})
		for whisker := range 5 {
			fan := float64(whisker)
			root := maths.V(.775-.006*fan, .037+.007*fan, side*(.072+.002*fan))
			previous := rig.HeadPoint(pose, root)
			for segment := 1; segment <= 3; segment++ {
				share := float64(segment) / 3
				tip := root.Add(maths.V((.022-.011*fan)*share, (.025-.011*fan)*share-.030*share*share, side*(.080+.008*fan)*share))
				at := rig.HeadPoint(pose, tip)
				middle := previous.Add(at).Scale(.5)
				turn := maths.Upright(at.Sub(previous))
				rod := maths.NewSubspace(middle, turn, 1)
				parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .00032, Thick: .00064, Height: at.Sub(previous).Len()}, Material: whiskerLook, Motion: thing.Still(rod)})
				previous = at
			}
		}
	}
	// Conservative static extent includes the whiskers and rotated corneas.
	box := maths.Box{Min: maths.V(-1.3, -.8, -.8), Max: maths.V(1.3, .8, .8)}
	return good{native: parts, extent: &box}, nil
}

type baySealWetPaint struct{ spots texture.Texture }

// At is the wet coat's colour at a point, with no footprint.
func (coat baySealWetPaint) At(point maths.Vec) maths.Light { return coat.Within(point, 0) }

// Within is the wet coat's colour at a point, over a footprint in metres:
// spots on a dark wet base, and the nose.
func (coat baySealWetPaint) Within(point maths.Vec, footprint float64) maths.Light {
	spots := texture.Sample(coat.spots, point, footprint)
	base := maths.RGB(.021, .019, .016)
	wet := base.Add(spots.Scale(.23))
	noseWidth := .028 + .024*maths.Smooth(.023, .077, point.Y)
	nose := 1 - maths.Smooth(.76, 1.02, math.Hypot((point.Y-.059)/.038, point.Z/noseWidth))
	nose *= maths.Smooth(.779, .804, point.X)
	nostril := 1 - maths.Smooth(.52, 1.02, math.Hypot((point.Y-.067+.33*(math.Abs(point.Z)-.022))/.007, (math.Abs(point.Z)-.022)/.013))
	noseColour := maths.RGB(.014, .012, .011).Scale(1 - .67*nostril)
	return wet.Scale(1 - nose).Add(noseColour.Scale(nose))
}
