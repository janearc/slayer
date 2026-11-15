package manufacture

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

// bookedStairs finds a design's booked flight, read only.
func bookedStairs(kind string) *stairFlight {
	for _, design := range yardDesigns {
		if design.Kind == kind {
			return design.Stairs
		}
	}
	return nil
}

// accessStairs is the access-stairs flight, built the way the book
// books it, for tests that may break their own copy.
func accessStairs() stairFlight {
	return stairFlight{
		Steps: 8, Rise: 0.22, Going: 0.3, Width: 1.3,
		Treads: stairTreads{Material: "steel", Finish: paintFinish,
			Depth: 0.28, Thickness: 0.1, First: stairSpot{Y: 0.02, Z: 0}},
		Stringers: &stairSlope{Material: "steel", Finish: paintFinish,
			Radius: 0.065, Offset: 0.6,
			Low:  stairSpot{Y: 0.08, Z: 0.14},
			High: stairSpot{Y: 1.59, Z: -2.1}},
		Rail: &stairSlope{Material: "steel", Finish: paintFinish,
			Radius: 0.035, Offset: 0.67,
			Low:  stairSpot{Y: 1.15, Z: 0.12},
			High: stairSpot{Y: 2.69, Z: -2.1}},
		Infill: &stairInfill{Bars: &stairUprights{
			Material: "steel", Finish: paintFinish,
			Count: 3, Radius: 0.035, Offset: 0.67,
			First: stairBar{Z: 0.1, Bottom: 0.2, Top: 1.18},
			YStep: 0.77, ZStep: -1.1}},
	}
}

// the stairs constructor rebuilds the access-stairs the book used to
// place by hand: the same eighteen parts, in the same order, to the
// last bit of every number, so its geometry records cannot move.
func TestTheStairsConstructorRebuildsAccessStairsPartForPart(t *testing.T) {
	expect := []yardPart{}
	for step := range 8 {
		expect = append(expect, yardPart{Material: "steel",
			Box: &yardBox{
				At: maths.V(-0.65, 0.02+float64(step)*0.22,
					-float64(step)*0.3),
				Size: maths.V(1.3, 0.1, 0.28)}})
	}
	for _, side := range [2]float64{-1, 1} {
		expect = append(expect,
			yardPart{Material: "steel", Rod: &yardRod{
				From: maths.V(side*0.6, 0.08, 0.14),
				To:   maths.V(side*0.6, 1.59, -2.1), Radius: 0.065}},
			yardPart{Material: "steel", Rod: &yardRod{
				From: maths.V(side*0.67, 1.15, 0.12),
				To:   maths.V(side*0.67, 2.69, -2.1), Radius: 0.035}})
		for _, bar := range [3]float64{0, 1, 2} {
			expect = append(expect, yardPart{Material: "steel",
				Rod: &yardRod{
					From:   maths.V(side*0.67, 0.2+bar*0.77, 0.1-bar*1.1),
					To:     maths.V(side*0.67, 1.18+bar*0.77, 0.1-bar*1.1),
					Radius: 0.035}})
		}
	}
	for name, flight := range map[string]stairFlight{
		"the booked design": *bookedStairs("access-stairs"),
		"the constructor":   accessStairs(),
	} {
		pieces, err := flight.parts()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(pieces) != len(expect) {
			t.Fatalf("%s laid %d parts, the hand-placed design %d",
				name, len(pieces), len(expect))
		}
		if !reflect.DeepEqual(pieces, expect) {
			for index := range expect {
				if !reflect.DeepEqual(pieces[index], expect[index]) {
					t.Errorf("%s part %d is %v, the design placed %v",
						name, index, pieces[index], expect[index])
				}
			}
		}
	}
}

