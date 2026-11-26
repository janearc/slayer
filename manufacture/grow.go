package manufacture

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
)

// slot is which of a thing's matters a part is made of. body is what
// a thing is mainly made of: a stone's stone, a can's metal, a plant's
// wood. leaf is what a plant's leaves are.
type slot int

const (
	body slot = iota
	leaf
)

// form is what one part of a thing is shaped like: a rule for a
// surface, cut as finely as asked. a tube and a lump are forms.
type form interface {
	skin(fine cut) surface
}

// part is one piece of a made thing: a form, where it sits in the
// thing's space, how finely it is cut, and which matter it is of.
type part struct {
	shape    form
	space    maths.Subspace
	fine     cut
	of       slot
	terminal bool // foliage carried by a terminal branchlet
	rigTurn  bool // upper enclosure; metadata only, no change to the specimen
}

// skin is the part's surface in the thing's space.
func (one part) skin() surface {
	return one.shape.skin(one.fine).placed(one.space)
}

// Joint and sleeve geometry is shared with branching forms.
type joint = branching.Joint
type sleeve = branching.Sleeve

var errSleeve = branching.ErrSleeve

func newSleeve(joints []joint, fine cut, of slot) (sleeve, error) {
	return branching.NewSleeve(joints, fine.around, int(of))
}

// Limb state and placement belong to the shared branching engine.
type limb = branching.Limb

var errLimb = branching.ErrLimb
var newLimb = branching.NewLimb
var scaleOf = branching.ScaleOf

// along is the limb's parts on runs already worked out, for a limb of
// a size: every piece in every sleeve, piece by piece, and within a
// piece sleeve by sleeve, which is ninja's order.
func limbAlong(made limb, of size, runs []place) ([]part, error) {
	var parts []part
	for piece, run := range runs {
		for _, skin := range made.Skins {
			one, err := sleevePiece(skin, piece, of, run)
			if err != nil {
				return nil, err
			}
			parts = append(parts, one)
		}
	}
	return parts, nil
}

// piece is one piece of the sleeve on a limb of a size: a tube from
// one joint's radius to the next's, where the piece runs.
func sleevePiece(skin sleeve, index int, of size, run place) (part, error) {
	door := newTube
	if skin.Flat {
		door = newBlade
	}
	shape, err := door(run.Size.Length,
		measure.Metres(skin.Joints[index].Radius(inScene(of.Length), inScene(of.Girth))),
		measure.Metres(skin.Joints[index+1].Radius(inScene(of.Length), inScene(of.Girth))))
	if err != nil {
		return part{}, err
	}
	return part{shape: shape, space: run.Space, fine: cut{around: skin.Sides},
		of: slot(skin.Role), terminal: skin.Terminal}, nil
}

// Recursive growth and arrangements belong to the shared branching engine.
type arrangement = branching.Arrangement
type carried = branching.Carried
type growth = branching.Growth

var errGrowth = branching.ErrGrowth
var newGrowth = branching.NewGrowth

func grownParts(grown growth, on place, from seed) ([]part, error) {
	return branching.Parts(grown, on, from, limbAlong)
}

// sleeved is a round sleeve cut so many sides round.
func sleeved(joints []joint, around int, of slot) (sleeve, error) {
	fine, err := cutInto(around)
	if err != nil {
		return sleeve{}, err
	}
	return newSleeve(joints, fine, of)
}

// Stem construction is shared with other branching recipes.
var stemCarrying = branching.StemCarrying

// trunkCarrying is the shared whorled-stem constructor.
var trunkCarrying = branching.TrunkCarrying
