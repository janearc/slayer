package texture

import (
	"image"
	"math"
	"sync"

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
	read := reader(pic.Image)
	sample := func(px, py int) maths.Light {
		px = ((px % w) + w) % w
		py = max(0, min(h-1, py))
		r, g, bl := read(b.Min.X+px, b.Min.Y+py)
		return maths.RGB(r, g, bl)
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

// linear8 is linear light for each srgb byte, as the 16-bit part the
// byte makes, so the table agrees with linear16 where the two overlap.
var linear8 = func() (t [256]float64) {
	for i := range t {
		t[i] = srgb.ToLinear(float64(i*0x101) / 65535)
	}
	return t
}()

// linear16 is linear light for each 16-bit srgb part, filled on first use.
// A color.Color that returns a channel above 65535 breaks the color.Color
// contract and is not supported: the table has no entry for it.
var linear16 = sync.OnceValue(func() *[65536]float64 {
	var t [65536]float64
	for i := range t {
		t[i] = srgb.ToLinear(float64(i) / 65535)
	}
	return &t
})

// reader is a way to read an image's pixels as linear light, each channel
// what srgb.ToLinear makes of the image's own 16-bit part over 65535. the
// common image types are read without boxing a colour; any other is read
// through its At, and every path converts by a table.
func reader(img image.Image) func(x, y int) (r, g, b float64) {
	switch m := img.(type) {
	case *image.RGBA:
		return func(x, y int) (float64, float64, float64) {
			c := m.RGBAAt(x, y)
			return linear8[c.R], linear8[c.G], linear8[c.B]
		}
	case *image.NRGBA:
		return func(x, y int) (float64, float64, float64) {
			c := m.NRGBAAt(x, y)
			r, g, b, _ := c.RGBA()
			t := linear16()
			return t[r], t[g], t[b]
		}
	case *image.Gray:
		return func(x, y int) (float64, float64, float64) {
			v := linear8[m.GrayAt(x, y).Y]
			return v, v, v
		}
	}
	return func(x, y int) (float64, float64, float64) {
		r, g, b, _ := img.At(x, y).RGBA()
		t := linear16()
		return t[r], t[g], t[b]
	}
}
