package manufacture

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The lander's sizes, as venera-lander.yaml gives them and where it says
// they were read or picked.
const (
	veneraRingRadius = 0.9   // read: Schroeder et al., Figure 17, impact ring radius
	veneraRingTube   = 0.09  // picked: by eye from Figure 1
	veneraLegCount   = 18    // read: "18 legs that connected the payload module to the impact ring"
	veneraModuleSize = 1.0   // read: "roughly one meter in diameter"
	veneraPlateWide  = 2.15  // read: Figure 17, drag plate radius, taken as across
	veneraPlateThick = 0.03  // picked: by eye from Figure 1
	veneraTop        = 1.97  // derived: ring tube, payload height, the module's width, plate and can
	veneraNear       = 1e-9  // picked: sums of a few lengths, to rounding
	veneraLegSlack   = 1e-12 // picked: how near the module's surface a leg's end must be
)

// The lander stands on the ground: its ring lowest at nought, the module
// clear of the ground, the plate on the module's top and the can on the
// plate, the whole as tall as the can's top and as wide as the plate.
func TestTheVeneraLanderStandsOnItsRing(t *testing.T) {
	made := mustMake(t, "venera-lander", 5)
	box := made.Bounds()
	if math.Abs(box.Min.Y) > veneraNear || math.Abs(box.Max.Y-veneraTop) > veneraNear ||
		!(box.Max.X > veneraPlateWide/2) {
		t.Errorf("the lander's box is %+v, want nought to %v up and past "+
			"the plate, %v out, to the arm on the ground", box, veneraTop,
			veneraPlateWide/2)
	}
	if every := partBounds(veneraStood("quick")); every.Max.X > box.Max.X+veneraNear ||
		every.Max.Z > box.Max.Z+veneraNear || every.Min.X < box.Min.X-veneraNear ||
		every.Min.Z < box.Min.Z-veneraNear {
		t.Errorf("the lander's parts reach %+v, outside its box %+v", every, box)
	}
	low, high := veneraSpans(t, made)
	if math.Abs(low["impact ring"]) > veneraNear {
		t.Errorf("the ring's lowest is %v, want the ground", low["impact ring"])
	}
	if !(low["payload module"] > 0) {
		t.Errorf("the module's underside is %v up: not clear of the ground",
			low["payload module"])
	}
	if math.Abs(high["payload module"]-low["drag plate"]) > veneraNear ||
		math.Abs(low["drag plate"]+veneraPlateThick-low["parachute can"]) > veneraNear ||
		!(high["drag plate"] < veneraTop) ||
		math.Abs(high["parachute can"]-veneraTop) > veneraNear {
		t.Errorf("the module ends at %v, the plate spans %v to %v, the can "+
			"%v to %v: want the plate on the module, the can on the plate's "+
			"middle and its top at %v", high["payload module"],
			low["drag plate"], high["drag plate"], low["parachute can"],
			high["parachute can"], veneraTop)
	}
}

// The plate is dished: its underside rises from its middle to its rim,
// and the lip round the rim is its highest point.
func TestTheVeneraPlateIsDishedWithALip(t *testing.T) {
	size := veneraRead
	profile := size.plateProfile()
	rim := size.Plate.Across / 2
	highest := 0.0
	for _, corner := range profile {
		highest = math.Max(highest, corner[1])
	}
	if !(size.plateUnder(rim) > size.plateUnder(0)) {
		t.Errorf("the plate's underside is %v at the rim and %v in the "+
			"middle: not dished", size.plateUnder(rim), size.plateUnder(0))
	}
	if math.Abs(highest-size.Plate.Dish-size.Plate.Lip) > veneraNear {
		t.Errorf("the plate's highest is %v, want the lip's top, %v",
			highest, size.Plate.Dish+size.Plate.Lip)
	}
}

