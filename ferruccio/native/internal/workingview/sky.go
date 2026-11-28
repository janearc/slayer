package workingview

import (
	"image"
	"math"

	"github.com/janearc/slayer/session/placement"
)

// fillSky replaces only unpainted working pixels. Opaque geometry and its
// edge colors are left byte-identical. The directional colors are sampled on
// a bounded grid, then use the same display gamma as simple working solids.
func fillSky(picture *image.NRGBA, scene *placement.DisplayScene) {
	if scene.SkyGradient == nil {
		return
	}
	width, height := picture.Bounds().Dx(), picture.Bounds().Dy()
	if width == 0 || height == 0 {
		return
	}
	const samples = 256
	across, down := min(width, samples), min(height, samples)
	sky := make([][3]uint8, across*down)
	for y := range down {
		v := (float64(y) + .5) / float64(down)
		for x := range across {
			u := (float64(x) + .5) / float64(across)
			light := scene.SampleSky(u, v)
			sky[y*across+x] = [3]uint8{skyChannel(light.R), skyChannel(light.G), skyChannel(light.B)}
		}
	}
	columns := make([]int, width)
	for x := range columns {
		columns[x] = x * across / width
	}
	for y := range height {
		colours := sky[(y*down/height)*across:][:across]
		row := picture.Pix[y*picture.Stride:][:width*4]
		for x, column := range columns {
			offset := x * 4
			if row[offset+3] == 0 {
				c := colours[column]
				row[offset], row[offset+1], row[offset+2], row[offset+3] = c[0], c[1], c[2], 255
			}
		}
	}
}

func skyChannel(linear float64) uint8 {
	return uint8(math.Round(255 * math.Pow(max(0, min(1, linear)), 1.0/2.2)))
}
