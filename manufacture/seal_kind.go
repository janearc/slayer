package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/pinnipeds"
	"github.com/janearc/slayer/scene/thing"
)

const (
	sealLoft = iota
	sealDetail
	sealBones
	sealBay
	sealWalrus
)

// sealKind dispatches the four distinct family constructors from one Kind adapter.
type sealKind struct {
	form    int
	loft    *pinnipeds.Loft
	profile *pinnipedProfile
	look    pinnipeds.SealLook
	walrus  *pinnipeds.Walrus
}

func (row sealKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	var result pinnipeds.Result
	var err error
	switch row.form {
	case sealLoft:
		result, err = row.loft.Build(from.number)
	case sealDetail:
		result, err = pinnipeds.Detailed(row.look, from.number)
	case sealBones:
		result, err = pinnipeds.Bones(*row.profile)
	case sealBay:
		result, err = pinnipeds.Bay()
	case sealWalrus:
		var parts []thing.Thing
		parts, err = row.walrus.Build()
		if err != nil {
			return good{}, err
		}
		if len(parts) == 0 {
			return good{}, errGood
		}
		bounds := transformed(parts[0].Shape.Bounds(), parts[0].At(0))
		for _, part := range parts[1:] {
			bounds = enclosing(bounds, transformed(part.Shape.Bounds(), part.At(0)))
		}
		result = pinnipeds.Result{Parts: parts, Bounds: bounds}
	}
	if err != nil {
		return good{}, err
	}
	return good{native: result.Parts, extent: &result.Bounds}, nil
}
