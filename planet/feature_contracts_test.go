package planet

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestGrainPatternsFilterFineStructureAndKeepTheirSeam(t *testing.T) {
	names := Patterns()
	if !slices.Equal(names, []string{"cells", "ridges", "smooth", "speckle"}) {
		t.Fatal(names)
	}
	names[0] = "changed"
	if Patterns()[0] != "cells" {
		t.Fatal("caller changed registry")
	}
	ring := mustRing(t, 100, 200, .3, .8, 0)
	for _, name := range Patterns() {
		t.Run(name, func(t *testing.T) {
			made, err := ring.Grained(name, 17, 1, 4, 2, 3, .6)
			if err != nil {
				t.Fatal(err)
			}
			low, hidden := made.grains.layered(150, 0, 1e6)
			if low != 0 || !(hidden > 0) {
				t.Fatalf("unresolved structure %g %g", low, hidden)
			}
			left, right := made.share(-150, -1e-8), made.share(-150, 1e-8)
			if math.Abs(left-right) > 1e-5 || left < 0 || math.IsNaN(left) {
				t.Fatalf("grain seam %g %g", left, right)
			}
			a, b := made.grains.layered(150, 2, 0)
			if math.IsNaN(a) || b != 0 {
				t.Fatal("fully resolved grains")
			}
		})
	}
	for _, bad := range []struct {
		name                              string
		small, large, slope, shear, clump float64
	}{
		{"unknown", 1, 4, 2, 3, .5}, {"smooth", 0, 4, 2, 3, .5}, {"smooth", 1, .5, 2, 3, .5},
		{"smooth", 1, 4, 0, 3, .5}, {"smooth", 1, 4, 2, .5, .5}, {"smooth", 1, 4, 2, 3, 1.1},
	} {
		if _, err := ring.Grained(bad.name, 1, bad.small, bad.large, bad.slope, bad.shear, bad.clump); err == nil {
			t.Fatal("bad grains accepted")
		}
	}
	whole, _ := NewZone(-90, 90, nil)
	body, err := NewBody(1, 1, 1, []Zone{whole})
	if err != nil {
		t.Fatal(err)
	}
	body, err = body.Ringed([]Ring{ring})
	if err != nil {
		t.Fatal(err)
	}
	seen, err := body.Seen(maths.V(0, 100, 0), .01)
	if err != nil {
		t.Fatal(err)
	}
	if body.rings[0].from.ray != 0 || seen.rings[0].from.ray != .01 {
		t.Fatal("Seen changed original ring")
	}
	if got := seen.rings[0].from.finest(100, 0); !(got > 1) {
		t.Fatal(got)
	}
	for _, test := range []struct {
		eye maths.Vec
		ray float64
	}{
		{maths.Vec{}, -1}, {maths.Vec{}, math.Inf(1)}, {maths.V(math.NaN(), 0, 0), 0}, {maths.V(0, math.Inf(1), 0), 0},
	} {
		if _, err := body.Seen(test.eye, test.ray); err == nil {
			t.Fatal("nonfinite vantage")
		}
	}
}

func TestMoonletGapsAndPropellersStayAtTheirOrbits(t *testing.T) {
	ring := mustRing(t, 100, 200, .3, .8, 0)
	for _, opening := range []float64{.5, 2} {
		made, err := ring.Cleared(7, 1, 1, 1, 2, opening, 1e15)
		if err != nil {
			t.Fatal(err)
		}
		moon := made.moonlets[0]
		x, z := moon.reach*math.Cos(moon.angle), moon.reach*math.Sin(moon.angle)
		if made.cleared(x, z) != 0 || made.cleared(1e6, 0) != 1 {
			t.Fatal("moonlet did not clear its neighbourhood")
		}
		if opening < 1 && made.cleared(-x, -z) != 0 {
			t.Fatal("open gap stopped halfway round orbit")
		}
		if opening > 1 && made.cleared(-x, -z) != 1 {
			t.Fatal("propeller cleared whole orbit")
		}
	}
	for _, bad := range []struct {
		count                            int
		small, large, slope, opens, mass float64
	}{
		{-1, 1, 2, 2, 1, 1}, {1, 0, 2, 2, 1, 1}, {1, 2, 1, 2, 1, 1}, {1, 1, 2, 1, 1, 1}, {1, 1, 2, 2, 0, 1}, {1, 1, 2, 2, 1, 0},
	} {
		if _, err := ring.Cleared(1, bad.count, bad.small, bad.large, bad.slope, bad.opens, bad.mass); err == nil {
			t.Fatal("invalid moonlet recipe")
		}
	}
	if edged(0) != 0 || edged(1.25) != .5 || edged(2) != 1 {
		t.Fatal("gap edge endpoints")
	}
	body := bareBody(t, 1, nil)
	if !(body.Mass() > 1e27 && body.Mass() < 2e27) || body.Radius() != 6e7 {
		t.Fatal("mass units")
	}
}

