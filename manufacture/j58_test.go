package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The engine's numbers as the sources give them, held here apart from
// the sizes file, so that the file is checked against them.
const (
	j58LengthRead   = 180 * j58Inch // read: 180 in, the 1966 engine register
	j58DiameterRead = 50 * j58Inch  // read: 50 in, the same page
	j58EnvelopeRead = 57 * j58Inch  // read: 4 ft 9 in, all-aero
	j58Inch         = 0.0254        // defined: an inch is 25.4 mm
	j58Stages       = 9             // read: "a nine-stage compressor", TM-104330 page 6
	j58Turbines     = 2             // read: "a two-stage turbine", page 6
	j58Cans         = 8             // read: "an eight-can combustor", page 6
	j58Tubes        = 6             // read: "six bleed-bypass tubes", page 6
	j58BleedStage   = 4             // read: "bled from the fourth stage", page 6
	j58Pressure     = 8.8           // read: overall pressure ratio, Wikipedia
	j58Near         = 1e-9          // picked: sums of a few lengths, to rounding
	j58Rounded      = 1e-3          // picked: the file gives the envelope to the millimetre
	j58Area         = 1e-10         // picked: square metres below which a face is a sliver
)

// j58Details is every detail the catalogue's menu gives the engine.
var j58Details = []string{"hull", "clumps", "quick", "demo"}

// j58Parts is the engine's parts lying alone at a detail, none shared.
func j58Parts(detail string) []part {
	return j58Read.laid(detail).object.parts
}

// j58PartsOf is the engine's parts of one look, at quick detail.
func j58PartsOf(of slot) []part {
	var found []part
	for _, each := range j58Parts("quick") {
		if each.of == of {
			found = append(found, each)
		}
	}
	return found
}

// j58Own is a point of the lying engine taken back into the engine's own
// space: its distance from the axis, and how far aft of the face it is.
func j58Own(point maths.Vec) (float64, float64) {
	own := j58Read.lying().In(point)
	return math.Hypot(own.X, own.Z), own.Y
}

// j58Reach is the furthest from the axis, the least aft and the most
// aft that some parts reach, in the engine's own space.
func j58Reach(parts []part) (float64, float64, float64) {
	out, front, back := spaceZero*1.0, math.Inf(1), math.Inf(-1)
	for _, each := range parts {
		for _, triangle := range each.skin() {
			for _, point := range triangle {
				radius, along := j58Own(point)
				out = math.Max(out, radius)
				front, back = math.Min(front, along), math.Max(back, along)
			}
		}
	}
	return out, front, back
}

// The file holds the numbers the sources give: the register's length
// and diameter, all-aero's envelope, and the report's counts.
func TestTheJ58FileHoldsItsCitedNumbers(t *testing.T) {
	size := j58Read
	if math.Abs(size.Length-j58LengthRead) > j58Near ||
		math.Abs(size.Diameter-j58DiameterRead) > j58Near ||
		math.Abs(size.Envelope-j58EnvelopeRead) > j58Rounded ||
		size.Pressure != j58Pressure {
		t.Errorf("the file gives %v long, %v and %v across at %v, want "+
			"%v, %v, %v and %v", size.Length, size.Diameter, size.Envelope,
			size.Pressure, j58LengthRead, j58DiameterRead, j58EnvelopeRead,
			j58Pressure)
	}
	counts := size.Counts
	if counts.Compressor != j58Stages || counts.Turbine != j58Turbines ||
		counts.Cans != j58Cans || counts.Tubes != j58Tubes ||
		counts.Bleed != j58BleedStage {
		t.Errorf("the file counts %+v", counts)
	}
}

