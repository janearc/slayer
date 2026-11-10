package manufacture

import "testing"

func TestSingleFireLayoutAcceptsItsExplicitOneSourceCount(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		spec := burnerFixture()
		spec.Layout = FireLayout{Pattern: "single", Count: count}
		_, err := Burn(spec)
		if count <= 1 && err != nil {
			t.Fatalf("single layout count %d refused: %v", count, err)
		}
		if count > 1 && err == nil {
			t.Fatal("single layout accepted two sources")
		}
	}
}
