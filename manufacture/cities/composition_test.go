package cities

import (
	"math"
	"strings"
	"testing"
)

func simpleComposition() Composition {
	return Composition{Version: 1, Name: "crossing", Parts: []CompositionPart{
		{ID: "leg", Type: "leg", At: [3]float64{0, 0, 0}, Size: [3]float64{2, 10, 2}, Ground: true},
		{ID: "body", Type: "body", At: [3]float64{0, 10, 0}, Size: [3]float64{2, 5, 2}, RestsOn: []string{"leg"}},
		{ID: "deck", Type: "deck", At: [3]float64{0, 15, 0}, Size: [3]float64{10, 1, 2}, RestsOn: []string{"body"}},
	}, Roads: []CompositionRoad{{ID: "road", Type: "straight", On: "deck", From: [3]float64{0, 16, 1}, To: [3]float64{10, 16, 1}, Width: 1}}}
}

func TestCompositionChecksContactsAndRoad(t *testing.T) {
	if err := simpleComposition().Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, want string
		change     func(*Composition)
	}{
		{"floating ground", "leg", func(c *Composition) { c.Parts[0].At[1] = 1 }},
		{"no contact", "body", func(c *Composition) { c.Parts[1].At[1] = 11 }},
		{"unknown support", "missing", func(c *Composition) { c.Parts[1].RestsOn = []string{"missing"} }},
		{"duplicate", "leg", func(c *Composition) { c.Parts[1].ID = "leg" }},
		{"cycle", "body", func(c *Composition) { c.Parts[1].RestsOn = []string{"deck"}; c.Parts[2].RestsOn = []string{"body"} }},
		{"not finite", "leg", func(c *Composition) { c.Parts[0].Size[0] = math.NaN() }},
		{"road off deck", "road", func(c *Composition) { c.Roads[0].To[0] = 11 }},
		{"vertical road", "road", func(c *Composition) { c.Roads[0].To = c.Roads[0].From }},
		{"unsupported type", "body", func(c *Composition) { c.Parts[1].Type = "magic" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := simpleComposition()
			tc.change(&c)
			err := c.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want refusal naming %s", err, tc.want)
			}
		})
	}
}

func TestCompositionDesignMakesCheckedParts(t *testing.T) {
	c := simpleComposition()
	d, err := c.Design()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Elements) != len(c.Parts) {
		t.Fatalf("got %d elements", len(d.Elements))
	}
	c.Parts[0].Ground = false
	if _, err := c.Design(); err == nil {
		t.Fatal("made an unsupported composition")
	}
}

func TestCompositionStayFanNeedsAnchoredEnds(t *testing.T) {
	c := simpleComposition()
	c.Parts[2].RestsOn = nil
	c.Parts[2].SupportedBy = []string{"fan"}
	c.Parts = append(c.Parts, CompositionPart{ID: "fan", Type: "stay-fan", Tower: "body", Deck: "deck", TowerFrom: [3]float64{0, 14, 0}, TowerTo: [3]float64{0, 15, 0}, DeckFrom: [3]float64{2, 15, 0}, DeckTo: [3]float64{8, 15, 0}, Count: 2, Thickness: .1})
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Parts[3].DeckTo[0] = 11
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "fan") {
		t.Fatalf("detached stay: %v", err)
	}
	c.Parts[3].DeckTo[0] = 8
	c.Parts[3].Tower = "deck"
	if err := c.Validate(); err == nil {
		t.Fatal("accepted self-supporting deck")
	}
}

func TestCompositionStoreyBand(t *testing.T) {
	c := simpleComposition()
	c.Parts[1].Type = "storey-band"
	c.Parts[1].Count = 5
	d, err := c.Design()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Elements) != 7 {
		t.Fatalf("five storeys and two other parts: got %d", len(d.Elements))
	}
	c.Parts[1].Count = 0
	if err := c.Validate(); err == nil {
		t.Fatal("accepted zero storeys")
	}
}

