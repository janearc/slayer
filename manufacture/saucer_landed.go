package manufacture

import (
	_ "embed"
	"fmt"
	"maps"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The MCRN Larson landed is the same saucer with its gear down and its
// stairs out: a tripod of legs under the hull and one flight of stairs
// from the belly disc to the ground, the flight built by the stairs
// constructor. The saucer in flight has neither, and is not changed by
// any of this.

//go:embed saucer-landed.yaml
var saucerLandedRaw []byte

var saucerLanded, saucerLandedError = readGear(saucerLandedRaw, saucerShape())

// The landed saucer's slots beyond the saucer's own: its legs, and then
// one for each material its stairs name, in the order they name them.
const (
	saucerLeg   slot = saucerPorthole + spaceOne
	saucerStair      = saucerLeg + spaceOne
)

// saucerGear is the landed saucer's gear as its yaml books it, and what
// reading it worked out: the stairs' parts, the slot each of their
// materials takes, and the looks the gear wears.
type saucerGear struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Legs saucerLegs `yaml:"legs"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Foot float64 `yaml:"foot"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Stairs stairFlight `yaml:"stairs"`

	pieces []yardPart
	slots  map[string]slot
	worn   looks
}

// saucerLegs is the tripod: how many legs, how far round the first one
// stands, where each meets the hull and where its foot stands, its
// brace, its pad, and what it is made of.
type saucerLegs struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Count int `yaml:"count"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Turn float64 `yaml:"turn"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Attach float64 `yaml:"attach"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Foot float64 `yaml:"foot"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Radius float64 `yaml:"radius"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Brace saucerBrace `yaml:"brace"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Pad saucerPad `yaml:"pad"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Material string `yaml:"material"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Finish string `yaml:"finish"`
}

// saucerBrace is the thinner strut beside a leg: where it meets the flat
// of the underside, and its radius.
type saucerBrace struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Attach float64 `yaml:"attach"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Radius float64 `yaml:"radius"`
}

// saucerPad is the round foot a leg stands on.
type saucerPad struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Radius float64 `yaml:"radius"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Thickness float64 `yaml:"thickness"`
}

// readGear reads the landed saucer's gear and checks all of it against
// the hull it hangs from, so that lowering it afterwards cannot fail:
// the legs' numbers, the stairs' parts, and a look for every material
// and finish named. Nonsense is refused, by name where there is one.
func readGear(raw []byte, size saucerSize) (saucerGear, error) {
	var gear saucerGear
	if err := yamlfile.Read("saucer-landed.yaml", raw, &gear); err != nil {
		return saucerGear{}, err
	}
	if err := gear.Legs.sound(size); err != nil {
		return saucerGear{}, err
	}
	pieces, err := gear.Stairs.parts()
	if err != nil {
		return saucerGear{}, err
	}
	gear.pieces = pieces
	legs, err := lookOf("legs", gear.Legs.Material, gear.Legs.Finish)
	if err != nil {
		return saucerGear{}, err
	}
	gear.worn = looks{saucerLeg: legs}
	gear.slots = map[string]slot{}
	finished := map[string]string{}
	for _, each := range gear.Stairs.wears() {
		if before, seen := finished[each.material]; seen {
			if before != each.finish {
				return saucerGear{}, fmt.Errorf(
					"manufacture: the saucer's stairs finish %s both %s and %s",
					each.material, before, each.finish)
			}
			continue
		}
		look, err := lookOf("stairs "+each.family, each.material, each.finish)
		if err != nil {
			return saucerGear{}, err
		}
		of := saucerStair + slot(len(gear.slots))
		finished[each.material] = each.finish
		gear.slots[each.material] = of
		gear.worn[of] = look
	}
	return gear, nil
}

// lookOf is the look of one family of the gear: its material and its
// finish, by their names, made into a look by thing. An unknown name is
// refused, naming the family.
func lookOf(family, material, finish string) (*thing.Material, error) {
	return surfaceOf("the saucer's "+family, material, finish)
}

