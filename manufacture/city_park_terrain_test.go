package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func testTerrainPark() CityPark {
	return CityPark{Name: "test park", At: maths.V(120, 0, -80), Width: measure.Metres(640), Depth: measure.Metres(2304)}
}

func TestParkTerrainHeightBoundsEdgesAndRelief(t *testing.T) {
	p := testTerrainPark()
	for _, at := range []maths.Vec{maths.V(p.At.X-p.Width.InMetres()/2, 0, p.At.Z), maths.V(p.At.X+p.Width.InMetres()/2, 0, p.At.Z), maths.V(p.At.X, 0, p.At.Z+p.Depth.InMetres()/2)} {
		if got := CityParkTerrainHeight(p, at); got != parkTerrainBase {
			t.Fatalf("boundary height %g", got)
		}
	}
	outside := maths.V(p.At.X+p.Width.InMetres(), 0, p.At.Z)
	if got := CityParkTerrainHeight(p, outside); got != 0 {
		t.Fatalf("outside park height %g", got)
	}
	maxHeight := 0.0
	for i := -15; i <= 15; i++ {
		for j := -40; j <= 40; j++ {
			offset := maths.V(float64(i)*18+.37, 0, float64(j)*24+.63)
			at := p.At.Add(offset)
			h := CityParkTerrainHeight(p, at)
			if h < parkTerrainBase || h > parkTerrainBase+parkTerrainMaxHeight {
				t.Fatalf("height %g out of bounds", h)
			}
			maxHeight = math.Max(maxHeight, h)
		}
	}
	if maxHeight < 2 {
		t.Fatalf("park remains flat: maximum %g", maxHeight)
	}
}

func TestParkTerrainMatchesItsActualTriangles(t *testing.T) {
	p := CityPark{Name: "small", At: maths.V(1, 0, 2), Width: measure.Metres(64), Depth: measure.Metres(80)}
	b := newCityBatch()
	parkTerrain(b, p)
	count := 0
	for _, role := range []slot{parkMeadow, parkWoodland, parkStone} {
		for _, tri := range b.pending[role] {
			centre := tri[0].Add(tri[1]).Add(tri[2]).Scale(1.0 / 3)
			if got := CityParkTerrainHeight(p, centre); math.Abs(got-centre.Y) > 1e-9 {
				t.Fatalf("placement %v height %g, triangle %g", centre, got, centre.Y)
			}
			if tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0])).Y <= 0 {
				t.Fatal("terrain triangle faces down")
			}
			count++
		}
	}
	if count != 2*16*20 {
		t.Fatalf("got %d faces, want %d", count, 2*16*20)
	}
}

func TestParkTerrainLeavesEveryPathAtGroundLevel(t *testing.T) {
	p := testTerrainPark()
	for _, path := range parkPaths(p) {
		for i := 1; i < len(path); i++ {
			for _, tval := range []float64{0, .25, .5, .75, 1} {
				centre := path[i-1].Scale(1 - tval).Add(path[i].Scale(tval))
				d := path[i].Sub(path[i-1])
				normal := maths.V(-d.Z, 0, d.X).Unit()
				for _, side := range []float64{-2, 0, 2} {
					at := centre.Add(normal.Scale(side))
					if got := CityParkTerrainHeight(p, at); math.Abs(got-parkTerrainBase) > 1e-10 {
						t.Fatalf("path point %v raised to %g", at, got)
					}
				}
			}
		}
	}
}

func TestParkTerrainKeepsPathGeometryAtConfiguredPedestrianScale(t *testing.T) {
	p := CityPark{Name: "small", At: maths.V(1, 0, 2), Width: measure.Metres(64), Depth: measure.Metres(80)}
	city := ClusteredCityLayout{parks: []CityPark{p}, parkRelief: true, parkPathWidth: measure.Metres(.016)}
	b := newCityBatch()
	city.landscape(b)
	paths := b.pending[cityPavement]
	if len(paths) == 0 {
		t.Fatal("park paths disappeared")
	}
	first := paths[0]
	shorter := math.Min(first[2].Sub(first[1]).Len(), first[0].Sub(first[2]).Len())
	width := math.Min(first[1].Sub(first[0]).Len(), shorter)
	if math.Abs(width-.016) > 1e-9 {
		t.Fatalf("path width %g, want 0.016 factory m", width)
	}
	if math.Abs(first[0].Y-.001) > 1e-12 {
		t.Fatalf("path elevation %g", first[0].Y)
	}
	if CityParkTerrainHeight(p, first[0]) >= first[0].Y {
		t.Fatal("path buried in terrain")
	}
}

