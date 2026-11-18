package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The boat's numbers as the sources give them, held here apart from the
// sizes file, so that the file is checked against them.
const (
	alfaLengthMeasured    = 81.4  // measured: russianships-project-705 line 278
	alfaBeamMeasured      = 9.5   // measured: russianships line 279
	alfaDraughtMeasured   = 6.24  // measured: russianships line 280
	alfaDraughtRead       = 7.6   // read, not shelved: Wikipedia's Alfa page
	alfaSurfacedMeasured  = 2300  // measured: tonnes, russianships line 275
	alfaSubmergedMeasured = 3150  // measured: tonnes, russianships line 276
	alfaReserveMeasured   = 0.3   // measured: "nearly 30 percent", Thamm page 3
	alfaSeaWater          = 1.025 // chosen: tonnes a cubic metre of sea water, a common figure; not on the shelf
	alfaHullVolume        = 4412  // measured: cubic metres the drawn body of revolution holds, by alfaImmersed
	alfaVolumeNear        = 0.005 // chosen: the share by which the drawn volume may move before the test sees it
	alfaFloatNear         = 0.05  // chosen: how far a share of the volume may stand from the displacements' share
	alfaSlices            = 4000  // chosen: slices the hull is cut into to sum its volume
	alfaNear              = 1e-9  // chosen: sums of a few lengths, to rounding
	alfaArea              = 1e-10 // chosen: square metres below which a face is a sliver
)

// alfaDetails is every detail the catalogue's menu gives the boat.
var alfaDetails = []string{"hull", "clumps", "quick", "demo"}

// alfaParts is the boat's parts at a detail, none shared.
func alfaParts(detail string) []part {
	return alfaRead.laid(detail).object.parts
}

// alfaPartsOf is the boat's parts of one look, at quick detail.
func alfaPartsOf(of slot) []part {
	var found []part
	for _, each := range alfaParts("quick") {
		if each.of == of {
			found = append(found, each)
		}
	}
	return found
}

// alfaImmersed is the volume of the hull's body of revolution below a
// height over the keel, and its whole volume, summed slice by slice:
// each slice a circle round the axis, cut by the line.
func alfaImmersed(size alfaSizes, height float64) (float64, float64) {
	below, whole := 0.0, 0.0
	step := size.hullEnd() / alfaSlices
	for slice := range alfaSlices {
		radius := size.hullRadius((float64(slice) + 0.5) * step)
		over := math.Max(-radius, math.Min(radius, height-size.radius()))
		// the half chord as a product of differences, which is nought,
		// not a fused multiply's residue below it, at the circle's edge
		chord := math.Sqrt((radius - over) * (radius + over))
		below += (radius*radius*math.Acos(-over/radius) + over*chord) * step
		whole += math.Pi * radius * radius * step
	}
	return below, whole
}

// The file holds the numbers the sources give: russianships' length,
// beam, draught and displacements, Wikipedia's draught beside it, and
// Thamm's reserve buoyancy.
func TestTheAlfaFileHoldsItsCitedNumbers(t *testing.T) {
	size := alfaRead
	if size.Length != alfaLengthMeasured || size.Beam != alfaBeamMeasured ||
		size.Draughts["russianships"] != alfaDraughtMeasured ||
		size.Draughts["wikipedia"] != alfaDraughtRead ||
		size.Draught != "russianships" {
		t.Errorf("the file gives %v long, %v in beam, draughts %v, %q the "+
			"default", size.Length, size.Beam, size.Draughts, size.Draught)
	}
	if size.Displacement.Surfaced != alfaSurfacedMeasured ||
		size.Displacement.Submerged != alfaSubmergedMeasured ||
		size.Reserve != alfaReserveMeasured {
		t.Errorf("the file displaces %+v with %v in reserve",
			size.Displacement, size.Reserve)
	}
}

// The hull and its boss run from the bow at minus half the length to
// the stern at plus half, as wide as the beam, the keel at nought and
// the top at the beam's height.
func TestTheAlfaHullIsAsLongAndAsWideAsCited(t *testing.T) {
	box := partBounds(append(alfaPartsOf(alfaHull), alfaPartsOf(alfaBoss)...))
	half := alfaLengthMeasured / 2
	if math.Abs(box.Min.Z+half) > alfaNear ||
		math.Abs(box.Max.Z-half) > alfaNear ||
		math.Abs(box.Max.X-alfaBeamMeasured/2) > alfaNear ||
		math.Abs(box.Min.X+alfaBeamMeasured/2) > alfaNear ||
		math.Abs(box.Min.Y) > alfaNear ||
		math.Abs(box.Max.Y-alfaBeamMeasured) > alfaNear {
		t.Errorf("the hull reaches %+v", box)
	}
}

