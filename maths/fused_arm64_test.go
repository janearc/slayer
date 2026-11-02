//go:build arm64

package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// Upright and LookAtUp answers for three inputs are held bit for bit.
//
// They are recorded from the code on arm64, and only arm64 was measured.
//
// arm64 fuses the multiplies and adds of Dot and Cross, so the bits depend on
// fused multiply-add.
//
// On amd64 at the default GOAMD64=v1 nothing fuses, so inlining unmeasured
// cannot move bits there and the directive needs holding only where fusion
// happens.
//
// The recorded bits would also differ on amd64, which is why this is the only
// test in an arm64 file. Each answer moves by a last bit if unmeasured is
// inlined. So this fails on arm64 if the directive that stops that is dropped.
func TestOutOfLineKeepsEveryBit(test *testing.T) {
	origin := vector(test, "origin")
	for _, name := range []string{"moved 1", "moved 2", "moved 3"} {
		from := vector(test, name)
		upright := orientation(test, "upright on "+name)
		if got := maths.Upright(from); got != upright {
			test.Errorf("%s: standing on it is %v, recorded %v", name, got, upright)
		}
		looking := orientation(test, "looking from "+name)
		if got := maths.LookAtUp(from, origin, maths.Up); got != looking {
			test.Errorf("%s: looking from it is %v, recorded %v", name, got, looking)
		}
	}
}
