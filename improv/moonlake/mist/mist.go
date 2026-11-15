package mist

import (
	"fmt"
	"math"
)

type Image struct {
	Width, Height int
	Scale         float64
	Pixels        []float32
}

type Config struct {
	Top, Bottom, Shore           float64
	Strength, Seconds            float64
	Seed                         uint64
	Colour                       [3]float64
	Layers                       float64
	MoonRef, WaterMask, TreeMask *Image
}

const MaxPixels = 16777216

// dimensions is true when width and height are positive, the picture holds
// no more than MaxPixels pixels, and twelve bytes a pixel fits in an int.
func dimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= MaxPixels/height && width <= int(^uint(0)>>1)/height/12
}

// finite is true when v is neither NaN nor an infinity.
func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// valid is true when image has good dimensions, a finite scale, and exactly
// three values per pixel. It does not check the pixel values.
func valid(image Image) bool {
	return dimensions(image.Width, image.Height) && finite(image.Scale) && len(image.Pixels) == image.Width*image.Height*3
}

// Apply lays mist over image and returns the result. The mist thins upward
// from the shore, is densest far across the lake, and is stronger near the
// moon's column; it moves with Seconds and Seed. It stays off the TreeMask,
// and below the shore it follows the WaterMask. It refuses an invalid image,
// a nonfinite config, Shore not between Bottom and Top, or Layers below one.
func Apply(image Image, config Config) (Image, error) {
	if !valid(image) {
		return Image{}, fmt.Errorf("invalid mist image")
	}
	for _, value := range []float64{config.Top, config.Bottom, config.Shore, config.Strength, config.Seconds, config.Layers, config.Colour[0], config.Colour[1], config.Colour[2]} {
		if !finite(value) {
			return Image{}, fmt.Errorf("nonfinite mist configuration")
		}
	}
	if config.Top <= config.Bottom || config.Shore <= config.Bottom || config.Shore >= config.Top || config.Layers < 1 {
		return Image{}, fmt.Errorf("invalid mist configuration")
	}
	width, height, pixels := image.Width, image.Height, image.Pixels
	top, bottom, shore, strength := config.Top, config.Bottom, config.Shore, config.Strength
	seed, seconds := config.Seed, config.Seconds
	span := top - bottom
	shoreRow := (top - shore) / span * float64(height)
	rise := 1.6 / span * float64(height)
	if !finite(span) || !finite(rise) || !finite(shoreRow) || shoreRow < 1 || shoreRow >= float64(height) {
		return Image{}, fmt.Errorf("invalid mist shore geometry")
	}
	for _, optional := range []struct {
		name  string
		image *Image
	}{
		{"moon reference", config.MoonRef}, {"water mask", config.WaterMask}, {"tree mask", config.TreeMask},
	} {
		if optional.image != nil && (!valid(*optional.image) || optional.image.Width != width || optional.image.Height != height) {
			return Image{}, fmt.Errorf("invalid %s image", optional.name)
		}
	}
	at := func(x, fromTop int) int { return ((height-1-fromTop)*width + x) * 3 }
	moon, column := moonOf(pixels, width, height, at, int(shoreRow))
	// with lights in the sky, the moon is taken from the plain plate
	if config.MoonRef != nil {
		moon, column = moonOf(config.MoonRef.Pixels, width, height, at, int(shoreRow))
	}
	lake := float64(height) - shoreRow
	// WATERMASK names a render of the lake glowing alone: below the far
	// shore, the mist lies only where it shows the water
	var water []float32
	most := float32(0)
	if config.WaterMask != nil {
		water = config.WaterMask.Pixels
		for _, v := range water {
			most = max(most, v)
		}
	}
	// TREEMASK names a render of the close spruces glowing alone: they
	// stand in front of all the mist, so none lies over them
	var trees []float32
	treesMost := float32(0)
	if config.TreeMask != nil {
		trees = config.TreeMask.Pixels
		for _, v := range trees {
			treesMost = max(treesMost, v)
		}
	}
	out := append([]float32(nil), pixels...)
	for r := 0; r < height; r++ {
		below := float64(r) - shoreRow
		if below < -rise {
			continue
		}
		near := math.Max(0, below/lake)
		// above the water it thins upward; on it, it is densest far off,
		// where a look runs through most of it
		up := smooth(-rise, 0, below)
		far := 0.35 + 0.65*math.Exp(-near*1.6)
		for x := 0; x < width; x++ {
			off := (float64(x) - column) / (0.28 * float64(width))
			forward := 0.25 + 0.75*math.Exp(-off*off)
			// the veil: a soft glow everywhere the mist is; its lengths in
			// a 1280-wide frame's pixels, scaled to this frame's
			k := float64(width) / 1280
			u := float64(x)/((260+500*near)*k) + 0.05*seconds
			v := float64(r) / ((30 + 120*near) * k)
			veil := 0.55 + 0.45*fbm(u, v, seed+11)
			// the wisps: curling, drifting with the air
			wu := float64(x)/((90+300*near)*k) + 0.12*seconds
			wv := float64(r)/((14+60*near)*k) - 0.04*seconds
			wisps := smooth(0.42, 0.72, curled(wu, wv, seconds, seed))
			d := up * far * forward * (0.45*veil + 0.75*wisps)
			to := at(x, r)
			if trees != nil && trees[to]+trees[to+1]+trees[to+2] > 0.3*treesMost {
				continue
			}
			if water != nil && below > 0 {
				// eased in over the first 30 rows below the far shore, so
				// the mist over the far wall meets the lake's with no line
				on := float64(water[to]+water[to+1]+water[to+2]) / float64(3*most)
				ease := smooth(0, 30, below)
				d *= (1 - ease) + ease*math.Min(1, 1.25*on)
			}
			d = thickened(d, config.Layers)
			for c := 0; c < 3; c++ {
				// moonlight on mist is a cool white, the disc's brightness
				// in MIST_COLOUR, not the low disc's amber
				mist := float32(strength * d * config.Colour[c] *
					(moon[0] + moon[1] + moon[2]) / 3)
				out[to+c] = out[to+c]*float32(1-0.6*math.Min(1, d)) + mist
			}
		}
	}
	return Image{Width: width, Height: height, Scale: image.Scale, Pixels: out}, nil
}