// sound is whether the legs can be built under a hull of this size: at
// least one, each of a real thickness, meeting the hull inside its rim
// and standing inside it too, the brace meeting the flat of the
// underside.
func (legs saucerLegs) sound(size saucerSize) error {
	if legs.Count < 1 {
		return fmt.Errorf("manufacture: a saucer on %d legs", legs.Count)
	}
	if !(legs.Radius > 0) || !(legs.Brace.Radius > 0) ||
		!(legs.Pad.Radius > 0) || !(legs.Pad.Thickness > 0) {
		return fmt.Errorf("manufacture: a saucer's leg of radius %v, "+
			"brace %v, pad %v by %v", legs.Radius, legs.Brace.Radius,
			legs.Pad.Radius, legs.Pad.Thickness)
	}
	if !(legs.Attach > 0 && legs.Attach < size.radius) ||
		!(legs.Foot > 0 && legs.Foot+legs.Pad.Radius < size.radius) {
		return fmt.Errorf("manufacture: a saucer's leg from %v m to %v m "+
			"out, under a hull %v m in radius", legs.Attach, legs.Foot,
			size.radius)
	}
	if !(legs.Brace.Attach > 0 && legs.Brace.Attach <= size.belly) {
		return fmt.Errorf("manufacture: a saucer's brace %v m out, off "+
			"the flat of the underside, which is %v m in radius",
			legs.Brace.Attach, size.belly)
	}
	return nil
}

// under is the height of the hull's underside at a distance from its
// axis: nought across the flat, then up the bowl to the rim.
func (size saucerSize) under(across float64) float64 {
	if across <= size.belly {
		return spaceZero
	}
	return size.bowl * (across - size.belly) / (size.radius - size.belly)
}

// ground is how far below the flat of the underside the ground lies: far
// enough that the top tread's top meets the underside of the belly disc.
// It is a height in the saucer's own frame, and so below nought.
func (gear saucerGear) ground(size saucerSize) float64 {
	treads := gear.Stairs.Treads
	climbed := treads.First.Y +
		float64(gear.Stairs.Steps-spaceOne)*gear.Stairs.Rise +
		treads.Thickness
	return -size.drop - climbed
}

// lower lowers the gear onto an assembly that already holds the saucer:
// each leg with its brace and its pad, then the stairs, moved from the
// flight's own frame to where its foot stands. It hands back the box the
// landed saucer fits in. A gear that was read cannot fail to lower.
func (gear saucerGear) lower(assembly *instrumentAssembly,
	size saucerSize) maths.Box {
	// heights are above the saucer's origin, wherever the size puts the
	// flat of the underside above it: landed, the ground comes out nought
	ground := size.raised(gear.ground(size))
	flat := maths.V(spaceZero, size.raised(spaceZero), spaceZero)
	legs := gear.Legs
	for each := range legs.Count {
		angle := legs.Turn*maths.Degree +
			fullTurn*float64(each)/float64(legs.Count)
		out := maths.V(math.Cos(angle), spaceZero, -math.Sin(angle))
		stands := out.Scale(legs.Foot).Add(maths.V(spaceZero, ground, spaceZero))
		ankle := stands.Add(maths.V(spaceZero, legs.Pad.Thickness, spaceZero))
		hip := out.Scale(legs.Attach).Add(flat).Add(
			maths.V(spaceZero, size.under(legs.Attach), spaceZero))
		assembly.rod(hip, ankle, legs.Radius, saucerLeg)
		assembly.rod(out.Scale(legs.Brace.Attach).Add(flat), ankle,
			legs.Brace.Radius, saucerLeg)
		assembly.rodAlong(stands,
			maths.V(spaceZero, legs.Pad.Thickness, spaceZero),
			legs.Pad.Radius, saucerLeg)
	}
	foot := maths.V(spaceZero, ground, gear.Foot)
	for _, piece := range gear.pieces {
		of := gear.slots[piece.Material]
		if piece.Box != nil {
			assembly.box(piece.Box.Size.X, piece.Box.Size.Y, piece.Box.Size.Z,
				piece.Box.At.Add(foot), of)
			continue
		}
		assembly.rod(piece.Rod.From.Add(foot), piece.Rod.To.Add(foot),
			piece.Rod.Radius, of)
	}
	maps.Copy(assembly.object.dressed, gear.worn)
	bounds := size.extent()
	bounds.Min.Y = ground
	return bounds
}

// landedGear is the landed saucer's gear, read once. A gear that will
// not read is the embedded file's fault, and nothing can be built.
func landedGear() saucerGear {
	if saucerLandedError != nil {
		panic(saucerLandedError)
	}
	return saucerLanded
}
