// Package diffusion approximates subsurface light transport with RGB
// irradiance filters in image space and on indexed meshes.
package diffusion

import (
	"errors"
	"math"

	"github.com/janearc/slayer/render/frame"
)

// Profile gives the Gaussian standard deviations in scene metres for each
// channel and the fraction of the diffuse response that travels below the surface.
type Profile struct {
	Radius [3]float64
	Share  float64
}

// Apply replaces a share of local diffuse illumination with diffused illumination.
// Beauty, unit-albedo irradiance and unlit albedo are aligned linear frames.
// Albedo coverage is zero off the selected tissue. Span is horizontal view width at
// unit camera depth; distance and profile radius use the same metric units.
func Apply(beauty, irradiance, albedo *frame.Frame, span float64, profile Profile) error {
	valid := func(n float64) bool { return n > 0 && !math.IsInf(n, 0) }
	if !valid(span) || !(profile.Share >= 0 && profile.Share <= 1) {
		return errors.New("diffusion: invalid view width or transport share")
	}
	for _, r := range profile.Radius {
		if !valid(r) {
			return errors.New("diffusion: radius must be positive and finite")
		}
	}
	for _, f := range []*frame.Frame{beauty, irradiance, albedo} {
		if f == nil || f.W != beauty.W || f.H != beauty.H || len(f.Pix) != f.W*f.H {
			return errors.New("diffusion: frames must have equal dimensions")
		}
	}
	for i, p := range albedo.Pix {
		if p.Cover > 0 && !valid(float64(irradiance.Pix[i].Depth)) {
			return errors.New("diffusion: visible tissue needs a positive finite depth")
		}
	}

	filtered := irradiance
	for axis := 0; axis < 2; axis++ {
		next := frame.New(beauty.W, beauty.H)
		for y := 0; y < beauty.H; y++ {
			for x := 0; x < beauty.W; x++ {
				i := y*beauty.W + x
				if albedo.Pix[i].Cover == 0 {
					continue
				}
				depth := float64(irradiance.Pix[i].Depth)
				pixel := depth * span / float64(beauty.W)
				dst := &next.Pix[i]
				var sum, total [3]float64
				radius := int(min(float64(max(beauty.W, beauty.H)), math.Ceil(3*max(profile.Radius[0], profile.Radius[1], profile.Radius[2])/pixel)))
				for j := -radius; j <= radius; j++ {
					xx, yy := x, y
					if axis == 0 {
						xx += j
					} else {
						yy += j
					}
					if xx < 0 || yy < 0 || xx >= beauty.W || yy >= beauty.H {
						continue
					}
					k := yy*beauty.W + xx
					if albedo.Pix[k].Cover == 0 {
						continue
					}
					lateral := float64(j) * pixel
					dz := depth - float64(irradiance.Pix[k].Depth)
					d2 := lateral*lateral + dz*dz
					source := filtered.Pix[k]
					light := [3]float64{float64(source.R), float64(source.G), float64(source.B)}
					for c, r := range profile.Radius {
						w := math.Exp(-.5 * d2 / (r * r))
						sum[c] += w * light[c]
						total[c] += w
					}
				}
				dst.R = float32(sum[0] / total[0])
				dst.G = float32(sum[1] / total[1])
				dst.B = float32(sum[2] / total[2])
			}
		}
		filtered = next
	}
	for i, d := range filtered.Pix {
		a, l := albedo.Pix[i], irradiance.Pix[i]
		s := float32(profile.Share)
		p := &beauty.Pix[i]
		p.R += s * a.R * (d.R - l.R)
		p.G += s * a.G * (d.G - l.G)
		p.B += s * a.B * (d.B - l.B)
	}
	return nil
}
