package main

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// tintedSun filters the direct lamp; the existing fill remains unchanged.
func tintedSun(sun set.Sun, power float64) maths.Light {
	red, green, blue := sun.Tint()
	return maths.RGB(red, green, blue).Scale(power)
}