// thickened is a pixel's mist, n layers of it: what one layer leaves
// clear, n of them leave to the nth power. Above one, a pixel's mist is
// already as thick as it comes, and keeps what is over.
func thickened(d, n float64) float64 {
	return 1 - math.Pow(1-math.Min(1, d), n) + math.Max(0, d-1)
}

// moonOf is the moon's colour, the mean of the brightest pixels above
// the shore, and the column they centre on.
func moonOf(pixels []float32, width, height int, at func(int, int) int,
	shore int) ([3]float64, float64) {
	most := 0.0
	for r := 0; r < shore; r++ {
		for x := 0; x < width; x++ {
			i := at(x, r)
			most = math.Max(most, float64(pixels[i]+pixels[i+1]+pixels[i+2]))
		}
	}
	var sum [3]float64
	count, columns := 0.0, 0.0
	for r := 0; r < shore; r++ {
		for x := 0; x < width; x++ {
			i := at(x, r)
			if float64(pixels[i]+pixels[i+1]+pixels[i+2]) < 0.5*most {
				continue
			}
			for c := 0; c < 3; c++ {
				sum[c] += float64(pixels[i+c])
			}
			count++
			columns += float64(x)
		}
	}
	for c := range sum {
		sum[c] /= count
	}
	return sum, columns / count
}

// curled is fractal noise looked up through itself, which swirls; its
// two warping fields slide against each other in time, so it curls.
func curled(u, v, seconds float64, seed uint64) float64 {
	qx := fbm(u+0.09*seconds, v, seed)
	qy := fbm(u+5.2-0.07*seconds, v+1.3, seed+1)
	return fbm(u+1.9*qx, v+0.8*qy, seed+2)
}

// fbm is five octaves of value noise, from 0 to about 1.
func fbm(u, v float64, seed uint64) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for octave := range 5 {
		sum += amp * noise(u, v, seed+uint64(octave)*7919)
		norm += amp
		u, v, amp = u*2.03, v*2.03, amp*0.5
	}
	return sum / norm
}

// noise is smooth value noise on a unit lattice.
func noise(u, v float64, seed uint64) float64 {
	x0, y0 := math.Floor(u), math.Floor(v)
	fx, fy := u-x0, v-y0
	sx, sy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	ix, iy := int64(x0), int64(y0)
	a := lattice(ix, iy, seed)
	b := lattice(ix+1, iy, seed)
	c := lattice(ix, iy+1, seed)
	d := lattice(ix+1, iy+1, seed)
	return a + (b-a)*sx + (c-a)*sy + (a-b-c+d)*sx*sy
}

// lattice is a point's random value, from 0 to 1.
func lattice(x, y int64, seed uint64) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ seed*0x165667B19E3779F9
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	return float64(h>>11) / float64(1<<53)
}

// smooth rises from 0 at a to 1 at b.
func smooth(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}
