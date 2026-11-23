//go:build !shapecount

package shape

// askedAbout is called each time a walk of the tree asks about a face. In
// an ordinary build it is empty, and the compiler removes the call; the
// shapecount build counts the calls, for the tests of the tree's walk.
func askedAbout() {}
