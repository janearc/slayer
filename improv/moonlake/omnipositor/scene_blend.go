package omnipositor

import (
	"fmt"
	"math"
	"runtime"
	"sync"

	"github.com/janearc/slayer/render/frame"
)

// CompositeTransport applies primary-ray transport, L = S + T * plate,
// before display development. Reduced transport is reconstructed with
// spatial weights and a relative-depth weight. The 2% depth falloff is an
// authored reconstruction choice, not a property of fog. Thin silhouettes
// missed by every coarse ray still need finer transport sampling.
func CompositeTransport(plate, scatter, transmission *frame.Frame) (*frame.Frame, error) {
	for _, input := range []*frame.Frame{plate, scatter, transmission} {
		if err := validSceneFrame(input); err != nil {
			return nil, err
		}
	}
	if scatter.W != transmission.W || scatter.H != transmission.H || plate.W%scatter.W != 0 || plate.H%scatter.H != 0 || plate.W/scatter.W != plate.H/scatter.H {
		return nil, fmt.Errorf("composite: transport dimensions must match and divide the plate equally")
	}
	for _, pixel := range transmission.Pix {
		if pixel.R < 0 || pixel.R > 1 || pixel.G < 0 || pixel.G > 1 || pixel.B < 0 || pixel.B > 1 {
			return nil, fmt.Errorf("composite: transmission outside [0,1]")
		}
	}
	out := frame.New(plate.W, plate.H)
	scale := plate.W / scatter.W
	var workers sync.WaitGroup
	count := min(runtime.GOMAXPROCS(0), plate.H)
	for worker := range count {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for y := worker; y < plate.H; y += count {
				for x := range plate.W {
					p := plate.At(x, y)
					var s, t frame.Pixel
					if scale == 1 {
						s, t = scatter.At(x, y), transmission.At(x, y)
					} else {
						s, t = reconstruct(scatter, transmission, x, y, scale, p.Depth)
					}
					p.R, p.G, p.B = s.R+t.R*p.R, s.G+t.G*p.G, s.B+t.B*p.B
					out.Set(x, y, p)
				}
			}
		}()
	}
	workers.Wait()
	return out, nil
}

// ResolveTransport averages a composited sample grid only after transport
// has been applied to each scenery sample. Averaging the plate's depths
// first would close the gaps between needles and other thin silhouettes.
func ResolveTransport(samples *frame.Frame, side int) (*frame.Frame, error) {
	if err := validSceneFrame(samples); err != nil {
		return nil, err
	}
	if side < 1 || side > 4 || samples.W%side != 0 || samples.H%side != 0 {
		return nil, fmt.Errorf("composite: sample grid must divide by rays per axis")
	}
	if side == 1 {
		return samples, nil
	}
	out := frame.New(samples.W/side, samples.H/side)
	share := 1 / float32(side*side)
	for y := range out.H {
		for x := range out.W {
			sum := out.At(x, y)
			for yy := range side {
				for xx := range side {
					p := samples.At(x*side+xx, y*side+yy)
					sum.R += p.R * share
					sum.G += p.G * share
					sum.B += p.B * share
					sum.Cover += p.Cover * share
					sum.Depth = min(sum.Depth, p.Depth)
				}
			}
			out.Set(x, y, sum)
		}
	}
	return out, nil
}

// validSceneFrame bounds allocation and refuses nonfinite radiance. A sky
// miss has positive infinite depth; negative or NaN depths are invalid.
func validSceneFrame(input *frame.Frame) error {
	if input == nil || input.W <= 0 || input.H <= 0 || input.W > (1<<24)/input.H || len(input.Pix) != input.W*input.H {
		return fmt.Errorf("composite: invalid frame dimensions")
	}
	for _, p := range input.Pix {
		if !finite(float64(p.R)) || !finite(float64(p.G)) || !finite(float64(p.B)) || math.IsNaN(float64(p.Depth)) || p.Depth < 0 {
			return fmt.Errorf("composite: invalid light or depth")
		}
	}
	return nil
}

// depthGap compares surfaces by relative distance; sky samples only match sky.
func depthGap(a, b float32) float64 {
	if math.IsInf(float64(a), 1) || math.IsInf(float64(b), 1) {
		if a == b {
			return 0
		}
		return math.Inf(1)
	}
	return math.Abs(float64(a)-float64(b)) / math.Max(1, math.Max(float64(a), float64(b)))
}

// reconstruct samples a small neighbourhood without pulling distant fog
// over a nearby surface. If no matching depth exists it chooses the closest
// depth, rather than dividing by zero or making up a transparent pixel.
func reconstruct(scatter, transmission *frame.Frame, x, y, scale int, depth float32) (frame.Pixel, frame.Pixel) {
	u, v := (float64(x)+0.5)/float64(scale)-0.5, (float64(y)+0.5)/float64(scale)-0.5
	cx, cy := int(math.Floor(u)), int(math.Floor(v))
	var sum [6]float64
	weight, best := 0.0, math.Inf(1)
	nearX, nearY := min(max(int(math.Round(u)), 0), scatter.W-1), min(max(int(math.Round(v)), 0), scatter.H-1)
	for yy := max(0, cy-1); yy <= min(scatter.H-1, cy+2); yy++ {
		for xx := max(0, cx-1); xx <= min(scatter.W-1, cx+2); xx++ {
			s, t := scatter.At(xx, yy), transmission.At(xx, yy)
			gap := depthGap(depth, s.Depth)
			dx, dy := float64(xx)-u, float64(yy)-v
			distance := dx*dx + dy*dy
			if gap < best {
				best, nearX, nearY = gap, xx, yy
			}
			w := math.Exp(-distance*2 - gap*gap/(2*0.02*0.02))
			weight += w
			for c, value := range [6]float32{s.R, s.G, s.B, t.R, t.G, t.B} {
				sum[c] += w * float64(value)
			}
		}
	}
	if weight < 1e-12 {
		return scatter.At(nearX, nearY), transmission.At(nearX, nearY)
	}
	return frame.Pixel{R: float32(sum[0] / weight), G: float32(sum[1] / weight), B: float32(sum[2] / weight)}, frame.Pixel{R: float32(sum[3] / weight), G: float32(sum[4] / weight), B: float32(sum[5] / weight)}
}
