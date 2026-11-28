package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestRefinePreservesBoundaryAndFaceCenter(t *testing.T) {
	g := geom{pts: []maths.Vec{v(0, 0, 0), v(1, 0, 0), v(1, 1, 0), v(0, 1, 0)}, polys: [][]int{{0, 1, 2, 3}}}
	out := refine(g)
	if len(out.pts) != 9 || len(out.faces) != 8 || out.pts[0] != v(.125, .125, 0) || out.pts[8] != v(.5, .5, 0) || !reflect.DeepEqual(out, refine(g)) {
		t.Fatal(out)
	}
}

func TestRestComplexionAndRegionalFilm(t *testing.T) {
	nose := v(0, 1.535, .08)
	cheek := v(.043, 1.526, .08)
	if oilFilm(nose) <= oilFilm(cheek) || oilFilm(cheek) < 0 || oilFilm(nose) > 1 {
		t.Fatal(oilFilm(nose), oilFilm(cheek))
	}
	changed := 0
	for i := 0; i < 60; i++ {
		p := cheek.Add(v(.00023*float64(i), 0, 0))
		a := complexion(p, 1)
		if a != complexion(p, 1) {
			t.Fatal("unstable skin pigment")
		}
		if a != complexion(p, 0) {
			changed++
		}
	}
	if changed == 0 || changed == 60 {
		t.Fatal("localized marks", changed)
	}
}

func TestRelaxedStudyHasTwoGroundContacts(t *testing.T) {
	body, _ := readObj("SMPLX-female.obj")
	body = refine(body)
	rig := standingPose(body, true)
	floor := [2]float64{1, 1}
	for _, p := range body.pts {
		if p.Y > .1 {
			continue
		}
		side := 0
		if p.X > 0 {
			side = 1
		}
		floor[side] = math.Min(floor[side], rig.Carry(p).Y)
	}
	if math.Abs(floor[0])+math.Abs(floor[1]) > 1e-8 {
		t.Fatal(floor)
	}
}