// The hull is one solid, closed and wound outward, so the water knows
// its inside from its outside; so is every other part.
func TestTheAlfaHullIsOneClosedOutwardSolid(t *testing.T) {
	hulls := alfaPartsOf(alfaHull)
	if len(hulls) != 1 {
		t.Fatalf("the hull is %d parts, want one", len(hulls))
	}
	for index, each := range alfaParts("quick") {
		skin := each.skin()
		if _, err := skin.meshed(); err != nil {
			t.Errorf("part %d, %s: %v", index, alfaNames[each.of], err)
		}
		if !(skin.held() > 0) {
			t.Errorf("part %d, %s, holds %v: wound inward", index,
				alfaNames[each.of], skin.held())
		}
	}
}

// The hull's volume is pinned: its body of revolution, the boss, sail
// and appendages left out, holds the drawn volume to within its
// tolerance, so a change to its form is seen here.
func TestTheAlfaHullVolumeIsPinned(t *testing.T) {
	_, whole := alfaImmersed(alfaRead, alfaBeamMeasured)
	if math.Abs(whole-alfaHullVolume) > alfaHullVolume*alfaVolumeNear {
		t.Errorf("the hull holds %v cubic metres, want %v", whole,
			alfaHullVolume)
	}
}

// No round hull of the measured length and beam holds as little as the
// boat displaces submerged: a spheroid that long and that wide, finer
// than a submarine's body, holds more than the displaced sea water. So
// the drawn hull is not fined down to the displacement; what it holds
// beyond it is taken as free-flooding space, the casing and the space
// between the hulls, which the displacement does not count. The
// anti-test of fitting the form to the displacement.
func TestNoRoundHullHoldsAsLittleAsTheAlfaDisplaces(t *testing.T) {
	half := alfaBeamMeasured / 2
	spheroid := 4.0 / 3 * math.Pi * half * half * alfaLengthMeasured / 2
	water := alfaSubmergedMeasured / alfaSeaWater
	_, whole := alfaImmersed(alfaRead, alfaBeamMeasured)
	if !(spheroid > water) || !(whole > spheroid) {
		t.Errorf("a spheroid of the boat's size holds %v, the boat "+
			"displaces %v, the hull holds %v", spheroid, water, whole)
	}
	t.Logf("spheroid %.0f, displaced %.0f, drawn hull %.0f cubic metres",
		spheroid, water, whole)
}

// The default draught floats the hull as the sources say: the share of
// its volume below the waterline is the surfaced displacement's share
// of the submerged, and what is left is Thamm's reserve buoyancy. The
// other draught, Wikipedia's, would leave half that reserve: the
// anti-test, and the reason 6.24 is the default.
func TestTheAlfaFloatsAtItsDefaultDraught(t *testing.T) {
	want := float64(alfaSurfacedMeasured) / alfaSubmergedMeasured
	below, whole := alfaImmersed(alfaRead, alfaRead.waterline())
	if math.Abs(below/whole-want) > alfaFloatNear ||
		math.Abs(spaceOne-below/whole-alfaReserveMeasured) > alfaFloatNear {
		t.Errorf("at %v the hull immerses %v of itself, want %v",
			alfaRead.waterline(), below/whole, want)
	}
	deeper, _ := alfaImmersed(alfaRead, alfaDraughtRead)
	if !(math.Abs(deeper/whole-want) > alfaFloatNear) {
		t.Errorf("at %v the hull immerses %v of itself, as near %v as the "+
			"default", alfaDraughtRead, deeper/whole, want)
	}
	t.Logf("immersed at %v: %.3f; at %v: %.3f; the displacements say %.3f",
		alfaRead.waterline(), below/whole, alfaDraughtRead, deeper/whole, want)
}