// Every stay leaves the module's surface and ends under the plate, not
// through it, and no two stays are the same stay. The anti-test of a
// plate floating on nothing.
func TestEveryVeneraStayHoldsThePlate(t *testing.T) {
	size := veneraRead
	stays := veneraPartsOf(t, veneraStay)
	if len(stays) != size.Plate.Stays {
		t.Fatalf("the plate has %d stays, want %d", len(stays), size.Plate.Stays)
	}
	middle := maths.Up.Scale(size.moduleMiddle())
	starts := map[maths.Vec]bool{}
	for index, each := range stays {
		start := each.space.Origin
		starts[start] = true
		if math.Abs(start.Sub(middle).Len()-veneraModuleSize/2) > veneraLegSlack {
			t.Errorf("stay %d starts %v from the module's middle", index,
				start.Sub(middle).Len())
		}
		if box := partBounds([]part{each}); !(box.Max.Y <= size.plateUnder(
			size.Plate.Across/2)+veneraNear) {
			t.Errorf("stay %d reaches %v, through the plate", index, box.Max.Y)
		}
	}
	if len(starts) != size.Plate.Stays {
		t.Errorf("the stays start at %d places, want %d", len(starts),
			size.Plate.Stays)
	}
}

// A taller can makes a taller lander: the numbers the test holds are the
// file's, and not a coincidence of the shape.
func TestAnotherCanMakesAnotherLanderHeight(t *testing.T) {
	taller := strings.Replace(string(veneraLanderRaw), "height: 0.7", "height: 0.8", 1)
	size, err := readVenera([]byte(taller))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(size.extent().Max.Y-veneraTop) < veneraNear {
		t.Errorf("a can 0.1 taller left the lander at %v", veneraTop)
	}
}

// veneraSpans is the lowest and the highest each named part of a made
// lander reaches.
func veneraSpans(t *testing.T, made good) (map[string]float64, map[string]float64) {
	t.Helper()
	parts, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	low := map[string]float64{}
	high := map[string]float64{}
	for _, each := range parts {
		name := each.Material.Name
		inside := each.Shape.Bounds()
		at := each.At(0).Origin.Y
		if _, seen := low[name]; !seen {
			low[name], high[name] = math.Inf(1), math.Inf(-1)
		}
		low[name] = math.Min(low[name], at+inside.Min.Y)
		high[name] = math.Max(high[name], at+inside.Max.Y)
	}
	return low, high
}

// There are as many legs as the file says, and each is one straight
// strut from the ring's middle circle, on the ground side, to the
// module's surface, higher and nearer the middle.
func TestEveryVeneraLegRunsFromTheRingToTheModule(t *testing.T) {
	size := veneraRead
	if size.Legs.Count != veneraLegCount {
		t.Fatalf("the file has %d legs, want %d", size.Legs.Count, veneraLegCount)
	}
	middle := maths.Up.Scale(size.moduleMiddle())
	for leg := range size.Legs.Count {
		from, to := size.legEnds(leg)
		radial := math.Hypot(from.X, from.Z)
		if math.Abs(radial-veneraRingRadius) > veneraNear ||
			math.Abs(from.Y-veneraRingTube) > veneraNear {
			t.Errorf("leg %d starts at %+v, not on the ring's middle circle", leg, from)
		}
		if reach := to.Sub(middle).Len(); math.Abs(reach-veneraModuleSize/2) > veneraLegSlack {
			t.Errorf("leg %d ends %v from the module's middle, want %v",
				leg, reach, veneraModuleSize/2)
		}
		if !(to.Y > from.Y) || !(math.Hypot(to.X, to.Z) < radial) {
			t.Errorf("leg %d does not go up and inward: %+v to %+v", leg, from, to)
		}
	}
}

