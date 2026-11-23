package shape

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The file the mesh tree benchmarks read, and the places in a face that
// the tests ask for by name.
const (
	meshtreeStreamsFile = "testdata/meshtree-streams.csv"
	meshtreeFirst       = 0 // defined: the first corner of a face
	meshtreeSecond      = 1 // defined: the second corner of a face
	meshtreeThird       = 2 // defined: the third corner of a face
)

// meshtreeRing is the place in a ball's corners of the one at slice
// around of ring i, the rings counted from the one under the north pole.
func meshtreeRing(slices, i, around int) int {
	return unity + (i-unity)*slices + around%slices
}

// lumpy is a ball with bumps on it, made of rings of slices: two faces
// for every slice of every ring, and one at each pole.
func lumpy(tester testing.TB, slices,
	rings int) ([]maths.Vec, []triangleCorners) {
	tester.Helper()
	height := trianglePicked(tester, "meshtree-bump-height")
	latitude := trianglePicked(tester, "meshtree-bump-latitude")
	longitude := trianglePicked(tester, "meshtree-bump-longitude")
	vs := []maths.Vec{maths.V(nought, unity, nought)}
	for i := unity; i < rings; i++ {
		th := math.Pi * float64(i) / float64(rings)
		for around := range slices {
			ph := pair * math.Pi * float64(around) / float64(slices)
			radius := unity + height*math.Sin(latitude*th)*
				math.Cos(longitude*ph)
			x := radius * math.Sin(th) * math.Cos(ph)
			y := radius * math.Cos(th)
			z := radius * math.Sin(th) * math.Sin(ph)
			vs = append(vs, maths.V(x, y, z))
		}
	}
	vs = append(vs, maths.V(nought, -unity, nought))
	south := len(vs) - unity
	var fs []triangleCorners
	for around := range slices {
		north := triangleCorners{nought,
			meshtreeRing(slices, unity, around+unity),
			meshtreeRing(slices, unity, around)}
		last := rings - unity
		below := triangleCorners{south, meshtreeRing(slices, last, around),
			meshtreeRing(slices, last, around+unity)}
		fs = append(fs, north, below)
		for i := unity; i < rings-unity; i++ {
			fs = append(fs, meshtreeBand(slices, i, around)...)
		}
	}
	return vs, fs
}

// meshtreeBand is the two faces of the slice around of the ring i,
// between it and the ring below.
func meshtreeBand(slices, i, around int) []triangleCorners {
	upperLeft := meshtreeRing(slices, i, around)
	upperRight := meshtreeRing(slices, i, around+unity)
	lowerRight := meshtreeRing(slices, i+unity, around+unity)
	lowerLeft := meshtreeRing(slices, i+unity, around)
	return []triangleCorners{{upperLeft, upperRight, lowerRight},
		{upperLeft, lowerRight, lowerLeft}}
}

// soup is faces of every size from a hundredth to four tenths, strewn
// through the cube from -1 to 1, overlapping as they fall.
func soup(rng *rand.Rand, faces int) ([]maths.Vec, []triangleCorners) {
	smallest := triangleChosen("meshtree-soup-smallest")
	spread := triangleChosen("meshtree-soup-spread")
	var vs []maths.Vec
	var fs []triangleCorners
	for range faces {
		mid := within(rng, unity)
		size := smallest * math.Pow(spread, rng.Float64())
		for range cornersOfAFace {
			offset := anyway(rng).Scale(size)
			vs = append(vs, mid.Add(offset))
		}
		fs = append(fs, triangleCorners{len(vs) - cornersOfAFace,
			len(vs) - pair, len(vs) - unity})
	}
	return vs, fs
}

// benchRays are rays from all round a ball, aimed into it.
func benchRays(tester testing.TB) []maths.Ray {
	tester.Helper()
	rng := triangleStream(tester, meshtreeStreamsFile, "bench")
	count := int(trianglePicked(tester, "meshtree-bench-rays"))
	rays := make([]maths.Ray, count)
	standoff := trianglePicked(tester, "meshtree-bench-standoff")
	aim := trianglePicked(tester, "meshtree-bench-aim")
	for i := range rays {
		rays[i] = aimed(anyway(rng).Scale(standoff), inBall(rng, aim))
	}
	return rays
}

// meshtreeBall is the lumpy ball the benchmarks are run against.
func meshtreeBall(tester testing.TB) ([]maths.Vec, []triangleCorners) {
	tester.Helper()
	slices := int(trianglePicked(tester, "meshtree-lumpy-slices"))
	rings := int(trianglePicked(tester, "meshtree-lumpy-rings"))
	return lumpy(tester, slices, rings)
}

// BenchmarkMeshHit is rays a second against a hundred thousand faces.
func BenchmarkMeshHit(bench *testing.B) {
	vs, fs := meshtreeBall(bench)
	built := made(bench, vs, fs, nil, false)
	rays, count := benchRays(bench), nought
	for bench.Loop() {
		built.Hit(rays[count%len(rays)], nought, forever)
		count++
	}
	bench.ReportMetric(float64(count)/bench.Elapsed().Seconds(), "rays/s")
	faces := float64(len(fs))
	bench.ReportMetric(faces, "faces")
}

// BenchmarkMeshNormal is normals a second, where the rays met the faces.
func BenchmarkMeshNormal(bench *testing.B) {
	vs, fs := meshtreeBall(bench)
	built := made(bench, vs, fs, nil, false)
	var at []maths.Vec
	for _, ray := range benchRays(bench) {
		if distance, ok := built.Hit(ray, nought, forever); ok {
			at = append(at, ray.Point(distance))
		}
	}
	count := nought
	for bench.Loop() {
		built.Normal(at[count%len(at)])
		count++
	}
	bench.ReportMetric(float64(count)/bench.Elapsed().Seconds(),
		"normals/s")
}

// BenchmarkNewMesh is checking and building a hundred thousand faces.
func BenchmarkNewMesh(bench *testing.B) {
	vs, fs := meshtreeBall(bench)
	for bench.Loop() {
		made(bench, vs, fs, nil, false)
	}
}
