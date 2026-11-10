//go:build darwin

package metal

import "testing"

const functions = `
#include <metal_stdlib>
using namespace metal;
kernel void first_value(device const float *p [[buffer(0)]],
 device float *out [[buffer(1)]], uint i [[thread_position_in_grid]]) {
 out[i]=p[3*i];
}
kernel void second_value(device const float *p [[buffer(0)]],
 device float *out [[buffer(1)]], uint i [[thread_position_in_grid]]) {
 out[i]=2*p[3*i];
}
`

func TestKernelSelectsFunctionsAndKeepsItsContext(t *testing.T) {
	kernel, err := Open(functions, "first_value")
	requireDevice(t, err)
	defer kernel.Close()
	points := [][3]float32{{2, 0, 0}, {5, 0, 0}}
	a, err := kernel.Points(points, nil)
	requireDevice(t, err)
	if err = kernel.Function("second_value"); err != nil {
		t.Fatal(err)
	}
	b, err := kernel.Points(points, nil)
	requireDevice(t, err)
	if len(a) != 2 || a[0] != 2 || a[1] != 5 || b[0] != 4 || b[1] != 10 {
		t.Fatal("function selection did not change GPU work")
	}
	if err = kernel.Function("first_value"); err != nil {
		t.Fatal(err)
	}
	c, err := kernel.Points(points, nil)
	if err != nil || c[0] != a[0] || c[1] != a[1] {
		t.Fatal("original entry point was lost")
	}
	if kernel.GPUTime() <= 0 || kernel.Device() == "" {
		t.Fatal("context profile was lost")
	}
}

func TestKernelRefusesMissingFunctionsWithoutLosingTheCurrentOne(t *testing.T) {
	kernel, err := Open(functions, "first_value")
	requireDevice(t, err)
	if err = kernel.Function("missing"); err == nil {
		t.Fatal("missing function accepted")
	}
	got, err := kernel.Points([][3]float32{{7, 0, 0}}, nil)
	if err != nil || got[0] != 7 {
		t.Fatal("failed selection changed the active pipeline")
	}
	kernel.Close()
	if err = kernel.Function("first_value"); err == nil {
		t.Fatal("closed context accepted a function")
	}
}
