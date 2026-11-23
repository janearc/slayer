package manufacture

import "testing"

func TestCassiniAndHuygensAreSeparateDetailedKinds(t *testing.T) {
	for _, name := range []string{"cassini", "huygens", "cassini-huygens"} {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatal(name, err)
		}
		made, err := makeOf(row, sow(1))
		if err != nil {
			t.Fatal(name, err)
		}
		total := 0
		for _, p := range made.parts {
			total += len(p.skin())
		}
		t.Logf("%s: %d parts, %d triangles", name, len(made.parts), total)
		if total < 1000 {
			t.Fatalf("%s: only %d faces", name, total)
		}
	}
}
