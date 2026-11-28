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

// retainedDeformingMaker prepares immutable rest data shared by all frames.
type retainedDeformingMaker interface {
	prepareDeformation(seed, string) (func(time.Duration) (good, error), error)
}

// Deforms reports whether this kind has a time-dependent authored skin.
func Deforms(row Kind) bool {
	_, direct := row.how.(deformingMaker)
	_, retained := row.how.(retainedDeformingMaker)
	return direct || retained
}

// MakeAtTime constructs a deforming kind at an absolute scene time. A caller
// owns the returned geometry and must not alter any shared material or mesh.
// A static kind is refused rather than silently pretending it deforms.
func MakeAtTime(row Kind, from seed, detail string, at time.Duration) (good, error) {
	sample, err := PrepareDeformation(row, from, detail)
	if err != nil {
		return good{}, err
	}
	return sample(at)
}

// PrepareDeformation makes a scene-time sampler. It retains rest data when
// the maker provides it. Each sample owns its frame geometry.
func PrepareDeformation(row Kind, from seed, detail string) (func(time.Duration) (good, error), error) {
	if !from.made() {
		return nil, errGood
	}
	valid := false
	for _, name := range Complexities() {
		valid = valid || name == detail
	}
	if !valid {
		return nil, fmt.Errorf("manufacture: complexity %q; choose %v", detail, Complexities())
	}
	var sample func(time.Duration) (good, error)
	switch maker := row.how.(type) {
	case retainedDeformingMaker:
		var err error
		sample, err = maker.prepareDeformation(from, detail)
		if err != nil {
			return nil, err
		}
	case deformingMaker:
		sample = func(at time.Duration) (good, error) {
			return maker.madeAtTime(from, detail, at)
		}
	default:
		return nil, fmt.Errorf("manufacture: %s has no deforming skin", row.name)
	}
	return func(at time.Duration) (good, error) {
		made, err := sample(at)
		if err != nil {
			return good{}, err
		}
		return snapshotDeformation(made, at), nil
	}, nil
}

func snapshotDeformation(made good, at time.Duration) good {
	// The ordinary good.Things path takes native part placements at zero.
	// Snapshot each native part at the requested absolute scene time first;
	// copying the slice keeps any source assembly owned by the maker intact.
	made.native = append([]thing.Thing(nil), made.native...)
	for i := range made.native {
		made.native[i].Motion = thing.Still(made.native[i].At(at))
	}
	return made
}
