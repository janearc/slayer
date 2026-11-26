package seasonal

import (
	"math"
	"testing"
)

func TestPlantCellCacheDistinguishesSameNamedProfiles(t *testing.T) {
	first := plantProfiles["valley-oak"]
	second := first
	second.LeafLength *= 2
	first.Kind = "cache-reuse-example"
	second.Kind = first.Kind
	a, err := first.cell("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.cell("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || a.bounds == b.bounds {
		t.Fatal("different foliage reused the same prototype")
	}
	var done = make(chan *plantCell, 2)
	for _, p := range []Profile{first, second} {
		go func(p Profile) { cell, _ := p.cell("quick", 0); done <- cell }(p)
	}
	x, y := <-done, <-done
	if x == nil || y == nil || x == y {
		t.Fatal("parallel profiles shared a prototype")
	}
}

func TestPlantCellsShareOnlyWithinBookAndByGeometry(t *testing.T) {
	firstBook, err := Read("first.yaml", raw, testPolicy)
	if err != nil {
		t.Fatal(err)
	}
	secondBook, err := Read("second.yaml", raw, testPolicy)
	if err != nil {
		t.Fatal(err)
	}
	profile, found := firstBook.Get("valley-oak")
	if !found {
		t.Fatal("missing valley oak")
	}
	altered := profile
	altered.Kind, altered.By, altered.Date = "different-name", "someone", "tomorrow"
	a, err := profile.cell("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := altered.cell("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("nongeometry metadata created a second cell")
	}
	other, found := secondBook.Get("valley-oak")
	if !found {
		t.Fatal("missing valley oak")
	}
	c, err := other.cell("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a == c {
		t.Fatal("separate profile books share a process-global cache")
	}
}

func TestPlantProfilesRejectNumbersThatCannotBeGrown(t *testing.T) {
	profile := plantProfiles["valley-oak"]
	profile.Scale.Maximum = 1e300
	if err := profile.valid(); err == nil {
		t.Fatal("huge profile accepted")
	}
	profile = plantProfiles["valley-oak"]
	plan, err := profile.Plan(sow(1))
	if err != nil {
		t.Fatal(err)
	}
	for _, spacing := range []float64{math.SmallestNonzeroFloat64, 1e-150} {
		profile.CellSpacing = spacing
		plant := profile
		if _, err := plant.Attachments(plan, "quick"); err == nil {
			t.Fatalf("spacing %g accepted", spacing)
		}
	}
}

func TestPlantBookSnapshotsPolicyAndRoots(t *testing.T) {
	for _, policy := range []*Policy{nil, {}, {Value: plantValue}, {Sow: sow}} {
		if _, err := Read("plants.yaml", raw, policy); err == nil {
			t.Fatal("accepted an incomplete policy")
		}
	}
	policy := &Policy{Value: plantValue, Sow: sow}
	book, err := Read("plants.yaml", raw, policy)
	if err != nil {
		t.Fatal(err)
	}
	first, ok := book.Get("avicennia-marina")
	if !ok || first.Roots == nil {
		t.Fatal("missing mangrove roots")
	}
	wantRadius := first.Roots.Radius
	first.Roots.Radius *= 2
	second, ok := book.Get("avicennia-marina")
	if !ok || second.Roots.Radius != wantRadius {
		t.Fatal("book shared a mutable roots pointer")
	}
	policy.Value = func(string) float64 { return -1 }
	policy.Sow = nil
	if err := second.valid(); err != nil {
		t.Fatalf("book policy changed: %v", err)
	}
	if _, err := second.cell("quick", 0); err != nil {
		t.Fatalf("book seed factory changed: %v", err)
	}
}
