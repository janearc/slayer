package mesh

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// body is corners and the faces among them, a solid or a sheet or
// several of either, before it is a mesh.
type body struct {
	corners []maths.Vec
	faces   [][triangleCorners]int
}

// cubeAt is the unit cube moved by a shift.
func cubeAt(shift maths.Vec) body {
	corners := cubeCorners()
	for i, corner := range corners {
		corners[i] = corner.Add(shift)
	}
	return body{corners: corners, faces: cubeFaces()}
}

// together is bodies put in one: each one's faces renumbered to follow
// the corners of the bodies before it.
func together(bodies ...body) body {
	var all body
	for _, each := range bodies {
		offset := len(all.corners)
		all.corners = append(all.corners, each.corners...)
		for _, at := range each.faces {
			all.faces = append(all.faces, [triangleCorners]int{
				at[first] + offset, at[second] + offset,
				at[third] + offset})
		}
	}
	return all
}

// turnedIn is a body with each of its faces turned the other way.
func turnedIn(of body) body {
	turned := body{corners: of.corners,
		faces: make([][triangleCorners]int, len(of.faces))}
	for i, at := range of.faces {
		turned.faces[i] = [triangleCorners]int{at[first], at[third],
			at[second]}
	}
	return turned
}

// scaledBy is a body with its corners all scaled about the origin.
func scaledBy(of body, share float64) body {
	scaled := body{corners: make([]maths.Vec, len(of.corners)),
		faces: of.faces}
	for i, corner := range of.corners {
		scaled.corners[i] = corner.Scale(share)
	}
	return scaled
}

// made is a body as a mesh, flat, or the error of it.
func (of body) made() (*Mesh, error) {
	return New(of.corners, of.faces, nil, false)
}

// prismOf is the solid of an outline of corners in the ground plane,
// stood up to a height, its caps fans from a middle point that sees every
// corner of the outline. Its faces turn out.
func prismOf(outline []maths.Vec, middle maths.Vec, height float64) body {
	up := maths.V(nought, height, nought)
	rings := len(outline)
	bottomMiddle, topMiddle := pair*rings, pair*rings+unity
	made := body{corners: ringsOf(outline, up, middle)}
	for i := range outline {
		next := (i + unity) % rings
		bottom, bottomNext := i, next
		top, topNext := rings+i, rings+next
		made.faces = append(made.faces,
			[triangleCorners]int{bottom, bottomNext, topNext},
			[triangleCorners]int{bottom, topNext, top},
			[triangleCorners]int{topMiddle, top, topNext},
			[triangleCorners]int{bottomMiddle, bottomNext, bottom})
	}
	every := make([]int, len(made.faces))
	for i := range every {
		every[i] = i
	}
	points := make([]face, len(made.faces))
	for i, at := range made.faces {
		points[i] = face{int32(at[first]), int32(at[second]),
			int32(at[third])}
	}
	if volume(made.corners, points, every) < nought {
		return turnedIn(made)
	}
	return made
}

// ringsOf is an outline, the outline raised by a rise, a middle point
// on the ground, and the middle raised: a prism's corners, bottom ring,
// top ring, bottom middle and top middle.
func ringsOf(outline []maths.Vec, rise, middle maths.Vec) []maths.Vec {
	corners := append([]maths.Vec{}, outline...)
	for _, corner := range outline {
		corners = append(corners, corner.Add(rise))
	}
	return append(corners, middle, middle.Add(rise))
}

// lSolid is a solid of an L shaped outline, two arms of one width and
// height, each as long as the arm-length picked: not convex, for the
// notch between its arms is in its hull and not in it.
func lSolid(test testing.TB) body {
	long, wide := picked(test, "arm-length"), picked(test, "arm-width")
	outline := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(long, nought, nought), maths.V(long, nought, wide),
		maths.V(wide, nought, wide), maths.V(wide, nought, long),
		maths.V(nought, nought, long)}
	middle := maths.V(wide/pair, nought, wide/pair)
	return prismOf(outline, middle, wide)
}

// plateOf is a flat plate closed by two sides: a fan of triangles about
// its middle, and its mirror, every edge run once each way. Its corners
// are not dyadic, and it is tilted, so the sum that finds its volume
// rounds.
func plateOf(test testing.TB) body {
	sides := pickedCount(test, "plate-sides")
	radius := picked(test, "plate-radius")
	slopeX, slopeZ := picked(test, "plate-slope-x"),
		picked(test, "plate-slope-z")
	plate := body{corners: []maths.Vec{maths.V(nought, nought, nought)}}
	for i := range sides {
		turn := pair * math.Pi * float64(i) / float64(sides)
		x, z := radius*math.Cos(turn), radius*math.Sin(turn)
		plate.corners = append(plate.corners,
			maths.V(x, slopeX*x+slopeZ*z, z))
	}
	for i := range sides {
		ring, next := unity+i, unity+(i+unity)%sides
		plate.faces = append(plate.faces,
			[triangleCorners]int{ring, next, nought})
	}
	for i := range sides {
		ring, next := unity+i, unity+(i+unity)%sides
		plate.faces = append(plate.faces,
			[triangleCorners]int{next, ring, nought})
	}
	return plate
}

