package snakes

import (
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func testProfile(t *testing.T, name string) Profile {
	t.Helper()
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := book.Profile(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func holds(box maths.Box, point maths.Vec) bool {
	return box.Min.X <= point.X && point.X <= box.Max.X &&
		box.Min.Y <= point.Y && point.Y <= box.Max.Y &&
		box.Min.Z <= point.Z && point.Z <= box.Max.Z
}

// Each checked species makes one closed, positively wound static body and
// separate eyes. Its analytic planning box includes every visible part.
func TestEverySnakeProfileMakesAClosedStaticBodyWithinItsExtent(t *testing.T) {
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	names := book.Species()
	if len(names) != 9 {
		t.Fatalf("species count = %d, want 9", len(names))
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			p, err := book.Profile(name)
			if err != nil {
				t.Fatal(err)
			}
			box, err := Extent(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Build(p)
			if err != nil {
				t.Fatal(err)
			}
			if got.Bounds != box {
				t.Fatalf("build and planning extents differ: %+v %+v", got.Bounds, box)
			}
			if len(got.Parts) != 3 {
				t.Fatalf("parts = %d, want body and two eyes", len(got.Parts))
			}
			skin, ok := got.Parts[0].Shape.(*mesh.Mesh)
			if !ok {
				t.Fatalf("body is %T", got.Parts[0].Shape)
			}
			if skin.Posed() || !skin.Closed() {
				t.Fatal("body must be a closed mesh without rest points")
			}
			points, faces := skin.Points(), skin.Faces()
			if geometry.SignedVolume6(points, faces) <= 0 {
				t.Fatal("body has no positive volume")
			}
			for i, face := range faces {
				a, b, c := points[face[0]], points[face[1]], points[face[2]]
				if b.Sub(a).Cross(c.Sub(a)).Len() <= 0 {
					t.Fatalf("face %d is degenerate", i)
				}
			}
			for _, part := range got.Parts {
				local := part.Shape.Bounds()
				at := part.At(0)
				for _, x := range []float64{local.Min.X, local.Max.X} {
					for _, y := range []float64{local.Min.Y, local.Max.Y} {
						for _, z := range []float64{local.Min.Z, local.Max.Z} {
							if point := at.Out(maths.V(x, y, z)); !holds(box, point) {
								t.Fatalf("part bounds %v outside %+v", point, box)
							}
						}
					}
				}
			}
			if got.Parts[0].Material == nil || got.Parts[0].Material.Texture == nil {
				t.Fatal("body has no procedural paint")
			}
		})
	}
}

// The two heavy snakes and the viper must not collapse to one default body.
func TestSpeciesDimensionsAndPatternsAreSelectedFromTheProfile(t *testing.T) {
	boa := testProfile(t, "boa-constrictor-occidentalis")
	titan := testProfile(t, "titanoboa-cerrejonensis")
	viper := testProfile(t, "bitis-gabonica")
	if !(titan.Length > boa.Length && boa.Length > viper.Length) {
		t.Fatalf("unexpected lengths titan=%g boa=%g viper=%g", titan.Length, boa.Length, viper.Length)
	}
	if viper.Head.Pit || viper.Pattern.Kind != "triangles" {
		t.Fatalf("Gaboon head/pattern lost: %+v %+v", viper.Head, viper.Pattern)
	}
	one, err := Build(boa)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Build(viper)
	if err != nil {
		t.Fatal(err)
	}
	if one.Parts[0].Material.Texture.At(maths.V(0, 0, 0)) == two.Parts[0].Material.Texture.At(maths.V(0, 0, 0)) {
		t.Fatal("distinct species paint collapsed")
	}
}

func TestSnakePaintRetainsItsOwnProfileSnapshot(t *testing.T) {
	p := testProfile(t, "boa-constrictor-occidentalis")
	made, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	paint := made.Parts[0].Material.Texture
	point := maths.V(0, 0, .02)
	before := paint.At(point)
	p.Pattern.Base = maths.RGB(1, 0, 1)
	p.Stations[len(p.Stations)/2].Width *= .5
	if after := paint.At(point); after != before {
		t.Fatalf("paint changed after profile mutation: %v to %v", before, after)
	}
}

func TestSnakeBuildAndExtentRefuseMutatedProfiles(t *testing.T) {
	p := testProfile(t, "bitis-gabonica")
	p.Length = math.NaN()
	if _, err := Extent(p); err == nil {
		t.Fatal("extent accepted a nonfinite length")
	}
	if _, err := Build(p); err == nil {
		t.Fatal("build accepted a nonfinite length")
	}
}

// Gaboon eye triangles and the illustrative python labial series are
// distinct markings. Gaboon vipers are viperines and have no loreal pits.
func TestGaboonTrianglesAndPythonLabialSeriesArePaired(t *testing.T) {
	viper := testProfile(t, "bitis-gabonica")
	python := testProfile(t, "python-bivittatus")
	if !python.Head.Pit {
		t.Fatal("python fixture has no labial markings")
	}
	for _, side := range []float64{-1, 1} {
		angle := math.Acos(viper.Head.EyeHeight) + .24
		centres := [2]float64{math.Max(0, viper.Head.EyeAt-.038), math.Min(.995, viper.Head.EyeAt+.006)}
		unmarked := viper
		unmarked.Pattern.Kind = "plain"
		for i, along := range centres {
			width, height := section(viper.Stations, along)
			point := maths.V((along-.5)*viper.Length, height*viper.Length*math.Cos(angle), side*width*viper.Length*math.Sin(angle))
			if !facialTriangles(viper.Head, along, side*angle) || (snakePaint{profile: viper}).At(point) == (snakePaint{profile: unmarked}).At(point) {
				t.Fatalf("viper side %g lost facial triangle %d", side, i)
			}
		}
		if facialTriangles(viper.Head, (centres[0]+centres[1])/2, side*angle) {
			t.Fatalf("viper side %g merged the two marks", side)
		}
		var along float64
		var width, height float64
		var point maths.Vec
		start := math.Max(0, python.Head.EyeAt-.035)
		end := math.Min(.998, python.Head.EyeAt+.018)
		angle = math.Acos(python.Head.EyeHeight) + .45
		unmarked = python
		unmarked.Head.Pit = false
		for i := 0; i < 5; i++ {
			along = start + (end-start)*float64(i)/4
			width, height = section(python.Stations, along)
			point = maths.V((along-.5)*python.Length, height*python.Length*math.Cos(angle), side*width*python.Length*math.Sin(angle))
			if (snakePaint{profile: python}).At(point) == (snakePaint{profile: unmarked}).At(point) {
				t.Fatalf("python side %g lost labial mark %d", side, i)
			}
		}
	}
}
