package hillside

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
)

// construction runs only on a stock miss. Static recipes retain their meshes.
func (stage *staging) construction(kind, recipe string,
) (*manufacture.Kind, error) {
	if recipe == "" {
		row, err := manufacture.KindNamed(kind)
		if err != nil {
			return nil, err
		}
		return &row, nil
	}
	row, found := stage.recipes.Recipe(recipe)
	if !found {
		return nil, fmt.Errorf("unknown construction recipe %q", recipe)
	}
	return row.Construction()
}

// constructed returns no stock on a construction refusal.
func (stage *staging) constructed(kind, recipe, detail string,
	seed uint64,
) (*kept, error) {
	return stage.constructedTaller(kind, recipe, detail, seed, 1, false)
}

// constructedTaller is constructed grown so many times taller, and its
// branches branched when asked: a whorled tree grown, not magnified, for
// improv/moonlake's -close and -giants. once as tall and unbranched is
// the kind as it is; anything but a whorled tree is refused.
func (stage *staging) constructedTaller(kind, recipe, detail string,
	seed uint64, times float64, branched bool,
) (*kept, error) {
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return nil, err
	}
	if times != 1 {
		grown, err := manufacture.Taller(*row, times)
		if err != nil {
			return nil, err
		}
		row = &grown
	}
	if branched {
		full, err := manufacture.Branched(*row)
		if err != nil {
			return nil, err
		}
		row = &full
	}
	one, err := manufacturedKind(stage.given, *row, int(seed), detail)
	if err != nil {
		return nil, err
	}
	return &one, nil
}
