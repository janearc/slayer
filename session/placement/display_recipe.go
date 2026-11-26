package placement

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

// errBurning is what a burning placement or a fire recipe is refused with.
var errBurning = errors.New("working display does not support burning placements")

// thingMesh is the mesh of one placement, from the cache when everything
// that decides it is already held there and made, then kept, when not. The
// caller holds the cache's lock.
func (cache *DisplayCache) thingMesh(read set.Set, each set.Thing) (*DisplayMesh, error) {
	row, digest, err := displayGate(read, each)
	if err != nil {
		return nil, err
	}
	key := displayKey{kind: each.Kind(), detail: each.Complexity(), recipe: digest, sceneSeed: read.Seed(), objectSeed: each.Seed()}
	if skin := cache.meshes[key]; skin != nil {
		return skin, nil
	}
	skin, err := displayManufactured(read.Seed(), row, each)
	if err != nil {
		return nil, err
	}
	cache.meshes[key] = skin
	return skin, nil
}

// displayGate is every check a placement must pass before the display makes
// its mesh, and costs no mesh: it is not burning and its construction is
// found. Scene and CheckDisplay both ask it, so they cannot disagree.
func displayGate(read set.Set, each set.Thing) (manufacture.Kind, string, error) {
	if each.Alight() {
		return manufacture.Kind{}, "", errBurning
	}
	row, digest, err := displayRow(read, each)
	if err != nil {
		return manufacture.Kind{}, "", err
	}
	if _, has := each.Scale(); has && manufacture.Deforms(row) {
		return manufacture.Kind{}, "", fmt.Errorf("scale is not yet supported for deforming geometry")
	}
	return row, digest, nil
}

// displayRow is the construction a placement is made from, with the digest
// of the recipe it came from. A placement with no recipe is made from its
// named kind and has no digest. A recipe is looked up in the set and built
// as staging builds it, so the display and the render make the same thing.
func displayRow(read set.Set, each set.Thing) (manufacture.Kind, string, error) {
	if each.Recipe() == "" {
		row, err := manufacture.KindNamed(each.Kind())
		return row, "", err
	}
	recipe, found := read.Recipe(each.Recipe())
	if !found {
		return manufacture.Kind{}, "", fmt.Errorf("unknown construction recipe %q", each.Recipe())
	}
	if _, burns := recipe.Fire(); burns {
		return manufacture.Kind{}, "", errBurning
	}
	row, err := recipe.Construction()
	if err != nil {
		return manufacture.Kind{}, "", err
	}
	return *row, recipe.Digest(), nil
}
