//go:build darwin

package fuelbed

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/metal"
)

func TestZeroTransportKeepsTheLegacyMetalParameterBlock(test *testing.T) {
	spec := failureSpec()
	legacy, err := NewMetal(spec)
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip(err)
	}
	if err != nil {
		test.Fatal(err)
	}
	defer legacy.Close()
	zero, err := NewMetalAdvected(spec, Transport{})
	if err != nil {
		test.Fatal(err)
	}
	defer zero.Close()
	if !bytes.Equal(legacy.params, zero.params) || len(zero.params) != 32 ||
		legacy.stepFunction() != zero.stepFunction() {
		test.Fatal("zero transport changed the legacy Metal ABI or entry point")
	}
}

func TestRoundedTransportChecksTheThermalSumAndEveryEncodedWeight(test *testing.T) {
	if err := validRoundedTransport(advectionWeights{centre: .5, west: .2, north: .3}); err != nil {
		test.Fatal(err)
	}
	for _, weights := range []advectionWeights{
		{centre: 1, west: .2}, {centre: -1}, {centre: math.NaN()},
		{centre: math.Inf(1)}, {centre: .5, burn: 2},
	} {
		if err := validRoundedTransport(weights); err == nil {
			test.Fatal("invalid encoded weights were accepted", weights)
		}
	}
}