func TestCompositionWindowRibbon(t *testing.T) {
	c := simpleComposition()
	c.Parts = append(c.Parts, CompositionPart{ID: "windows", Type: "window-ribbon", On: "body", Face: "north", At: [3]float64{0, 11, -.1}, Size: [3]float64{2, 2, .1}, BayWidth: 1, LitShare: .5, Heat: 3500, Glow: .3})
	d, err := c.DesignSeed(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Elements) != 5 {
		t.Fatalf("two windows and three solids: %d", len(d.Elements))
	}
	c.Parts[3].At[2] = -1
	if err := c.Validate(); err == nil {
		t.Fatal("accepted unattached window")
	}
	c.Parts[3].At[2] = -.1
	c.Parts[3].BayWidth = 0
	if err := c.Validate(); err == nil {
		t.Fatal("accepted zero bay width")
	}
}

func TestCompositionHangingBandNeedsContact(t *testing.T) {
	c := simpleComposition()
	c.Parts = append(c.Parts, CompositionPart{ID: "hanging", Type: "storey-band", At: [3]float64{2, 13, 0}, Size: [3]float64{3, 2, 2}, Count: 2, HangsFrom: []string{"deck"}})
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Parts[3].At[1] = 12
	if err := c.Validate(); err == nil {
		t.Fatal("accepted detached band")
	}
}

func TestCompositionRampAndRoad(t *testing.T) {
	c := Composition{Version: 1, Name: "ramp", Parts: []CompositionPart{
		{ID: "foot", Type: "leg", At: [3]float64{0, 0, 0}, Size: [3]float64{2, 1, 2}, Ground: true},
		{ID: "head", Type: "leg", At: [3]float64{100, 0, 0}, Size: [3]float64{2, 5, 2}, Ground: true},
		{ID: "rise", Type: "ramp", From: [3]float64{1, 1, 1}, To: [3]float64{101, 5, 1}, Width: 1, Thickness: .1, FromSupport: "foot", ToSupport: "head"},
	}, Roads: []CompositionRoad{{ID: "road", Type: "straight", On: "rise", From: [3]float64{1, 1, 1}, To: [3]float64{101, 5, 1}, Width: 1}}}
	if _, err := c.Design(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*Composition)
	}{
		{"grade", func(c *Composition) { c.Parts[1].Size[1] = 5.1; c.Parts[2].To[1] = 5.1; c.Roads[0].To[1] = 5.1 }},
		{"detached", func(c *Composition) { c.Parts[2].To[0] = 105 }},
		{"road leaves ramp", func(c *Composition) { c.Roads[0].Width = 2 }},
		{"road discontinuity", func(c *Composition) { c.Roads[0].To[0] = 100 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c2 := c
			c2.Parts = append([]CompositionPart(nil), c.Parts...)
			c2.Roads = append([]CompositionRoad(nil), c.Roads...)
			tc.change(&c2)
			if err := c2.Validate(); err == nil {
				t.Fatal("accepted invalid ramp or road")
			}
		})
	}
}

func TestCompositionConnectedRoadSegments(t *testing.T) {
	c := simpleComposition()
	c.Roads[0].To = [3]float64{5, 16, 1}
	c.Roads[0].Next = "road-b"
	c.Roads = append(c.Roads, CompositionRoad{ID: "road-b", Type: "straight", On: "deck", From: [3]float64{5, 16, 1}, To: [3]float64{10, 16, 1}, Width: 1})
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Roads[1].From[0] = 6
	if err := c.Validate(); err == nil {
		t.Fatal("accepted gap in road")
	}
	c.Roads[1].From[0] = 5
	c.Roads[1].Next = "road"
	if err := c.Validate(); err == nil {
		t.Fatal("accepted cyclic road")
	}
	c.Roads[1].Next = "missing"
	if err := c.Validate(); err == nil {
		t.Fatal("accepted missing next road")
	}
}
