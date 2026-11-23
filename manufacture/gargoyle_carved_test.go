package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/weathering"
)

func TestTheSpoutMarkerIsNamedAsTheWeatheringLooksForIt(t *testing.T) {
	if spoutMaterial != weathering.SpoutMaterial {
		t.Errorf("the marker is %q, the weathering looks for %q",
			spoutMaterial, weathering.SpoutMaterial)
	}
}

func TestEveryGargoyleCarriesOneHiddenMarkerAtItsSpout(t *testing.T) {
	for _, name := range gargoyleKindNames(t) {
		made := mustMake(t, name, 1)
		spouts, err := Spouts(name, 1)
		if err != nil {
			t.Fatal(err)
		}
		var markers int
		for _, piece := range made.native {
			if piece.Material.Name != spoutMaterial {
				continue
			}
			markers++
			if !piece.Hidden() {
				t.Errorf("%s: the marker is drawn", name)
			}
			if at := piece.At(0).Origin; at.Sub(spouts[0]).Len() > gargoyleTol {
				t.Errorf("%s: the marker is at %v, the spout at %v", name, at,
					spouts[0])
			}
		}
		if markers != 1 {
			t.Errorf("%s has %d markers, not one", name, markers)
		}
	}
}

func TestNoFaceOfACarvedCreatureIsWhollyInTheWall(t *testing.T) {
	for _, name := range gargoyleKindNames(t) {
		for _, piece := range mustMake(t, name, 1).native {
			whole, is := piece.Shape.(*mesh.Mesh)
			if !is {
				continue
			}
			bare := soupOfMesh(whole)
			for _, face := range bare.faces {
				if !bare.reaches(face, maths.Local()) {
					t.Fatalf("%s: a face is wholly behind the wall", name)
				}
			}
		}
	}
}

func TestAGraniteGargoyleIsFlecksAndConcreteIsBoardMarkedBothFaceted(t *testing.T) {
	for name, stone := range gargoyleStones() {
		look := stone.look(name, 1, 1)
		if look.Crinkle == nil || !(look.Crinkle.Facet > 0) {
			t.Errorf("%s has no chisel facets", name)
		}
		_, boarded := look.Texture.(texture.Boarded)
		_, dusted := look.Texture.(texture.Dusted)
		if boarded != (name == "concrete") || dusted != (name == "granite") {
			t.Errorf("%s: boarded %v, dusted %v", name, boarded, dusted)
		}
	}
}

func TestAStoneAtAScaleKeepsItsLookInTheThingsOwnSpace(t *testing.T) {
	stone := gargoyleStones()["concrete"]
	whole := stone.look("whole", 1, 1).Texture.(texture.Boarded)
	tenth := stone.look("tenth", 1, 0.1).Texture.(texture.Boarded)
	if math.Abs(tenth.Board-whole.Board*10) > gargoyleTol {
		t.Errorf("a tenth-size thing has boards %v, want %v", tenth.Board,
			whole.Board*10)
	}
}

func TestAWingFarFromTheBodyIsRaisedAndSweptBackAndOneNearItIsLeftAlone(t *testing.T) {
	near := maths.V(0.3, 0.1, wingHinge-0.1)
	if foldedWing(near) != near {
		t.Errorf("a point inside the hinge moved to %v", foldedWing(near))
	}
	at := maths.V(0, 0, wingHinge)
	if got := foldedWing(at); got.Sub(at).Len() > gargoyleTol {
		t.Errorf("the hinge moved to %v: the wing is torn", got)
	}
	tip := foldedWing(maths.V(0, 0, 4))
	if !(tip.Y > 1) || !(tip.X < 0) || !(tip.Z < 4) || !(tip.Z > wingHinge) {
		t.Errorf("a wing tip is at %v: not raised, swept back and drawn in", tip)
	}
}

func TestBothWingsFoldAsMirrorsOfEachOther(t *testing.T) {
	for _, reach := range []float64{1, 2, 3.5} {
		left := foldedWing(maths.V(0.4, 0.05, -reach))
		right := foldedWing(maths.V(0.4, 0.05, reach))
		if math.Abs(left.X-right.X) > gargoyleTol ||
			math.Abs(left.Y-right.Y) > gargoyleTol ||
			math.Abs(left.Z+right.Z) > gargoyleTol {
			t.Errorf("at %v the wings are %v and %v", reach, left, right)
		}
	}
}

func TestTheFoldedHatzIsNarrowerThanTheStudyAndTheStudyIsNotChanged(t *testing.T) {
	rest, err := hatzGargoyleRest()
	if err != nil {
		t.Fatal(err)
	}
	before := rest[0].Shape.Bounds()
	folded, err := hatzGargoyleStudy()
	if err != nil || len(folded) != len(rest) {
		t.Fatalf("%d pieces, %v", len(folded), err)
	}
	after := folded[0].Shape.Bounds()
	if !(after.Max.Z < before.Max.Z) || !(after.Max.Y > before.Max.Y) {
		t.Errorf("span %v to %v, height %v to %v: not folded up",
			before.Max.Z, after.Max.Z, before.Max.Y, after.Max.Y)
	}
	if again := rest[0].Shape.Bounds(); again != before {
		t.Errorf("folding changed the study: %v to %v", before, again)
	}
}

func TestAFoldedSurfaceGetsAUnitNormalAtEveryCorner(t *testing.T) {
	corners := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	normals := smoothed(corners, [][3]int{{0, 1, 2}})
	for index, normal := range normals {
		if math.Abs(normal.Len()-1) > gargoyleTol || normal.Z < 0.99 {
			t.Errorf("corner %d has normal %v", index, normal)
		}
	}
}

func TestASmallFeatureIsCarvedLargerAboutItsMiddleAndALargePieceIsNot(t *testing.T) {
	place := maths.NewSubspace(maths.V(1, 2, 3), maths.Identity, 2)
	eye := coarsened(shape.Sphere{Radius: 0.05}, place)
	if math.Abs(eye.Scale-2*coarseGrowth) > gargoyleTol || eye.Origin != place.Origin {
		t.Errorf("an eye is at %v, scale %v", eye.Origin, eye.Scale)
	}
	body := coarsened(shape.Sphere{Radius: 1}, place)
	if body != place {
		t.Errorf("a body was changed to %+v", body)
	}
}

func TestAFeatureOfAPlaceWithNoScaleIsGrownFromOne(t *testing.T) {
	eye := coarsened(shape.Sphere{Radius: 0.05}, maths.Local())
	if math.Abs(eye.Scale-coarseGrowth) > gargoyleTol {
		t.Errorf("scale %v, want %v", eye.Scale, coarseGrowth)
	}
}

func TestTheSpoutMarkerIsASphereNotAMesh(t *testing.T) {
	marker := gargoyleCreature{}.marker(maths.Local())
	if _, is := marker.Shape.(shape.Sphere); !is {
		t.Errorf("the marker is a %T", marker.Shape)
	}
}
