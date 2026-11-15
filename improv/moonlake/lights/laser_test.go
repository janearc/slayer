package lights

import (
	"math"
	"slices"
	"testing"
)

func laserFixture() (Image, Image, LaserConfig, LaserInput) {
	sky, front, _, search := searchFixture()
	eye := float64(1129.6 - 1076)
	config := LaserConfig{Azimuth: 109.65, EyeOverLake: eye, WaterZ: -eye, Reach: 4000,
		FarTree:    search.FarTree,
		ShoreLeft:  Vec{263652.4 - 263104.5, 2148337.2 - 2148423.5, -eye + 8},
		ShoreRight: Vec{263578.1 - 263104.5, 2148129.0 - 2148423.5, -eye + 8}}
	rows := make([]int, sky.Width)
	for x := range rows {
		rows[x] = 8 + x%3
	}
	input := LaserInput{Open: search.Open, CloseTrees: search.CloseTrees, Rim: rows,
		Camera: search.Camera, Moon: search.Moon, Seconds: 1.25, ShoreRow: 11.25,
		Cues: LaserCues{Rig: -1, Sweep: -1, Gate: 1, GateLeft: -1, GateRight: -1, Beams: 10, Parity: -1, Figure: "fan"}}
	return sky, front, config, input
}

func TestLaserCapturedOutputs(t *testing.T) {
	wants := map[string]string{
		"free":    "ee903347512a990bfa7debcfa193aa08d2ab3f9efc41b9ef215f43b58996ce9a",
		"beam":    "b2bdc130fa462bc879cf8c8baf377f63e66d1f9a78a4876a7cbeeb6fda964130",
		"pair":    "8f8425e2bb236e76f7ad91873363a22c95b8ef6e1a484ba772f6e0afb4588302",
		"fan":     "7d0a63b0b5ef789af0f742c4230f795b003b60702468c183e6b9530e27844df0",
		"fan64":   "80a1d73b7a977de47cbc954eff08c692ec06463ab38f795030186664af2908d2",
		"scan":    "66bf86103e8d08d25e2b758c31414198189fa1bbaa5ac9d745ac8a0bf9921bab",
		"cone":    "f5bcc2d3a3a6259055b6ff4aadb7f3540f1fbffdc549568a73f469583c2e57ae",
		"wave":    "1a55969b085724202ecadb272d36309825cb0a1992f6edeb23da62160bff96bb",
		"unknown": "7d0a63b0b5ef789af0f742c4230f795b003b60702468c183e6b9530e27844df0",
		"split":   "8a3ce92838e4cfd8ac9bf5bd22dabcc3f9cfd5d4514192e7564a796b3d15803f",
		"banks":   "16c64be9f92dc85139fefaf556f76d5bba62ecd14c0201aafc55cf67bfd6aa01",
		"deck":    "3774943582878e73de45067bf4edb054664445bb3dd6b323c536a78ec5cee6c3",
		"off":     "2d8a802ea3ab86c0eb987a46ba2a5afed69cc91ad44cbba597b841694208f406",
		"blocked": "83a6a60aac15a799d4a8024be4521d3ba11a99f2d073fbf4bac40b450e4cef50",
		"behind":  "2d8a802ea3ab86c0eb987a46ba2a5afed69cc91ad44cbba597b841694208f406",
	}
	for name, want := range wants {
		t.Run(name, func(t *testing.T) {
			sky, front, config, input := laserFixture()
			if name != "free" {
				input.Cues.Rig, input.Cues.Sweep, input.Cues.FigurePhase = 0.7, 0.23, 0.37
				input.Cues.HasColour, input.Cues.Colour = true, [3]float64{0.3, 0.8, 1.2}
				input.Cues.Figure = name
			}
			switch name {
			case "split":
				input.Cues.Figure, input.Cues.Parity = "fan", 1
			case "banks":
				input.Cues.Figure, input.Cues.GateLeft, input.Cues.GateRight = "wave", 0.009, 0.6
			case "deck":
				config.RigDeck = 20.4
			case "off":
				input.Cues.Rig = 0
			case "blocked":
				for i := range input.CloseTrees {
					input.CloseTrees[i] = true
				}
			case "behind":
				input.Camera.Forward = input.Camera.Forward.Scale(-1)
			}
			AddLasers(sky, front, config, input)
			if got := effectDigest(append(sky.Light, front.Light...)); got != want {
				t.Fatalf("digest=%s want %s", got, want)
			}
		})
	}
}

