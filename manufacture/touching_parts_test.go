package manufacture

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

type touchingSmoothBox struct{ box }

func (made touchingSmoothBox) normals(skin surface) [][3]maths.Vec { return flatly(skin) }

type coordinateTexture struct{}

func (coordinateTexture) At(p maths.Vec) maths.Light { return maths.RGB(p.X, p.Y, p.Z) }

type invalidSurface struct{}

func (invalidSurface) skin(cut) surface { return surface{{maths.Vec{}, maths.Vec{}, maths.V(1, 0, 0)}} }

func TestTouchingPartsKeepGeometryNormalsTextureAndPlacement(t *testing.T) {
	form, err := newBox(measure.Metres(1), measure.Metres(1), measure.Metres(1))
	if err != nil {
		t.Fatal(err)
	}
	material := &thing.Material{Name: "coordinate texture", Texture: coordinateTexture{}}
	made := good{parts: []part{
		{shape: touchingSmoothBox{form}, of: body},
		{shape: form, of: body, space: maths.Subspace{Origin: maths.V(1, 0, 1)}},
	}, dressed: looks{body: material}}
	skin, normals := made.skins()[body], made.shadings()[body]
	_, original := skin.shaded(normals)
	if original == nil {
		t.Fatal("fixture must share a four-face edge")
	}
	at := maths.NewSubspace(maths.V(13, -7, 22), maths.Rotation(maths.YAxis, 0.7), 2)
	if _, strict := made.thingsStrict(at); strict == nil || strict.Error() != original.Error() {
		t.Fatalf("strict placement lost original error: %v, want %v", strict, original)
	}
	things, err := made.things(at)
	if err != nil {
		t.Fatal(err)
	}
	if len(things) != 2 {
		t.Fatalf("got %d separate things", len(things))
	}
	offset := 0
	for i, one := range things {
		solid, ok := one.Shape.(*mesh.Mesh)
		if !ok || one.Material != material || one.At(0) != at {
			t.Fatalf("thing %d lost mesh, material, or placement", i)
		}
		for face := 0; face < solid.FaceCount(); face++ {
			a, b, c := solid.Corners(face)
			if face3 := faceType(a, b, c); face3 != skin[offset+face] {
				t.Fatalf("thing %d face %d changed", i, face)
			}
			got, smooth := solid.CornerNormals(face)
			if !smooth || got != normals[offset+face] {
				t.Fatalf("thing %d face %d normals changed", i, face)
			}
			if one.Material.Texture.At(a) != material.Texture.At(skin[offset+face][0]) {
				t.Fatal("texture coordinate changed")
			}
		}
		offset += solid.FaceCount()
	}
	if offset != len(skin) {
		t.Fatalf("got %d faces, want %d", offset, len(skin))
	}
}

func faceType(a, b, c maths.Vec) face { return face{a, b, c} }

func TestTouchingPartsDoNotHideInvalidConstituents(t *testing.T) {
	form, err := newBox(measure.Metres(1), measure.Metres(1), measure.Metres(1))
	if err != nil {
		t.Fatal(err)
	}
	made := good{parts: []part{{shape: form}, {shape: invalidSurface{}}}, dressed: looks{body: &thing.Material{Name: "test"}}}
	_, original := made.skins()[body].meshed()
	if original == nil {
		t.Fatal("fixture must be invalid")
	}
	_, got := made.things(maths.Subspace{})
	if got == nil || got.Error() != original.Error() {
		t.Fatalf("got %v, want original %v", got, original)
	}
}

// A malformed rounded form must not turn an original validation error into a
// panic or a successful partial conversion.
type badNormalBox struct{ box }

func (made badNormalBox) normals(skin surface) [][3]maths.Vec { return flatly(skin[:len(skin)-1]) }

func TestTouchingPartsPreserveMalformedNormalError(t *testing.T) {
	form, err := newBox(measure.Metres(1), measure.Metres(1), measure.Metres(1))
	if err != nil {
		t.Fatal(err)
	}
	made := good{parts: []part{{shape: badNormalBox{form}}, {shape: form, space: maths.Subspace{Origin: maths.V(1, 0, 1)}}}, dressed: looks{body: &thing.Material{Name: "test"}}}
	_, original := made.skins()[body].shaded(made.shadings()[body])
	if original == nil {
		t.Fatal("fixture needs mismatched normals")
	}
	_, got := made.things(maths.Subspace{})
	if got == nil || got.Error() != original.Error() {
		t.Fatalf("got %v, want original %v", got, original)
	}
}