// The cases run from the face to the engine's length, and their widest,
// a flange of the afterburner, is the read diameter.
func TestTheJ58CasesAreAsLongAndAsWideAsTheRegister(t *testing.T) {
	var cases []part
	for _, of := range []slot{j58Frame, j58Compressor, j58Burner,
		j58Turbine, j58Afterburner, j58Flange, j58Nozzle} {
		cases = append(cases, j58PartsOf(of)...)
	}
	out, front, back := j58Reach(cases)
	if math.Abs(out-j58DiameterRead/2) > j58Near ||
		math.Abs(front) > j58Near || math.Abs(back-j58LengthRead) > j58Near {
		t.Errorf("the cases reach %v from the axis, from %v to %v aft; "+
			"want %v, from nought to %v", out, front, back,
			j58DiameterRead/2, j58LengthRead)
	}
}

// Every part is inside the engine's mount, its envelope, and inside the
// made engine's box.
func TestEveryJ58PartIsInsideItsMountAndBox(t *testing.T) {
	mount := j58Read.mount()
	out, front, back := j58Reach(j58Parts("quick"))
	if out > mount.Radius+j58Near || front < mount.Front-j58Near ||
		back > j58Read.Length+j58Near {
		t.Errorf("the parts reach %v out, from %v to %v aft; the mount "+
			"is %+v", out, front, back, mount)
	}
	box := mustMake(t, "j58", 5).Bounds()
	every := partBounds(j58Parts("quick"))
	if every.Min.X < box.Min.X-j58Near || every.Max.X > box.Max.X+j58Near ||
		every.Min.Y < -j58Near || every.Max.Y > box.Max.Y+j58Near ||
		every.Min.Z < box.Min.Z-j58Near || every.Max.Z > box.Max.Z+j58Near {
		t.Errorf("the parts reach %+v, outside the box %+v", every, box)
	}
}

// j58Rings is the blades of one look sorted into rings by how far aft
// each stands.
func j58Rings(of slot) map[float64]int {
	rings := map[float64]int{}
	for _, each := range j58PartsOf(of) {
		_, along := j58Own(each.space.Origin)
		rings[math.Round(along/j58Rounded)*j58Rounded]++
	}
	return rings
}

// There are nine rings of compressor blades, as many to a ring as the
// file says, all inside the compressor.
func TestTheJ58HasNineRingsOfCompressorBlades(t *testing.T) {
	size := j58Read
	rings := j58Rings(j58Blade)
	if len(rings) != j58Stages {
		t.Fatalf("the compressor has %d rings, want %d", len(rings),
			j58Stages)
	}
	for stage, count := range size.Blades.Compressor {
		along := size.along(size.stageShare(stage))
		if got := rings[math.Round(along/j58Rounded)*j58Rounded]; got != count {
			t.Errorf("stage %d has %d blades, want %d", stage+1, got, count)
		}
		if along < size.Stations.Frame || along > size.Stations.Compressor {
			t.Errorf("stage %d stands %v aft, outside the compressor",
				stage+1, along)
		}
	}
}

// No blade reaches its case: each blade's tip is short of the case's
// inside where it stands, and its root inside the hub. The anti-test of
// a blade standing through the case.
func TestEveryJ58BladeClearsItsCase(t *testing.T) {
	size := j58Read
	for index, each := range j58PartsOf(j58Blade) {
		_, along := j58Own(each.space.Origin)
		share := (along - size.Stations.Frame) /
			(size.Stations.Compressor - size.Stations.Frame)
		out, _, _ := j58Reach([]part{each})
		if !(out < size.tip(share)) {
			t.Errorf("blade %d reaches %v, the case is %v", index, out,
				size.tip(share))
		}
		root, _ := j58Own(each.space.Origin)
		if !(root < size.hub(share)) {
			t.Errorf("blade %d's root is %v out, the hub %v", index, root,
				size.hub(share))
		}
	}
}

