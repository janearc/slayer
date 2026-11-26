package manufacture

import (
	"bytes"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
)

// Anatomy belongs to the selected species, not a global cached Humboldt mesh.
func TestSquidProfilesKeepDistinctConcurrentAnatomies(t *testing.T) {
	changed := bytes.Replace(squidProfilesRaw, []byte("mantle-radius: 0.255"), []byte("mantle-radius: 0.34"), 1)
	if bytes.Equal(changed, squidProfilesRaw) {
		t.Fatal("profile has no mantle anatomy")
	}
	first, err := ReadSquidProfiles("humboldt.yaml", squidProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadSquidProfiles("wide.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	for i := range changed {
		changed[i] = 0
	}
	var bounds [2]maths.Box
	var errs [2]error
	var group sync.WaitGroup
	for i, book := range []*SquidProfiles{first, second} {
		group.Add(1)
		go func() {
			defer group.Done()
			parts, err := book.NewHumboldtSquid(0)
			if err != nil {
				errs[i] = err
				return
			}
			bounds[i] = parts[0].Shape.Bounds()
		}()
	}
	group.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if bounds[0] == bounds[1] {
		t.Fatal("different mantle anatomy shared one rest mesh")
	}
	again, err := first.NewHumboldtSquid(0)
	if err != nil || again[0].Shape.Bounds() != bounds[0] {
		t.Fatalf("first profile changed after constructing the second: %v", err)
	}
}

// Each accepted extreme must still produce a closed surface in the bounded
// scaffold grid, not merely pass the numerical reader.
func TestSquidAnatomyLimitsProduceClosedSurfaces(t *testing.T) {
	for _, tc := range []struct{ from, to string }{
		{"mantle-radius: 0.255", "mantle-radius: 0.12"},
		{"mantle-radius: 0.255", "mantle-radius: 0.36"},
		{"head-width: 0.21", "head-width: 0.15"},
		{"head-width: 0.21", "head-width: 0.3"},
		{"fin-span: 0.25", "fin-span: 0.18"},
		{"fin-span: 0.25", "fin-span: 0.34"},
		{"arm-length: 1.0", "arm-length: 0.85"},
		{"arm-length: 1.0", "arm-length: 1.15"},
		{"tentacle-length: 1.48", "tentacle-length: 1.25"},
		{"tentacle-length: 1.48", "tentacle-length: 1.65"},
	} {
		t.Run(tc.to, func(t *testing.T) {
			raw := bytes.Replace(squidProfilesRaw, []byte(tc.from), []byte(tc.to), 1)
			if bytes.Equal(raw, squidProfilesRaw) {
				t.Fatal("anatomy fixture not found")
			}
			book, err := ReadSquidProfiles("limit.yaml", raw)
			if err != nil {
				t.Fatal(err)
			}
			parts, err := book.NewHumboldtSquid(0)
			if err != nil {
				t.Fatal(err)
			}
			skin, ok := parts[0].Shape.(*mesh.Mesh)
			if !ok {
				t.Fatalf("unexpected skin %T", parts[0].Shape)
			}
			edges := map[[2]int]int{}
			for _, face := range skin.Faces() {
				for i := range 3 {
					a, b := face[i], face[(i+1)%3]
					if a > b {
						a, b = b, a
					}
					edges[[2]int{a, b}]++
				}
			}
			for edge, count := range edges {
				if count != 2 {
					t.Fatalf("edge %v has %d incident faces", edge, count)
				}
			}
		})
	}
}

func TestSquidProfilesSelectSizedAndColouredSpecimens(t *testing.T) {
	original := bytes.Clone(squidProfilesRaw)
	custom := bytes.Replace(original, []byte("kind: humboldt-squid"), []byte("kind: illustrated-squid"), 1)
	custom = bytes.Replace(custom, []byte("mantle-metres: 0.8"), []byte("mantle-metres: 1.6"), 1)
	custom = bytes.Replace(custom, []byte("coat: {r: 0.32, g: 0.09, b: 0.085}"), []byte("coat: {r: 0.5, g: 0.2, b: 0.1}"), 1)
	if bytes.Equal(original, custom) {
		t.Fatal("profile fixture was not changed")
	}
	book, err := ReadSquidProfiles("custom.yaml", custom)
	if err != nil {
		t.Fatal(err)
	}
	// The checked book must not refer to the caller's mutable input.
	for i := range custom {
		custom[i] = 'x'
	}
	made, err := book.NewSquid("illustrated-squid", .3)
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewHumboldtSquid(.3)
	if err != nil {
		t.Fatal(err)
	}
	if len(made) != len(base) {
		t.Fatal("species changed the shared body structure")
	}
	large, small := made[0].Shape.(*mesh.Mesh).Points(), base[0].Shape.(*mesh.Mesh).Points()
	if len(large) != len(small) {
		t.Fatal("size changed topology")
	}
	for i := range large {
		if large[i].Sub(small[i].Scale(2)).Len() > 1e-12 {
			t.Fatalf("skin point %d lost scale", i)
		}
	}
	if got := made[0].Material.Colour; got != maths.RGB(.5, .2, .1) {
		t.Fatalf("profile coat %v", got)
	}
	point := maths.V(.1, .2, .3)
	if got, old := made[0].Material.Texture.At(point), base[0].Material.Texture.At(point); got == old {
		t.Fatal("profile colour did not reach the patterned material")
	}
	if _, err := book.NewHumboldtSquid(0); err == nil {
		t.Fatal("missing species fell back to defaults")
	}
}

func TestSquidProfilesRejectInvalidOrAmbiguousData(t *testing.T) {
	for _, tc := range []struct{ name, from, to string }{
		{"missing mantle", "    mantle-metres: 0.8\n", ""},
		{"null mantle", "mantle-metres: 0.8", "mantle-metres: null"},
		{"zero mantle", "mantle-metres: 0.8", "mantle-metres: 0"},
		{"too large", "mantle-metres: 0.8", "mantle-metres: 1e300"},
		{"nonfinite", "mantle-metres: 0.8", "mantle-metres: .nan"},
		{"missing anatomy", "    anatomy: {mantle-radius: 0.255, head-width: 0.21, fin-span: 0.25, arm-length: 1.0, tentacle-length: 1.48}\n", ""},
		{"missing mantle radius", "mantle-radius: 0.255, ", ""},
		{"null head width", "head-width: 0.21", "head-width: null"},
		{"nonfinite fin", "fin-span: 0.25", "fin-span: .inf"},
		{"short arm", "arm-length: 1.0", "arm-length: 0.1"},
		{"long tentacle", "tentacle-length: 1.48", "tentacle-length: 4"},
		{"missing colour", "r: 0.32, ", ""},
		{"null colour", "r: 0.32", "r: null"},
		{"negative mottle", "r: 0.035", "r: -0.01"},
		{"out of range colour", "r: 0.32", "r: 0.99"},
		{"missing source", "status: picked", "status: read"},
		{"blank author", "by: Humboldt squid illustration study", "by: ' '"},
		{"unknown field", "mantle-metres: 0.8", "mantle-metres: 0.8\n    unexpected: 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := bytes.Replace(squidProfilesRaw, []byte(tc.from), []byte(tc.to), 1)
			if bytes.Equal(raw, squidProfilesRaw) {
				t.Fatal("missing mutation fixture")
			}
			if _, err := ReadSquidProfiles("bad.yaml", raw); err == nil {
				t.Fatal("accepted invalid squid profile")
			}
		})
	}
	entry := squidProfilesRaw[bytes.Index(squidProfilesRaw, []byte("  - kind:")):]
	duplicate := append(bytes.Clone(squidProfilesRaw), entry...)
	if _, err := ReadSquidProfiles("duplicate.yaml", duplicate); err == nil || !strings.Contains(err.Error(), "duplicate squid") {
		t.Fatalf("duplicate profile: %v", err)
	}
	for _, suffix := range []string{"\n---\nprofiles: []\n", "\n---\n[broken"} {
		if _, err := ReadSquidProfiles("two.yaml", append(bytes.Clone(squidProfilesRaw), suffix...)); err == nil {
			t.Fatal("accepted a trailing document")
		}
	}
	if _, err := ReadSquidProfiles("empty.yaml", nil); err == nil {
		t.Fatal("accepted an empty book")
	}
	if _, err := ReadSquidProfiles("large.yaml", make([]byte, (1<<20)+1)); err == nil {
		t.Fatal("accepted an oversized book")
	}
	for _, size := range []float64{.001, 101, math.MaxFloat64} {
		if _, err := squidWithLength(size, 0); err == nil {
			t.Fatalf("accepted unsafe mantle %g", size)
		}
	}
	if _, err := NewSquid("not-a-species", 0); err == nil {
		t.Fatal("accepted an unknown default squid species")
	}
	var book *SquidProfiles
	if _, err := book.NewSquid("humboldt-squid", 0); err == nil {
		t.Fatal("accepted a nil book")
	}
	if _, err := defaultSquidProfiles.NewHumboldtSquid(math.NaN()); err == nil {
		t.Fatal("accepted a nonfinite phase")
	}
}