// The waterline is the named draught's height over the keel, given to
// the water by Floating, and the other name moves it; a kind that does
// not float says so.
func TestTheAlfaWaterlineIsItsNamedDraught(t *testing.T) {
	afloat, floats := Floating("alfa")
	if !floats || math.Abs(afloat.Waterline.InMetres()-alfaDraughtMeasured) >
		alfaNear || afloat.Hull != "hull" || afloat.Wet == nil ||
		afloat.Wet.Name != "wet hull" {
		t.Errorf("the alfa floats %v: %+v", floats, afloat)
	}
	other := strings.Replace(string(alfaRaw), "draught: russianships",
		"draught: wikipedia", 1)
	size, err := readAlfa([]byte(other))
	if err != nil {
		t.Fatal(err)
	}
	if size.waterline() != alfaDraughtRead {
		t.Errorf("wikipedia's draught puts the waterline at %v",
			size.waterline())
	}
	if _, floats := Floating("j58"); floats {
		t.Errorf("the j58 floats")
	}
}

// The wet hull is a look the boat has and no face wears: the water
// decides where the hull is wet.
func TestTheAlfaWetHullIsWornByNoFace(t *testing.T) {
	if alfaRead.dressed[alfaWet] == nil {
		t.Fatalf("the boat has no wet hull look")
	}
	if worn := alfaPartsOf(alfaWet); len(worn) != 0 {
		t.Errorf("%d parts wear the wet hull", len(worn))
	}
}

// The finish is a choice: bare titanium by default, its metal under it,
// and painted the file's colour when the file says so, the boss with
// the hull. Both have a wet look.
func TestTheAlfaFinishIsAChoice(t *testing.T) {
	bare := alfaRead.dressed
	if bare[alfaHull].Metal == nil || bare[alfaBoss].Metal == nil ||
		bare[alfaWet].Metal == nil {
		t.Errorf("the bare hull is not metal: %+v", bare[alfaHull])
	}
	painted := strings.Replace(string(alfaRaw), "finish: bare",
		"finish: painted", 1)
	size, err := readAlfa([]byte(painted))
	if err != nil {
		t.Fatal(err)
	}
	hull, want := size.dressed[alfaHull], maths.RGB(0.012, 0.012, 0.014)
	if hull.Metal != nil || hull.Colour != want ||
		size.dressed[alfaBoss].Colour != want ||
		size.dressed[alfaWet].Colour == want {
		t.Errorf("the painted hull is %+v", hull)
	}
}

func TestTheAlfaHeroKindIsBlackWithoutChangingTheDefault(t *testing.T) {
	ordinary, hero := mustMake(t, alfaKind, 3), mustMake(t, alfaHeroKind, 3)
	if ordinary.dressed[alfaHull].Metal == nil ||
		hero.dressed[alfaHull].Metal != nil ||
		hero.dressed[alfaHull].Colour != maths.RGB(.012, .012, .014) {
		t.Fatalf("default hull %+v, hero hull %+v; only hero should use the black coat",
			ordinary.dressed[alfaHull], hero.dressed[alfaHull])
	}
	ordinaryFloat, ordinaryOK := Floating(alfaKind)
	heroFloat, heroOK := Floating(alfaHeroKind)
	if !ordinaryOK || !heroOK || ordinaryFloat.Wet.Colour == heroFloat.Wet.Colour {
		t.Errorf("ordinary/hero afloat looks are not independently dressed: %+v %+v",
			ordinaryFloat, heroFloat)
	}
}

func TestTheAlfaKindsAreReportedAsSubmarines(t *testing.T) {
	for _, kind := range []maker{alfaBoat{}, alfaBoat{finish: "painted"}} {
		if family := familyOf(kind); family != "submarine" {
			t.Errorf("Alfa maker family is %q, want submarine", family)
		}
	}
}

// The sail stands on the hull: one solid, its foot wholly inside the
// hull, its front where the file says, its top the last level's height
// over the hull's top.
func TestTheAlfaSailStandsOnTheHull(t *testing.T) {
	size := alfaRead
	sails := alfaPartsOf(alfaSail)
	if len(sails) != 1 {
		t.Fatalf("the sail is %d parts, want one", len(sails))
	}
	box, sections := partBounds(sails), size.Sail.Sections
	top := size.Beam + sections[len(sections)-1].Up
	if math.Abs(box.Max.Y-top) > alfaNear ||
		math.Abs(box.Min.Y-size.Beam-sections[0].Up) > alfaNear ||
		math.Abs(box.Min.Z-size.Sail.Front+size.Length/2) > alfaNear {
		t.Errorf("the sail reaches %+v; its top should be %v", box, top)
	}
	for _, corner := range size.sailRings()[0] {
		aft := corner.Z + size.Length/2
		if !(math.Hypot(corner.X, corner.Y-size.radius()) <
			size.hullRadius(aft)) {
			t.Errorf("the sail's foot at %+v is outside the hull", corner)
		}
	}
}

