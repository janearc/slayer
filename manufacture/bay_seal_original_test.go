package manufacture_test

import (
	"crypto/sha256"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// These reference values were captured from named.Get("seal-bay-study") at
// c9441d50b6f816c50b9b5f39f4bc0fb26488bb6c, before that scene was retired
// from the registry. The reference fingerprint writes each original vertex,
// rounded to a micrometre, as "x,y,z;", followed by each original face as
// "a,b,c;", in mesh order, and hashes the bytes using SHA-256. Keeping this
// independent reference lets the specimen be checked without reviving a
// retired named scene or deriving the expected values from the new maker.
func TestBaySealMatchesOriginalBayScene(t *testing.T) {
	kind, err := manufacture.KindNamed("bay-seal")
	if err != nil {
		t.Fatal(err)
	}
	specimen, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(407), "quick")
	if err != nil {
		t.Fatal(err)
	}
	parts, err := specimen.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 33 {
		t.Fatalf("seal has %d parts; original had 33", len(parts))
	}
	skin, ok := parts[0].Shape.(*mesh.Mesh)
	if !ok {
		t.Fatalf("seal skin has shape %T; original had *mesh.Mesh", parts[0].Shape)
	}
	if len(skin.Points()) != 1818 || len(skin.Faces()) != 3548 {
		vertices := len(skin.Points())
		faces := len(skin.Faces())
		t.Fatalf("skin has %d vertices and %d faces; original had 1818 and 3548", vertices, faces)
	}
	hash := sha256.New()
	for _, p := range skin.Points() {
		microX := int64(math.Round(p.X * 1e6))
		microY := int64(math.Round(p.Y * 1e6))
		microZ := int64(math.Round(p.Z * 1e6))
		fmt.Fprintf(hash, "%d,%d,%d;", microX, microY, microZ)
	}
	for _, face := range skin.Faces() {
		fmt.Fprintf(hash, "%d,%d,%d;", face[0], face[1], face[2])
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != "9d7fb3ff151d4706f45ebee1cb8a583ab6781067e6243814471e003b715dd330" {
		t.Fatalf("skin fingerprint differs from original bay scene: %s", got)
	}
	coat := parts[0].Material
	if coat.Name != "wet dark harbor seal" || coat.Colour != maths.RGB(.068, .063, .055) || coat.Gloss != .42 || coat.Satin != .33 || coat.Index != 1.43 || coat.Texture == nil {
		t.Fatal("coat settings differ from original bay scene")
	}
	for _, sample := range []struct {
		point maths.Vec
		want  maths.Light
	}{
		{maths.V(0, .2, 0), maths.RGB(.08011844044800001, .07688680627200001, .06649700121600001)},
		{maths.V(-.4, .1, .12), maths.RGB(.0698925300167751, .06613534389536761, .056553991364735684)},
		{maths.V(.81, .059, .02), maths.RGB(.014, .012, .011)},
	} {
		got := coat.Texture.At(sample.point)
		if math.Abs(got.R-sample.want.R) > 1e-12 || math.Abs(got.G-sample.want.G) > 1e-12 || math.Abs(got.B-sample.want.B) > 1e-12 {
			t.Fatalf("paint differs from original at %v: got %v, want %v", sample.point, got, sample.want)
		}
	}
	counts := map[string]int{}
	for _, part := range parts {
		if part.Material == nil {
			t.Fatal("seal part has no material")
		}
		counts[part.Material.Name]++
	}
	for name, want := range map[string]int{"wet dark harbor seal": 1, "harbor seal eye": 2, "wet muzzle whiskers": 30} {
		if counts[name] != want {
			t.Fatalf("%s count = %d; original had %d", name, counts[name], want)
		}
	}
}
