//go:build draw || shapecount

package shape

import "github.com/janearc/slayer/maths"

// every is the nearest face the ray meets, found by asking every face.
func every(built *Mesh, ray maths.Ray, near,
	far float64) (float64, int, bool) {
	asking, best, which := ready(ray), far, -unity
	for i := range built.tris {
		if distance, _, ok := asking.meet(&built.tris[i], near,
			best); ok {
			best, which = distance, i
		}
	}
	if which < nought {
		return nought, -unity, false
	}
	return best, which, true
}
