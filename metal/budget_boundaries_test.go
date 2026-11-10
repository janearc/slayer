//go:build darwin

package metal

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// reservedBytes supplies a real oversized input without filling physical RAM.
// Anonymous pages remain untouched except for one early-failure sentinel.
// Inputs also fail the next check if a mutant removes their size guard, so
// such a mutant never copies the reservation or walks billions of entries.
func reservedBytes(t *testing.T, count int) []byte {
	t.Helper()
	data, err := syscall.Mmap(-1, 0, count, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := syscall.Munmap(data); err != nil {
			t.Error(err)
		}
	})
	return data
}

func TestGridInputLimitPrecedesOutputAllocation(t *testing.T) {
	kernel, err := Open(fillGrid, "fill_grid")
	requireDevice(t, err)
	defer kernel.Close()
	input := reservedBytes(t, math.MaxInt32+1)
	_, err = kernel.Grid(math.MaxInt, 1, 1, 1, [][]byte{input})
	if !errors.Is(err, errTooBig) {
		t.Fatalf("input must be refused before the invalid output: %v", err)
	}
}

func TestNativeVertexByteLimitPrecedesScanning(t *testing.T) {
	count := (math.MaxInt32/48 + 1) * 48
	data := reservedBytes(t, count)
	binary.LittleEndian.PutUint32(data, math.Float32bits(float32(math.NaN())))
	_, err := NewAcceleration(data, 16)
	if err == nil || !strings.Contains(err.Error(), "invalid triangle vertex layout") {
		t.Fatalf("oversized aligned triangle data must precede the NaN: %v", err)
	}
}

func TestNativePlacementLimitPrecedesReadingThePlacements(t *testing.T) {
	plan, err := NewAcceleration(nil, 16)
	if err != nil {
		t.Fatal(err)
	}
	triangle, err := plan.Triangles(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	place, err := Placed(triangle, translated(0, 0, 0), 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Instances(place); err != nil {
		t.Fatal(err)
	}
	count := math.MaxInt32
	data := reservedBytes(t, count*int(unsafe.Sizeof(Placement{})))
	places := unsafe.Slice((*Placement)(unsafe.Pointer(&data[0])), count)
	// Zeroed matrices are singular, so removing the count guard fails early
	// with the wrong diagnostic rather than allocating a huge result.
	if _, err := plan.Instances(places...); err == nil ||
		!strings.Contains(err.Error(), "too many instance placements") {
		t.Fatalf("placement limit must precede invalid transforms: %v", err)
	}
}