// The sail is drawn smooth: its levels are the file's and a spline's
// between them, each higher than the last, and every corner of its
// sides between its foot and its crown, but its blunt trailing edge,
// is shaded by one normal, so no face's edge shows.
func TestTheAlfaSailIsSmooth(t *testing.T) {
	size := alfaRead
	levels := size.sailLevels()
	want := (len(size.Sail.Sections)-1)*alfaSailSteps + 1
	if len(levels) != want {
		t.Errorf("the sail has %d levels, want %d", len(levels), want)
	}
	for at := 1; at < len(levels); at++ {
		if !(levels[at].Up > levels[at-1].Up) {
			t.Errorf("level %d is at %v, under the last", at, levels[at].Up)
		}
	}
	sail := alfaPartsOf(alfaSail)[0]
	loft := sail.shape.(alfaLoft)
	skin := loft.skin(sail.fine)
	normals := loft.normals(skin)
	inner := map[maths.Vec]bool{}
	for _, ring := range loft.rings[1 : len(loft.rings)-1] {
		for index, corner := range ring {
			inner[corner] = index != alfaFoilSteps && index != alfaFoilSteps+1
		}
	}
	given := map[maths.Vec]maths.Vec{}
	for index, one := range skin {
		for corner, point := range one {
			normal := normals[index][corner]
			if seen, met := given[point]; inner[point] && met &&
				seen.Sub(normal).Len() > alfaNear {
				t.Fatalf("the sail at %+v is shaded %+v and %+v", point,
					seen, normal)
			}
			given[point] = normal
		}
	}
}

// The hull shows no rings: every corner of it from the bow's point to
// its end is shaded by one normal, whichever face it is drawn for, so
// the faces' edges along the nose and tail do not show; only the edge
// round the flat end of the hull, inside the boss, stays sharp.
func TestTheAlfaHullShowsNoRings(t *testing.T) {
	size := alfaRead
	hull := alfaPartsOf(alfaHull)[0]
	shape := hull.shape.(alfaTurned)
	skin := shape.skin(hull.fine)
	normals := shape.normals(skin)
	given := map[maths.Vec]maths.Vec{}
	for index, one := range skin {
		for corner, point := range one {
			if point.Y >= size.hullEnd()-alfaNear {
				continue // the flat end, and its sharp rim
			}
			normal := normals[index][corner]
			if seen, met := given[point]; met &&
				seen.Sub(normal).Len() > alfaNear {
				t.Fatalf("the hull at %+v is shaded %+v and %+v", point,
					seen, normal)
			}
			given[point] = normal
		}
	}
	bow := maths.V(0, 0, 0)
	if normal, met := given[bow]; !met || normal.Y > -1+alfaNear {
		t.Errorf("the bow's point is shaded %+v, want straight ahead",
			normal)
	}
}

// The fins, rudders, stern planes and bow planes are hydrofoils: each
// root's section is NACA's, sharp at no point, its nose at the axis of
// its chord, as thick as the file says at three tenths of its chord,
// and a hundredth of that at its trailing edge, and it is shaded
// smooth. The anti-test of a flat blade.
func TestTheAlfaFinsAndPlanesAreHydrofoils(t *testing.T) {
	size := alfaRead
	thick := map[slot]float64{alfaFin: size.Stern.Thick,
		alfaRudder: size.Stern.Thick * alfaFlapThick,
		alfaFlap:   size.Stern.Thick * alfaFlapThick,
		alfaPlane:  size.Planes.Thick}
	for of, through := range thick {
		for index, each := range alfaPartsOf(of) {
			if _, smooth := each.shape.(rounded); !smooth {
				t.Errorf("%s %d is shaded flat", alfaNames[of], index)
			}
			root := each.shape.(alfaLoft).rings[0]
			widest := 0.0
			for _, corner := range root {
				widest = math.Max(widest, math.Abs(corner.X))
			}
			trailing := math.Abs(root[alfaFoilSteps].X)
			if root[0].X != 0 || math.Abs(widest-through/2) > through/100 ||
				math.Abs(trailing-through*alfaNaca(1)) > alfaNear {
				t.Errorf("%s %d's section is %v at its nose, %v at its "+
					"widest and %v at its tail, for %v through",
					alfaNames[of], index, root[0].X, widest, trailing,
					through)
			}
		}
	}
	if math.Abs(alfaNaca(0.3)-0.5) > 0.001 || alfaNaca(0) != 0 {
		t.Errorf("the NACA section is %v at three tenths and %v at its nose",
			alfaNaca(0.3), alfaNaca(0))
	}
}

