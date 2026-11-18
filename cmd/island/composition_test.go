package main

import (
	"strings"
	"testing"
)

// goodSite is a site that passes every composition rule. Tests change one
// field of a copy and expect the matching refusal.
func goodSite() *sitePlan {
	forest := &forestPlan{
		GeometryMinimum: "crown", CrownAreaMatch: true, CrownAreaSide: 16,
		EdgeFade: 10, HeightFade: 5, Thresholds: [4]float64{1, 2, 3, 4},
		Bounds:        localBounds{MinEast: -100, MinSouth: -50, MaxEast: 100, MaxSouth: 50},
		MinimumHeight: 0, MaximumHeight: 100,
	}
	forest.Mixture = append(forest.Mixture, struct {
		Kind   string  `yaml:"kind" json:"kind"`
		Weight float64 `yaml:"weight" json:"weight"`
	}{Kind: "spruce", Weight: 1})
	site := &sitePlan{
		Objects: []objectPlan{{Name: "a", Kind: "can"}, {Name: "b", Kind: "log"}},
		Forest:  forest,
		Atmosphere: []mistPlan{{FloorSide: 4, EdgeFade: 10,
			Bounds: localBounds{MinEast: 0, MinSouth: 0, MaxEast: 100, MaxSouth: 60}}},
	}
	site.Budget.MistFloors = 1024
	return site
}

// TestCheckCompositionAcceptsGoodPlans checks that plans with and without a
// site, and with every optional section, pass.
func TestCheckCompositionAcceptsGoodPlans(t *testing.T) {
	if err := (plan{}).checkComposition(); err != nil {
		t.Errorf("empty plan: %v", err)
	}
	good := plan{Site: goodSite()}
	good.Clouds = []cloudPlan{{Low: 1000, High: 2000}}
	good.Landcover = []heightBand{{Below: 100}, {Below: 300}}
	good.LandcoverTransition = 50
	good.VolumeLighting = &volumeLightingPlan{Model: "isotropic-single", Samples: 8}
	if err := good.checkComposition(); err != nil {
		t.Errorf("good plan: %v", err)
	}
}

// TestCheckCompositionRefusals changes one thing in an otherwise good plan
// and checks the refusal that names it.
func TestCheckCompositionRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(*plan)
		want   string
	}{
		{"odd sky samples", func(p *plan) { p.VolumeLighting = &volumeLightingPlan{Samples: 3} }, "even"},
		{"cloud heights equal", func(p *plan) { p.Clouds = []cloudPlan{{Low: 5, High: 5}} }, "cloud layer heights must increase"},
		{"landcover not increasing", func(p *plan) { p.Landcover = []heightBand{{Below: 5}, {Below: 5}} }, "landcover height boundaries must increase"},
		{"landcover transitions overlap", func(p *plan) {
			p.Landcover = []heightBand{{Below: 100}, {Below: 150}}
			p.LandcoverTransition = 30
		}, "landcover transitions overlap"},
		{"transition with one band", func(p *plan) {
			p.Landcover = []heightBand{{Below: 100}}
			p.LandcoverTransition = 1
		}, "needs adjacent bands"},
		{"unknown object kind", func(p *plan) { p.Site.Objects[0].Kind = "dragon" }, "dragon"},
		{"duplicate object name", func(p *plan) { p.Site.Objects[1].Name = "a" }, `duplicate site object "a"`},
		{"crown match without crown floor", func(p *plan) { p.Site.Forest.GeometryMinimum = "hull" }, "crown material-area matching"},
		{"crown match without side", func(p *plan) { p.Site.Forest.CrownAreaSide = 0 }, "crown material-area matching"},
		{"unknown forest kind", func(p *plan) { p.Site.Forest.Mixture[0].Kind = "dragon" }, "dragon"},
		{"empty forest footprint", func(p *plan) { p.Site.Forest.Bounds.MaxEast = -100 }, "invalid forest footprint"},
		{"forest band reversed", func(p *plan) { p.Site.Forest.MinimumHeight = 200 }, "invalid forest footprint"},
		{"forest edge fade too wide", func(p *plan) { p.Site.Forest.EdgeFade = 51 }, "forest edge fade"},
		{"forest height fades overlap", func(p *plan) { p.Site.Forest.HeightFade = 51 }, "elevation fades overlap"},
		{"thresholds not increasing", func(p *plan) { p.Site.Forest.Thresholds = [4]float64{1, 2, 2, 4} }, "thresholds must increase"},
		{"mist floor over allowance", func(p *plan) { p.Site.Atmosphere[0].FloorSide = 12 }, "mist floor grid exceeds"},
		{"empty mist footprint", func(p *plan) { p.Site.Atmosphere[0].Bounds.MaxSouth = 0 }, "invalid mist footprint"},
		{"mist edge fade too wide", func(p *plan) { p.Site.Atmosphere[0].EdgeFade = 31 }, "mist edge fade"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := plan{Site: goodSite()}
			tc.change(&p)
			err := p.checkComposition()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

// TestMistFloorAllowanceZeroIsUnlimited checks that a zero cache allowance
// switches the payload check off.
func TestMistFloorAllowanceZeroIsUnlimited(t *testing.T) {
	p := plan{Site: goodSite()}
	p.Site.Budget.MistFloors = 0
	p.Site.Atmosphere[0].FloorSide = 4000
	if err := p.checkComposition(); err != nil {
		t.Errorf("unlimited allowance refused: %v", err)
	}
}

// TestValidLocalBounds checks that both axes must have positive extent.
func TestValidLocalBounds(t *testing.T) {
	cases := []struct {
		bounds localBounds
		want   bool
	}{
		{localBounds{0, 0, 1, 1}, true},
		{localBounds{1, 0, 1, 1}, false},
		{localBounds{0, 1, 1, 1}, false},
		{localBounds{2, 0, 1, 1}, false},
		{localBounds{0, 2, 1, 1}, false},
	}
	for _, tc := range cases {
		if got := validLocalBounds(tc.bounds); got != tc.want {
			t.Errorf("%+v: got %v, want %v", tc.bounds, got, tc.want)
		}
	}
}
