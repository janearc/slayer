//go:build draw && darwin

package metal

import (
	"errors"
	"testing"
)

// doubler is a kernel of its own, so the test's sources are in the cache
// only when the test has put them there.
const doubler = `
#include <metal_stdlib>
using namespace metal;
kernel void double_points(device const packed_float3 *points [[buffer(0)]],
                          device float *out [[buffer(1)]],
                          constant float &unused [[buffer(2)]],
                          uint id [[thread_position_in_grid]]) {
	out[id] = 2 * points[id].x;
}
`

// TestSharedLibraryCompilesOnce opens the same source twice and a second
// source once, and reads the cache: the pair adds two entries, not three.
func TestSharedLibraryCompilesOnce(t *testing.T) {
	before := cachedLibraries()
	first, err := Open(doubler, "double_points")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	again, err := Open(doubler, "double_points")
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	other, err := Open(doubler+"\n// a distinct source\n", "double_points")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if grown := cachedLibraries() - before; grown != 2 {
		t.Fatalf("three opens of two sources grew the cache by %d, "+
			"want 2", grown)
	}
}
