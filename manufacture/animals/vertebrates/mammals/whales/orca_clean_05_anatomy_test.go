package whales

import (
	"github.com/janearc/slayer/manufacture/geometry"
	"math"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// TestOrcaFieldHoldsTheNamedParts checks that the field is solid where
// each anatomical part stands and clear where the animal has none.
func TestOrcaFieldHoldsTheNamedParts(t *testing.T) {
	// The paddle's middle: the root, a share orcaFlipperAt of the way
	// along the span. The paddle is .11 m thick there, so a point a
	// hand's breadth off its plane is outside it.
	paddle := func(side float64) maths.Vec {
		return maths.V(orcaFlipperRootX+orcaFlipperSpan.X*orcaFlipperAt,
			orcaFlipperRootY+orcaFlipperSpan.Y*orcaFlipperAt,
			side*(orcaFlipperRootZ+orcaFlipperSpan.Z*orcaFlipperAt))
	}
	solid := map[string]maths.Vec{
		"trunk":          maths.V(0, 0, 0),
		"dorsal fin":     maths.V(.55, 1.2, 0),
		"flipper":        paddle(1),
		"fluke lobe":     maths.V(-2.65, orcaFlukeLevel, .40),
		"fluke tip":      maths.V(-2.95, orcaFlukeLevel, .80),
		"forehead":       maths.V(2.45, .30, 0),
		"dorsal keel":    maths.V(-2.2, .262, 0),
		"flipper (left)": paddle(-1),
	}
	for name, at := range solid {
		if orcaField(at) >= 0 {
			t.Errorf("%s: field %.4f at %v, want inside", name, orcaField(at), at)
		}
	}
	clear := map[string]maths.Vec{
		"beside the dorsal fin": maths.V(.55, 1.2, .2),
		"in the fluke notch":    maths.V(-2.72, orcaFlukeLevel, 0),
		"behind the fluke tip":  maths.V(-3.15, orcaFlukeLevel, .95),
		"above the back":        maths.V(0, 1.0, .4),
		"far away":              maths.V(5, 5, 5),
	}
	for name, at := range clear {
		if orcaField(at) <= 0 {
			t.Errorf("%s: field %.4f at %v, want outside", name, orcaField(at), at)
		}
	}
}

// TestOrcaFieldIsMirrorSymmetric checks that, apart from the marks, the
// left of the animal is the right turned about the midline.
func TestOrcaFieldIsMirrorSymmetric(t *testing.T) {
	for _, at := range []maths.Vec{
		maths.V(1.4, -.4, .5), maths.V(-2.6, 0, .6), maths.V(2.5, -.2, .3),
		maths.V(.5, 1.0, .05), maths.V(2.6, -.12, .25),
	} {
		mirror := maths.V(at.X, at.Y, -at.Z)
		if math.Abs(orcaField(at)-orcaField(mirror)) > 1e-12 {
			t.Errorf("field differs across the midline at %v", at)
		}
	}
}

// TestOrcaFinRootsAreFilleted checks that the dorsal fin thickens into
// the back: its section through the back is thicker than higher up.
func TestOrcaFinRootsAreFilleted(t *testing.T) {
	thickness := func(height float64) float64 {
		for z := 0.; z < .3; z += .002 {
			if orcaField(maths.V(.62, height, z)) >= 0 {
				return z
			}
		}
		return .3
	}
	// The back is at .641 m here and the fillet reaches .12 m above it:
	// .68 m is inside it, 1.1 m is well clear and is the fin's own.
	const filleted, free = .68, 1.1
	low, high := thickness(filleted), thickness(free)
	if low <= high {
		t.Fatalf("fin root half thickness %.3f is not more than %.3f above", low, high)
	}
}

// TestOrcaFlukesAreNotched checks the median notch: the trailing edge
// at the midline lies ahead of the trailing edge a little way out.
func TestOrcaFlukesAreNotched(t *testing.T) {
	_, middle := orcaFlukeEdges(0)
	_, out := orcaFlukeEdges(orcaFlukeCleft + .05)
	if middle <= out {
		t.Fatalf("no notch: trailing edge %.3f at the midline, %.3f out", middle, out)
	}
}

// orcaTestTooth is the field just above the base and just short of the
// tip of tooth index along one side: even teeth of the upper jaw point
// down, odd teeth of the lower point up.
func orcaTestTooth(index int) (base, tip float64) {
	across := orcaTeethFirst + float64(index)*orcaTeethStep
	line := orcaMouthLine(across)
	s := orcaTrunkProfile().at(across)
	z := s.wide*math.Sqrt(1-(line-s.centre)*(line-s.centre)/
		(s.high*s.high)) - orcaToothSink
	direction := 1.
	if index%2 == 0 {
		direction = -1
	}
	root := line - direction*orcaToothRoot
	return orcaTeeth(maths.V(across, root+direction*.004, z)),
		orcaTeeth(maths.V(across, root+direction*(orcaToothHeight-.002), z))
}

// TestOrcaTeethAreCones checks each tooth along a side is wide at the
// base and a point at the tip, the jaws taking them in turn.
func TestOrcaTeethAreCones(t *testing.T) {
	for index := 0; index < orcaTeethCount; index++ {
		base, tip := orcaTestTooth(index)
		if base >= 0 {
			t.Errorf("tooth %d: base %.4f is not inside", index, base)
		}
		if tip > orcaToothRadius {
			t.Errorf("tooth %d: tip %.4f is not a point", index, tip)
		}
	}
}

// Anti-test: there is no tooth past the last one along a side.

func TestOrcaHasNoToothPastTheLast(t *testing.T) {
	base, _ := orcaTestTooth(orcaTeethCount + 1)
	if base < 0 {
		t.Fatal("a tooth stands past the last along a side")
	}
}

// TestOrcaSkinIsOneClosedSurface checks the cut skin is a single closed
// piece, wound outward, with the relief small beside the body.
func TestOrcaSkinIsOneClosedSurface(t *testing.T) {
	parts, err := orcaSwimming(0, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatalf("%d parts, want one", len(parts))
	}
	skin := parts[0].Shape.(*mesh.Mesh)
	if !skin.Closed() {
		t.Fatal("skin is open")
	}
	volume := 0.
	for i := 0; i < skin.FaceCount(); i++ {
		a, b, c := skin.Corners(i)
		volume += a.Dot(b.Cross(c)) / 6
	}
	if volume < 1 || volume > 10 {
		t.Fatalf("enclosed volume %.3f cubic metres is not an orca's", volume)
	}
	rest := orcaRest()
	limit := orcaRidgeHeight + orcaRakeDepth + orcaScarDepth
	for i, p := range rest.points {
		if rest.relieved[i].Sub(p).Len() > limit+1e-9 {
			t.Fatalf("relief at %v is %.5f, beyond %.5f", p,
				rest.relieved[i].Sub(p).Len(), limit)
		}
	}
}

// TestOrcaMarkingsFollowTheBody checks the colours by place: white belly,
// black back, grey saddle, white eye patch and dark eye.
func TestOrcaMarkingsFollowTheBody(t *testing.T) {
	profile, _ := Default().Get("orca")
	paint := orca05Paint{model: defaultOrcaModel(), palette: profile.Palette}
	bright := func(p maths.Vec) float64 { c := paint.At(p); return (c.R + c.G + c.B) / 3 }
	if bright(maths.V(0, -.55, 0)) < .5 {
		t.Error("belly is not white")
	}
	// The back's crest ahead of the dorsal fin, where the grey saddle
	// (which lies behind the fin, from x = .45 back) has not begun. The
	// crest at x = 0 is the saddle's.
	const ahead = 1.0
	section := orcaTrunkProfile().at(ahead)
	if bright(maths.V(ahead, section.centre+section.high, 0)) > .05 {
		t.Error("back is not black")
	}
	if g := bright(maths.V(-.3, .60, 0)); g < .15 || g > .4 {
		t.Errorf("saddle brightness %.3f is not grey", g)
	}
	patch := maths.V(orcaPatchX, orcaPatchY, orcaTrunkProfile().at(orcaPatchX).wide*.9)
	if bright(patch) < .5 {
		t.Error("eye patch is not white")
	}
	if bright(orcaEyeCentre()) > .06 {
		t.Error("eye is not dark")
	}
}

// orcaTestTailTop is the place of the topmost rest point of the tail
// stock between x = -2.0 and -1.9, a point the swim moves.
func orcaTestTailTop() int {
	rest := orcaRest()
	best := -1
	for i, p := range rest.points {
		if p.X > -2 && p.X < -1.9 &&
			(best < 0 || p.Y > rest.points[best].Y) {
			best = i
		}
	}
	return best
}

// orcaTestPaintPoint is where the tracer would paint a ray that meets
// the skin at its point i in the swim at phase: the mesh's rest point
// for that hit, and the posed point the ray was aimed at.
func orcaTestPaintPoint(t *testing.T, phase float64,
	i int) (rest, posed maths.Vec) {
	t.Helper()
	parts, err := orcaSwimming(phase, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	skin := parts[0].Shape.(*mesh.Mesh)
	points := skin.Points()
	normal := geometry.UnitNormals(points, orcaRest().faces)[i]
	start := points[i].Add(normal.Scale(.05))
	ray := maths.NewRay(start, normal.Neg())
	met, ok := skin.Meet(ray, 0, 1)
	if !ok {
		t.Fatalf("a ray at point %d missed the skin", i)
	}
	rest, found := skin.RestPointAt(met)
	if !found {
		t.Fatal("the posed skin carries no rest points")
	}
	return rest, points[i]
}

// TestOrcaMarkingsAreNotADecal checks that the pattern moves with the
// skin: a ray that meets one point of the skin in two poses is painted
// at that point's rest place both times, though the point has moved.
// Paint read at the posed place, a decal, fails it.
func TestOrcaMarkingsAreNotADecal(t *testing.T) {
	i := orcaTestTailTop()
	want := orcaRest().points[i]
	first, firstPosed := orcaTestPaintPoint(t, 0, i)
	later, laterPosed := orcaTestPaintPoint(t, math.Pi/2, i)
	if firstPosed.Sub(laterPosed).Len() < .001 {
		t.Fatal("the swim did not move the tail point")
	}
	if !first.Near(want, 1e-6) || !later.Near(want, 1e-6) {
		t.Fatalf("painted at %v and %v, not the rest point %v", first,
			later, want)
	}
	profile, _ := Default().Get("orca")
	paint := orca05Paint{model: defaultOrcaModel(), palette: profile.Palette}
	if paint.At(first) != paint.At(later) {
		t.Fatal("the colour changed with the pose")
	}
}

// TestOrcaJointsMoveOnlyTheTail checks the swim's joints leave the head
// and the flipper's root where they are and carry the tail with the
// last caudal.
func TestOrcaJointsMoveOnlyTheTail(t *testing.T) {
	bones, err := orcaSkeleton()
	if err != nil {
		t.Fatal(err)
	}
	spans := orcaAxis(bones)
	frames := bones.Frames(orcaSwimPose(bones, 1.3))
	shoulder := maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ)
	for _, still := range []maths.Vec{maths.V(2, .1, .3), shoulder} {
		moved := skeleton.Carry(still, frames, orcaBind(bones, spans, still))
		if !moved.Near(still, 1e-9) {
			t.Fatalf("%v moved to %v", still, moved)
		}
	}
	// behind the last caudal's middle a point is held by it alone, so it
	// goes where that caudal's frame takes it. By hand the sum of the
	// caudals' turns times their arms is about .022 m at phase 1.3; a
	// quarter of that is asked for, so the sum can be off by much and a
	// tail held still still fails.
	tail := maths.V(-2.9, 0, .5)
	carried := skeleton.Carry(tail, frames, orcaBind(bones, spans, tail))
	last := orcaMust(bones, "Ca24")
	if !carried.Near(frames[last].Apply(tail), 1e-9) {
		t.Fatalf("tail went to %v, not with the last caudal", carried)
	}
	if carried.Sub(tail).Len() < .022/4 {
		t.Fatalf("tail moved %.4f m", carried.Sub(tail).Len())
	}
}
