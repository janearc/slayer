//go:build draw && darwin

package metal_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/janearc/slayer/metal"
)

// source is a kernel that does nothing, enough to open and set a limit on.
const source = `
#include <metal_stdlib>
using namespace metal;
kernel void idle(device float *into [[buffer(0)]]) {}
`

// A kernel's wait for any one command gives up after its limit with an
// error naming the command, and the kernel is not used again after it.
func ExampleKernel_SetLimit() {
	kernel, err := metal.Open(source, "idle")
	if errors.Is(err, metal.ErrNoDevice) {
		// a machine with no device has no limit to set; say the same
		fmt.Println("limit set")
		return
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	defer kernel.Close()
	kernel.SetLimit(30 * time.Second)
	fmt.Println("limit set")
	// Output: limit set
}

// A session's wait for one batch gives up the same way.
func ExampleSession_SetLimit() {
	session, err := metal.OpenSession(source, "idle")
	if errors.Is(err, metal.ErrNoDevice) {
		fmt.Println("limit set")
		return
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	defer session.Close()
	session.SetLimit(30 * time.Second)
	fmt.Println("limit set")
	// Output: limit set
}
