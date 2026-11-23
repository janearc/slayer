package manufacture

import "testing"

func TestNewHorizonsRegisteredAndHasThreeSourcedParts(t *testing.T) {
	row, err := kindNamed("new-horizons")
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(made.parts) != 3 {
		t.Fatalf("got %d parts", len(made.parts))
	}
	got := []int{7764, 10192, 4258}
	for i, p := range made.parts {
		if n := len(p.skin()); n != got[i] {
			t.Fatalf("part %d: %d faces, want %d", i, n, got[i])
		}
	}
}

func TestNewHorizonsVTADIsRegistered(t *testing.T) {
	row, err := kindNamed("new-horizons-vtad")
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(made.parts) != 12 {
		t.Fatalf("%d parts", len(made.parts))
	}
	total := 0
	for _, p := range made.parts {
		total += len(p.skin())
	}
	if total != 8260 {
		t.Fatalf("%d faces", total)
	}
}