// The rescue sphere is inside the sail: every corner of it is inside
// the sail's section at its height, and it is as wide as the file says.
func TestTheAlfaRescueSphereIsInsideTheSail(t *testing.T) {
	size := alfaRead
	spheres := alfaPartsOf(alfaSphere)
	if len(spheres) != 1 {
		t.Fatalf("there are %d rescue spheres, want one", len(spheres))
	}
	for _, triangle := range spheres[0].skin() {
		for _, point := range triangle {
			half := size.sailHalf(point.Y, point.Z+size.Length/2)
			if !(math.Abs(point.X) < half) {
				t.Fatalf("the sphere at %+v is outside the sail, %v wide "+
					"there", point, half)
			}
		}
	}
	box := partBounds(spheres)
	if math.Abs(box.Max.Y-box.Min.Y-size.Sphere.Diameter) > alfaNear {
		t.Errorf("the sphere is %v tall, want %v", box.Max.Y-box.Min.Y,
			size.Sphere.Diameter)
	}
}

// alfaInside is whether a point of the boat's space is inside the hull.
func alfaInside(size alfaSizes, point maths.Vec) bool {
	return math.Hypot(point.X, point.Y-size.radius()) <
		size.hullRadius(point.Z+size.Length/2)
}

// alfaRoot is a lofted part's first ring, its root, in the boat's space.
func alfaRoot(each part) []maths.Vec {
	var root []maths.Vec
	for _, corner := range each.shape.(alfaLoft).rings[0] {
		root = append(root, each.space.Out(corner))
	}
	return root
}

// The stern is cruciform: four fins, a rudder behind each upright one
// and a stern plane behind each stabiliser, each rooted inside the hull,
// the fins reaching up, down and across as the file says; and a bow
// plane out to either side, rooted inside the hull too.
func TestTheAlfaHasACruciformStern(t *testing.T) {
	size := alfaRead
	counts := map[slot]int{alfaFin: 4, alfaRudder: 2, alfaFlap: 2,
		alfaPod: 2, alfaPlane: 2}
	for of, want := range counts {
		if got := len(alfaPartsOf(of)); got != want {
			t.Errorf("the boat has %d of its %s, want %d", got, alfaNames[of],
				want)
		}
	}
	for _, of := range []slot{alfaFin, alfaRudder, alfaFlap, alfaPlane} {
		for index, each := range alfaPartsOf(of) {
			for _, corner := range alfaRoot(each) {
				if !alfaInside(size, corner) {
					t.Errorf("%s %d's root at %+v is outside the hull",
						alfaNames[of], index, corner)
				}
			}
		}
	}
	fins := partBounds(alfaPartsOf(alfaFin))
	stern := size.Stern
	if math.Abs(fins.Max.Y-size.radius()-stern.Up) > 1e-6 ||
		math.Abs(size.radius()-fins.Min.Y-stern.Down) > 1e-6 ||
		math.Abs(fins.Max.X-stern.Across) > 1e-6 ||
		math.Abs(fins.Max.Z+size.Length/2-size.trailing()) > alfaNear {
		t.Errorf("the fins reach %+v", fins)
	}
	flaps := partBounds(append(alfaPartsOf(alfaRudder),
		alfaPartsOf(alfaFlap)...))
	if !(flaps.Min.Z > fins.Max.Z) {
		t.Errorf("the rudders start at %v, ahead of the fins' end, %v",
			flaps.Min.Z, fins.Max.Z)
	}
	planes := partBounds(alfaPartsOf(alfaPlane))
	if math.Abs(planes.Max.X-size.planeTip()) > 1e-6 {
		t.Errorf("the bow planes reach %v, want %v", planes.Max.X,
			size.planeTip())
	}
}

