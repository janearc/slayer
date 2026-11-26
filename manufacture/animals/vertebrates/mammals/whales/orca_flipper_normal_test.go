package whales

import (
	"math"
	"runtime"
	"testing"
)

// The cross product's X terms nearly cancel. The chosen evaluation retains
// the original binding even if an unrelated package initializer changes how
// the compiler fuses arithmetic.
func TestOrcaFlipperNormalRetainsBindingPrecision(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("precision witness was captured on Darwin/ARM64")
	}
	const original = 1.053477413162646e-17
	if math.Float64bits(orcaFlipperNormal.X) != math.Float64bits(original) {
		t.Fatalf("flipper normal X changed: %.17g", orcaFlipperNormal.X)
	}
}