// The legs are not all one leg: they stand three to a foot, each foot
// at its own place round the ring, and each leg meets the module at its
// own place. The anti-test of the above.
func TestVeneraLegsAreEvenlyRound(t *testing.T) {
	size := veneraRead
	feet := map[maths.Vec]int{}
	tops := map[maths.Vec]bool{}
	for leg := range size.Legs.Count {
		from, to := size.legEnds(leg)
		feet[from]++
		tops[to] = true
	}
	if len(feet) != size.Legs.Count/veneraLegsPerFoot {
		t.Errorf("the legs stand on %d feet, want %d", len(feet),
			size.Legs.Count/veneraLegsPerFoot)
	}
	for foot, legs := range feet {
		if legs != veneraLegsPerFoot {
			t.Errorf("the foot at %+v has %d legs, want %d", foot, legs,
				veneraLegsPerFoot)
		}
	}
	if len(tops) != size.Legs.Count {
		t.Errorf("the legs meet the module at %d places, want %d",
			len(tops), size.Legs.Count)
	}
	_, oppositeTo := size.legEnds(size.Legs.Count / 2)
	_, firstTo := size.legEnds(0)
	if math.Abs(oppositeTo.X+firstTo.X) > veneraNear ||
		math.Abs(oppositeTo.Z+firstTo.Z) > veneraNear {
		t.Errorf("opposite legs end at %+v and %+v: not opposite", oppositeTo, firstTo)
	}
}

// Everything it wears is a matte coat: no gloss, no bare metal and no
// film, one look for every part named, each with its own name.
func TestTheVeneraLanderIsPaintedMatte(t *testing.T) {
	dressed, err := veneraRead.looks()
	if err != nil {
		t.Fatal(err)
	}
	for where, look := range dressed {
		if look.Gloss != 0 || look.Metal != nil || look.Film != nil {
			t.Errorf("look %d: gloss %v, metal %v and film %v, want a "+
				"matte coat", where, look.Gloss, look.Metal, look.Film)
		}
		if look.Name != veneraNames[where] {
			t.Errorf("look %d is named %q, want %q", where, look.Name,
				veneraNames[where])
		}
	}
	if len(dressed) != len(veneraNames) {
		t.Errorf("the lander wears %d looks, want %d", len(dressed),
			len(veneraNames))
	}
}

// The coats are not one coat: the module is not the can's colour, and
// the aerial is lighter than the can it is wound on. The anti-test of
// the above.
func TestTheVeneraCoatsAreNotAllOneColour(t *testing.T) {
	dressed, err := veneraRead.looks()
	if err != nil {
		t.Fatal(err)
	}
	module, can := *dressed[veneraModule], *dressed[veneraCan]
	module.Name, can.Name = "", ""
	if reflect.DeepEqual(module, can) {
		t.Error("the module and the can wear the same coat")
	}
	if !(dressed[veneraAerial].Colour.Luminance() >
		dressed[veneraCan].Colour.Luminance()) {
		t.Error("the aerial is no lighter than the can")
	}
	if !strings.Contains(string(veneraLanderRaw), "picked") {
		t.Error("the file does not say which sizes are picked")
	}
}

// The module wears one band round its middle, standing proud of it and
// lower than higher; two camera windows; and its fittings round the
// equator, every other one a box.
func TestTheVeneraModuleWearsItsBandWindowsAndFittings(t *testing.T) {
	size := veneraRead
	bands := veneraPartsOf(t, veneraBand)
	if len(bands) != 1 {
		t.Fatalf("the module wears %d bands, want one", len(bands))
	}
	span := partBounds(bands)
	middle := size.moduleMiddle()
	if !(span.Min.Y < middle && span.Max.Y > middle) ||
		!(middle-span.Min.Y > span.Max.Y-middle) ||
		math.Abs(span.Max.X-veneraModuleSize/2-size.Module.Proud) > veneraNear {
		t.Errorf("the band spans %v to %v and reaches %v out, want it round "+
			"%v, more below than above, proud by %v", span.Min.Y, span.Max.Y,
			span.Max.X, middle, size.Module.Proud)
	}
	if windows := veneraPartsOf(t, veneraWindow); len(windows) != veneraWindows {
		t.Errorf("the module has %d camera windows, want %d", len(windows),
			veneraWindows)
	}
	fittings := veneraPartsOf(t, veneraBox)
	boxes := 0
	for _, each := range fittings {
		if _, isBox := each.shape.(box); isBox {
			boxes++
		}
	}
	if len(fittings) != size.Module.Fittings ||
		boxes != (size.Module.Fittings+1)/veneraBoxEvery {
		t.Errorf("the equator has %d fittings, %d of them boxes, want %d "+
			"and %d", len(fittings), boxes, size.Module.Fittings,
			(size.Module.Fittings+1)/veneraBoxEvery)
	}
}