func TestRingPhosphorsRespondToPowerExposureAndThickness(t *testing.T) {
	ring := mustRing(t, 100, 200, .3, .8, 0)
	glowing, err := ring.Glowing(map[string]float64{"csi-tl": .5, "zns-ag": .5})
	if err != nil {
		t.Fatal(err)
	}
	xrays, err := NewXrays(1e10, 30)
	if err != nil {
		t.Fatal(err)
	}
	slab, shine, err := NewRingGlow(glowing, 2, 50, xrays, 1)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(150, 0, 0)
	first := shine.At(point)
	if !(first.G > 0 && slab.Density(point) > 0) || slab.Density(maths.V(150, 10, 0)) != 0 {
		t.Fatal("slab did not confine light")
	}
	if slab.Bounds().Min != maths.V(-200, -1, -200) || slab.Bounds().Max != maths.V(200, 1, 200) {
		t.Fatal("glow bounds")
	}
	_, thicker, err := NewRingGlow(glowing, 4, 50, xrays, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !close(thicker.At(point).Scale(2), first, 1e-12) {
		t.Fatal("thickness changed integrated emission")
	}
	double, err := NewXrays(2e10, 30)
	if err != nil {
		t.Fatal(err)
	}
	_, brighter, _ := NewRingGlow(glowing, 2, 50, double, 1)
	if !close(brighter.At(point), first.Scale(2), 1e-12) {
		t.Fatal("x-ray power was not linear")
	}
	if shine.At(maths.Vec{}) != (maths.Light{}) || slab.Density(maths.V(250, 0, 0)) != 0 {
		t.Fatal("glow outside ring")
	}
	for _, shares := range []map[string]float64{{"unknown": .2}, {"csi-tl": -1}, {"csi-tl": 2}} {
		if _, err := ring.Glowing(shares); err == nil {
			t.Fatal("invalid phosphor")
		}
	}
	plain, err := ring.Glowing(nil)
	if err != nil || plain.glow.yield != 0 {
		t.Fatal("empty phosphor recipe")
	}
	_, dark, _ := NewRingGlow(plain, 2, 50, xrays, 1)
	if dark.At(point) != (maths.Light{}) {
		t.Fatal("ice glows without phosphors")
	}
	for _, bad := range [][2]float64{{-1, 30}, {1, 0}, {1, 90}} {
		if _, err := NewXrays(bad[0], bad[1]); err == nil {
			t.Fatal("bad x-rays")
		}
	}
	for _, bad := range [][2]float64{{0, 50}, {1, 0}} {
		if _, _, err := NewRingGlow(ring, bad[0], bad[1], xrays, 1); err == nil {
			t.Fatal("bad glow slab")
		}
	}
}

func TestThermalClearingsAreDarkAtZeroAndBrighterWhenHotter(t *testing.T) {
	body := bareBody(t, 1, nil)
	for _, bad := range [][2]float64{{0, 300}, {300, 200}, {math.NaN(), 300}} {
		if _, err := body.Heated(bad[0], bad[1]); err == nil {
			t.Fatal("bad heat")
		}
	}
	warm, err := body.Heated(200, 800)
	if err != nil {
		t.Fatal(err)
	}
	face := NewFace(warm, maths.V(0, 1, 0), maths.V(0, 0, 1), maths.V(0, 0, 1e9), even(), 1)
	if radiance(5000, 0) != 0 || radiance(5000, -1) != 0 || face.glowed(0, 4000, 6000) != 0 {
		t.Fatal("cold blackbody emitted light")
	}
	if !(radiance(5000, 800) > radiance(5000, 200) && face.glowed(1, 4000, 6000) > face.glowed(.5, 4000, 6000) && face.glowed(.5, 4000, 6000) > 0) {
		t.Fatal("hotter clearing did not brighten")
	}
}

func TestStormRecipesKeepBoundsAndSeeds(t *testing.T) {
	storms, err := NewStorms(5, .5, 20, 2, 2, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if storms.jet(10) != 0 || storms.rough(10) != 1 {
		t.Fatal("calm jets")
	}
	jet, err := storms.Jetted(30, 40)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(jet.jet(10)-30) > 1e-12 || math.Abs(jet.jet(-10)+30) > 1e-12 || jet.rough(10) >= jet.rough(0) {
		t.Fatal("jet direction or core turbulence")
	}
	for _, bad := range [][2]float64{{-1, 40}, {361, 40}, {30, 0}, {30, 181}} {
		if _, err := storms.Jetted(bad[0], bad[1]); err == nil {
			t.Fatal("bad jet")
		}
	}
	for _, bad := range [][4]float64{{-91, 0, 1, 1}, {0, 0, 0, 1}, {0, math.NaN(), 1, 1}, {0, 0, 1, math.NaN()}} {
		if _, err := NewSpot(bad[0], bad[1], bad[2], bad[3]); err == nil {
			t.Fatal("bad spot")
		}
	}
	for _, base := range []Storms{storms, jet} {
		made, err := base.Swarmed(9, 32, 1, 5, 2)
		if err != nil {
			t.Fatal(err)
		}
		same, _ := base.Swarmed(9, 32, 1, 5, 2)
		other, _ := base.Swarmed(10, 32, 1, 5, 2)
		if !reflect.DeepEqual(made, same) || reflect.DeepEqual(made.spots, other.spots) || len(base.spots) != 0 {
			t.Fatal("seed or value semantics")
		}
		for _, spot := range made.spots {
			if math.Abs(spot.latitude) > 75 || math.Abs(spot.longitude) > 180 || spot.radius < 1 || spot.radius > 5 || math.Abs(spot.turn) != 2.5 {
				t.Fatal("spot bounds")
			}
		}
	}
	for _, bad := range []struct {
		count               int
		small, large, slope float64
	}{{-1, 1, 2, 2}, {2001, 1, 2, 2}, {1, 0, 2, 2}, {1, 1, 31, 2}, {1, 1, 2, 1}} {
		if _, err := storms.Swarmed(1, bad.count, bad.small, bad.large, bad.slope); err == nil {
			t.Fatal("bad swarm")
		}
	}
	if jet.turning(0) != -1 || jet.turning(20) != 1 {
		t.Fatal("jet shear sign")
	}
	for _, bad := range []struct {
		turb, eddy, stretch, edge float64
		folds                     int
	}{{-1, 1, 1, 0, 0}, {0, 0, 1, 0, 0}, {0, 1, 0, 0, 0}, {0, 1, 1, -1, 0}, {0, 1, 1, 0, -1}, {0, 1, 1, 0, 9}} {
		if _, err := NewStorms(1, bad.turb, bad.eddy, bad.stretch, bad.edge, bad.folds, nil); err == nil {
			t.Fatal("bad storms")
		}
	}
	ways := frameOf(maths.V(0, 1, 0), maths.V(0, 0, 10))
	p := ways.toward(10, 0)
	moved := jet.sheared(p, ways, marks{})
	if math.Abs(moved.Len()-1) > 1e-12 || moved.Sub(p).Len() < .1 {
		t.Fatal("shear did not carry unit direction")
	}
	if storms.layers(1, maths.V(1, 2, 3), 1e6, 16) != 0 {
		t.Fatal("unresolved turbulence not filtered")
	}
	flow := storms.flow(5, maths.V(1, 2, 3), 0)
	if math.Abs(flow.Dot(maths.V(1, 2, 3))) > 1e-10 {
		t.Fatal("flow left sphere tangent")
	}
	folded, stretch := storms.folded(maths.V(0, 1, 0), 0, 1)
	if !finite(folded) || !(stretch > 0) {
		t.Fatal("polar fold lost its tangent")
	}
	for _, eye := range []maths.Vec{{}, maths.V(0, 10, 0)} {
		frame := frameOf(maths.V(0, 1, 0), eye)
		if math.Abs(frame.front.Len()-1) > 1e-12 || math.Abs(frame.east.Dot(frame.pole)) > 1e-12 {
			t.Fatal("polar frame")
		}
	}
}

type recordingFlow struct{ latitude, longitude, spin float64 }

func (f *recordingFlow) At(latitude, longitude float64) (float64, float64, float64) {
	f.latitude, f.longitude = latitude, longitude
	return 0, f.spin, 5
}

func TestFlowUsesRadiansAndFadesAtPolarWalls(t *testing.T) {
	flow := &recordingFlow{spin: 1}
	body := bareBody(t, 1, nil).Flowing(flow)
	face := NewFace(body, maths.V(0, 1, 0), maths.V(0, 0, 1), maths.V(0, 0, 1e9), even(), 1)
	for _, latitude := range []float64{30, -30, 80} {
		up := face.ways.toward(latitude, -90)
		origin, lift, marked := face.flowed(up)
		if math.Abs(flow.latitude-latitude*maths.Degree) > 1e-12 || math.Abs(flow.longitude-1.5*math.Pi) > 1e-12 {
			t.Fatal("flow coordinate units")
		}
		if latitude == 80 {
			if math.Abs(origin-80) > 1e-10 || lift != 0 {
				t.Fatal("wall fade")
			}
		} else if origin != 0 || marked.age != 5 || lift != map[bool]float64{true: 1, false: -1}[latitude < 0] {
			t.Fatal("hemisphere spin")
		}
	}
	if face.At(maths.V(0, 0, 6e7)).G <= 0 {
		t.Fatal("flowing face lost reflected light")
	}
}

func TestHazeAndRingTintsHoldTheirEndpoints(t *testing.T) {
	haze, err := NewHaze(.2, .8, 0)
	if err != nil {
		t.Fatal(err)
	}
	haze, err = haze.Tinted(.2)
	if err != nil {
		t.Fatal(err)
	}
	if haze.albedo.at(300) != .2 || haze.albedo.at(800) != .8 || math.Abs(haze.albedo.at(550)-.5) > 1e-12 {
		t.Fatal("haze tint")
	}
	ring := mustRing(t, 1, 2, .3, .8, 0)
	ring, err = ring.Tinted(.2)
	if err != nil {
		t.Fatal(err)
	}
	if ring.albedo.at(400) != .2 || ring.albedo.at(700) != .8 {
		t.Fatal("ring tint")
	}
	if _, err := haze.Tinted(2); err == nil {
		t.Fatal("invalid haze tint")
	}
	if _, err := ring.Tinted(-1); err == nil {
		t.Fatal("invalid ring tint")
	}
}

func TestStormColumnsBlendWithoutSeamsAndCookingChangesTheirLight(t *testing.T) {
	gas := mustGas(t, "clear", flat(0), .02)
	clear := layerOf(t, gas, 0, 100, Haze{})
	haze, err := NewHaze(.5, .5, 0)
	if err != nil {
		t.Fatal(err)
	}
	cloudy := layerOf(t, gas, 0, 100, haze)
	south, _ := NewZone(-90, 0, []Layer{clear})
	north, _ := NewZone(0, 90, []Layer{cloudy})
	body, err := NewBody(6e7, 25, 1, []Zone{south, north})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ cap, clearing []Layer }{{nil, []Layer{clear}}, {[]Layer{cloudy}, nil}} {
		if _, err := body.Marked(test.cap, test.clearing); err == nil {
			t.Fatal("empty marked column")
		}
	}
	for _, test := range []struct {
		column   []Layer
		strength float64
	}{{nil, .5}, {[]Layer{clear}, -1}, {[]Layer{clear}, 2}} {
		if _, err := body.Churned(test.column, test.strength); err == nil {
			t.Fatal("bad churn")
		}
	}
	body, err = body.Marked([]Layer{cloudy}, []Layer{clear})
	if err != nil {
		t.Fatal(err)
	}
	body, err = body.Churned([]Layer{cloudy}, .5)
	if err != nil {
		t.Fatal(err)
	}
	storms, err := NewStorms(1, .2, 10, 2, 2, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	anti, err := NewSpot(10, 0, 8, -1)
	if err != nil {
		t.Fatal(err)
	}
	cyclone, err := NewSpot(-10, 0, 8, -1)
	if err != nil {
		t.Fatal(err)
	}
	storms.spots = []Spot{anti, cyclone}
	body = body.Stormy(storms)
	if _, err := bareBody(t, 1, nil).Cooked([]Layer{cloudy}, 10); err == nil {
		t.Fatal("cooking accepted a zone with no layer")
	}
	body, err = body.Cooked([]Layer{cloudy}, 10)
	if err != nil {
		t.Fatal(err)
	}
	face := NewFace(body, maths.V(0, 1, 0), maths.V(0, 0, 1), maths.V(0, 0, 1e9), even(), 1)
	if face.capped < 0 || face.cleared < 0 || face.churned < 0 || len(face.ladders) != 2 {
		t.Fatal("special columns not prepared")
	}
	for _, latitude := range []float64{-10, -1, 0, 1, 10} {
		up := face.ways.toward(latitude, 0)
		light := face.Within(up.Scale(body.Radius()), 1)
		if !(light.G > 0) || math.IsInf(light.G, 0) {
			t.Fatal("storm lost finite reflected light")
		}
	}
	whirls := storms.vortices(face.ways)
	for i, whirl := range whirls {
		_, mark := swirled(whirl.middle, whirls)
		if mark.held != 1 || (i == 0 && mark.capped < .99) || (i == 1 && mark.cleared < .99) {
			t.Fatalf("vortex marks %+v", mark)
		}
		if !math.IsInf(whirl.within(whirl.middle.Neg()), 1) {
			t.Fatal("vortex crossed far hemisphere")
		}
	}
	for _, lat := range []float64{-1, 1} {
		_, _, share := face.blend(lat)
		if math.Abs(share-.25) > 1e-12 {
			t.Fatal("band edge is not symmetric")
		}
	}
	flow := &recordingFlow{}
	flowed := NewFace(body.Flowing(flow), maths.V(0, 1, 0), maths.V(0, 0, 1), maths.V(0, 0, 1e9), even(), 1)
	// The same material darkens as its air cooks; caps and churn stay separate.
	mixed := mix{here: 0, rung: 0}
	fresh := flowed.from(mixed, 0, 0, 1, 1, -1)
	mixed.rung = 14
	mixed.toward = .5
	cooked := flowed.from(mixed, 0, 0, 1, 1, -1)
	if !(cooked < fresh) {
		t.Fatalf("cooked haze %g, fresh %g", cooked, fresh)
	}
	columns := []Zone{}
	for _, pair := range [][2]float64{{-90, -30}, {-30, 30}, {30, 90}} {
		z, err := NewZone(pair[0], pair[1], []Layer{clear})
		if err != nil {
			t.Fatal(err)
		}
		columns = append(columns, z)
	}
	three, err := NewBody(1, 1, 1, columns)
	if err != nil {
		t.Fatal(err)
	}
	separate := NewFace(three, maths.V(0, 1, 0), maths.V(0, 0, 1), maths.V(0, 0, 10), even(), 1)
	if separate.neighbour(-20) != 0 || separate.neighbour(20) != 2 || separate.neighbour(-60) != 1 || separate.neighbour(60) != 1 {
		t.Fatal("nearest neighbouring zone")
	}
}

func TestRingletsVaryDepthWithoutChangingTheRingRecipe(t *testing.T) {
	plain := mustRing(t, 100, 200, .3, .8, 0)
	made, err := plain.Ringleted(7, .5, 10)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[float64]bool{}
	for i := range 100 {
		depth := made.share(100+float64(i), 0)
		if depth < 0 || math.IsNaN(depth) {
			t.Fatal("invalid ringlet depth")
		}
		seen[depth] = true
	}
	if len(seen) < 50 {
		t.Fatal("ringlets did not vary across the ring")
	}
	if plain.share(150, 0) != 1 {
		t.Fatal("ringlets changed the original ring")
	}
	if _, err := plain.Ringleted(7, -1, 10); err == nil {
		t.Fatal("negative ringlet strength")
	}
	body := bareBody(t, 1, nil)
	ring := mustRing(t, 7e7, 8e7, .3, .8, 0)
	body, err = body.Ringed([]Ring{ring})
	if err != nil {
		t.Fatal(err)
	}
	rings := body.Rings()
	rings[0] = Ring{}
	if got := body.Rings(); len(got) != 1 || got[0].inner != 7e7 {
		t.Fatal("caller changed internal rings")
	}
}

func TestAirglowReachesTheNightFaceOnlyInsideItsBand(t *testing.T) {
	body, err := bareBody(t, 1, nil).Airglowing(2, 10, .2)
	if err != nil {
		t.Fatal(err)
	}
	face := NewFace(body, maths.V(0, 1, 0), maths.V(0, 0, -1), maths.V(0, 0, 1e9), even(), 1)
	if light := face.At(maths.V(0, 0, body.Radius())); !(light.G > 0) {
		t.Fatal("sodium airglow never reached the night face")
	}
	outside := face.ways.toward(30, 0).Scale(body.Radius())
	if light := face.At(outside); light != (maths.Light{}) {
		t.Fatalf("airglow escaped its band: %+v", light)
	}
}
