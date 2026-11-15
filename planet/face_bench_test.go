package planet

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func BenchmarkFaceWithin(b *testing.B) {
	body := bareBody(b, 1, nil)
	up := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), up, up.Scale(1e9), even(), 2)
	b.ReportAllocs()
	b.ResetTimer()
	var sink maths.Light
	for i := 0; i < b.N; i++ {
		f := float64(i%50) / 100
		sink = face.Within(maths.V(f, f/2, 1).Unit().Scale(6e7), 1e5)
	}
	_ = sink
}

// BenchmarkFaceWithinOld is the old Within, for the same minute.
func BenchmarkFaceWithinOld(b *testing.B) {
	body := bareBody(b, 1, nil)
	up := maths.V(0, 0, 1)
	face := NewFace(body, maths.V(0, 1, 0), up, up.Scale(1e9), even(), 2)
	b.ReportAllocs()
	b.ResetTimer()
	var sink maths.Light
	for i := 0; i < b.N; i++ {
		f := float64(i%50) / 100
		sink = oldWithin(face, maths.V(f, f/2, 1).Unit().Scale(6e7), 1e5)
	}
	_ = sink
}

// giantProbes are points of the giant's face that send light to the eye,
// from a seed, so each timed call does the whole of its work.
func giantProbes(b testing.TB, face *Face) []probe {
	b.Helper()
	var lit []probe
	for _, each := range probesOf(face, 5, 4000) {
		if got := face.Within(each.at, each.footprint); got.R+got.G+got.B > 0 {
			lit = append(lit, each)
		}
	}
	return lit
}

// BenchmarkFaceWithinGiant is Within on a face as full as Melancholia's: its
// zones, storms, churn, cap and clearing, cooked air, heat and airglow, over a
// flow, in the eye's band count.
func BenchmarkFaceWithinGiant(b *testing.B) {
	star := maths.V(0.8, 0.1, 0.5)
	face := NewFace(stormyBody(b, true), maths.V(0, 1, 0.4), star,
		maths.V(-1.4e8, 3e7, 5e7), even(), 0.9)
	probes := giantProbes(b, face)
	b.ReportAllocs()
	b.ResetTimer()
	var sink maths.Light
	for i := 0; i < b.N; i++ {
		each := probes[i%len(probes)]
		sink = face.Within(each.at, each.footprint)
	}
	_ = sink
}

// BenchmarkFaceWithinGiantOld is the same points through the reference.
func BenchmarkFaceWithinGiantOld(b *testing.B) {
	star := maths.V(0.8, 0.1, 0.5)
	face := NewFace(stormyBody(b, true), maths.V(0, 1, 0.4), star,
		maths.V(-1.4e8, 3e7, 5e7), even(), 0.9)
	probes := giantProbes(b, face)
	b.ReportAllocs()
	b.ResetTimer()
	var sink maths.Light
	for i := 0; i < b.N; i++ {
		each := probes[i%len(probes)]
		sink = refWithin(face, each.at, each.footprint)
	}
	_ = sink
}

// BenchmarkFaceWithinGiantPaired asks the face and the reference the same
// point one after the other, timing each, so a busy machine slows both alike:
// it reports the face's and the reference's nanoseconds a call, and the ratio.
func BenchmarkFaceWithinGiantPaired(b *testing.B) {
	star := maths.V(0.8, 0.1, 0.5)
	face := NewFace(stormyBody(b, true), maths.V(0, 1, 0.4), star,
		maths.V(-1.4e8, 3e7, 5e7), even(), 0.9)
	probes := giantProbes(b, face)
	var now, was time.Duration
	var sink maths.Light
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		each := probes[i%len(probes)]
		start := time.Now()
		sink = face.Within(each.at, each.footprint)
		mid := time.Now()
		sink = refWithin(face, each.at, each.footprint)
		end := time.Now()
		now, was = now+mid.Sub(start), was+end.Sub(mid)
	}
	_ = sink
	b.ReportMetric(float64(now)/float64(b.N), "new-ns/op")
	b.ReportMetric(float64(was)/float64(b.N), "ref-ns/op")
	b.ReportMetric(float64(was)/float64(now), "ref/new")
}
