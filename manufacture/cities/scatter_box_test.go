package cities

import "testing"

func TestAScatterOverARectangleStaysInsideItAndIsTheSameEachTime(t *testing.T) {
	t.Parallel()
	box := Quarter{West: 100, South: 200, East: 160, North: 330}
	ground := Ground{Box: &box}
	first, err := scatter{}.Sites(ground, Item{Kind: "boat"}, Seed(9), 40)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := scatter{}.Sites(ground, Item{Kind: "boat"}, Seed(9), 40)
	other, _ := scatter{}.Sites(ground, Item{Kind: "boat"}, Seed(10), 40)
	same, differs := true, false
	for index, site := range first.Sites {
		if !box.holds(site.East, site.North) {
			t.Errorf("site %d, %v east %v north, is outside %+v", index,
				site.East, site.North, box)
		}
		same = same && site == again.Sites[index]
		differs = differs || site != other.Sites[index]
	}
	if len(first.Sites) != 40 || !same || !differs {
		t.Errorf("40 sites: got %d, same seed the same: %v, another seed "+
			"another: %v", len(first.Sites), same, differs)
	}
}

func TestAScatterOverAWedgeIsAsItWasBeforeRectanglesWereOffered(t *testing.T) {
	t.Parallel()
	ground := Ground{East: 10, North: 20, From: 0, To: 0, Near: 5, Far: 50}
	laid, err := scatter{}.Sites(ground, Item{Kind: "boat"}, Seed(3), 25)
	if err != nil {
		t.Fatal(err)
	}
	for index, site := range laid.Sites {
		if !ground.contains(site.East, site.North) {
			t.Errorf("site %d, %v east %v north, is outside the wedge", index,
				site.East, site.North)
		}
	}
}

func TestAScatterWithNothingToLayIsRefused(t *testing.T) {
	t.Parallel()
	box := Quarter{West: 0, South: 0, East: 10, North: 10}
	if _, err := (scatter{}).Sites(Ground{Box: &box}, Item{Kind: "boat"},
		Seed(1), 0); err == nil {
		t.Error("a scatter of none was laid")
	}
}
