package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/theropods"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// AllosaurusStudy makes the embedded walking reconstruction.
func AllosaurusStudy(seconds float64, at maths.Subspace) ([]thing.Thing, error) {
	return theropods.DefaultStudy(seconds, at)
}