// The camera windows look down and out on opposite sides, not both the
// same way. The anti-test of the above.
func TestTheVeneraCamerasLookDownOnOppositeSides(t *testing.T) {
	windows := veneraPartsOf(t, veneraWindow)
	if len(windows) != veneraWindows {
		t.Fatalf("the module has %d camera windows", len(windows))
	}
	first, second := windows[0].space, windows[1].space
	for _, each := range []maths.Subspace{first, second} {
		if !(each.Orientation.Y.Y < 0) {
			t.Errorf("a camera looks along %+v, not down", each.Orientation.Y)
		}
	}
	if !(first.Origin.X*second.Origin.X < 0) {
		t.Errorf("the cameras stand at %+v and %+v, not on opposite sides",
			first.Origin, second.Origin)
	}
}

// The ring wears its teeth round its outer edge: as many as the file
// says, each above the ring's middle and inside its outer edge, each
// standing taller than it is thick.
func TestTheVeneraRingIsToothed(t *testing.T) {
	size := veneraRead
	var teeth []part
	for _, each := range veneraPartsOf(t, veneraRing) {
		if _, isTooth := each.shape.(instrumentPlate); isTooth {
			teeth = append(teeth, each)
		}
	}
	if len(teeth) != size.Ring.Teeth {
		t.Fatalf("the ring has %d teeth, want %d", len(teeth), size.Ring.Teeth)
	}
	for index, each := range teeth {
		span := partBounds([]part{each})
		out := math.Hypot(each.space.Origin.X, each.space.Origin.Z)
		if !(span.Min.Y > size.Ring.Tube) || !(out < size.ringOut()) ||
			!(out > size.Ring.Radius) ||
			math.Abs(span.Max.Y-span.Min.Y-size.Ring.Tooth.Tall) > veneraNear {
			t.Errorf("tooth %d spans %v to %v up, %v out: want above %v, "+
				"between %v and %v out, %v tall", index, span.Min.Y,
				span.Max.Y, out, size.Ring.Tube, size.Ring.Radius,
				size.ringOut(), size.Ring.Tooth.Tall)
		}
	}
}

// The teeth point up, not down or out: each one's highest point is its
// tip, over its middle. The anti-test of teeth lying flat.
func TestTheVeneraTeethPointUp(t *testing.T) {
	for index, each := range veneraPartsOf(t, veneraRing) {
		if _, isTooth := each.shape.(instrumentPlate); !isTooth {
			continue
		}
		up := each.space.Orientation.Out(maths.V(0, 0, -1))
		if math.Abs(up.Y-1) > veneraNear {
			t.Errorf("tooth %d points along %+v, not up", index, up)
		}
	}
}

// The arm lies flat on the ground out from the ring: two rails, its
// rungs, a hinge and a pad, level, its near end at the ring's wall and
// its far end beyond the ring, resting on the ground and not sunk into
// it or floating over it. Propped up from the ring, it read as steps.
func TestTheVeneraArmLiesFlatOnTheGround(t *testing.T) {
	size := veneraRead
	arm := veneraPartsOf(t, veneraArm)
	// two rails, the rungs, the hinge and the pad
	if want := 2 + size.Arm.Rungs + 1 + 1; len(arm) != want {
		t.Fatalf("the arm has %d parts, want %d", len(arm), want)
	}
	hinge, run := size.armRun()
	end := hinge.Add(run)
	near := math.Hypot(hinge.X, hinge.Z)
	if math.Abs(end.Y-size.Arm.Rail) > veneraNear || math.Abs(run.Y) > veneraNear ||
		!(math.Hypot(end.X, end.Z) > size.ringOut()) ||
		!(near > size.Ring.Radius && near < size.ringOut()) ||
		math.Abs(run.Len()-size.Arm.Length) > veneraNear {
		t.Errorf("the arm runs %+v from %+v: want %v long, level, from "+
			"the ring's wall out beyond the ring", run, hinge, size.Arm.Length)
	}
	if span := partBounds(arm); math.Abs(span.Min.Y) > veneraNear ||
		span.Max.Y > size.Arm.Rail+size.Arm.Rail+veneraNear {
		t.Errorf("the arm spans %v to %v up, want flat on the ground",
			span.Min.Y, span.Max.Y)
	}
}