// The annulus narrows by the read pressure ratio: the face's area over
// the last stage's is the air's density ratio, and the hub rises and the
// tip falls stage by stage.
func TestTheJ58AnnulusNarrowsAsThePressureRises(t *testing.T) {
	size := j58Read
	area := func(share float64) float64 {
		tip, hub := size.tip(share), size.hub(share)
		return tip*tip - hub*hub
	}
	if ratio := area(0) / area(1); math.Abs(ratio-math.Pow(j58Pressure,
		1/j58Gamma)) > j58Near {
		t.Errorf("the annulus narrows by %v", ratio)
	}
	if math.Abs(size.hub(0)-size.Radii.Spinner) > j58Near {
		t.Errorf("the hub at the face is %v, the spinner %v", size.hub(0),
			size.Radii.Spinner)
	}
	for stage := 1; stage < size.Counts.Compressor; stage++ {
		before, after := size.stageShare(stage-1), size.stageShare(stage)
		if !(size.hub(after) > size.hub(before)) ||
			!(size.tip(after) < size.tip(before)) {
			t.Errorf("stage %d does not narrow", stage+1)
		}
	}
}

// The face is open: the vanes stand round the spinner, each from inside
// it out into the front frame's wall, and nothing else stands in front
// of the frame's aft end but the spinner's nose.
func TestTheJ58FaceHasItsVanesRoundTheSpinner(t *testing.T) {
	size := j58Read
	vanes := j58PartsOf(j58Vane)
	if len(vanes) != size.Vanes.Count {
		t.Fatalf("the face has %d vanes, want %d", len(vanes), size.Vanes.Count)
	}
	for index, each := range vanes {
		root, along := j58Own(each.space.Origin)
		out, front, _ := j58Reach([]part{each})
		if !(root < size.noseRadius(front)) || !(out > size.Radii.Face) ||
			!(out < size.Radii.Flange) || along > size.Stations.Frame {
			t.Errorf("vane %d runs from %v to %v out at %v aft", index,
				root, out, along)
		}
	}
	_, nose, _ := j58Reach(j58PartsOf(j58Spinner))
	if math.Abs(nose-j58NoseTip) > j58Near {
		t.Errorf("the spinner's point is %v aft, want %v", nose, j58NoseTip)
	}
}

// There are six bypass tubes, each from the bleed collar at the fourth
// stage, its foot in the compressor's wall, aft into the afterburner
// case's wall at the augmentor inlet, all evenly round.
func TestTheJ58HasSixBypassTubesFromTheFourthStage(t *testing.T) {
	size := j58Read
	share := size.stageShare(j58BleedStage - 1)
	port, enter := size.along(share), size.Stations.Turbine+size.Tubes.Enter
	starts := map[maths.Vec]bool{}
	for tube := range j58Tubes {
		run := size.tubeRun(size.tubeTurn(tube))
		first, last := run[0], run[len(run)-1]
		foot, end := math.Hypot(first.X, first.Z), math.Hypot(last.X, last.Z)
		if math.Abs(first.Y-port) > j58Near || math.Abs(last.Y-enter) > j58Near {
			t.Errorf("tube %d runs from %v to %v aft, want %v to %v", tube,
				first.Y, last.Y, port, enter)
		}
		if !(foot > size.tip(share)) || !(foot < size.tip(share)+size.Radii.Wall) ||
			!(end < size.afterburner()) ||
			!(end > size.afterburner()-size.Radii.Wall) {
			t.Errorf("tube %d's ends, %v and %v out, are not in the walls",
				tube, foot, end)
		}
		starts[first] = true
	}
	if len(starts) != j58Tubes || len(j58PartsOf(j58Tube)) != j58Tubes*9 {
		tubeParts := len(j58PartsOf(j58Tube))
		t.Errorf("the tubes start at %d places in %d parts, want %d in %d",
			len(starts), tubeParts, j58Tubes, j58Tubes*9)
	}
	one, opposite := size.tubeRun(size.tubeTurn(0))[0],
		size.tubeRun(size.tubeTurn(j58Tubes / 2))[0]
	if math.Abs(one.X+opposite.X) > j58Near || math.Abs(one.Z+opposite.Z) > j58Near {
		t.Errorf("opposite tubes start at %+v and %+v", one, opposite)
	}
}

// j58Span is how far aft a part starts and ends, and how far out it
// reaches, in the engine's own space.
type j58Span struct{ front, back, out float64 }

