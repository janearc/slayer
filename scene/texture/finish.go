package texture

import "github.com/janearc/slayer/maths"

// Finish describes visible surface relief and sheen independently of paint.
// FinishAt returns the gradient of a small height field in chart units and
// a Satin value in [0, 1]. A mesh with rest points uses its rest chart.
// Wider footprints should suppress slopes that a pixel cannot resolve.
type Finish interface {
	FinishAt(point maths.Vec, footprint float64) (slope maths.Vec, satin float64)
}