// The lens caps lie on the ground beyond the ring, not on the lander,
// and not both in one place; and the chart's patches are four colours,
// not one. The anti-test of instruments drawn as one thing.
func TestTheVeneraCapsAndChartAreApart(t *testing.T) {
	size := veneraRead
	caps := veneraPartsOf(t, veneraCap)
	if len(caps) != veneraWindows {
		t.Fatalf("there are %d lens caps, want %d", len(caps), veneraWindows)
	}
	for index, each := range caps {
		out := math.Hypot(each.space.Origin.X, each.space.Origin.Z)
		if !(out > size.ringOut()) || math.Abs(each.space.Origin.Y) > veneraNear {
			t.Errorf("lens cap %d lies at %+v, not on the ground beyond "+
				"the ring", index, each.space.Origin)
		}
	}
	if caps[0].space.Origin.Sub(caps[1].space.Origin).Len() < veneraNear {
		t.Error("both lens caps lie in one place")
	}
	dressed := veneraRead.dressed
	colours := map[maths.Light]bool{}
	for _, patch := range veneraPatches {
		colours[dressed[patch].Colour] = true
		if found := veneraPartsOf(t, patch); len(found) != 1 {
			t.Errorf("patch %d is drawn %d times, want once", patch, len(found))
		}
	}
	if len(colours) != len(veneraPatches) {
		t.Errorf("the chart's patches are %d colours, want %d",
			len(colours), len(veneraPatches))
	}
}

// The ring's holes run round its outer wall at the tube's middle, each
// just proud of the wall; the skirt is a drum and its slats, round the
// module at the height the legs meet it.
func TestTheVeneraRingIsHoledAndTheModuleSkirted(t *testing.T) {
	size := veneraRead
	holes := veneraPartsOf(t, veneraHole)
	if len(holes) != size.Ring.Holes {
		t.Fatalf("the ring has %d holes, want %d", len(holes), size.Ring.Holes)
	}
	for index, each := range holes {
		span := partBounds([]part{each})
		// how far out from the ring's axis the drum's farthest corner is;
		// the corners of its box, turned round, reach farther than it does
		reach := 0.0
		for _, triangle := range each.skin() {
			for _, point := range triangle {
				reach = math.Max(reach, math.Hypot(point.X, point.Z))
			}
		}
		if !(reach > size.ringOut()) || !(reach < size.ringOut()+size.Ring.Hole) ||
			!(span.Min.Y < size.Ring.Tube && span.Max.Y > size.Ring.Tube) {
			t.Errorf("hole %d spans %+v, not on the ring's outer wall", index, span)
		}
	}
	skirt := veneraPartsOf(t, veneraSkirt)
	if len(skirt) != size.Skirt.Slats+1 {
		t.Errorf("the skirt has %d parts, want a drum and %d slats",
			len(skirt), size.Skirt.Slats)
	}
	_, top := size.legEnds(1)
	if span := partBounds(skirt); !(span.Min.Y < top.Y && span.Max.Y > top.Y) {
		t.Errorf("the skirt spans %v to %v, not round where the legs meet "+
			"the module, %v", span.Min.Y, span.Max.Y, top.Y)
	}
}

// The holes are not one hole drawn over and over: each is at its own
// place round the ring. The anti-test of the above.
func TestTheVeneraRingHolesAreSpreadRound(t *testing.T) {
	places := map[maths.Vec]bool{}
	for _, each := range veneraPartsOf(t, veneraHole) {
		places[each.space.Origin] = true
	}
	if len(places) != veneraRead.Ring.Holes {
		t.Errorf("the holes are at %d places, want %d", len(places),
			veneraRead.Ring.Holes)
	}
}

