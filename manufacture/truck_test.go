package manufacture

import (
	"math"
	"testing"
)

// Each truck stands in a box of its published size, length by width by
// height, every matter it is made of inside that box, its tyres on the
// ground; and it rolls on as many tyres as its axles and twins say,
// with the HEMTT's spare besides.
func TestEveryTruckStandsInItsPublishedSize(t *testing.T) {
	cases := []struct {
		name             string
		long, wide, tall float64
		tyres            int
	}{
		{"hemtt", 10.389, 2.438, 3.003, 8 + 1},
		{"kamaz", 9.27, 2.55, 3.11, 6},
		{"unimog", 5.95, 2.402, 2.873, 4},
		{"tatra", 10.72, 2.5, 3.42, 8},
		{"scania-r", 6.05, 2.55, 3.53, 2 + 4},
		{"daf-xf", 6.1, 2.55, 3.95, 2 + 4},
	}
	rows := truckRows()
	if len(rows) != len(cases) {
		t.Fatalf("%d trucks, want %d", len(rows), len(cases))
	}
	for _, want := range cases {
		row, ok := rows[want.name].(truck)
		if !ok {
			t.Fatalf("no truck named %s", want.name)
		}
		made, err := row.made(sow(3))
		if err != nil {
			t.Fatalf("%s: %v", want.name, err)
		}
		box := *made.extent
		size := box.Max.Sub(box.Min)
		if !(math.Abs(size.Z-want.long) <= 1e-9 && math.Abs(size.X-want.wide) <= 1e-9 &&
			math.Abs(size.Y-want.tall) <= 1e-9) {
			t.Errorf("%s stands %.3f long, %.3f wide, %.3f tall", want.name,
				size.Z, size.X, size.Y)
		}
		for of, group := range made.groups {
			if !within(group.Bounds(), box) {
				t.Errorf("%s: matter %d reaches %v, outside its %v", want.name,
					of, group.Bounds(), box)
			}
		}
		rubber := made.groups[truckRubber]
		if rubber == nil {
			t.Fatalf("%s has no tyres", want.name)
		}
		if got := len(rubber.Members()); got != want.tyres {
			t.Errorf("%s rolls on %d tyres, want %d", want.name, got, want.tyres)
		}
		if low := rubber.Bounds().Min.Y; !(math.Abs(low) <= 1e-6) {
			t.Errorf("%s's tyres reach %v m below the ground", want.name, -low)
		}
	}
}

// The axles stand as far apart as published: the Unimog's 3.85 m, the
// Tatra's 2.15, 3.66 and 1.45 m, the KAMAZ's 3.69 and 1.32, the DAF's
// 3.8; and the HEMTT's 5.334 m between the middles of its pairs.
func TestTrucksAxlesStandAsPublished(t *testing.T) {
	gaps := map[string][]float64{
		"unimog": {3.85},
		"tatra":  {2.15, 3.66, 1.45},
		"kamaz":  {3.69, 1.32},
		"daf-xf": {3.8},
	}
	hemtt, err := hemttRow().rigged()
	if err != nil {
		t.Fatal(err)
	}
	a := hemtt.axles
	if got := (a[0]+a[1])/2 - (a[2]+a[3])/2; !(math.Abs(got-5.334) <= 1e-9) {
		t.Errorf("the HEMTT's pairs are %v m apart, want 5.334", got)
	}
	for name, want := range gaps {
		at, err := truckRows()[name].(truck).rigged()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for k, gap := range want {
			if got := at.axles[k] - at.axles[k+1]; !(math.Abs(got-gap) <= 1e-9) {
				t.Errorf("%s's axles %d and %d are %v m apart, want %v", name,
					k+1, k+2, got, gap)
			}
		}
	}
}

// The Tatra stands 410 mm clear of the ground, as its sheet says: its
// lowest steel is the foot of the differentials under its backbone,
// lower than its hubs. The Unimog's axle beams are as thick as put
// their undersides 460 mm up.
func TestTrucksStandAsClearOfTheGroundAsPublished(t *testing.T) {
	made, err := tatraRow().made(sow(3))
	if err != nil {
		t.Fatal(err)
	}
	if low := made.groups[truckChassis].Bounds().Min.Y; !(math.Abs(low-0.41) <= 1e-9) {
		t.Errorf("the Tatra's lowest steel is %v m up, want 0.41", low)
	}
	unimog, err := unimogRow().rigged()
	if err != nil {
		t.Fatal(err)
	}
	if under := unimog.radius - unimog.beam(); !(math.Abs(under-0.46) <= 1e-9) {
		t.Errorf("the Unimog's axles are %v m up underneath, want 0.46", under)
	}
}

// A truck that cannot stand is refused: a seed never sown, no axles,
// axles out of order, tyres wider than the truck, a bed lower than its
// tyres, a cab longer than the truck, and a bonnet above the roof.
func TestATruckThatCannotStandIsRefused(t *testing.T) {
	if _, err := hemttRow().made(seed{}); err != errGood {
		t.Errorf("an unsown seed: got %v, want %v", err, errGood)
	}
	broken := []func(*truck){
		func(row *truck) { row.axles = nil },
		func(row *truck) { row.axles[1], row.axles[2] = row.axles[2], row.axles[1] },
		func(row *truck) { row.track = truckMetres(2.4) },
		func(row *truck) { row.frame = truckMetres(1.0) },
		func(row *truck) { row.cab.long = truckMetres(11) },
		func(row *truck) {
			row.cab.nose, row.cab.bonnet = truckMetres(0.7), truckMetres(3.5)
		},
		func(row *truck) { row.twins = 5 },
		func(row *truck) { row.clearance = truckMetres(0.7) },
	}
	for i, breaks := range broken {
		row := hemttRow()
		breaks(&row)
		if _, err := row.made(sow(3)); err != errTruck {
			t.Errorf("broken truck %d: got %v, want %v", i, err, errTruck)
		}
	}
}
