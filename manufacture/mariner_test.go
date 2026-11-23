package manufacture

import (
	"testing"

	"github.com/janearc/slayer/mesh"
)

func TestMariner10RegisteredDetailedAndBounded(t *testing.T) {
	row, err := kindNamed("mariner-10")
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if made.PartCount() < 600 {
		t.Fatalf("only %d components", made.PartCount())
	}
	faces := 0
	for _, p := range made.parts {
		faces += len(p.skin())
	}
	for _, group := range made.groups {
		for _, member := range group.Members() {
			faces += member.Mesh.(*mesh.Mesh).FaceCount()
		}
	}
	t.Logf("%d components, %d faces", made.PartCount(), faces)
	if faces < 6000 {
		t.Fatalf("only %d faces", faces)
	}
	bounds := made.Bounds()
	if bounds.Max.X-bounds.Min.X != 6.8 {
		t.Fatalf("solar span: %v", bounds)
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