// A thruster pod holds each stabiliser's tip: every corner of the tip
// is inside a pod.
func TestTheAlfaPodsHoldTheStabilisersTips(t *testing.T) {
	size := alfaRead
	pods := alfaPartsOf(alfaPod)
	for index, each := range alfaPartsOf(alfaFin) {
		rings := each.shape.(alfaLoft).rings
		tip := rings[len(rings)-1]
		level := math.Abs(each.space.Out(tip[0]).Y - size.radius())
		if level > 1e-6 {
			continue // an upright fin
		}
		for _, corner := range tip {
			point := each.space.Out(corner)
			held := false
			for _, pod := range pods {
				own := pod.space.In(point)
				radius := math.Hypot(own.X, own.Z)
				held = held || (radius < size.Pods.Radius && own.Y > 0 &&
					own.Y < size.Pods.Long)
			}
			if !held {
				t.Errorf("fin %d's tip at %+v is in no pod", index, point)
			}
		}
	}
}

// Nothing fixed stands outside the measured beam or below the keel:
// only the bow planes, drawn out, reach past the beam.
func TestTheAlfaIsWithinItsBeamAndAboveItsKeel(t *testing.T) {
	var fixed []part
	for _, each := range alfaParts("quick") {
		if each.of != alfaPlane {
			fixed = append(fixed, each)
		}
	}
	box := partBounds(fixed)
	if box.Max.X > alfaBeamMeasured/2+alfaNear ||
		box.Min.X < -alfaBeamMeasured/2-alfaNear || box.Min.Y < -alfaNear {
		t.Errorf("the boat reaches %+v, past its beam or its keel", box)
	}
}

// alfaFromAxis is how far a point of the boat's space is from the hull's
// axis.
func alfaFromAxis(size alfaSizes, point maths.Vec) float64 {
	return math.Hypot(point.X, point.Y-size.radius())
}

// There is one propeller of as many blades as the file says, its tips
// on the diameter, its roots inside the boss, every blade on the boss's
// straight part and aft of the rudders and stern planes.
func TestTheAlfaHasOnePropeller(t *testing.T) {
	size := alfaRead
	blades := alfaPartsOf(alfaBlade)
	if len(blades) != size.Propeller.Blades {
		t.Fatalf("the propeller has %d blades, want %d", len(blades),
			size.Propeller.Blades)
	}
	flaps := partBounds(append(alfaPartsOf(alfaRudder),
		alfaPartsOf(alfaFlap)...))
	for index, each := range blades {
		out := 0.0
		for _, triangle := range each.skin() {
			for _, point := range triangle {
				out = math.Max(out, alfaFromAxis(size, point))
				aft := point.Z + size.Length/2
				if aft <= size.hullEnd() ||
					aft >= size.Length-size.Hull.Stern ||
					point.Z <= flaps.Max.Z {
					t.Fatalf("blade %d reaches %v aft, off the boss", index,
						aft)
				}
			}
		}
		if math.Abs(out-size.Propeller.Diameter/2) > 1e-9 {
			t.Errorf("blade %d reaches %v out, want %v", index, out,
				size.Propeller.Diameter/2)
		}
		for _, corner := range alfaRoot(each) {
			if !(alfaFromAxis(size, corner) < size.Hull.Stern) {
				t.Errorf("blade %d's root at %+v is outside the boss", index,
					corner)
			}
		}
	}
}

func TestTheAlfaHasSourcedFiveBladeMainAndTwoAuxiliaryScrews(t *testing.T) {
	if alfaRead.Propeller.Blades != 5 || alfaRead.Propeller.Skew != 0 {
		t.Fatalf("main screw has %d blades and %v degrees skew; want sourced five-blade non-skewback form",
			alfaRead.Propeller.Blades, alfaRead.Propeller.Skew)
	}
	auxiliary := alfaPartsOf(alfaAuxBlade)
	if alfaRead.Auxiliary.Blades != 2 || len(auxiliary) != 2*alfaRead.Auxiliary.Blades {
		t.Fatalf("auxiliary screws have %d blades each and %d blade meshes; want two each on two pods",
			alfaRead.Auxiliary.Blades, len(auxiliary))
	}
}

func TestTheAlfaDemoAddsPhotoDerivedDeckDetails(t *testing.T) {
	quick, demo := alfaRead.laid("quick").object.parts,
		alfaRead.laid("demo").object.parts
	count := func(parts []part, of slot) int {
		found := 0
		for _, each := range parts {
			if each.of == of {
				found++
			}
		}
		return found
	}
	for _, of := range []slot{alfaMast, alfaFitting} {
		if count(quick, of) != 0 || count(demo, of) == 0 {
			t.Errorf("%s counts quick=%d demo=%d; details belong to demo only",
				alfaNames[of], count(quick, of), count(demo, of))
		}
	}
}

