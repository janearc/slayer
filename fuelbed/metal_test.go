//go:build darwin

package fuelbed_test

import (
	"github.com/janearc/slayer/fuelbed"
	"math"
	"testing"
)

func TestMetalEvolvesHeatAndFuelLikeTheReference(t *testing.T) {
	s := fixtureWithHot()
	cpu, e := fuelbed.New(s)
	if e != nil {
		t.Fatal(e)
	}
	device, e := fuelbed.NewMetal(s)
	requireFuelDevice(t, e)
	defer device.Close()
	for _, at := range []float64{0, 1, 10, 2, 10} {
		want, e := cpu.At(at)
		if e != nil {
			t.Fatal(e)
		}
		got, e := device.At(at)
		if e != nil {
			t.Fatal(e)
		}
		for i, c := range got {
			if math.Abs(c.Excess-want[i].Excess) > .1 || math.Abs(c.Fuel-want[i].Fuel) > .0001 {
				t.Fatal(at, i, c, want[i])
			}
		}
	}
	device.Close()
	if _, e := device.At(1); e == nil {
		t.Fatal("closed device accepted")
	}
}
func TestMetalDoesNotInventFuelOrIgnition(t *testing.T) {
	s := fixture()
	for i := range s.Initial {
		s.Initial[i].Fuel = 1
	}
	s.Initial[4].Fuel = 0
	device, e := fuelbed.NewMetal(s)
	requireFuelDevice(t, e)
	defer device.Close()
	got, e := device.At(20)
	if e != nil {
		t.Fatal(e)
	}
	for i, c := range got {
		if c.Excess != 0 || c.Fuel != s.Initial[i].Fuel {
			t.Fatal(c)
		}
	}
	if _, e := device.At(-1); e == nil {
		t.Fatal("negative time")
	}
	s.Cell = 0
	if _, e := fuelbed.NewMetal(s); e == nil {
		t.Fatal("invalid grid")
	}
}

func TestMetalKeepsNeighbourStateAcrossMultipleThreadgroups(t *testing.T) {
	s := fixture()
	s.Width = 47
	s.Depth = 35
	s.Initial = make([]fuelbed.Cell, s.Width*s.Depth)
	for z := 0; z < s.Depth; z++ {
		for x := 0; x < s.Width; x++ {
			i := z*s.Width + x
			s.Initial[i] = fuelbed.Cell{Excess: float64((x*11 + z*19) % 900), Fuel: float64((x+z)%4) / 3}
		}
	}
	cpu, e := fuelbed.New(s)
	if e != nil {
		t.Fatal(e)
	}
	device, e := fuelbed.NewMetal(s)
	requireFuelDevice(t, e)
	defer device.Close()
	want, e := cpu.At(3)
	if e != nil {
		t.Fatal(e)
	}
	got, e := device.At(3)
	if e != nil {
		t.Fatal(e)
	}
	for i, c := range got {
		if math.Abs(c.Excess-want[i].Excess) > .1 || math.Abs(c.Fuel-want[i].Fuel) > .0001 {
			t.Fatal("threadgroup boundary", i, c, want[i])
		}
	}
}
