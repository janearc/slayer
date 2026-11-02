package moonlight

import (
	"fmt"
	"math"
	"os"

	"github.com/janearc/slayer/maths"
	"golang.org/x/image/tiff"
)

// A Face is the full moon's face as a texture: a map of its brightness from
// place to place, read from an equirectangular mosaic, times the glow its
// geometric albedo gives it as a whole.
//
// the map gives only the pattern: its luminance, scaled so its mean over the
// whole moon is one. the colour and the level are the glow's, from the physics,
// so the maria are darker and the highlands lighter and the moon as a whole is
// as bright as its albedo says.
//
// its near side faces the earth, longitude zero toward the eye, and its north
// is toward the sky's up: picked, until an ephemeris gives the moon's libration
// and the angle of its axis.
type Face struct {
	width, height    int
	bright           []float32
	glow             maths.Light
	toward, up, east maths.Vec
}

// NewFace is the face from a mosaic file, its longitude zero in the middle and
// north at the top, for a moon a way from the eye, y up, that glows as given.
func NewFace(path string, way maths.Vec, glow maths.Light) (*Face, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	picture, err := tiff.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	box := picture.Bounds()
	face := &Face{width: box.Dx(), height: box.Dy(), glow: glow,
		bright: make([]float32, box.Dx()*box.Dy())}
	weighed, sum := 0.0, 0.0
	for y := range face.height {
		weight := math.Cos((0.5 - (float64(y)+0.5)/float64(face.height)) *
			math.Pi)
		for x := range face.width {
			red, green, blue, _ := picture.At(box.Min.X+x, box.Min.Y+y).RGBA()
			lum := 0.2126*linear(red) + 0.7152*linear(green) +
				0.0722*linear(blue)
			face.bright[y*face.width+x] = float32(lum)
			sum += lum * weight
			weighed += weight
		}
	}
	if !(sum > 0) {
		return nil, fmt.Errorf("%s: a mosaic with no light in it", path)
	}
	mean := float32(sum / weighed)
	for i := range face.bright {
		face.bright[i] /= mean
	}
	face.orient(way)
	return face, nil
}

// orient sets which way the face's middle looks, back toward the eye,
// and its north and its east as the eye sees them.
func (face *Face) orient(way maths.Vec) {
	way = way.Unit()
	face.toward = way.Neg()
	up := maths.V(0, 1, 0)
	up = up.Sub(way.Scale(up.Dot(way))).Unit()
	face.up = up
	face.east = way.Cross(up).Unit()
}

// linear is a sixteen-bit srgb sample as linear light, zero to one.
func linear(sample uint32) float64 {
	value := float64(sample) / 65535
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

// At is the face's glow at a point of the moon's own space: the
// mosaic's brightness at the point's latitude and longitude, times the
// glow.
func (face *Face) At(p maths.Vec) maths.Light {
	d := p.Unit()
	latitude := math.Asin(math.Max(-1, math.Min(1, d.Dot(face.up))))
	longitude := math.Atan2(d.Dot(face.east), d.Dot(face.toward))
	x := int((longitude/(2*math.Pi) + 0.5) * float64(face.width))
	y := int((0.5 - latitude/math.Pi) * float64(face.height))
	x = max(0, min(face.width-1, x))
	y = max(0, min(face.height-1, y))
	return face.glow.Scale(float64(face.bright[y*face.width+x]))
}
