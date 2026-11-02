//go:build darwin

package main

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/metal"
)

// the same hillside on the gpu is the picture the cpu draws, night
// and lamps and all: the kernel's cone lamp and emit's fall agree, on
// things the manufactory made. the two are held to a thousandth of
// the picture's light; gpu's own tests hold them far closer on scenes
// of their own.
func TestTheGPUDrawsTheCommandsHillsideAsTheCPUDoes(t *testing.T) {
	arguments := append(only(map[string]string{"boulder": "3",
		"fern": "4", "spruce": "1", "campfire": "1", "spotlight": "1"}),
		"-night")
	given := mustParse(t, arguments...)
	world, made, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	slow, err := draw(world, given, made.across)
	if err != nil {
		t.Fatal(err)
	}
	given.onGPU = true
	fast, err := draw(world, given, made.across)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	apart := 0.0
	for index, pixel := range slow.Pix {
		other := fast.Pix[index]
		apart += math.Abs(float64(pixel.R-other.R)) +
			math.Abs(float64(pixel.G-other.G)) +
			math.Abs(float64(pixel.B-other.B))
	}
	if whole := light(slow); whole <= 0 || apart > whole/1000 {
		t.Errorf("the gpu's picture is %v apart from the cpu's %v of "+
			"light", apart, whole)
	}
}