// j58Outside is how far out the widest of some parts reaches where it
// stands so far aft: the cases, flanges, bosses and manifold under the
// tubes.
func j58Outside(spans []j58Span, along float64) float64 {
	widest := 0.0
	for _, span := range spans {
		if along >= span.front && along <= span.back {
			widest = math.Max(widest, span.out)
		}
	}
	return widest
}

// The tubes' rise and run clear every case, flange and boss under them,
// and nothing of a tube reaches past the read envelope, which a tube's
// outside reaches to within its facets.
func TestEveryJ58TubeClearsTheCases(t *testing.T) {
	size := j58Read
	var spans []j58Span
	for _, of := range []slot{j58Frame, j58Compressor, j58Burner, j58Turbine,
		j58Afterburner, j58Flange, j58Boss, j58Manifold} {
		for _, each := range j58PartsOf(of) {
			out, front, back := j58Reach([]part{each})
			spans = append(spans, j58Span{front, back, out})
		}
	}
	radius := size.Tubes.Across / 2
	for tube := range j58Tubes {
		run := size.tubeRun(size.tubeTurn(tube))
		for step := 1; step < 3; step++ {
			for sample := range 20 {
				at := run[step].Sub(run[step+1]).Scale(float64(sample) / 20).Add(
					run[step+1])
				under := math.Hypot(at.X, at.Z) - radius
				if wall := j58Outside(spans, at.Y); !(under > wall) {
					t.Errorf("tube %d at %v aft is %v out, over a wall %v out",
						tube, at.Y, under, wall)
				}
			}
		}
	}
	out, _, _ := j58Reach(j58PartsOf(j58Tube))
	if out > j58EnvelopeRead/2+j58Rounded || out < j58EnvelopeRead/2-j58Rounded {
		t.Errorf("the tubes reach %v out, want the envelope's %v", out,
			j58EnvelopeRead/2)
	}
}

// What shows of the eight cans: eight bosses round the burner's drum,
// each at its own place, and the manifold that feeds them.
func TestTheJ58BurnerShowsItsEightCans(t *testing.T) {
	size := j58Read
	bosses := j58PartsOf(j58Boss)
	if len(bosses) != j58Cans || len(j58PartsOf(j58Manifold)) != 1 {
		manifolds := len(j58PartsOf(j58Manifold))
		t.Fatalf("the burner has %d bosses and %d manifolds, want %d and 1",
			len(bosses), manifolds, j58Cans)
	}
	seen := map[maths.Vec]bool{}
	for index, each := range bosses {
		root, along := j58Own(each.space.Origin)
		if math.Abs(along-size.burnerMiddle()) > j58Near ||
			!(root < size.Radii.Burner) {
			t.Errorf("boss %d stands %v out at %v aft", index, root, along)
		}
		seen[each.space.Origin] = true
	}
	if len(seen) != j58Cans {
		t.Errorf("the bosses stand at %d places, want %d", len(seen), j58Cans)
	}
}

// The turbine has two rings of blades inside its case, as many to a
// ring as the file says, each clear of the case.
func TestTheJ58HasTwoTurbineStages(t *testing.T) {
	size := j58Read
	rings := j58Rings(j58TurbineBlade)
	if len(rings) != j58Turbines {
		t.Fatalf("the turbine has %d rings, want %d", len(rings), j58Turbines)
	}
	for stage, count := range size.Turbine.Blades {
		along := size.turbineAlong(stage)
		if got := rings[math.Round(along/j58Rounded)*j58Rounded]; got != count {
			t.Errorf("turbine stage %d has %d blades, want %d", stage+1, got,
				count)
		}
	}
	out, front, back := j58Reach(j58PartsOf(j58TurbineBlade))
	if !(out < size.Radii.Turbine-size.Radii.Wall) ||
		front < size.Stations.Burner || back > size.Stations.Turbine {
		t.Errorf("the turbine blades reach %v out, from %v to %v aft", out,
			front, back)
	}
}

