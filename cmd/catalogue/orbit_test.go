package main

import (
	"bytes"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
)

const orbitTolerance = 1e-10

func TestCatalogueOrbitYawTurnsAroundTheEntryVertical(test *testing.T) {
	for _, angle := range []float64{42, -37} {
		test.Run(fmtAngle(angle), func(test *testing.T) {
			plan := defaultPlan(test)
			entry := entry{Kind: "coast-redwood-shoot", Yaw: 19, Up: "x"}
			base := specimenFor(test, plan, entry)
			plan.Orbit.Yaw = angle
			turned := specimenFor(test, plan, entry)
			centre := centreOf(base)
			before := base.eye.Origin.Sub(centre)
			after := turned.eye.Origin.Sub(centre)
			want := maths.Rotation(maths.XAxis,
				angle*maths.Degree).Out(before)
			nearVector(test, after, want)
			preservesDistance(test, plan, base, turned)
		})
	}
}

func TestCatalogueOrbitYawTurnsAroundYForAnUprightEntry(test *testing.T) {
	plan := defaultPlan(test)
	entry := entry{Kind: "keck-i", Yaw: 23}
	base := specimenFor(test, plan, entry)
	plan.Orbit.Yaw = -51
	turned := specimenFor(test, plan, entry)
	baseCentre := centreOf(base)
	want := maths.Rotation(maths.YAxis, -51*maths.Degree).
		Out(base.eye.Origin.Sub(baseCentre))
	turnedCentre := centreOf(turned)
	nearVector(test, turned.eye.Origin.Sub(turnedCentre), want)
	preservesDistance(test, plan, base, turned)
}

func TestCatalogueOrbitPitchUsesTheTurnedCameraRight(test *testing.T) {
	for _, angle := range []float64{31, -24} {
		test.Run(fmtAngle(angle), func(test *testing.T) {
			plan := defaultPlan(test)
			entry := entry{Kind: "coast-redwood-shoot", Yaw: -23, Up: "x"}
			base := specimenFor(test, plan, entry)
			plan.Orbit.Yaw = 28
			plan.Orbit.Pitch = angle
			turned := specimenFor(test, plan, entry)
			centre := centreOf(base)
			direction := base.eye.Origin.Sub(centre)
			direction = maths.Rotation(maths.XAxis,
				28*maths.Degree).Out(direction)
			frame, err := maths.View(centre.Add(direction), centre, maths.XAxis)
			if err != nil {
				test.Fatal(err)
			}
			right := frame.Orientation.X
			want := maths.Rotation(right.Neg(), angle*maths.Degree).
				Out(direction)
			nearVector(test, turned.eye.Origin.Sub(centre), want)
			preservesDistance(test, plan, base, turned)
		})
	}
}

func TestCatalogueOrbitZeroKeepsTheAuthoredCameraExactly(test *testing.T) {
	plan := defaultPlan(test)
	entry := entry{Kind: "coast-redwood-shoot", Yaw: 17, Up: "x"}
	authored := specimenFor(test, plan, entry)
	plan.Orbit = orbit{}
	zero := specimenFor(test, plan, entry)
	if *authored.eye != *zero.eye {
		test.Fatal("zero orbit changed the authored eye", authored.eye, zero.eye)
	}
}

func TestCatalogueOrbitClampsElevationAndKeepsFiniteEye(test *testing.T) {
	plan := defaultPlan(test)
	plan.Orbit.Pitch = 80
	shot := specimenFor(test, plan, entry{Kind: "coast-redwood-shoot", Up: "x"})
	centre := centreOf(shot)
	direction := shot.eye.Origin.Sub(centre)
	elevation := math.Asin(direction.Dot(maths.XAxis) / direction.Len())
	if math.Abs(elevation/maths.Degree-85) > orbitTolerance {
		test.Fatal("orbit did not clamp away from the pole", elevation/maths.Degree)
	}
	for _, component := range []float64{shot.eye.Origin.X, shot.eye.Origin.Y,
		shot.eye.Origin.Z, shot.eye.Orientation.X.X,
		shot.eye.Orientation.X.Y, shot.eye.Orientation.X.Z} {
		if math.IsNaN(component) || math.IsInf(component, 0) {
			test.Fatal("orbit produced a non-finite camera", shot.eye)
		}
	}
}

func TestCatalogueOrbitClampDoesNotFlipItsAzimuth(test *testing.T) {
	plan := defaultPlan(test)
	entry := entry{Kind: "coast-redwood-shoot", Up: "x"}
	base := specimenFor(test, plan, entry)
	centre := centreOf(base)
	direction := base.eye.Origin.Sub(centre)
	elevation := math.Asin(direction.Dot(maths.XAxis) / direction.Len())
	pitch := 85*maths.Degree - elevation
	want := maths.Rotation(base.eye.Orientation.X.Neg(), pitch).
		Out(direction)
	plan.Orbit.Pitch = 80
	turned := specimenFor(test, plan, entry)
	nearVector(test, turned.eye.Origin.Sub(centre), want)
}