func TestTheAlfaDemoMastsStandAboveTheSailCrown(t *testing.T) {
	demo := alfaRead.laid("demo").object.parts
	var masts []part
	for _, each := range demo {
		if each.of == alfaMast {
			masts = append(masts, each)
		}
	}
	if len(masts) != 4 {
		t.Fatalf("there are %d masts, want the four photo-derived antenna forms", len(masts))
	}
	top := alfaRead.Beam + alfaRead.Sail.Sections[len(alfaRead.Sail.Sections)-1].Up
	box := partBounds(masts)
	if box.Min.Y < top-alfaNear || box.Max.Y <= top {
		t.Errorf("mast bounds %+v do not rise from the sail crown at %v", box, top)
	}
}

// The blades are pitched as the file says: at every section, the chord
// from the leading edge to the middle of the trailing edge runs round
// and aft at the helix angle of the pitch at that radius.
func TestTheAlfaPropellerHasItsPitch(t *testing.T) {
	size := alfaRead
	screw := size.Propeller
	for index, ring := range size.blade().rings {
		lead := ring[0]
		trail := ring[alfaFoilSteps].Add(ring[alfaFoilSteps+1]).Scale(0.5)
		radius := math.Hypot(lead.X, lead.Y)
		round := radius * (math.Atan2(-trail.X, trail.Y) -
			math.Atan2(-lead.X, lead.Y))
		got := math.Atan2(trail.Z-lead.Z, round)
		want := math.Atan(screw.Pitch * screw.Diameter / (2 * math.Pi * radius))
		if math.Abs(got-want) > 1e-6 {
			t.Errorf("section %d at %v is pitched %v, want %v", index, radius,
				got, want)
		}
	}
}

// The boat flattens into the meshes a renderer takes at every detail.
func TestTheAlfaFlattensAtEveryDetail(t *testing.T) {
	for _, detail := range alfaDetails {
		made, err := alfaBoat{}.madeAt(Sow(5), detail)
		if err != nil {
			t.Fatalf("%s: %v", detail, err)
		}
		if _, err := made.Things(maths.Local()); err != nil {
			t.Errorf("%s: %v", detail, err)
		}
	}
}

// No face of any part, at any detail, is a sliver of no area.
func TestEveryAlfaFaceHasArea(t *testing.T) {
	for _, detail := range alfaDetails {
		for index, each := range alfaParts(detail) {
			for _, triangle := range each.skin() {
				across := triangle[1].Sub(triangle[0]).Cross(
					triangle[2].Sub(triangle[0])).Len() / 2
				if !(across > alfaArea) {
					t.Fatalf("%s: part %d has a face of %v", detail, index,
						across)
				}
			}
		}
	}
}

// A longer boat makes a longer boat: the numbers the tests hold are the
// file's, and not a coincidence of the shape.
func TestAnotherLengthMakesAnotherAlfa(t *testing.T) {
	longer := strings.Replace(string(alfaRaw), "length: 81.4",
		"length: 85", 1)
	size, err := readAlfa([]byte(longer))
	if err != nil {
		t.Fatal(err)
	}
	box := partBounds(size.laid("quick").object.parts)
	if math.Abs(box.Max.Z-box.Min.Z-alfaLengthMeasured) < alfaNear {
		t.Errorf("a longer boat left the box %v long", alfaLengthMeasured)
	}
}

// alfaRefused is a change to the sizes file, and a word the refusal of
// it names.
type alfaRefused struct{ was, is, names string }

