package main

import (
	"flag"
	"image/png"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/bioforms/assembly"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"testing"
)

func TestGazeFixatesBothEyesAndLimitsExtremeTargets(t *testing.T) {
	identity := func(p maths.Vec) maths.Vec { return p }
	target := v(-.41, 1.63, 1.14)
	g, e := solveGaze(target, identity)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		p := headRestEye(i)
		neutral := yawPitch(float64(1-2*i)*.0162, .077).Out(maths.ZAxis)
		axis := g.eye(p.Add(neutral.Scale(.012)), i).Sub(g.eye(p, i)).Unit()
		if mis := axis.Sub(target.Sub(g.eyes[i]).Unit()).Len(); mis > 1e-8 {
			t.Fatal(i, mis)
		}
	}
	far, e := solveGaze(v(1, 15, -1), identity)
	if e != nil || far.yaw > .90 || far.pitch > .48 || far.yaw < .89 {
		t.Fatal(far, e)
	}
	if _, e := solveGaze(v(0, 1.55250, .05863), identity); e == nil {
		t.Fatal("accepted target inside eyes")
	}
	for _, s := range []string{"look", "nan,2,3", "2,inf,3"} {
		if _, e := vector(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestFigureCameraFollowsTransformedPartsAndHeading(t *testing.T) {
	actor := maths.NewSubspace(v(11, 3, -9), maths.Rotation(maths.YAxis, 1.55), 1)
	local := maths.NewSubspace(v(.21, .73, .16), maths.Identity, 1.2)
	part := thing.Thing{Shape: shape.Sphere{Radius: .65}, Motion: thing.Still(local)}
	world := thing.NewWorld()
	placed := placeFigure(world, "alex", []thing.Thing{part}, .8, actor)
	c := placed.view(nil)
	if c.name != "alex-camera" || len(world.Things) != 1 {
		t.Fatal(c, world)
	}
	if delta := c.at.Sub(actor.Out(local.Origin)).Len(); delta > 1e-7 {
		t.Fatal(delta, c.at)
	}
	if c.from.Sub(c.at).Unit().Dot(actor.Orientation.Out(maths.ZAxis)) < .95 {
		t.Fatal("camera no longer faces actor")
	}
	eye := c.eye()
	for _, x := range []float64{-.65, .65} {
		for _, y := range []float64{-.65, .65} {
			for _, z := range []float64{-.65, .65} {
				localPoint := eye.In(actor.Out(local.Out(v(x, y, z))))
				half := -localPoint.Z * math.Tan(c.wide/2)
				if half <= 0 || math.Abs(localPoint.X) >= half || math.Abs(localPoint.Y) >= half/.8 {
					t.Fatal(localPoint)
				}
			}
		}
	}
	explicit := figureCamera{name: "authored", from: v(0, 2, 4), at: v(0, 1, 0), wide: .45}
	if got := placed.view(&explicit); got != explicit {
		t.Fatal("lost authored camera", got)
	}
}
func TestGazeCarriesHeadwearBelowTheNeckAsOnePart(t *testing.T) {
	body, eyes := readObj("SMPLX-female.obj")
	rig := standing(body)
	f := standardFigure()
	f.Hair = false
	f.Anchors = rig.Anchors
	target := v(.7, 1.6, .95)
	var e error
	f.Gaze, e = solveGaze(target, rig.Carry)
	if e != nil {
		t.Fatal(e)
	}
	rest := []maths.Vec{v(.03, 1.39, -.05), v(.035, 1.39, -.05), v(.03, 1.396, -.05)}
	f.JointParts = []jointPart{{joint: "head", surface: assembly.Part{Rest: rest, Faces: [][3]int{{0, 1, 2}}, Material: &thing.Material{Name: "hat ribbon"}}}}
	parts, e := f.build(body, eyes, rig.Carry)
	if e != nil {
		t.Fatal(e)
	}
	points := parts[len(parts)-1].Shape.(*mesh.Mesh).Points()
	for i, p := range rest {
		want := f.Gaze.head(rig.Anchors["head"](p), 1)
		if points[i].Sub(want).Len() > 1e-10 {
			t.Fatal(i, points[i], want)
		}
	}
	if _, e := f.jointPoint("unknown", rest[0]); e == nil {
		t.Fatal("unknown joint accepted")
	}
	f.JointParts[0].joint = "unknown"
	if _, e := f.build(body, eyes, rig.Carry); e == nil {
		t.Fatal("unknown joint attachment accepted")
	}
}

func TestProfilesPreserveFootContactAndHeadLength(t *testing.T) {
	ref := standardFigure()
	foot, knee, hip, head, top := v(.10, 0, .01), v(.09, .46, .03), v(.17, .84, .02), v(.03, 1.45, .05), v(.03, 1.65, .05)
	for _, name := range []string{"compact", "long-legged"} {
		f := ref
		var e error
		f.Shape, e = bodyProfile(name)
		if e != nil {
			t.Fatal(e)
		}
		fit := func(p maths.Vec) maths.Vec { return f.fit(p, p) }
		if fit(foot).Y != 0 || math.Abs(fit(top).Y-fit(head).Y-.20) > 1e-10 {
			t.Fatal(name, fit(foot), fit(top), fit(head))
		}
		if name == "compact" && (fit(knee).Y >= knee.Y || fit(hip).X <= hip.X) {
			t.Fatal("compact profile lost local proportions")
		}
		if name == "long-legged" && (fit(knee).Y <= knee.Y || fit(hip).X >= hip.X) {
			t.Fatal("long profile lost local proportions")
		}
	}
	if _, e := bodyProfile("unknown"); e == nil {
		t.Fatal("unknown profile")
	}
}
func TestHairAndCameraChoicesThroughCommand(t *testing.T) {
	saveArgs, saveFlags := os.Args, flag.CommandLine
	defer func() { os.Args = saveArgs; flag.CommandLine = saveFlags }()
	for _, variation := range [][]string{{"-camera", "plate", "-pose", "gather", "-hairstyle", "rooted", "-profile", "broad-shouldered", "-natural-skin", "-light-study", "bounce", "-study-pass", "2", "-skin-sheen", "matte"}, {"-alley", "-camera", "alley", "-study-pass", "2", "-hairstyle", "layered"}, {"-study-pass", "2", "-skin-sheen", "matte", "-light-study", "albedo", "-sss=false", "-hair=false"}, {"-study-pass", "2", "-skin-sheen", "original", "-hair=false", "-soft-studio"}, {"-hairstyle", "legacy-bob", "-profile", "compact", "-look-at", "-.33,1.65,1"}, {"-alley", "-camera", "alley", "-profile", "long-legged", "-hairstyle", "curls"}, {"-camera", "figure", "-at", "1,0,-1", "-heading", "30"}} {
		path := filepath.Join(t.TempDir(), "variant.png")
		os.Args = append([]string{"portrait", "-obj", "SMPLX-female.obj", "-width", "30", "-out", path}, variation...)
		flag.CommandLine = flag.NewFlagSet("portrait", flag.PanicOnError)
		main()
		file, e := os.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		cfg, e := png.DecodeConfig(file)
		file.Close()
		if e != nil || cfg.Width != 30 {
			t.Fatal(cfg, e)
		}
	}
	for _, variation := range [][]string{{"-light-study", "unknown"}, {"-hairstyle", "rooted", "-crown-fall", "1"}, {"-hairstyle", "rooted", "-follicle-front", "2"}, {"-pose", "unknown"}, {"-study-pass", "3"}, {"-skin-sheen", "unknown"}, {"-hairstyle", "unknown"}, {"-camera", "alley"}, {"-camera", "unknown"}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("accepted", variation)
				}
			}()
			os.Args = append([]string{"portrait", "-obj", "SMPLX-female.obj", "-hair=false"}, variation...)
			flag.CommandLine = flag.NewFlagSet("portrait", flag.PanicOnError)
			main()
		}()
	}
}
func TestHairRefusesMissingScalp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("accepted missing scalp")
		}
	}()
	scalp(geom{pts: []maths.Vec{v(4, 4, 4), v(5, 4, 4), v(5, 5, 4)}, faces: [][3]int{{0, 1, 2}}})
}
