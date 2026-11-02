package geom

import "math"

type Vec3 [3]float64

type Triangle struct {
	V        [3]Vec3
	Material int
	Clear    bool
}

// Add is the sum of v and w, part by part.
func (v Vec3) Add(w Vec3) Vec3 {
	return Vec3{v[0] + w[0], v[1] + w[1], v[2] + w[2]}
}

// Sub is v less w, part by part.
func (v Vec3) Sub(w Vec3) Vec3 {
	return Vec3{v[0] - w[0], v[1] - w[1], v[2] - w[2]}
}

// Scale is v with every part multiplied by s.
func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{v[0] * s, v[1] * s, v[2] * s}
}

// Dot is the dot product of v and w.
func (v Vec3) Dot(w Vec3) float64 {
	return v[0]*w[0] + v[1]*w[1] + v[2]*w[2]
}

// Cross is the cross product of v and w.
func (v Vec3) Cross(w Vec3) Vec3 {
	return Vec3{
		v[1]*w[2] - v[2]*w[1],
		v[2]*w[0] - v[0]*w[2],
		v[0]*w[1] - v[1]*w[0],
	}
}

// Length is the distance from the origin to v.
func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

// Normalize is v scaled to length one; the zero vector stays zero.
func (v Vec3) Normalize() Vec3 {
	len := v.Length()
	if len == 0 {
		return Vec3{}
	}
	return v.Scale(1 / len)
}

type Box struct {
	Min, Max Vec3
}

// EmptyBox is a box that holds nothing, so the first point fitted sets both
// corners.
func EmptyBox() Box {
	inf := math.Inf(1)
	return Box{
		Min: Vec3{inf, inf, inf},
		Max: Vec3{-inf, -inf, -inf},
	}
}

// Fit is b grown, if it must, to hold p.
func (b Box) Fit(p Vec3) Box {
	return Box{
		Min: Vec3{
			math.Min(b.Min[0], p[0]),
			math.Min(b.Min[1], p[1]),
			math.Min(b.Min[2], p[2]),
		},
		Max: Vec3{
			math.Max(b.Max[0], p[0]),
			math.Max(b.Max[1], p[1]),
			math.Max(b.Max[2], p[2]),
		},
	}
}

// Size is the vector from the low corner to the high corner.
func (b Box) Size() Vec3 {
	return b.Max.Sub(b.Min)
}

// BoundsOfTriangles is the smallest box that holds every vertex of triangles.
func BoundsOfTriangles(triangles []Triangle) Box {
	bounds := EmptyBox()
	for _, tri := range triangles {
		for _, v := range tri.V {
			bounds = bounds.Fit(v)
		}
	}
	return bounds
}

// Normal is the unit normal of the triangle v0, v1, v2, by the right-hand rule.
func Normal(v0, v1, v2 Vec3) Vec3 {
	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)
	return edge1.Cross(edge2).Normalize()
}