// The primary nozzle closes from the afterburner's end to its exit at
// the engine's length, with a rib on each flap and its actuators from
// the aft flange.
func TestTheJ58NozzleClosesToItsExit(t *testing.T) {
	size := j58Read
	out, front, back := j58Reach(j58PartsOf(j58Nozzle))
	if math.Abs(back-j58LengthRead) > j58Near ||
		!(front < size.Stations.Afterburner) ||
		math.Abs(out-size.nozzleOutside(size.Stations.Afterburner)) > j58Near {
		t.Errorf("the nozzle reaches %v out, from %v to %v aft", out, front,
			back)
	}
	exit := size.nozzleOutside(size.Length) - size.Radii.Wall
	if math.Abs(exit-size.Nozzle.Exit) > j58Near ||
		!(exit < size.afterburner()-size.Radii.Wall) {
		t.Errorf("the nozzle's exit is %v, want %v and narrower than the "+
			"case", exit, size.Nozzle.Exit)
	}
	flaps, actuators := j58PartsOf(j58Flap), j58PartsOf(j58Actuator)
	if len(flaps) != size.Nozzle.Flaps || len(actuators) != size.Nozzle.Actuators {
		t.Fatalf("the nozzle has %d flaps and %d actuators, want %d and %d",
			len(flaps), len(actuators), size.Nozzle.Flaps, size.Nozzle.Actuators)
	}
	_, front, back = j58Reach(flaps)
	if front < size.Stations.Afterburner || back > size.Length {
		t.Errorf("the flaps run from %v to %v aft, off the nozzle", front, back)
	}
	for index, each := range actuators {
		_, along := j58Own(each.space.Origin)
		if math.Abs(along-size.Stations.Afterburner-size.Radii.Seam/2) > j58Near {
			t.Errorf("actuator %d starts %v aft, not on the aft flange", index,
				along)
		}
	}
}

// What shows through the nozzle: the tail cone ends ahead of the spray
// rings, and every spray ring and flame holder is inside the nozzle's
// exit, one of each on every ring.
func TestTheJ58AfterburnerShowsThroughTheNozzle(t *testing.T) {
	size := j58Read
	burner := size.Afterburner
	rings := len(burner.Rings)
	if len(j58PartsOf(j58Spray)) != rings || len(j58PartsOf(j58Holder)) != rings ||
		len(j58PartsOf(j58Strut)) != burner.Struts*2 {
		sprays := len(j58PartsOf(j58Spray))
		holders := len(j58PartsOf(j58Holder))
		struts := len(j58PartsOf(j58Strut))
		t.Fatalf("the afterburner has %d spray rings, %d holders and %d "+
			"struts", sprays, holders,
			struts)
	}
	for _, of := range []slot{j58Spray, j58Holder} {
		out, front, _ := j58Reach(j58PartsOf(of))
		if !(out < size.Nozzle.Exit) || !(front > size.Stations.Turbine+burner.Cone) {
			t.Errorf("the %s reaches %v out from %v aft, outside the exit "+
				"%v or into the cone", j58Names[of], out, front, size.Nozzle.Exit)
		}
	}
	_, _, cone := j58Reach(j58PartsOf(j58Cone))
	if math.Abs(cone-size.Stations.Turbine-burner.Cone) > j58Near {
		t.Errorf("the tail cone ends %v aft", cone)
	}
}

// The engine is held by four lugs, two a side, level with its axis at
// the mount's two stations, each from inside its case's wall out to its
// pin, which the mount gives a host.
func TestTheJ58HasFourMountLugs(t *testing.T) {
	size := j58Read
	lugs, pins := j58PartsOf(j58Lug), size.mount().Pins
	if len(lugs) != 4 || len(pins) != 4 {
		t.Fatalf("the engine has %d lugs and %d pins, want 4 of each",
			len(lugs), len(pins))
	}
	for index, pin := range pins {
		if math.Abs(pin.Z) > j58Near ||
			math.Abs(math.Abs(pin.X)-size.Mounts.Reach) > j58Near {
			t.Errorf("pin %d stands at %+v, not level beside the axis",
				index, pin)
		}
		root, along := j58Own(lugs[index].space.Origin)
		out, _, _ := j58Reach([]part{lugs[index]})
		if math.Abs(along-pin.Y) > j58Near || !(root < size.caseOutside(along)) ||
			out < size.Mounts.Reach-j58Near {
			t.Errorf("lug %d runs from %v to %v out at %v aft", index, root,
				out, along)
		}
	}
	if pins[0].Y != size.Mounts.Forward || pins[2].Y != size.Mounts.Aft ||
		pins[0].X != -pins[1].X {
		t.Errorf("the pins stand at %+v", pins)
	}
}