func TestParkTerrainDefaultParksAreNativeBatchedGeometry(t *testing.T) {
	spec := DefaultClusteredCity()
	city := ClusteredCityLayout{parks: spec.Parks, parkRelief: true, parkPathWidth: measure.Metres(.016)}
	batch := newCityBatch()
	city.landscape(batch)
	made, err := batch.finish()
	if err != nil {
		t.Fatal(err)
	}
	faces := 0
	for _, role := range []slot{parkMeadow, parkWoodland, parkStone, parkService, parkPaving} {
		if group := made.groups[role]; group != nil {
			faces += group.FaceCount()
		}
	}
	if faces < 200000 {
		t.Fatalf("park terrain has %d faces, expected full dimensional parks", faces)
	}
	serviceFaces := 0
	for _, role := range []slot{parkService, parkPaving} {
		if group := made.groups[role]; group != nil {
			serviceFaces += group.FaceCount()
		}
	}
	t.Logf("default parks without trees: %d of %d park terrain triangles use service finishes (%.2f%%; triangle count proxy, not citywide union)", serviceFaces, faces, 100*float64(serviceFaces)/float64(faces))
	if paths := made.groups[cityPavement]; paths == nil || paths.FaceCount() < 500 {
		t.Fatal("pedestrian paths lost from the native batches")
	}
}

func TestParkTerrainLongPathPointMatchesOriginalSegments(t *testing.T) {
	p := testTerrainPark()
	path := parkPaths(p)[1]
	for _, fraction := range []float64{0, .1, .5, .5003, .8, 1} {
		got := CityParkLongPathAt(p, fraction)
		index := min(int(fraction*64), 63)
		blend := fraction*64 - float64(index)
		want := path[index].Scale(1 - blend).Add(path[index+1].Scale(blend))
		want.Y = .0011
		if !got.Near(want, 1e-12) {
			t.Fatalf("path position %v, want %v", got, want)
		}
		if h := CityParkTerrainHeight(p, got); h >= got.Y {
			t.Fatalf("path buried at %v, ground %g", got, h)
		}
	}
}

func TestParkTerrainParksDoNotRepeatTheSameHills(t *testing.T) {
	a := testTerrainPark()
	b := a
	b.At = b.At.Add(maths.V(3000, 0, 1800))
	different := false
	for _, offset := range []maths.Vec{maths.V(100, 0, 300), maths.V(-170, 0, 400), maths.V(130, 0, -400)} {
		first, second := CityParkTerrainHeight(a, a.At.Add(offset)), CityParkTerrainHeight(b, b.At.Add(offset))
		if math.Abs(first-second) > 1e-6 {
			different = true
		}
	}
	if !different {
		t.Fatal("separate parks repeated identical relief")
	}
}

func TestParkTerrainUrbanSurfacePreservesGrovePathsTreesAndRelief(t *testing.T) {
	park := CityPark{Name: "small", At: maths.V(0, 0, 0), Width: measure.Metres(128), Depth: measure.Metres(160)}
	trees := []CityTree{{Bounds: maths.Box{Min: maths.V(39, 0, 39), Max: maths.V(41, 10, 41)}}}
	buckets := parkInfillTreeBuckets(trees)
	paths := parkPaths(park)
	nx, nz, sx, sz := parkTerrainGrid(park)
	startX, startZ := -park.Width.InMetres()/2, -park.Depth.InMetres()/2
	service, living := 0, 0
	for j := 0; j < nz; j++ {
		for i := 0; i < nx; i++ {
			x, z := startX+float64(i)*sx, startZ+float64(j)*sz
			role := parkUrbanTerrainRole(park, paths, buckets, x, z, sx, sz)
			cell := maths.Box{Min: maths.V(x, 0, z), Max: maths.V(x+sx, 0, z+sz)}
			if role == parkService || role == parkPaving {
				service++
				if parkInfillMeetsTrees(cell, buckets) || boxesOverlapXZ(cell, maths.Box{Min: maths.V(-4.2, 0, -3.2), Max: maths.V(4.2, 0, 3.2)}) {
					t.Fatal("service surface covers tree or camera grove")
				}
				if parkTerrainPathDistance(paths, x+sx/2, z+sz/2) <= math.Hypot(sx, sz)/2+.05 {
					t.Fatal("service surface covers path")
				}
			} else {
				living++
			}
		}
	}
	if service == 0 || living == 0 {
		t.Fatalf("expected both service and living surfaces, got %d, %d", service, living)
	}
	batch := newCityBatch()
	parkUrbanTerrain(batch, park, trees)
	faces := 0
	for _, role := range []slot{parkMeadow, parkWoodland, parkStone, parkService, parkPaving} {
		for _, tri := range batch.pending[role] {
			centre := tri[0].Add(tri[1]).Add(tri[2]).Scale(1.0 / 3)
			if math.Abs(CityParkTerrainHeight(park, centre)-centre.Y) > 1e-9 {
				t.Fatal("service surface changed relief")
			}
			faces++
		}
	}
	if faces != 2*nx*nz {
		t.Fatalf("urban surface lost geometry: %d faces, want %d", faces, 2*nx*nz)
	}
}
