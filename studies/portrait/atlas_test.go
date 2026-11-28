package main

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestGesturePairsMoveDifferentLandmarksAndGroundBothSoles(t *testing.T) {
	body, _ := readObj("SMPLX-female.obj")
	seen := map[string][]maths.Vec{}
	for name, s := range repertoire {
		model := articulated(body, true, s)
		var marks []maths.Vec
		for _, r := range []maths.Vec{v(-.75, 1.29, -.06), v(.75, 1.29, -.06), v(-.085, .465, -.02), v(.085, .465, -.02), v(0, .88, -.02)} {
			marks = append(marks, model.Carry(r))
		}
		for other, prior := range seen {
			different := false
			for i, p := range marks {
				if p.Sub(prior[i]).Len() > .045 {
					different = true
				}
			}
			if !different {
				t.Fatalf("%s duplicates %s", name, other)
			}
		}
		seen[name] = marks
		low := [2]float64{1, 1}
		for _, p := range body.pts {
			if p.Y < .12 {
				j := 0
				if p.X > 0 {
					j = 1
				}
				low[j] = math.Min(low[j], model.Carry(p).Y)
			}
		}
		if math.Abs(low[0])+math.Abs(low[1]) > 1e-7 {
			t.Fatal(name, low)
		}
	}
	if len(seen) != 8 {
		t.Fatal(len(seen))
	}
}

func TestFolliclesOriginateOnTheDistributedHead(t *testing.T) {
	body, _ := readObj("SMPLX-female.obj")
	skull, err := mesh.New(body.pts, body.faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	region := follicleRegion{.98, 1.52, 1.93}
	if region.edge(0) != region.front || math.Abs(region.edge(math.Pi)-region.nape) > 1e-12 {
		t.Fatal("region boundaries")
	}
	fibres := rootedGroom(body, region, .08)[0]
	centre := v(0, 1.551, -.025)
	for base := 0; base < len(fibres.Rest); base += 4 * 17 * 97 {
		var root maths.Vec
		for _, p := range fibres.Rest[base : base+4] {
			root = root.Add(p.Scale(.25))
		}
		d := root.Sub(centre).Unit()
		ray := maths.Ray{At: centre.Add(d.Scale(.22)), Dir: d.Neg()}
		hit, ok := skull.Meet(ray, 0, .22)
		if !ok || root.Sub(ray.Point(hit.Along())).Len() > .00027 {
			t.Fatal(base, "root missed true scalp")
		}
	}
	for i, face := range body.faces {
		body.faces[i] = [3]int{face[2], face[1], face[0]}
	}
	reversed := rootedGroom(body, region, .08)[0]
	if reversed.Rest[0].Sub(fibres.Rest[0]).Len() > 1e-9 {
		t.Fatal("hair direction depends on skull winding")
	}
	skin := rgb(.56, .32, .21)
	if region.roots(v(.04, 1.33, .02), skin) != skin || region.roots(v(0, 1.64, 0), skin) == skin {
		t.Fatal("follicle pigment region")
	}
}

func TestRootedGroomRejectsMissingScalp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("accepted missing cranial surface")
		}
	}()
	rootedGroom(geom{pts: []maths.Vec{v(4, 4, 4), v(5, 4, 4), v(5, 5, 4)}, faces: [][3]int{{0, 1, 2}}}, follicleRegion{.98, 1.52, 1.93}, 0)
}