// The bolt rings hold as many bolts as the file says: one row round the
// plate's lip, two along the band, one round the cap; and every bolt on
// the band stands on its wall, out from the module.
func TestTheVeneraBoltRingsAreAllThere(t *testing.T) {
	size := veneraRead
	bolts := veneraPartsOf(t, veneraBolt)
	want := size.Bolts.Plate + size.Bolts.Band + size.Bolts.Band + size.Bolts.Cap
	if len(bolts) != want {
		t.Fatalf("there are %d bolts, want %d", len(bolts), want)
	}
	wall := veneraModuleSize/2 + size.Module.Proud
	onBand := 0
	for _, each := range bolts {
		out := math.Hypot(each.space.Origin.X, each.space.Origin.Z)
		if math.Abs(out-wall+size.Bolt/2) < veneraNear {
			onBand++
		}
	}
	if onBand != size.Bolts.Band+size.Bolts.Band {
		t.Errorf("%d bolts stand on the band's wall, want %d", onBand,
			size.Bolts.Band+size.Bolts.Band)
	}
}

// The plate's seams stand on its top and not under it or through it:
// the anti-test of seams drawn at the module's top.
func TestTheVeneraPlateSeamsStandOnItsTop(t *testing.T) {
	size := veneraRead
	for index, each := range veneraPartsOf(t, veneraPlate) {
		if index == 0 {
			continue // the plate itself
		}
		span := partBounds([]part{each})
		if !(span.Min.Y > size.moduleTop()) {
			t.Errorf("seam %d reaches down to %v, under the plate's top",
				index, span.Min.Y)
		}
	}
}

// The cables run outside the module, never through it: one loop and so
// many runs over the shoulder from each round fitting; the pipes run
// from under the plate to the module; the booms' heads stand clear of
// the ground.
func TestTheVeneraCablesPipesAndBoomsRunOutside(t *testing.T) {
	size := veneraRead
	cables := veneraPartsOf(t, veneraCable)
	round := size.Module.Fittings / veneraBoxEvery
	if len(cables) != 1+round*veneraCableSteps {
		t.Errorf("there are %d cable parts, want a loop and %d runs",
			len(cables), round*veneraCableSteps)
	}
	middle := maths.Up.Scale(size.moduleMiddle())
	for index, each := range cables[1:] {
		if reach := each.space.Origin.Sub(middle).Len(); !(reach >
			veneraModuleSize/2) {
			t.Errorf("cable run %d starts %v from the module's middle, "+
				"inside it", index, reach)
		}
	}
	pipes := veneraPartsOf(t, veneraPipe)
	if span := partBounds(pipes); len(pipes) != 2*len(veneraPipeTurns) ||
		span.Max.Y > size.moduleTop()+size.Plate.Dish+size.Pipe {
		t.Errorf("%d pipe runs reaching %v up, want %d under the plate",
			len(pipes), span.Max.Y, 2*len(veneraPipeTurns))
	}
	heads := veneraPartsOf(t, veneraHead)
	if len(heads) != len(veneraBoomTurns) {
		t.Fatalf("there are %d boom heads, want %d", len(heads),
			len(veneraBoomTurns))
	}
	if span := partBounds(heads); !(span.Min.Y > size.ringTop()) {
		t.Errorf("a boom head reaches %v, down to the ring", span.Min.Y)
	}
}

// veneraAerialTurns is the aerial's turns round the can, without its
// lead: the parts of it turned about the axis.
func veneraAerialTurns(t *testing.T) []part {
	t.Helper()
	var turns []part
	for _, each := range veneraPartsOf(t, veneraAerial) {
		if _, turned := each.shape.(roundProfile); turned {
			turns = append(turns, each)
		}
	}
	return turns
}

// The aerial's lead winds down from its lowest turn to the plate, and
// the can carries its stiffeners and lugs; the lead ends on the plate
// and not under it.
func TestTheVeneraCanHasItsLeadStiffenersAndLugs(t *testing.T) {
	size := veneraRead
	aerial := veneraPartsOf(t, veneraAerial)
	if lead := len(aerial) - size.Can.Ribs; lead != veneraLeadSteps {
		t.Errorf("the aerial's lead is %d runs, want %d", lead, veneraLeadSteps)
	}
	can := veneraPartsOf(t, veneraCan)
	// the core and the cap, and the stiffeners and lugs
	if want := 2 + size.Can.Stiffeners + size.Can.Lugs; len(can) != want {
		t.Errorf("the can has %d parts, want %d", len(can), want)
	}
	if span := partBounds(aerial); !(span.Min.Y >= size.plateTop()-veneraNear) {
		t.Errorf("the aerial reaches %v, under the plate's top %v",
			span.Min.Y, size.plateTop())
	}
}