func TestFigureCapturedPaths(t *testing.T) {
	wants := map[string]string{
		"beam":    "a24b51b6e204f71c5cf78f0187259456bf45c1884e0296827f8af6abd055f8a2",
		"pair":    "e39cb47a745cd43b0111950ab7d2cb847b8e3cd7adf97cd564be503d3ff4ee17",
		"fan":     "0cf754f08d4696bb488d4db7fc0417923dd486c2e633c43c8c1c1d120d6d72e9",
		"fan64":   "cf91e489005e6e050a1de9aadd92706b57ec21ed68ce0861b349e148d3e457be",
		"scan":    "04f0f48fa88fd7e3e879479f5cac1d082d9ab9bcd927248a36d87dc78a1ec6b7",
		"cone":    "2e0324cd4859e25026a13919a9cab1cb5c1b898f9e923c570fcb6657f7442002",
		"wave":    "a28c1622743fc4c3a9c46962ea8d52942399950718b57e8ab293650a0084a60e",
		"unknown": "0cf754f08d4696bb488d4db7fc0417923dd486c2e633c43c8c1c1d120d6d72e9",
		"zero":    "68f962b48b11fe16e782b2e07047abb2dfbfdc78a67e39410e00508331a5d1e7",
		"one":     "2ce6e9dc4ef5790a79db6a9c3733d814c09e937aa5a15d01dc59612a83d106fb",
	}
	for name, want := range wants {
		beams := 10
		if name == "zero" {
			beams = 0
		}
		if name == "one" {
			beams = 1
		}
		ways, weight, split := FigureWays(name, 289.65, -3.2, 27.5, beams, 0.37)
		values := []float64{weight, float64(len(ways))}
		if split {
			values = append(values, 1)
		} else {
			values = append(values, 0)
		}
		for _, way := range ways {
			values = append(values, way.X, way.Y, way.Z)
		}
		if got := effectDigest(values); got != want {
			t.Fatalf("%s digest=%s want %s", name, got, want)
		}
	}
}

func TestBeamLaserCapturedEffects(t *testing.T) {
	wants := map[string]string{
		"beam":             "38e4dc2a17777564f97d9308ce1b339bf348c7c481c5aa2bebce9d41b9c16fa8",
		"near":             "800782e9fd6de824e18ec89b01fe197e6ebd549ef001846c1caf4023898a32bc",
		"close":            "2d8a802ea3ab86c0eb987a46ba2a5afed69cc91ad44cbba597b841694208f406",
		"water":            "2d8a802ea3ab86c0eb987a46ba2a5afed69cc91ad44cbba597b841694208f406",
		"aperture":         "45eb63bde912a9de63e8b3f8cc97e0cf23456287912b6ab5f297d6df8579a121",
		"aperture-blocked": "2d8a802ea3ab86c0eb987a46ba2a5afed69cc91ad44cbba597b841694208f406",
	}
	for name, want := range wants {
		sky, front, config, input := laserFixture()
		origin := Toward(config.Azimuth, 0).Scale(150).Add(Vec{0, 0, -1})
		colour := [3]float64{0.3, 0.8, 1.2}
		switch name {
		case "aperture", "aperture-blocked":
			clear(input.CloseTrees)
			if name == "aperture-blocked" {
				for i := range input.CloseTrees {
					input.CloseTrees[i] = true
				}
			}
			input.ShoreRow = 9.25
			AddLaserAperture(sky, front, input, origin, colour)
		default:
			if name == "near" {
				clear(input.Open)
			}
			if name == "close" {
				for i := range input.CloseTrees {
					input.CloseTrees[i] = true
				}
			}
			if name == "water" {
				origin.Z = config.WaterZ - 1
			}
			AddLaserBeam(sky, front, config, input, origin, Toward(config.Azimuth+4, -2), colour)
		}
		if got := effectDigest(append(sky.Light, front.Light...)); got != want {
			t.Fatalf("%s digest=%s want %s", name, got, want)
		}
	}
	_, _, config, _ := laserFixture()
	values := []float64{}
	for _, p := range []Vec{{0, 0, -54}, {12, -34, -20}, {-80, 11, 60}} {
		values = append(values, LaserFog(p, 1.25, config.WaterZ))
	}
	if got, want := effectDigest(values), "2d7ab3ca2dfd8b82e47c7099146c1b25fc7b0d7e5833f25802e8737f03a85ef9"; got != want {
		t.Fatalf("fog=%s want %s", got, want)
	}
}

