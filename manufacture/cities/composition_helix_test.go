package cities

import (
	"math"
	"testing"
)

func TestHelixMeshGradeAndSupport(t *testing.T) {
	p := CompositionPart{ID: "spiral", Type: "helix", Centre: [3]float64{0, 0, 0}, Radius: 100, Turns: 1, Rise: 20, Width: 4, Thickness: 1, Clearance: 3, Segments: 64, FromSupport: "low", ToSupport: "high"}
	a, b := p.helixPoint(0, 0), p.helixPoint(1, 0)
	c := Composition{Version: 1, Name: "spiral", Parts: []CompositionPart{
		{ID: "low", Type: "leg", At: [3]float64{97, 0, -3}, Size: [3]float64{6, 1, 6}, Ground: true},
		{ID: "high", Type: "leg", At: [3]float64{97, 0, -3}, Size: [3]float64{6, 21, 6}, Ground: true}, p,
	}, Roads: []CompositionRoad{{ID: "road", Type: "helix", On: "spiral", From: a, To: b, Width: 4}}}
	c.Parts[2].Centre[1] = 1
	a, b = c.Parts[2].helixPoint(0, 0), c.Parts[2].helixPoint(1, 0)
	c.Roads[0].From = a
	c.Roads[0].To = b
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	mesh, err := (helixForm{c.Parts[2]}).Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	if len(mesh.Triangles) != 64*8+4 || !closed(mesh) {
		t.Fatalf("helix mesh has %d faces or is open", len(mesh.Triangles))
	}
	cases := []struct {
		name   string
		change func(*Composition)
	}{
		{"steep", func(c *Composition) { c.Parts[2].Rise = 40 }},
		{"detached", func(c *Composition) { c.Parts[1].At[0] = 110 }},
		{"no clearance", func(c *Composition) { c.Parts[2].Clearance = 30 }},
		{"too wide", func(c *Composition) { c.Roads[0].Width = 5 }},
		{"nonfinite", func(c *Composition) { c.Parts[2].Radius = math.NaN() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := c
			d.Parts = append([]CompositionPart(nil), c.Parts...)
			d.Roads = append([]CompositionRoad(nil), c.Roads...)
			tc.change(&d)
			if err := d.Validate(); err == nil {
				t.Fatal("accepted invalid helix")
			}
		})
	}
}

func TestHelixRejectsBadParametersAndChecksBothDirections(t *testing.T) {
	p := CompositionPart{ID: "spiral", Type: "helix", Centre: [3]float64{0, 1, 0}, Radius: 100, Turns: -1, Rise: 20, Width: 4, Thickness: 1, Clearance: 3, Segments: 64}
	if err := p.helixCheck(); err != nil {
		t.Fatal(err)
	}
	m, err := (helixForm{p}).Mesh("quick")
	if err != nil || !closed(m) {
		t.Fatalf("clockwise mesh: %v", err)
	}
	cases := []func(*CompositionPart){func(p *CompositionPart) { p.Segments = 3 }, func(p *CompositionPart) { p.Turns = 0 }, func(p *CompositionPart) { p.Width = 201 }, func(p *CompositionPart) { p.Clearance = 0 }, func(p *CompositionPart) { p.Rise = 0 }}
	for _, change := range cases {
		bad := p
		change(&bad)
		if err := bad.helixCheck(); err == nil {
			t.Fatalf("accepted invalid helix %+v", bad)
		}
	}
}

func TestRoadJoinNeedsAlignedWidth(t *testing.T) {
	c := simpleComposition()
	c.Roads[0].To = [3]float64{5, 16, 1}
	c.Roads[0].Next = "turn"
	c.Roads = append(c.Roads, CompositionRoad{ID: "turn", Type: "straight", On: "deck", From: [3]float64{5, 16, 1}, To: [3]float64{5, 16, 1.5}, Width: 1})
	if err := c.Validate(); err == nil {
		t.Fatal("accepted a corner with mismatched edges")
	}
}

func TestHelixRequiresTurnAlignedTessellation(t *testing.T) {
	p := CompositionPart{ID: "spiral", Type: "helix", Centre: [3]float64{0, 1, 0}, Radius: 100, Turns: 1.5, Rise: 30, Width: 4, Thickness: 1, Clearance: 3, Segments: 65}
	if err := p.helixCheck(); err == nil {
		t.Fatal("accepted mismatched tessellation across turns")
	}
	p.Segments = 96
	if err := p.helixCheck(); err != nil {
		t.Fatal(err)
	}
}

func TestRoadJoinChecksActualEdgeCoordinates(t *testing.T) {
	c := Composition{Version: 1, Name: "wide", Parts: []CompositionPart{{ID: "deck", Type: "deck", At: [3]float64{-1, 0, -2000}, Size: [3]float64{4000, 1, 4000}, Ground: true}}}
	c.Roads = []CompositionRoad{{ID: "a", Type: "straight", On: "deck", From: [3]float64{0, 1, 0}, To: [3]float64{100, 1, 0}, Width: 1000, Next: "b"}, {ID: "b", Type: "straight", On: "deck", From: [3]float64{100, 1, 0}, To: [3]float64{200, 1, .001}, Width: 1000}}
	if err := c.Validate(); err == nil {
		t.Fatal("accepted an edge gap on a wide road")
	}
	c.Roads[1].To[2] = 0
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestHelixBoundedCoordinatesAndClearanceMargin(t *testing.T) {
	p := CompositionPart{ID: "bound", Type: "helix", Centre: [3]float64{999898, 0, 0}, Radius: 100, Turns: 1, Rise: 20, Width: 4, Thickness: 1, Clearance: 3, Segments: 64}
	if err := p.helixCheck(); err != nil {
		t.Fatal(err)
	}
	p.Centre[0] = 999898.01
	if err := p.helixCheck(); err == nil {
		t.Fatal("accepted extent beyond a million metres")
	}
	p.Centre[0] = 0
	p.StartAngle = 2*math.Pi + .001
	if err := p.helixCheck(); err == nil {
		t.Fatal("accepted unbounded start angle")
	}
	p.StartAngle = 0
	p.Turns = 1.50000000001
	p.Segments = 96
	p.Rise = 30
	if err := p.helixCheck(); err == nil {
		t.Fatal("accepted approximately aligned turns")
	}
	p.Turns = 1
	p.Segments = 64
	p.Rise = 4.0005
	if err := p.helixCheck(); err == nil {
		t.Fatal("accepted clearance inside numerical margin")
	}
}
