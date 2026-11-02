package texture

import (
	"image"
	"math"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// Picture is an image wrapped round the y axis as the map of a sphere: an
// equirectangular picture, longitude across and latitude down, as nasa
// publishes the blue marble and the black marble.
//
// At a direction from the centre it is the picture's colour there, read as srgb
// and given back as linear light, blended between the four nearest pixels. Turn
// moves the picture's left edge round the sphere, in radians, and Gain scales
// the light, zero read as one.
//
// it is wrapped mirrored: its longitude grows from +X toward +Z, where the
// package's spin, and the earth's, run from +X toward -Z, so a map's west lands
// east and the world seen is its mirror. it was drawn so, and is not turned
// here. a picture of no pixels is dark.
type Picture struct {
	Image image.Image
	Turn  float64
	Gain  float64
}

// At is the picture's light in the direction of p.
func (pic Picture) At(p maths.Vec) maths.Light {
	if pic.Image == nil || pic.Image.Bounds().Empty() {
		return maths.Light{}
	}
	d := p.Unit()
	lon := math.Atan2(d.Z, d.X) + pic.Turn
	lat := math.Asin(math.Max(-1, math.Min(1, d.Y)))
	u := lon/(2*math.Pi) + 0.5
	u -= math.Floor(u)
	v := 0.5 - lat/math.Pi
	b := pic.Image.Bounds()
	w, h := b.Dx(), b.Dy()
	x, y := u*float64(w)-0.5, v*float64(h)-0.5
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	sample := func(px, py int) maths.Light {
		px = ((px % w) + w) % w
		py = max(0, min(h-1, py))
		r, g, bl, _ := pic.Image.At(b.Min.X+px, b.Min.Y+py).RGBA()
		return maths.RGB(srgb.ToLinear(float64(r)/65535),
			srgb.ToLinear(float64(g)/65535),
			srgb.ToLinear(float64(bl)/65535))
	}
	top := sample(x0, y0).Scale(1 - fx).Add(sample(x0+1, y0).Scale(fx))
	bottom := sample(x0, y0+1).Scale(1 - fx).
		Add(sample(x0+1, y0+1).Scale(fx))
	l := top.Scale(1 - fy).Add(bottom.Scale(fy))
	if pic.Gain != 0 {
		l = l.Scale(pic.Gain)
	}
	return l
}

// Water is whether the picture is water in the direction of p: blue
// over red and green, as the sea is in a map of the earth by day.
func (pic Picture) Water(p maths.Vec) bool {
	c := pic.At(p)
	return c.B > c.R*1.15 && c.B > c.G*1.05
}
