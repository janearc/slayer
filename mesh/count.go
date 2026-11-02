package mesh

// the kinds of work a walk of the tree does, for a test to count.
const (
	entered = iota // a node looked in, not passed over
	tried          // a face asked
)

// counted, when a test sets it, is told of each piece of work a walk
// does. nothing sets it outside tests: on the real path it is one load
// and one compare of a nil, per node and per face.
//
// a test that sets it must not run in parallel with any test that
// walks a mesh: the variable is the package's, and shared.
var counted func(kind int)

// count tells counted of one piece of work, if it is listening.
func count(kind int) {
	if counted != nil {
		counted(kind)
	}
}