func TestCatalogueOrbitClampsAtBothYUpPolesWithoutFlipping(test *testing.T) {
	for _, elevation := range []float64{70, -70} {
		test.Run(fmtAngle(elevation), func(test *testing.T) {
			plan := defaultPlan(test)
			entry := entry{Kind: "keck-i", Pitch: &elevation}
			base := specimenFor(test, plan, entry)
			centre := centreOf(base)
			direction := base.eye.Origin.Sub(centre)
			requested := 80.0
			if elevation < 0 {
				requested = -80
			}
			wantElevation := elevation + requested
			if wantElevation > orbitLimit {
				wantElevation = orbitLimit
			} else if wantElevation < -orbitLimit {
				wantElevation = -orbitLimit
			}
			turn := wantElevation - elevation
			want := maths.Rotation(base.eye.Orientation.X.Neg(),
				turn*maths.Degree).Out(direction)
			plan.Orbit.Pitch = requested
			turned := specimenFor(test, plan, entry)
			nearVector(test, turned.eye.Origin.Sub(centre), want)
		})
	}
}

func TestCatalogueOrbitLeavesItsAuthoredPlanUntouched(test *testing.T) {
	plan := defaultPlan(test)
	before := append([]byte(nil), plan.raw...)
	setting := plan.Camera
	distance := plan.Distance
	entry := entry{Kind: "coast-redwood-shoot", Yaw: 17, Up: "x"}
	plan.Orbit = orbit{Yaw: 22, Pitch: -11}
	if _, err := plan.construct(entry, "quick"); err != nil {
		test.Fatal(err)
	}
	if !bytes.Equal(before, plan.raw) || plan.Camera != setting ||
		plan.Distance != distance {
		test.Fatal("orbit modified authored plan values")
	}
}

func TestCatalogueOrbitRefusesNonfiniteAndOutOfRangeValues(test *testing.T) {
	for _, values := range []orbit{
		{Yaw: math.NaN()}, {Pitch: math.Inf(1)},
		{Yaw: 181}, {Pitch: -81},
	} {
		plan := defaultPlan(test)
		plan.Orbit = values
		if err := plan.check(); err == nil {
			test.Fatal("accepted invalid orbit", values)
		}
	}
}

func TestCatalogueOrbitPlanSchemaRequiresBothFiniteBoundedAngles(test *testing.T) {
	for _, suffix := range []string{
		"orbit: {yaw: 12}\n",
		"orbit: {yaw: '12', pitch: 0}\n",
		"orbit: {yaw: 0, pitch: 0, roll: 2}\n",
		"orbit: {yaw: 181, pitch: 0}\n",
		"orbit: {yaw: 0, pitch: -81}\n",
	} {
		copied := append([]byte(nil), planRaw...)
		badPlan := append(copied, []byte("\n"+suffix)...)
		if _, err := readPlan("bad-orbit.yaml", badPlan); err == nil {
			test.Fatal("schema accepted invalid orbit", suffix)
		}
	}
	kept := append([]byte(nil), planRaw...)
	orbited := append(kept,
		[]byte("\norbit: {yaw: -180, pitch: 80}\n")...)
	valid, err := readPlan("orbit.yaml", orbited)
	if err != nil || valid.Orbit != (orbit{Yaw: -180, Pitch: 80}) {
		test.Fatal("schema rejected boundary orbit", valid, err)
	}
}

func specimenFor(test *testing.T, plan *plan, entry entry) *specimen {
	test.Helper()
	shot, err := plan.construct(entry, "quick")
	if err != nil {
		test.Fatal(err)
	}
	return shot
}

func centreOf(shot *specimen) maths.Vec {
	return shot.bounds.Min.Add(shot.bounds.Max).Scale(halfExtent)
}

func preservesDistance(test *testing.T, plan *plan, before, after *specimen) {
	test.Helper()
	if before.bounds != after.bounds {
		test.Fatal("orbit changed the physical target bounds")
	}
	if !reflect.DeepEqual(before.world.Lamps, after.world.Lamps) ||
		!reflect.DeepEqual(before.world.Sky, after.world.Sky) {
		test.Fatal("orbit changed specimen lighting")
	}
	want := before.eye.Origin.Sub(centreOf(before)).Len()
	got := after.eye.Origin.Sub(centreOf(after)).Len()
	if math.Abs(got-want) > orbitTolerance ||
		math.Abs(got-before.bounds.Max.Sub(before.bounds.Min).Len()*
			halfExtent*plan.Distance) > orbitTolerance {
		test.Fatal("orbit changed physical camera distance", want, got)
	}
}

func nearVector(test *testing.T, got, want maths.Vec) {
	test.Helper()
	if got.Sub(want).Len() > orbitTolerance {
		test.Fatal("wrong orbit direction", got, want)
	}
}

func fmtAngle(angle float64) string {
	return strconv.FormatFloat(angle, 'f', 0, 64)
}