// A solid of an L shaped outline is a mesh, closed, with an inside in each
// arm and none in the notch between them, though the notch is inside the
// box and the hull the solid fills.
func TestANonConvexSolidHasNoInsideInItsNotch(t *testing.T) {
	made, err := lSolid(t).made()
	if err != nil {
		t.Fatal(err)
	}
	if !made.Closed() {
		t.Fatal("the L shaped solid is not closed")
	}
	long, wide := picked(t, "arm-length"), picked(t, "arm-width")
	middling := (wide + long) / pair
	arm := maths.V(wide/pair, wide/pair, middling)
	notch := maths.V(middling, wide/pair, middling)
	if !made.inside(arm) || !(made.Distance(arm) < nought) {
		t.Errorf("a point in an arm is not inside: %v", made.Distance(arm))
	}
	if made.inside(notch) || !(made.Distance(notch) > nought) {
		t.Errorf("a point in the notch is inside: %v",
			made.Distance(notch))
	}
}

// Two cubes of the same size nested, wound alike, are two solids, one in
// the other, and are accepted: a point between their shells is inside the
// outer, and a point in the inner is inside both.
func TestNestedSolidsWoundAlikeAreAcceptedAndHaveAnInside(t *testing.T) {
	inner := scaledBy(unitCube(), picked(t, "cavity-share"))
	made, err := together(unitCube(), inner).made()
	if err != nil {
		t.Fatal(err)
	}
	between := maths.V(picked(t, "between-reach"), nought, nought)
	if !made.Closed() || !made.inside(between) ||
		!made.inside(middleOfCube()) {
		t.Errorf("nested solids: closed %v", made.Closed())
	}
}

// unitCube is the cube of side one about the origin, as a body.
func unitCube() body { return cubeAt(middleOfCube()) }

// middleOfCube is the middle of the unit cube, the origin.
func middleOfCube() maths.Vec { return maths.V(nought, nought, nought) }

// A hollow solid, whose cavity is a second shell turned in toward the
// hollow, is refused as a closed piece inside out, naming the first face
// of the cavity: the check for a closed piece reads a cavity as one.
func TestACavityIsRefusedAsInsideOut(t *testing.T) {
	shell := scaledBy(unitCube(), picked(t, "cavity-share"))
	_, err := together(unitCube(), turnedIn(shell)).made()
	if !errors.Is(err, errInsideOut) {
		t.Fatalf("a cavity: %v", err)
	}
	cavityFirst := len(cubeFaces())
	want := fmt.Sprintf("face %d", cavityFirst)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the error does not name %q: %v", want, err)
	}
}

// A closed flat plate, a fan of triangles and its mirror, holds nothing,
// and is not inside out. Its volume is nought by the arithmetic and may
// come out a rounding below it, because the faces' terms are summed in
// the order of the faces; this test says it must not be refused for that.
func TestAClosedFlatPlateIsNotInsideOut(t *testing.T) {
	_, err := plateOf(t).made()
	if errors.Is(err, errInsideOut) {
		t.Errorf("a closed flat plate is refused as inside out: %v", err)
	}
}

// Corners are one place only to the bit: a cube with one corner on one
// face an ulp off has an open edge, is accepted, and is not closed, so its
// middle is not inside it and the distance to it has no sign. The same
// cube with no corner moved is closed, and its middle is inside.
func TestACornerAnUlpOffLeavesTheCubeOpenWithNoWord(t *testing.T) {
	exact, err := unitCube().made()
	if err != nil || !exact.Closed() {
		t.Fatalf("the cube: closed %v, %v", exact != nil, err)
	}
	middle := middleOfCube()
	if !exact.inside(middle) || !(exact.Distance(middle) < nought) {
		t.Error("the middle of the exact cube is not inside it")
	}
	moved := unitCube()
	corner := moved.corners[pickedCount(t, "nudged-corner")]
	nudged := math.Nextafter(corner.X, math.Inf(plusInfinity))
	moved.corners = append(moved.corners, maths.V(nudged, corner.Y, corner.Z))
	moved.faces[pickedCount(t, "nudged-face")][third] = len(moved.corners) -
		unity
	made, err := moved.made()
	if err != nil {
		t.Fatalf("a corner an ulp off is refused: %v", err)
	}
	if made.Closed() || made.inside(middle) ||
		!(made.Distance(middle) > nought) {
		t.Errorf("closed %v, inside %v, distance %v", made.Closed(),
			made.inside(middle), made.Distance(middle))
	}
}
