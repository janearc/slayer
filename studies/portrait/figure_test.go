package main

import (
	"flag"
	"github.com/janearc/slayer/bioforms/assembly"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestIndependentProportionsAndGrounding(t *testing.T) {
	f := standardFigure()
	f.Height = 1.1
	f.Hips = 1.2
	f.Shoulders = .85
	for _, c := range []struct{ p, w maths.Vec }{{v(.2, .85, 0), v(.24, .935, 0)}, {v(.2, 1.32, 0), v(.17, 1.452, 0)}, {v(.2, 0, 0), v(.2, 0, 0)}} {
		if q := f.fit(c.p, c.p); q.Sub(c.w).Len() > 1e-9 {
			t.Fatal(c, q)
		}
	}
	body, _ := readObj("SMPLX-female.obj")
	rig := standing(body)
	carry := rig.Carry
	for _, c := range []struct {
		name   string
		origin maths.Vec
	}{{"left-wrist", v(.646, 1.29, -.05)}, {"right-wrist", v(-.646, 1.29, -.05)}, {"head", v(0, 1.42, -.025)}, {"pelvis", v(0, .88, -.02)}} {
		anchor := rig.Anchors[c.name]
		a := anchor(c.origin)
		b := anchor(c.origin.Add(maths.XAxis.Scale(.07)))
		if a.Sub(carry(c.origin)).Len() > 1e-10 || math.Abs(a.Sub(b).Len()-.07) > 1e-10 {
			t.Fatal(c.name, a, b)
		}
	}
	floor := [2]float64{1, 1}
	for _, p := range body.pts {
		if p.Y > .10 {
			continue
		}
		i := 0
		if p.X > 0 {
			i = 1
		}
		floor[i] = math.Min(floor[i], carry(p).Y)
	}
	if math.Abs(floor[0]) > 1e-8 || math.Abs(floor[1]) > 1e-8 {
		t.Fatal(floor)
	}
}

func TestRenderEntryPointsAndOptionalParts(t *testing.T) {
	// Both pictures exercise the same constructor and the normal command entry
	// point. The close-up keeps the hair; the alley removes it and changes fit.
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args = oldArgs; flag.CommandLine = oldFlags }()
	for _, alley := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "study.png")
		args := []string{"portrait", "-width", "36", "-obj", "SMPLX-female.obj", "-out", path}
		if alley {
			args = append(args, "-alley", "-hair=false", "-hip-scale", "1.06", "-mark-strength", "0")
		}
		os.Args = args
		flag.CommandLine = flag.NewFlagSet("portrait", flag.PanicOnError)
		main()
		for _, name := range []string{path, strings.TrimSuffix(path, ".png") + "-surface.png"} {
			f, e := os.Open(name)
			if e != nil {
				t.Fatal(e)
			}
			picture, e := png.Decode(f)
			f.Close()
			if e != nil || picture.Bounds().Dx() != 36 || picture.Bounds().Dy() != 45 {
				t.Fatal(name, picture, e)
			}
		}
	}
}
func TestDecorationsUseTheirOwnBoneAndFigureFit(t *testing.T) {
	body, eyes := readObj("SMPLX-female.obj")
	rig := standing(body)
	anchor := rig.Anchors["left-wrist"]
	f := standardFigure()
	f.Height = 1.06
	f.Hair = false
	p := v(.646, 1.29, -.05)
	r := []maths.Vec{p, p.Add(v(.01, 0, 0)), p.Add(v(0, .01, 0))}
	m := &thing.Material{Name: "test cuff"}
	f.Decorations = []assembly.Part{{Rest: r, Faces: [][3]int{{0, 1, 2}}, Material: m, Carry: anchor}}
	made, e := f.build(body, eyes, rig.Carry)
	if e != nil || len(made) != 4 {
		t.Fatal(len(made), e)
	}
	cuff := made[3]
	if cuff.Material != m {
		t.Fatal("missing cuff")
	}
	pts := cuff.Shape.(*mesh.Mesh).Points()
	for i, q := range pts {
		if q.Sub(f.fit(r[i], anchor(r[i]))).Len() > 1e-10 {
			t.Fatal(q)
		}
	}
}

func TestAlleyConstructionAndInputRefusals(t *testing.T) {
	for _, axis := range []maths.Vec{maths.ZAxis, maths.YAxis} {
		a, b := pipeFrame(axis)
		if a.Dot(axis) != 0 || b.Dot(axis) != 0 || a.Dot(b) != 0 {
			t.Fatal(axis, a, b)
		}
	}
	if !dryFooting(0, 0) || dryFooting(.9, 0) {
		t.Fatal("missing dry contact patch or curb runoff")
	}
	if skin(v(.04, 1.52, .06)) != skinWithMarks(v(.04, 1.52, .06), 1) {
		t.Fatal("default skin changed")
	}
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args = oldArgs; flag.CommandLine = oldFlags }()
	for _, args := range [][]string{{"-obj", "missing-model.obj"}, {"-obj", "SMPLX-female.obj", "-height-scale", "0"}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("accepted", args)
				}
			}()
			os.Args = append([]string{"portrait"}, args...)
			flag.CommandLine = flag.NewFlagSet("portrait", flag.PanicOnError)
			main()
		}()
	}
}
