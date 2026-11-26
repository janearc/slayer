package geometry

import "github.com/janearc/slayer/maths"

// SignedVolume6 is six times the signed volume enclosed by triangular faces.
// The sign identifies the winding of a closed surface.
func SignedVolume6(points []maths.Vec, faces [][3]int) float64 {
	volume := 0.0
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		volume += a.Dot(b.Cross(c))
	}
	return volume
}

// ReverseFaces reverses each triangle's winding in place.
func ReverseFaces(faces [][3]int) {
	for i := range faces {
		faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
	}
}
