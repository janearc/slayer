package mesh_test

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// What the test counts in, each defined by what it counts.
const (
	nought  = 0 // defined: no amount, and the first place of anything
	unity   = 1 // defined: one, and the second place
	pair    = 2 // defined: two, and the third place
	corners = 3 // defined: a triangle has three corners
)

// The files the test reads: its picked numbers, and the posed corners.
const (
	restNumbers = "testdata/numbers-rest_points.yaml"
	posedFile   = "testdata/rest_points-posed.csv"
	floatBits   = 64 // defined: the bits of a float64
)

// chosen is a number the test picks, by its name in its numbers file.
func chosen(t *testing.T, name string) float64 {
	t.Helper()
	raw, err := os.ReadFile(restNumbers)
	if err != nil {
		t.Fatal(err)
	}
	all, err := numbers.Read(restNumbers, raw)
	if err != nil {
		t.Fatal(err)
	}
	number, err := all.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// posedCorners are the corners of the posed face, read from their file, a
// row each: x, y and z.
func posedCorners(t *testing.T) []maths.Vec {
	t.Helper()
	opened, err := os.Open(posedFile)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	all, err := csv.NewReader(opened).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var posed []maths.Vec
	for _, row := range all[unity:] {
		var part [corners]float64
		for place := range part {
			part[place], err = strconv.ParseFloat(row[unity+place], floatBits)
			if err != nil {
				t.Fatal(err)
			}
		}
		posed = append(posed, maths.V(part[nought], part[unity], part[pair]))
	}
	return posed
}

// A point on a posed face is paired with the point of the undeformed
// face at the same weights, and moving the rest points after the mesh is
// made does not move it.
func TestRestPointFollowsTheSamePlaceOnADeformedFace(t *testing.T) {
	rest := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(nought, unity, nought)}
	face := [][corners]int{{nought, unity, pair}}
	skin, err := mesh.NewWithRestPoints(posedCorners(t), rest, face, nil,
		false)
	if err != nil {
		t.Fatal(err)
	}
	moved := chosen(t, "rest-points-moved")
	rest[unity] = maths.V(moved, moved, moved)
	at := maths.V(chosen(t, "rest-points-ray-x"),
		chosen(t, "rest-points-ray-y"), chosen(t, "rest-points-ray-z"))
	ray := maths.NewRay(at, maths.V(nought, nought, -unity))
	met, hit := skin.Meet(ray, nought, chosen(t, "rest-points-far"))
	if !hit {
		t.Fatal("ray missed the posed face")
	}
	got, hasRest := skin.RestPointAt(met)
	want := maths.V(chosen(t, "rest-points-want-x"),
		chosen(t, "rest-points-want-y"), nought)
	if !hasRest || got.Sub(want).Len() > chosen(t, "rest-points-slack") {
		t.Fatalf("surface paint point is %v (%t), want %v", got, hasRest,
			want)
	}
}
