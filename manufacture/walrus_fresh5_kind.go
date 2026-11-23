package manufacture

import "github.com/janearc/slayer/scene/thing"

// walrusFresh5Kind exposes the walrus in one pose, or, when bones is set,
// its skeleton alone at rest, or, when volumes is set, the volumes over
// the skeleton before any detail. This kind does not animate.
type walrusFresh5Kind struct {
	bones, volumes bool
	pose           walrusPose
}

// made makes the walrus, or its bones, from a sown seed.
func (row walrusFresh5Kind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, err := row.parts()
	if err != nil {
		return good{}, err
	}
	if len(parts) == 0 {
		return good{}, errGood
	}
	extent := transformed(parts[0].Shape.Bounds(), parts[0].At(0))
	for _, part := range parts[1:] {
		extent = enclosing(extent, transformed(part.Shape.Bounds(), part.At(0)))
	}
	return good{native: parts, extent: &extent}, nil
}

// parts is the walrus in its pose, or its skeleton.
func (row walrusFresh5Kind) parts() ([]thing.Thing, error) {
	if row.bones {
		return WalrusFresh5Skeleton()
	}
	if row.volumes {
		return walrusFresh5Volumes()
	}
	return walrusFresh5(row.pose, 0, 0)
}
