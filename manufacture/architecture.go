package manufacture

// a tree is an architecture, a foliage and its numbers (doc/families.md). the
// architecture is how it branches, a growth rule written once; the foliage is
// what its last shoots bear; the numbers are its row's. a species of a known
// architecture and foliage is a row and no code.
//
// the branching is an architecture and not a form: a form is already the shape
// of one part . whorled is the only architecture so far, so there is no
// architecture interface yet; it comes with the second, 'ohi'a's, at families
// step 5.

// quality is how much of a tree is made, never which tree: one species
// and one seed are one tree at every quality, its limbs the same, cut
// more or less finely.
//
// This internal quality describes the legacy sleeve
// scaffold. Complexities names the public representations, including the
// recursive foliage described in doc/conifer-foliage.md.
type quality struct{ rung rung }

// rung is one of the qualities doc/families.md names.
type rung int

// quick is the legacy sleeve scaffold, face for face.
const quick rung = iota

// atQuick makes the legacy scaffold before a foliage representation is chosen.
var atQuick = quality{rung: quick}

// round is the shape of a sleeve before it is cut: its joints, foot to
// tip, and how many sides go round it.
type round struct {
	joints []joint
	sides  int
}

// girths is joints that each keep a share of the limb's girth.
func girths(shares ...float64) []joint {
	joints := make([]joint, len(shares))
	for at, share := range shares {
		joints[at] = joint{ofGirth: share}
	}
	return joints
}

// lengths is joints that are each a share of the limb's length out.
func lengths(shares ...float64) []joint {
	joints := make([]joint, len(shares))
	for at, share := range shares {
		joints[at] = joint{ofLength: share}
	}
	return joints
}

// tapering is a trunk's two joints: its whole girth at its foot, and a
// hundredth of its length at its tip.
func tapering() []joint {
	return []joint{{ofGirth: 1}, {ofLength: 0.01}}
}

// in is the sleeve of this shape, of a matter.
func (shape round) in(of slot) (sleeve, error) {
	return sleeved(shape.joints, shape.sides, of)
}

// foliage is what a tree's last shoots bear, at a quality.
type foliage interface {
	bears(at quality) (sleeve, error)
}

// sleeveFoliage is the foliage every tree here has worn since ninja's
// conifer: a sleeve of leaf round each shoot. Redwood and sequoia retain
// these as construction descriptors, then replace them with needled shoots.
type sleeveFoliage round

// bears is the sleeve of leaf a shoot wears.
func (shape sleeveFoliage) bears(quality) (sleeve, error) {
	return round(shape).in(leaf)
}

// limbing is how a limb of an architecture is made: how many pieces it
// is in, and how far each piece droops.
type limbing struct {
	pieces int
	droop  float64
}

// branchlets is what a whorled tree's branches carry, when they carry
// any: shoots on every piece of a branch, so many to a piece, each of
// its own pieces and droop, set by a habit (norwayShoots' half turn is
// what makes them two rows), and in a foliage of their own.
type branchlets struct {
	per   span
	shoot limbing
	set   habit
	wear  foliage
}

// whorled is a trunk with tiers of branches at its nodes: halle and
// oldeman's model of the spruces, simplified. its trunk is one piece;
// each branch is in bark and clad in a foliage, and every piece of it
// carries branchlets when the tree has them.
type whorled struct {
	trunk, bark round
	branch      limbing
	clad        foliage
	shoots      *branchlets
}

// grown is a whorled tree: so many whorls, so many branches to its
// lowest, set by a habit, and so many branchlets to a piece of a branch
// when it has branchlets, at a quality.
func (tree whorled) grown(count, lowest, sprays int, set habit,
	at quality,
) (growth, error) {
	trunk, err := tree.trunk.in(body)
	if err != nil {
		return growth{}, err
	}
	bark, err := tree.bark.in(body)
	if err != nil {
		return growth{}, err
	}
	clad, err := tree.clad.bears(at)
	if err != nil {
		return growth{}, err
	}
	branch, err := tree.branching(bark, clad, sprays, at)
	if err != nil {
		return growth{}, err
	}
	return trunkCarrying(trunk, count, lowest, set, branch)
}

// branching is one branch of the tree: in its bark and clad, carrying
// its branchlets when it has them.
func (tree whorled) branching(bark, clad sleeve, sprays int,
	at quality,
) (growth, error) {
	if tree.shoots == nil {
		bough, err := newLimb(tree.branch.pieces, 1, tree.branch.droop,
			bark, clad)
		if err != nil {
			return growth{}, err
		}
		return newGrowth(bough)
	}
	shoots, err := newWhorls(sprays, 1, tree.shoots.set)
	if err != nil {
		return growth{}, err
	}
	worn, err := tree.shoots.wear.bears(at)
	if err != nil {
		return growth{}, err
	}
	worn.terminal = true
	return stemCarrying(tree.branch.pieces, tree.branch.droop,
		[]sleeve{bark, clad}, shoots, tree.shoots.shoot.pieces,
		tree.shoots.shoot.droop, []sleeve{worn})
}
