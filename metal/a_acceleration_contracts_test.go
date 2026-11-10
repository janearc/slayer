//go:build darwin

package metal

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

// stridedTriangle retains three positions and an arbitrary padding sentinel.
// Padding is outside the promised float3 position and must not be validated.
func stridedTriangle(stride int, padding uint32) []byte {
	data := make([]byte, 3*stride)
	for vertex, position := range [][3]float32{{-1, -1, 0}, {1, -1, 0}, {0, 1, 0}} {
		for offset := 0; offset+4 <= stride; offset += 4 {
			binary.LittleEndian.PutUint32(data[vertex*stride+offset:], padding)
		}
		for axis, value := range position {
			binary.LittleEndian.PutUint32(data[vertex*stride+axis*4:], math.Float32bits(value))
		}
	}
	return data
}

func TestNativeLayoutAcceptsCompactPositionsAndIgnoresPadding(t *testing.T) {
	for _, stride := range []int{12, 16, 20, 28} {
		data := stridedTriangle(stride, math.Float32bits(float32(math.NaN())))
		plan, err := NewAcceleration(data, stride)
		if err != nil {
			t.Fatalf("valid float3 at stride %d: %v", stride, err)
		}
		triangle, err := plan.Triangles(0, 1)
		if err != nil || triangle != 0 {
			t.Fatalf("stride %d first triangle: %d, %v", stride, triangle, err)
		}
		instance, err := Placed(triangle, translated(0, 0, -5), 3)
		if err != nil {
			t.Fatal(err)
		}
		root, err := plan.Instances(instance)
		if err != nil || root != 1 {
			t.Fatalf("stride %d root: %d, %v", stride, root, err)
		}
	}
}

func TestNativeLayoutChecksEveryPositionComponent(t *testing.T) {
	for _, stride := range []int{12, 16, 20} {
		for vertex := range 3 {
			for axis := range 3 {
				for _, invalid := range []float32{
					float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)),
				} {
					data := stridedTriangle(stride, 0)
					binary.LittleEndian.PutUint32(data[vertex*stride+axis*4:], math.Float32bits(invalid))
					if _, err := NewAcceleration(data, stride); err == nil ||
						!strings.Contains(err.Error(), "nonfinite triangle position") {
						t.Fatalf("stride %d vertex %d axis %d value %g: %v", stride, vertex, axis, invalid, err)
					}
				}
			}
		}
	}
}

func TestNativeLayoutRefusesIncompleteRecordsAndTriangles(t *testing.T) {
	for _, test := range []struct{ stride, bytes int }{
		{12, 12}, {12, 24}, {16, 16}, {16, 32},
		{12, 35}, {16, 47}, {13, 39}, {14, 42}, {15, 45},
	} {
		if _, err := NewAcceleration(make([]byte, test.bytes), test.stride); err == nil ||
			!strings.Contains(err.Error(), "invalid triangle vertex layout") {
			t.Fatalf("stride %d bytes %d: %v", test.stride, test.bytes, err)
		}
	}
}

// Every column participates in invertibility, including off-diagonal terms.
// These integer matrices have exact determinants independent of rounding.
func TestNativePlacementsAcceptInvertibleAndRefuseSingularBases(t *testing.T) {
	for _, test := range []struct {
		name   string
		matrix [12]float32
		valid  bool
	}{
		{"cyclic", [12]float32{0, 1, 0, 0, 0, 1, 1, 0, 0}, true},
		{"swap", [12]float32{0, 1, 0, 1, 0, 0, 0, 0, 1}, true},
		{"shear", [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 10}, true},
		{"dependent", [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 9}, false},
		{"duplicate", [12]float32{1, 2, 3, 1, 2, 3, 4, 5, 7}, false},
	} {
		_, err := Placed(0, test.matrix, 3)
		if (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}
	for component := range 12 {
		for _, invalid := range []float32{
			float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)),
		} {
			matrix := translated(2, 3, 4)
			matrix[component] = invalid
			if _, err := Placed(0, matrix, 3); err == nil {
				t.Fatalf("accepted nonfinite matrix component %d: %g", component, invalid)
			}
		}
	}
}

func TestNativePlacementsRefuseExactDependentIntegerColumns(t *testing.T) {
	columns := [][3]float32{
		{1, 2, 3}, {-2, 3, 5}, {7, -1, 2}, {3, 0, -4}, {0, -5, 1},
	}
	for _, a := range columns {
		for _, b := range columns {
			for _, factor := range []float32{-2, -1, 0, 1, 2} {
				// c=a+factor*b makes the basis singular. All operations
				// and triple products are exact small integers.
				c := [3]float32{a[0] + factor*b[0], a[1] + factor*b[1], a[2] + factor*b[2]}
				matrix := [12]float32{a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2]}
				if _, err := Placed(0, matrix, 3); err == nil {
					t.Fatalf("accepted dependent columns %v, %v, %v", a, b, c)
				}
			}
		}
	}
}

func TestNativeRootImmediatelyAfterItsLastNodeIsRefused(t *testing.T) {
	plan, err := NewAcceleration(stridedTriangle(12, 0), 12)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Triangles(0, 1); err != nil {
		t.Fatal(err)
	}
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	if err := kernel.Accelerate(plan, 1); err == nil ||
		!strings.Contains(err.Error(), "instance root") {
		t.Fatalf("root immediately after the last node: %v", err)
	}
	if err := kernel.Accelerate(nil, 0); err == nil ||
		!strings.Contains(err.Error(), "instance root") {
		t.Fatalf("nil root plan: %v", err)
	}
}
