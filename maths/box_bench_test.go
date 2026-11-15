package maths

import "testing"

// BenchmarkBoxHit is one box asked by a ray that crosses it, which is
// the question a tree walk asks of every branch.
func BenchmarkBoxHit(b *testing.B) {
	box := Box{Min: V(-1, -2, -3), Max: V(2, 1, 4)}
	ray := Ray{At: V(-5, -1, -8), Dir: V(0.4, 0.3, 0.8).Unit()}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		box.Hit(ray, 0, 1e9)
	}
}

// BenchmarkBoxMeets is BenchmarkBoxHit's box and ray asked by Meets, with the
// reciprocal taken once outside the loop, as a tree walk takes it.
func BenchmarkBoxMeets(b *testing.B) {
	box := Box{Min: V(-1, -2, -3), Max: V(2, 1, 4)}
	ray := Ray{At: V(-5, -1, -8), Dir: V(0.4, 0.3, 0.8).Unit()}
	inverse := ray.Dir.Inverse()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		box.Meets(ray, inverse, 0, 1e9)
	}
}
