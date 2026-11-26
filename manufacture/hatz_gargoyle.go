package manufacture

import (
	"fmt"
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// hatzGargoyleKind is a small, static stone ornament made from the final
// Hatz feedback sculpture's true rest shape. It is an exploratory building
// detail, not an anatomically approved animal or a finished gargoyle design.
type hatzGargoyleKind struct{}

// The field extraction is costly and the resulting rest mesh is immutable;
// reuse it between roof berths, while keeping each placement separate. Current
// face budgets conservatively count every placed native part independently.
var hatzGargoyleRest = sync.OnceValues(func() ([]thing.Thing, error) {
	return hatzDensityStudy(0, false, maths.Local())
})

// made is the first study at rest, as a gargoyle. It refuses a seed that
// was not made.
func (hatzGargoyleKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, err := hatzGargoyleRest()
	if err != nil {
		return good{}, err
	}
	if len(parts) != 1 {
		return good{}, fmt.Errorf("hatz gargoyle: expected one sculpted surface, got %d", len(parts))
	}
	stone := &thing.Material{Name: "hatz gargoyle stone", Colour: maths.RGB(.39, .37, .33), Satin: .12}
	part := parts[0]
	part.Material = stone
	// A quarter-size copy of the animal study fits a roof berth; the statue
	// remains a wing-spread experiment and has no animated poses.
	const scale = .25
	base := -scale * part.Shape.Bounds().Min.Y
	part.Motion = thing.Still(maths.NewSubspace(maths.V(0, base, 0), maths.Identity, scale))
	return good{native: []thing.Thing{part}}, nil
}

// madeAt is made at any detail: the study has one representation.
func (row hatzGargoyleKind) madeAt(from seed, detail string) (good, error) {
	// The first study has one representation at each requested detail; the
	// shape and shade are deliberately not yet a finished ornament.
	return row.made(from)
}