// The former Humboldt coat is an independent formula, including its exact
// noise seed, frequency and arithmetic order.
func TestHumboldtSquidProfileKeepsOriginalCoat(t *testing.T) {
	made, err := NewHumboldtSquid(0)
	if err != nil {
		t.Fatal(err)
	}
	if got := made[0].Material.Colour; got != maths.RGB(.32, .09, .085) {
		t.Fatalf("base coat changed: %v", got)
	}
	for _, point := range []maths.Vec{maths.Vec{}, maths.V(.1, .2, .3), maths.V(-.4, .2, -.1)} {
		n := noise.Fractal(1711, point.X*35, point.Y*35, point.Z*35, 4)
		want := maths.RGB(.32+.035*n, .09+.018*n, .085+.015*n)
		if got := made[0].Material.Texture.At(point); got != want {
			t.Fatalf("coat at %v = %v, want %v", point, got, want)
		}
	}
}

func TestSquidLengthLimitsConstructFiniteDetails(t *testing.T) {
	finite := func(p maths.Vec) bool {
		return !math.IsNaN(p.X) && !math.IsInf(p.X, 0) &&
			!math.IsNaN(p.Y) && !math.IsInf(p.Y, 0) &&
			!math.IsNaN(p.Z) && !math.IsInf(p.Z, 0)
	}
	for _, length := range []float64{.01, 100} {
		parts, err := squidWithLength(length, .3)
		if err != nil {
			t.Fatalf("length %g: %v", length, err)
		}
		if len(parts) != 292 {
			t.Fatalf("length %g: %d parts", length, len(parts))
		}
		for i, part := range parts {
			box := part.Shape.Bounds()
			if !finite(box.Min) || !finite(box.Max) || !finite(part.At(0).Origin) {
				t.Fatalf("length %g: nonfinite part %d", length, i)
			}
		}
	}
}