// veneraPartsOf is every part of the lander, as drawn, in one look.
func veneraPartsOf(t *testing.T, of slot) []part {
	t.Helper()
	var found []part
	for _, each := range veneraStood("quick") {
		if each.of == of {
			found = append(found, each)
		}
	}
	return found
}

// The aerial is wound round the can as many times as the file says,
// every turn above the plate and below the cap, reaching out as far as
// the can is wide.
func TestTheVeneraAerialIsWoundRoundTheCan(t *testing.T) {
	size := veneraRead
	ribs := veneraAerialTurns(t)
	if len(ribs) != size.Can.Ribs {
		t.Fatalf("the aerial has %d turns, want %d", len(ribs), size.Can.Ribs)
	}
	capFoot := size.overall() - size.Can.Cap
	for turn, each := range ribs {
		box := partBounds([]part{each})
		if !(box.Min.Y > size.plateTop()) || !(box.Max.Y < capFoot) {
			t.Errorf("turn %d spans %v to %v, want inside %v to %v",
				turn, box.Min.Y, box.Max.Y, size.plateTop(), capFoot)
		}
		if math.Abs(box.Max.X-size.Can.Across/2) > veneraNear {
			t.Errorf("turn %d reaches %v out, want %v", turn, box.Max.X,
				size.Can.Across/2)
		}
	}
}

// The aerial's turns are not one turn drawn many times: each stands at
// its own height. The anti-test of the above.
func TestTheVeneraAerialTurnsAreStacked(t *testing.T) {
	heights := map[float64]bool{}
	for _, each := range veneraAerialTurns(t) {
		heights[each.space.Origin.Y] = true
	}
	if len(heights) != veneraRead.Can.Ribs {
		t.Errorf("the aerial's turns stand at %d heights, want %d",
			len(heights), veneraRead.Can.Ribs)
	}
}