// the stairs constructor refuses nonsense instead of building it: no
// steps, no rise, no going, no width, no depth, no thickness, no
// uprights, no radius, no offset, and an infill that is neither bars
// nor a panel.
func TestTheStairsConstructorRefusesNonsenseParameters(t *testing.T) {
	broken := map[string]func(*stairFlight){
		"no steps":           func(f *stairFlight) { f.Steps = 0 },
		"no rise":            func(f *stairFlight) { f.Rise = 0 },
		"no going":           func(f *stairFlight) { f.Going = 0 },
		"no width":           func(f *stairFlight) { f.Width = 0 },
		"no tread depth":     func(f *stairFlight) { f.Treads.Depth = 0 },
		"no tread thickness": func(f *stairFlight) { f.Treads.Thickness = 0 },
		"no stringer radius": func(f *stairFlight) { f.Stringers.Radius = 0 },
		"no stringer offset": func(f *stairFlight) { f.Stringers.Offset = 0 },
		"no rail radius":     func(f *stairFlight) { f.Rail.Radius = 0 },
		"no bar count":       func(f *stairFlight) { f.Infill.Bars.Count = 0 },
		"no bar radius":      func(f *stairFlight) { f.Infill.Bars.Radius = 0 },
		"no bar offset":      func(f *stairFlight) { f.Infill.Bars.Offset = 0 },
		"no infill":          func(f *stairFlight) { f.Infill = &stairInfill{} },
		"bars and a panel": func(f *stairFlight) {
			f.Infill.Panel = &stairPanel{Material: "steel",
				Finish: paintFinish, Thickness: 0.02, Offset: 0.67,
				Bottom: 0.2, Top: 1.18, Front: 0.1, Back: -2.1}
		},
	}
	for name, breakIt := range broken {
		flight := accessStairs()
		breakIt(&flight)
		if _, err := flight.parts(); err == nil {
			t.Errorf("%s built a flight anyway", name)
		}
	}
}

// the stairs constructor lays what the access-stairs does not use:
// posts under the rail, and a panel instead of bars, each side after
// the parts the book's flight names, in the same order.
func TestTheStairsConstructorLaysPostsAndAPanel(t *testing.T) {
	flight := accessStairs()
	flight.Posts = &stairUprights{Material: "steel",
		Finish: paintFinish, Count: 2, Radius: 0.045, Offset: 0.67,
		First: stairBar{Z: 0.1, Bottom: 0.2, Top: 1.18},
		YStep: 0.77, ZStep: -1.1}
	flight.Infill = &stairInfill{Panel: &stairPanel{Material: "steel",
		Finish: paintFinish, Thickness: 0.02, Offset: 0.67,
		Bottom: 0.2, Top: 1.18, Front: 0.1, Back: -2.1}}
	pieces, err := flight.parts()
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 8+2*(1+1+2+1) {
		t.Fatalf("the flight laid %d parts, not %d", len(pieces), 8+2*5)
	}
	panel := pieces[12]
	if panel.Box == nil || panel.Material != "steel" {
		t.Fatalf("the panel did not follow the posts: %v", panel)
	}
	if panel.Box.Size.X != 0.02 || panel.Box.Size.Y != 0.98 ||
		panel.Box.Size.Z != 2.2 {
		t.Fatalf("the panel is %v big", panel.Box.Size)
	}
	if panel.Box.At.X != -0.68 || panel.Box.At.Y != 0.2 ||
		panel.Box.At.Z != -2.1 {
		t.Fatalf("the panel sits at %v", panel.Box.At)
	}
}

// the yard refuses a flight that names a material it does not carry or
// a finish its look does not wear, by name, and builds nothing.
func TestTheYardRefusesStairsThatNameWhatItDoesNotWear(t *testing.T) {
	for name, breakIt := range map[string]func(*stairFlight){
		"unknown material": func(f *stairFlight) {
			f.Treads.Material = "brass"
		},
		"another finish": func(f *stairFlight) {
			f.Treads.Finish = "flat"
		},
		"unknown finish": func(f *stairFlight) {
			f.Rail.Finish = "lacquered"
		},
	} {
		flight := accessStairs()
		breakIt(&flight)
		row := yardEquipment{Kind: "access-stairs", Stairs: &flight,
			Minimum: maths.V(-0.75, 0, -2.25),
			Maximum: maths.V(0.75, 2.85, 0.4)}
		if _, err := row.madeAt(Sow(5), "quick"); err == nil {
			t.Errorf("%s built stairs anyway", name)
		}
	}
}
