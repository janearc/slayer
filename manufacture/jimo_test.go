package manufacture

import "testing"

func TestJIMORegisteredDetailedAndBounded(t *testing.T) {
	row, err := kindNamed("jimo")
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(made.parts) < 600 {
		t.Fatalf("only %d components", len(made.parts))
	}
	faces := 0
	for _, p := range made.parts {
		faces += len(p.skin())
	}
	t.Logf("%d components, %d faces", len(made.parts), faces)
	if faces < 6000 {
		t.Fatalf("only %d faces", faces)
	}
	bounds := made.Bounds()
	if bounds.Max.X-bounds.Min.X != 11.6 {
		t.Fatalf("radiator span: %v", bounds)
	}
	for _, p := range made.parts {
		for _, f := range p.skin() {
			for _, point := range f {
				q := point
				if q.X < bounds.Min.X || q.X > bounds.Max.X || q.Y < bounds.Min.Y || q.Y > bounds.Max.Y || q.Z < bounds.Min.Z || q.Z > bounds.Max.Z {
					t.Fatalf("point outside bounds: %v", q)
				}
			}
		}
	}
}
