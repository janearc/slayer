package rings

// The numbers the package picked. Each is picked by the quasar slice,
// 2026-11-12, open to cutting.
const (
	// steps is how many steps a ray may take before it is given up as
	// captured: the tracer's own budget (bends in trace/field.go).
	steps = 40000

	// core is the radius, in masses, inside which a ray has come from
	// the middle: the tracer's own (core in trace/field.go).
	core = 0.2

	// farOut is how far past the eye, as a share of the eye's own
	// radius, a ray going outward is let go: past the eye and the disc,
	// with nothing left to cross.
	farOut = 2.0

	// halvings is how many times an edge of a band, or the edge of the
	// shadow, is halved toward: past fifty, float64 has nothing left.
	halvings = 60

	// scanRays is how many rays a band's search sends out, spaced
	// evenly in the logarithm of the angle past the shadow's edge.
	scanRays = 1200

	// nearest and widest are the least and greatest angle past the
	// shadow's edge, in radians, a band's search looks at: a later
	// image than the third lies inside the nearest.
	nearest = 1e-11
	widest  = 0.3
)
