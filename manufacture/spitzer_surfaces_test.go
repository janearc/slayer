package manufacture

import "testing"

func TestSpitzerRefusesMalformedAndUnknownCellSurfaces(t *testing.T) {
	before := spitzerSurfaceRaw
	t.Cleanup(func() { spitzerSurfaceRaw = before })
	row, err := KindNamed("spitzer")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"material: [", "material: absent\nfinish: polished\n"} {
		spitzerSurfaceRaw = []byte(raw)
		made, err := Make(row, Sow(1))
		if err == nil || made.PartCount() != 0 {
			t.Fatal("invalid solar surface left a partial spacecraft", err)
		}
	}
}
