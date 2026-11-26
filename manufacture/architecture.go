package manufacture

import "github.com/janearc/slayer/manufacture/plants/conifers"

type round = conifers.Round
type limbing = conifers.Limb
type branchlets = conifers.Branchlets
type whorled = conifers.Architecture

// girths is joints that each keep a share of the limb's girth.
func girths(shares ...float64) []joint {
	joints := make([]joint, len(shares))
	for at, share := range shares {
		joints[at] = joint{OfGirth: share}
	}
	return joints
}

// lengths is joints that are each a share of the limb's length out.
func lengths(shares ...float64) []joint {
	joints := make([]joint, len(shares))
	for at, share := range shares {
		joints[at] = joint{OfLength: share}
	}
	return joints
}
