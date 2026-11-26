package whales

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// These are the picked linear reflectances used before the profile change.
func TestOrcaPaletteKeepsOriginalValues(t *testing.T) {
	p, ok := Default().Get("orca")
	if !ok {
		t.Fatal("missing orca")
	}
	want := Palette{Basis: "picked", Black: maths.RGB(.016, .020, .024), White: maths.RGB(.72, .74, .70),
		Saddle: maths.RGB(.20, .21, .22), Pale: maths.RGB(.34, .34, .35), Iris: maths.RGB(.040, .025, .015),
		Pupil: maths.RGB(.004, .004, .005), Gum: maths.RGB(.050, .020, .020), Ivory: maths.RGB(.78, .75, .66)}
	if p.Palette != want {
		t.Fatalf("default palette changed: %+v", p.Palette)
	}
}

func TestOrcaPalettesAreIndependentOfSharedAnatomy(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("black: {r: 0.016, g: 0.020, b: 0.024}"), []byte("black: {r: 0.116, g: 0.120, b: 0.124}"), 1)
	if bytes.Equal(raw, profileRaw) {
		t.Fatal("missing palette fixture")
	}
	raw = bytes.Replace(raw, []byte("satin: 0.12"), []byte("satin: 0.4"), 1)
	a, err := Read("original.yaml", profileRaw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Read("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := a.Get("orca")
	changed, _ := b.Get("orca")
	if original.model != changed.model {
		t.Fatal("same anatomy did not reuse the canonical model")
	}
	first, err := original.Swim(0, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	second, err := changed.Swim(0, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	oldSkin, oldOK := first[0].Shape.(*mesh.Mesh)
	newSkin, newOK := second[0].Shape.(*mesh.Mesh)
	if !oldOK || !newOK || !reflect.DeepEqual(oldSkin.Points(), newSkin.Points()) || !reflect.DeepEqual(oldSkin.Faces(), newSkin.Faces()) {
		t.Fatal("changing palette changed the swim geometry")
	}
	if first[0].Material.Satin != .12 || second[0].Material.Satin != .4 {
		t.Fatal("profile satin did not reach the family material")
	}
	if first[0].Material.Colour != original.Palette.Black || second[0].Material.Colour != changed.Palette.Black {
		t.Fatal("profile colour did not reach the material")
	}
	point := maths.V(1, .55, 0)
	before := first[0].Material.Texture.At(point)
	if before == second[0].Material.Texture.At(point) {
		t.Fatal("changed palette did not change skin texture")
	}
	changed.Palette.Black = maths.RGB(1, 1, 1)
	if first[0].Material.Texture.At(point) != before || second[0].Material.Colour != maths.RGB(.116, .120, .124) {
		t.Fatal("a returned profile changed a constructed specimen")
	}
	again, _ := b.Get("orca")
	if again.Palette.Black != maths.RGB(.116, .120, .124) {
		t.Fatal("profile exposed book data")
	}
	hop, err := again.SpyHop(maths.Local())
	if err != nil || hop[0].Material.Satin != .4 || hop[0].Material.Colour != again.Palette.Black || hop[0].Material.Texture.At(point) != second[0].Material.Texture.At(point) {
		t.Fatalf("spy-hop lost its profile palette: %v", err)
	}
}

func TestOrcaPaletteAcceptsSourcedBasis(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("basis: picked"), []byte("basis: read\n      source: descriptive source"), 1)
	if bytes.Equal(raw, profileRaw) {
		t.Fatal("missing palette basis")
	}
	book, err := Read("sourced.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	profile, _ := book.Get("orca")
	if profile.Palette.Basis != "read" || profile.Palette.Source != "descriptive source" {
		t.Fatal("palette provenance lost")
	}
}

func TestOrcaPaletteRefusesMalformedData(t *testing.T) {
	for _, tc := range []struct{ from, to, field string }{
		{"basis: picked", "basis: invented", "palette"},
		{"basis: picked", "basis: read", "palette"},
		{"r: 0.016", "r: -0.1", "black"},
		{"r: 0.016", "r: 1.1", "black"},
		{"r: 0.016", "r: .nan", "black"},
		{"g: 0.020", "g: .inf", "black"},
		{"b: 0.024", "b: null", "black"},
		{"black: {r: 0.016, g: 0.020, b: 0.024}", "black: {r: 0.016, g: 0.020}", "black"},
		{"black:", "unknown:", "black"},
		{"basis: picked", "basis: picked\n      unknown: 1", "unknown"},
	} {
		raw := bytes.Replace(profileRaw, []byte(tc.from), []byte(tc.to), 1)
		if bytes.Equal(raw, profileRaw) {
			t.Fatalf("missing fixture %q", tc.from)
		}
		_, err := Read("bad.yaml", raw)
		if err == nil || !strings.Contains(err.Error(), tc.field) {
			t.Errorf("%s: got %v", tc.field, err)
		}
	}
	start := bytes.Index(profileRaw, []byte("    palette:\n"))
	end := bytes.Index(profileRaw, []byte("    # Axial position"))
	if start < 0 || end <= start {
		t.Fatal("missing palette block")
	}
	missing := append(bytes.Clone(profileRaw[:start]), profileRaw[end:]...)
	if _, err := Read("missing.yaml", missing); err == nil {
		t.Fatal("accepted missing palette")
	}
}