// alfaRefusals is every change to the sizes file that must be refused.
var alfaRefusals = map[string]alfaRefused{
	"no length": {
		"length: 81.4", "length: 0", "length"},
	"no beam": {
		"beam: 9.5 ", "beam: 0 ", "beam"},
	"an unnamed draught": {
		"draught: russianships", "draught: lloyds", "lloyds"},
	"a draught past the hull": {
		"wikipedia: 7.6", "wikipedia: 9.5", "as deep"},
	"a long nose": {
		"nose: 14", "nose: 60", "does not fit"},
	"a wide stern": {
		"stern: 0.9", "stern: 4.8", "does not fit"},
	"a short boss": {
		"boss: 2.5", "boss: 0.9", "does not fit"},
	"a hollow nose": {
		"bluntness: 2", "bluntness: 0.5", "powers"},
	"an unknown finish": {
		"finish: bare", "finish: gilded", "gilded"},
	"no alloy said": {
		"alloy: \"not found\"", "alloy: \"\"", "alloy"},
	"an unknown metal": {
		"dry: {material: titanium, finish: weathered}",
		"dry: {material: unobtainium, finish: weathered}", "unobtainium"},
	"a colour past one": {
		"colour: [0.012, 0.012, 0.014]", "colour: [1.012, 0.012, 0.014]",
		"colour"},
	"read nowhere": {
		"read: >-", "unread: >-", "unread"},
	"a foot outside the hull": {
		"up: -1.6", "up: -0.2", "sail"},
	"levels out of order": {
		"up: 3, lead", "up: 0.2, lead", "sail"},
	"a sail on the nose": {
		"front: 24", "front: 2", "sail"},
	"a big sphere": {
		"diameter: 3.2", "diameter: 4.4", "sphere"},
	"a sphere through the top": {
		"up: 1.9", "up: 3.6", "sphere"},
	"a plane through the hull": {
		"thick: 0.4", "thick: 9", "bow planes"},
	"planes past the bow": {
		"aft: 9", "aft: 1", "bow planes"},
	"fins past the boss": {
		"aft: 67.5", "aft: 73", "stern"},
	"a tip longer than its root": {
		"tip: 3.6", "tip: 6.5", "stern"},
	"an upper fin in the hull": {
		"up: 4.2", "up: 2.5", "stern"},
	"stabilisers past the beam": {
		"across: 4.2", "across: 4.4", "beam"},
	"a lower fin below the keel": {
		"down: 4.2", "down: 4.9", "keel"},
	"thin pods": {
		"radius: 0.45", "radius: 0.2", "pods"},
	"short pods": {
		"long: 4.2", "long: 3", "pods"},
	"no blades": {
		"blades: 5", "blades: 0", "propeller"},
	"a propeller past the keel": {
		"diameter: 4.2", "diameter: 9.6", "propeller"},
	"a propeller in the boss": {
		"diameter: 4.2", "diameter: 1.7", "propeller"},
	"a propeller off the boss": {
		"aft: 0.7", "aft: 0.2", "propeller"},
	"broad blades": {
		"chord: 1.1", "chord: 2", "propeller"},
}

// Sizes that cannot be built are refused when the file is read, by the
// name of what is wrong.
func TestAnAlfaThatCannotBeBuiltIsRefused(t *testing.T) {
	whole := string(alfaRaw)
	for name, each := range alfaRefusals {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		_, err := readAlfa([]byte(strings.Replace(whole, each.was, each.is, 1)))
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

// alfaAccepted is every change one step inside a limit, which must be
// read: the anti-test of the refusals, so that each refusal is the limit
// and not the change.
var alfaAccepted = map[string]struct{ was, is string }{
	"the other draught":       {"draught: russianships", "draught: wikipedia"},
	"a painted hull":          {"finish: bare", "finish: painted"},
	"a draught just inside":   {"wikipedia: 7.6", "wikipedia: 9.4"},
	"a longer nose":           {"nose: 14", "nose: 23"},
	"a stern just in":         {"stern: 0.9", "stern: 1.1"},
	"a straight-sided taper":  {"taper: 1.8", "taper: 1"},
	"a smaller sphere":        {"diameter: 3.2", "diameter: 2.0"},
	"a taller sail":           {"up: 4.6", "up: 5.5"},
	"a sail further aft":      {"front: 24", "front: 28"},
	"stabilisers nearly out":  {"across: 4.2", "across: 4.29"},
	"a lower fin to the keel": {"down: 4.2", "down: 4.7"},
	"fins further forward":    {"aft: 67.5", "aft: 60"},
	"a broader tip":           {"tip: 3.6", "tip: 4.1"},
	"four-blade alternate":    {"blades: 5", "blades: 4"},
	"a wider propeller":       {"diameter: 4.2", "diameter: 9.4"},
	"small skew":              {"skew: 0", "skew: 5"},
}

// Sizes one step inside each limit are read.
func TestAnAlfaOneStepInsideItsLimitsIsRead(t *testing.T) {
	whole := string(alfaRaw)
	for name, each := range alfaAccepted {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		changed := strings.Replace(whole, each.was, each.is, 1)
		if _, err := readAlfa([]byte(changed)); err != nil {
			t.Errorf("%s: refused as %v", name, err)
		}
	}
}