// The engine flattens into the meshes a renderer takes at every detail:
// no two of its parts of one look share an edge run the same way.
func TestTheJ58FlattensAtEveryDetail(t *testing.T) {
	for _, detail := range j58Details {
		made, err := j58Engine{}.madeAt(Sow(5), detail)
		if err != nil {
			t.Fatalf("%s: %v", detail, err)
		}
		if _, err := made.Things(maths.Local()); err != nil {
			t.Errorf("%s: %v", detail, err)
		}
	}
}

// No face of any part, at any detail, is a sliver of no area.
func TestEveryJ58FaceHasArea(t *testing.T) {
	for _, detail := range j58Details {
		for index, each := range j58Parts(detail) {
			for _, triangle := range each.skin() {
				across := triangle[1].Sub(triangle[0]).Cross(
					triangle[2].Sub(triangle[0])).Len() / 2
				if !(across > j58Area) {
					t.Fatalf("%s: part %d has a face of %v", detail, index,
						across)
				}
			}
		}
	}
}

// The blades and vanes are kept as shared groups, each blade's mesh
// stored once a stage, and the cases stay whole parts, shaded smooth.
func TestTheJ58SharesItsBlades(t *testing.T) {
	made := mustMake(t, "j58", 5)
	for _, of := range []slot{j58Blade, j58Vane} {
		group := made.groups[of]
		if group == nil {
			t.Fatalf("the %s is not a shared group", j58Names[of])
		}
	}
	if made.groups[j58Afterburner] != nil {
		t.Errorf("the afterburner case is a group; a round case is a part")
	}
}

// Every part wears its own look, titanium's metal under it, and the file
// says what each is really made of.
func TestTheJ58WearsWhatItIsMadeOf(t *testing.T) {
	dressed, err := j58Read.looks()
	if err != nil {
		t.Fatal(err)
	}
	if len(dressed) != len(j58Names) {
		t.Errorf("the engine wears %d looks, want %d", len(dressed),
			len(j58Names))
	}
	for where, look := range dressed {
		if look.Name != j58Names[where] || look.Metal == nil {
			t.Errorf("look %d is %q with metal %v", where, look.Name,
				look.Metal)
		}
	}
}

// A longer engine makes a longer engine: the numbers the tests hold are
// the file's, and not a coincidence of the shape.
func TestAnotherLengthMakesAnotherJ58(t *testing.T) {
	longer := strings.Replace(string(j58Raw), "length: 4.572", "length: 4.8", 1)
	size, err := readJ58([]byte(longer))
	if err != nil {
		t.Fatal(err)
	}
	box := size.extentIn(size.lying())
	if math.Abs(box.Max.X-box.Min.X-j58LengthRead) < j58Near {
		t.Errorf("a longer engine left the box %v long", j58LengthRead)
	}
}