// Sizes that cannot be built are refused when the file is read, by the
// name of what is wrong.
func TestAVeneraLanderThatCannotBeBuiltIsRefused(t *testing.T) {
	whole := string(veneraLanderRaw)
	refused := map[string]struct{ was, is, names string }{
		"no shell":              {"shell: 2.4", "shell: 0", "shell"},
		"no ring":               {"radius: 0.9", "radius: 0", "ring radius"},
		"no tube":               {"tube: 0.09", "tube: 0", "ring tube"},
		"no strut":              {"strut: 0.03", "strut: 0", "leg strut"},
		"no module":             {"across: 1.0", "across: 0", "module across"},
		"no plate":              {"thick: 0.03", "thick: 0", "plate thick"},
		"no can":                {"height: 0.7", "height: 0", "can height"},
		"no mass":               {"mass: 660", "mass: 0", "mass"},
		"two legs":              {"count: 18", "count: 2", "legs"},
		"vessel as wide":        {"vessel: 0.8", "vessel: 1.0", "vessel"},
		"plate as wide as it":   {"across: 2.15", "across: 2.4", "plate"},
		"ring as wide as it":    {"radius: 0.9", "radius: 1.2", "ring"},
		"module under ground":   {"height: 0.65", "height: 0.4", "ground"},
		"read nowhere":          {"read: >-", "unread: >-", "unread"},
		"module with no height": {"height: 0.65", "height: 0", "module height"},
		"core as wide as it":    {"core: 0.7", "core: 0.8", "core"},
		"cap as tall as it":     {"cap: 0.06", "cap: 0.7", "cap"},
		"no turns of aerial":    {"ribs: 10", "ribs: 0", "ribs"},
		"no coat for aerial":    {"  aerial: {red", "  antenna: {red", "aerial has no coat"},
		"a coat over one":       {"aerial: {red: 0.8", "aerial: {red: 1.8", "aerial coat"},
		"a lip under the plate": {"lip: 0.06", "lip: 0.03", "lip"},
		"a rim past the hub":    {"rim: 0.05", "rim: 0.8", "rim"},
		"hub as wide as it":     {"hub: 0.6", "hub: 1.0", "hub"},
		"two stays":             {"stays: 12", "stays: 2", "stays"},
		"window as wide":        {"window: 0.09", "window: 0.14", "window"},
		"legs not in threes":    {"count: 18", "count: 19", "each foot"},
		"an arm too thick":      {"rail: 0.008", "rail: 0.09", "arm"},
		"no holes":              {"holes: 72", "holes: 0", "holes"},
		"no lugs":               {"lugs: 4", "lugs: 0", "lugs"},
		"no bolts on the plate": {"plate: 48", "plate: 0", "bolt rings"},
		"no bolt size":          {"bolt: 0.008", "bolt: 0", "bolt"},
		"a hole as big as it":   {"hole: 0.012", "hole: 0.09", "ring hole"},
		"a skirt as wide":       {"across: 0.86", "across: 1.0", "skirt"},
		"an arm with no rungs":  {"rungs: 9", "rungs: 0", "rungs"},
		"a cap too deep":        {"deep: 0.07", "deep: 0.2", "lens caps"},
		"a missing patch":       {"  chart blue: {red", "  chart navy: {red", "chart blue has no coat"},
		"no teeth":              {"teeth: 90", "teeth: 0", "teeth"},
		"no ring width":         {"wide: 0.22", "wide: 0", "ring wide"},
		"a wide ring":           {"wide: 0.22", "wide: 0.6", "ring"},
		"no fittings":           {"fittings: 12", "fittings: 0", "fittings"},
		"a box clear of it":     {"box: {wide: 0.1", "box: {wide: 0.6", "box"},
	}
	for name, each := range refused {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		edited := strings.Replace(whole, each.was, each.is, 1)
		_, err := readVenera([]byte(edited))
		if err == nil {
			t.Errorf("%s: read without refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: refused as %q, which does not name %q", name, err, each.names)
		}
	}
}

// Sizes one step inside each limit are read: the anti-test of the
// refusals above, so that each refusal is the limit and not the change.
func TestAVeneraLanderOneStepInsideItsLimitsIsRead(t *testing.T) {
	whole := string(veneraLanderRaw)
	accepted := map[string]struct{ was, is string }{
		"three legs":            {"count: 18", "count: 3"},
		"a small strut":         {"strut: 0.03", "strut: 0.01"},
		"vessel just inside":    {"vessel: 0.8", "vessel: 0.99"},
		"plate just inside":     {"across: 2.15", "across: 2.39"},
		"ring just inside":      {"radius: 0.9", "radius: 1.08"},
		"one tooth":             {"teeth: 90", "teeth: 1"},
		"six legs":              {"count: 18", "count: 6"},
		"one rung":              {"rungs: 9", "rungs: 1"},
		"a shallow cap":         {"deep: 0.07", "deep: 0.01"},
		"module just off floor": {"height: 0.65", "height: 0.42"},
		"a small shell":         {"shell: 2.4", "shell: 2.2"},
		"one turn of aerial":    {"ribs: 10", "ribs: 1"},
		"core just inside":      {"core: 0.7", "core: 0.79"},
		"a white aerial":        {"aerial: {red: 0.8", "aerial: {red: 1"},
		"three stays":           {"stays: 12", "stays: 3"},
		"one fitting":           {"fittings: 12", "fittings: 1"},
		"a lip just above":      {"lip: 0.06", "lip: 0.031"},
	}
	for name, each := range accepted {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		edited := strings.Replace(whole, each.was, each.is, 1)
		if _, err := readVenera([]byte(edited)); err != nil {
			t.Errorf("%s: refused as %v", name, err)
		}
	}
	if _, err := readVenera([]byte(whole)); err != nil {
		t.Errorf("the file as it stands is refused: %v", err)
	}
}