func TestLaserHeadGeometry(t *testing.T) {
	_, _, config, _ := laserFixture()
	heads := LaserHeads(config)
	if heads[0].Bank != 0 || heads[1].Bank != 1 || heads[2].Bank != 0 || heads[3].Bank != 1 {
		t.Fatal("bank ordering changed")
	}
	if heads[0].Flip != 0 || heads[1].Flip != 1 || heads[2].Flip != 1 || heads[3].Flip != 0 {
		t.Fatal("split ordering changed")
	}
	if heads[2].Aim != Bearing(config.ShoreLeft, config.ShoreRight) || heads[3].Aim != Bearing(config.ShoreRight, config.ShoreLeft) {
		t.Fatal("shore aims changed")
	}
	if heads[2].Origin != config.ShoreLeft || heads[3].Origin != config.ShoreRight {
		t.Fatal("shore geometry changed")
	}
	config.RigDeck = 20.4
	deck := LaserHeads(config)
	for i, ground := range []float64{1180.2, 1191.3, 1076.1, 1079.4} {
		if deck[i].Origin.Z != ground-(1076+config.EyeOverLake)+config.RigDeck || deck[i].Origin.X != heads[i].Origin.X || deck[i].Origin.Y != heads[i].Origin.Y {
			t.Fatalf("tower %d geometry changed", i)
		}
	}
}

func TestLaserExplicitInputsConcurrentIsolationAndRouting(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run("instance", func(t *testing.T) {
			t.Parallel()
			sky, front, config, input := laserFixture()
			if custom {
				config.RigDeck, config.Reach = 20.4, 1200
				input.Cues.Figure, input.Cues.GateLeft = "cone", 0.3
				input.Camera.Focal *= 0.7
			}
			original := slices.Clone(sky.Light)
			open, trees, rows := slices.Clone(input.Open), slices.Clone(input.CloseTrees), slices.Clone(input.Rim)
			AddLasers(sky, front, config, input)
			wantSky, wantFront := slices.Clone(sky.Light), slices.Clone(front.Light)
			if slices.Equal(wantSky, original) || !slices.ContainsFunc(wantFront, func(v float64) bool { return v > 0 }) {
				t.Fatal("fixture did not exercise both layers")
			}
			for y := range sky.Height {
				for x := range sky.Width {
					i := sky.At(x, y)
					if float64(y) >= input.ShoreRow && !slices.Equal(sky.Light[i:i+3], original[i:i+3]) {
						t.Fatal("below-shore light in sky")
					}
					if float64(y) < input.ShoreRow && slices.ContainsFunc(front.Light[i:i+3], func(v float64) bool { return v != 0 }) {
						t.Fatal("above-shore light in front")
					}
				}
			}
			for range 4 {
				copy(sky.Light, original)
				clear(front.Light)
				AddLasers(sky, front, config, input)
				if !slices.Equal(sky.Light, wantSky) || !slices.Equal(front.Light, wantFront) {
					t.Fatal("instance output changed")
				}
			}
			if !slices.Equal(input.Open, open) || !slices.Equal(input.CloseTrees, trees) || !slices.Equal(input.Rim, rows) {
				t.Fatal("input masks mutated")
			}
			for _, gate := range []float64{0, 0.009} {
				copy(sky.Light, original)
				clear(front.Light)
				input.Cues.GateLeft, input.Cues.GateRight = gate, gate
				AddLasers(sky, front, config, input)
				if !slices.Equal(sky.Light, original) || slices.ContainsFunc(front.Light, func(v float64) bool { return v != 0 }) {
					t.Fatal("inactive gates changed output")
				}
			}
		})
	}
}

func TestBeamLaserFogBoundary(t *testing.T) {
	if LaserFog(Vec{0, 0, -1}, 0, 0) != 0 {
		t.Fatal("fog below water")
	}
	for _, height := range []float64{0, 10, 100, 1000} {
		got := LaserFog(Vec{0, 0, height}, 1.25, 0)
		if math.IsNaN(got) || got < 0 || got > math.Exp(-height/90) {
			t.Fatalf("fog at %v=%v", height, got)
		}
	}
}