// Sizes that cannot be built are refused when the file is read, by the
// name of what is wrong.
func TestAJ58ThatCannotBeBuiltIsRefused(t *testing.T) {
	whole := string(j58Raw)
	refused := map[string]struct{ was, is, names string }{
		"no length":        {"length: 4.572", "length: 0", "length"},
		"no wall":          {"wall: 0.025", "wall: 0", "wall"},
		"no stages":        {"compressor: 9 ", "compressor: 0 ", "counts"},
		"a missing row":    {"compressor: 9 ", "compressor: 10 ", "rows"},
		"bleed past it":    {"bleed: 4", "bleed: 12", "bleeds"},
		"no tubes":         {"tubes: 6", "tubes: 0", "counts"},
		"a bladeless row":  {"[32, 34", "[0, 34", "stage 1"},
		"stations swapped": {"burner: 2.29", "burner: 1.7", "stations"},
		"past its length":  {"afterburner: 4.21", "afterburner: 4.6", "stations"},
		"a narrow envelope": {"envelope: 1.448", "envelope: 1.2",
			"envelope"},
		"a wide burner":   {"burner: 0.58", "burner: 0.62", "case"},
		"a widening face": {"exit: 0.47", "exit: 0.55", "compressor"},
		"a fat spinner":   {"spinner: 0.2", "spinner: 0.55", "spinner"},
		"a closed exit":   {"pressure: 8.8", "pressure: 80", "no room"},
		"long vanes":      {"chord: 0.12", "chord: 0.3", "vanes"},
		"no alloy said": {"alloy: \"not found\"\n  spinner",
			"alloy: \"\"\n  spinner", "alloy"},
		"an unknown metal": {"material: titanium\n    finish: frosted",
			"material: unobtainium\n    finish: frosted", "unobtainium"},
		"read nowhere": {"read: >-", "unread: >-", "unread"},
		"fat tubes":    {"across: 0.075", "across: 0.1", "bypass tubes"},
		"tubes past the end": {"enter: 0.25", "enter: 1.7",
			"bypass tubes"},
		"a short turbine row": {"blades: [60, 64]", "blades: [60]",
			"turbine"},
		"tall bosses":       {"height: 0.04", "height: 0.09", "can bosses"},
		"a fat turbine hub": {"hub: 0.42", "hub: 0.57", "turbine"},
		"a widening nozzle": {"exit: 0.45", "exit: 0.6", "nozzle"},
		"no flaps":          {"flaps: 16", "flaps: 0", "nozzle"},
		"rings swapped": {"rings: [0.25, 0.42]", "rings: [0.42, 0.25]",
			"rings"},
		"a long cone":   {"cone: 0.6", "cone: 0.7", "afterburner"},
		"no struts":     {"struts: 6", "struts: 0", "struts"},
		"a thick sheet": {"sheet: 0.008", "sheet: 0.03", "sheet"},
		"pins outside":  {"reach: 0.7", "reach: 0.73", "mounts"},
		"pins inside":   {"reach: 0.7", "reach: 0.6", "mounts"},
		"a mount in the frame": {"forward: 1.65", "forward: 0.2",
			"mounts"},
		"a mount on the flange": {"aft: 3.91", "aft: 4.2", "mounts"},
	}
	for name, each := range refused {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		edited := strings.Replace(whole, each.was, each.is, 1)
		_, err := readJ58([]byte(edited))
		if err == nil {
			t.Errorf("%s: read without refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: refused as %q, which does not name %q", name,
				err, each.names)
		}
	}
}

// Sizes one step inside each limit are read: the anti-test of the
// refusals above, so that each refusal is the limit and not the change.
func TestAJ58OneStepInsideItsLimitsIsRead(t *testing.T) {
	whole := string(j58Raw)
	accepted := map[string]struct{ was, is string }{
		"bleed at the last": {"bleed: 4", "bleed: 9"},
		"one tube":          {"tubes: 6", "tubes: 1"},
		"a burner just in":  {"burner: 0.58", "burner: 0.6"},
		"a slim spinner":    {"spinner: 0.2", "spinner: 0.1"},
		"short vanes":       {"chord: 0.12", "chord: 0.05"},
		"a wider envelope":  {"envelope: 1.448", "envelope: 1.5"},
		"thinner tubes":     {"across: 0.075", "across: 0.05"},
		"a turbine hub in":  {"hub: 0.42", "hub: 0.55"},
		"a wider exit":      {"exit: 0.45", "exit: 0.5"},
		"one ring":          {"rings: [0.25, 0.42]", "rings: [0.3]"},
		"pins further out":  {"reach: 0.7", "reach: 0.72"},
	}
	for name, each := range accepted {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		changed := strings.Replace(whole, each.was, each.is, 1)
		if _, err := readJ58([]byte(changed)); err != nil {
			t.Errorf("%s: refused as %v", name, err)
		}
	}
}
