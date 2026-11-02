package shortcut

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// Threshold is the transmittance below which a ray may stop: what is
// left of its light, at most keep times top, the brightest light the
// scene makes, develops through the tone to less than half an eight-bit
// step: a pixel near a step's edge may still flip by one.
//
// The error of stopping is bounded by the remainder, keep times top,
// which is under that step by construction; in an opaque band a ray
// ends within a few steps.
//
// It is found once a picture by bisection through the tone itself, so
// a filmic shoulder or an exposure moves it as it should.
func Threshold(tone colour.Tone, top float64) float64 {
	if top <= 0 {
		top = 1
	}
	shows := func(keep float64) bool {
		f := frame.New(1, 1)
		l := keep * top
		f.Set(0, 0, frame.Pixel{R: float32(l), G: float32(l), B: float32(l), Cover: 1})
		c := tone.Develop(f).RGBAAt(0, 0)
		return c.R > 0 || c.G > 0 || c.B > 0
	}
	lo, hi := 0.0, 1.0
	if !shows(hi) {
		return hi
	}
	for i := 0; i < 40; i++ {
		mid := (lo + hi) / 2
		if shows(mid) {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
}

// Remainder is the most a ray with transmittance keep could still add
// to a pixel, with top the brightest light: the bound on the cutoff's
// error.
func Remainder(keep, top float64) float64 { return keep * top }

// Brightest is the brightest light a world can hand a ray, as far as
// its lamps say: the largest channel of any lamp's strength, or one
// when it has none, which a glowing thing may exceed and the budget
// then absorbs.
func Brightest(lamps []maths.Light) float64 {
	top := 0.0
	for _, l := range lamps {
		for _, c := range []float64{l.R, l.G, l.B} {
			if c > top {
				top = c
			}
		}
	}
	if top <= 0 {
		return 1
	}
	return top
}
