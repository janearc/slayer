package manufacture

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/scene/thing"
)

// deformingMaker constructs a fresh local skin for the requested absolute
// scene time. It must not mutate geometry retained by any previous call.
type deformingMaker interface {
	madeAtTime(seed, string, time.Duration) (good, error)
}

// Deforms reports whether this kind has a time-dependent authored skin.
func Deforms(row Kind) bool {
	_, ok := row.how.(deformingMaker)
	return ok
}

// MakeAtTime constructs a deforming kind at an absolute scene time. A caller
// owns the returned geometry and must not alter any shared material or mesh.
// A static kind is refused rather than silently pretending it deforms.
func MakeAtTime(row Kind, from seed, detail string, at time.Duration) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	valid := false
	for _, name := range Complexities() {
		valid = valid || name == detail
	}
	if !valid {
		return good{}, fmt.Errorf("manufacture: complexity %q; choose %v", detail, Complexities())
	}
	maker, ok := row.how.(deformingMaker)
	if !ok {
		return good{}, fmt.Errorf("manufacture: %s has no deforming skin", row.name)
	}
	made, err := maker.madeAtTime(from, detail, at)
	if err != nil {
		return good{}, err
	}
	// The ordinary good.Things path takes native part placements at zero.
	// Snapshot each native part at the requested absolute scene time first;
	// copying the slice keeps any source assembly owned by the maker intact.
	made.native = append([]thing.Thing(nil), made.native...)
	for i := range made.native {
		made.native[i].Motion = thing.Still(made.native[i].At(at))
	}
	return made, nil
}
