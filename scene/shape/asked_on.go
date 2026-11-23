//go:build shapecount

package shape

// facesAskedAbout is how many faces walks have asked about since a test
// last set it to nought. It is not safe for walks on several goroutines.
var facesAskedAbout int

// askedAbout counts one face asked about, in the shapecount build.
func askedAbout() { facesAskedAbout++ }
