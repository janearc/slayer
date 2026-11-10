package oceantex

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Vec is a direction or a place: x east, y north, z up.
type Vec struct{ X, Y, Z float64 }

// unit is the same direction at length one.
func (v Vec) unit() Vec {
	l := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return Vec{v.X / l, v.Y / l, v.Z / l}
}

// dot is the dot product.
func (v Vec) dot(o Vec) float64 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

// cross is the cross product.
func (v Vec) cross(o Vec) Vec {
	return Vec{v.Y*o.Z - v.Z*o.Y, v.Z*o.X - v.X*o.Z, v.X*o.Y - v.Y*o.X}
}

// plus is both vectors together.
func (v Vec) plus(o Vec) Vec { return Vec{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }

// times is the vector k times as long.
func (v Vec) times(k float64) Vec { return Vec{v.X * k, v.Y * k, v.Z * k} }

// Scene is everything about the sea and its light that does not depend on
// where it is seen from.
type Scene struct {
	Sea Sea
	// Salinity in parts per thousand and Celsius set the water's
	// refractive index. The open ocean is near 35.
	Salinity, Celsius float64
	// Chlorophyll is how much is in the water, in milligrams a cubic
	// metre. Zero is the clearest water there could be, a violet blue.
	Chlorophyll float64
	// Sun points from the sea toward the sun. It need not be unit length.
	// SunAt sets it, and the two spectra below, from a place and a time.
	Sun Vec
	// Lamp is the sun's spectrum at the sea, power by wavelength in
	// nanometres, as libtheme's swatch.D65 and swatch.Blackbody give one.
	// Nil is D65.
	Lamp map[int]float64
	// White is the spectrum whose luminance is called 1. Nil is the Lamp
	// itself, so that the sun's irradiance is the unit whatever the sun
	// is doing. SunAt sets it to the sun above the air, so that a low sun
	// is dim.
	White map[int]float64
	// Sky is the radiance arriving from a direction, as a swatch whose
	// luminance is in units of White's irradiance per steradian. Nil is a
	// black sky. This package has no model of the sky; the caller brings
	// one.
	Sky func(from Vec) swatch.Swatch
	// Shore is the ground the sea meets, if it meets any: sea floor
	// that changes the waves over it, and land that stands out of them.
	// Nil is open sea. Only View draws a shore.
	Shore *Shore
}

// Texture is a rendered picture: the radiance each pixel receives, as
// linear amounts of the sRGB lamps, per unit of the white's irradiance on
// a surface facing it. A perfect white matte surface under an overhead
// sun would read 1/pi in each lamp.
type Texture struct {
	Width, Height int
	// Light is row by row from the bottom of the picture up.
	Light [][3]float64
	// Exposure multiplies the light when the texture is used as an image.
	Exposure float64
}

// ColorModel is image/color's 8-bit model; srgb.RGB converts itself.
func (t *Texture) ColorModel() color.Model { return color.RGBAModel }

// Bounds is the picture's rectangle.
func (t *Texture) Bounds() image.Rectangle {
	return image.Rect(0, 0, t.Width, t.Height)
}

// At is one pixel on a screen, through libtheme's sRGB space, which clips
// what a screen cannot show. Row zero is the top.
func (t *Texture) At(x, y int) color.Color {
	l := t.Light[(t.Height-1-y)*t.Width+x]
	e := t.Exposure
	c, _ := srgb.FromSwatch(srgb.FromLight(l[0]*e, l[1]*e, l[2]*e))
	return c
}

// lighting is a scene worked out as far as it can be before any pixel:
// the colours of the sun and the water, and the sea's roughness.
type lighting struct {
	sun              Vec
	sunlight, water  light
	index            float64
	along, across    float64 // all of the sea's slope variance [CM54]
	windSin, windCos float64
	foam, down       float64
	sky              func(Vec) swatch.Swatch
}

// lighting works the scene out as far as it goes before any pixel, and
// refuses a scene whose sun is below the horizon.
func (scene Scene) lighting() (*lighting, error) {
	if !(scene.Sun.Z > 0) {
		return nil, errors.New("oceantex: the sun must be up")
	}
	lamp := scene.Lamp
	if lamp == nil {
		lamp = swatch.D65()
	}
	white := scene.White
	if white == nil {
		white = lamp
	}
	l := &lighting{sun: scene.Sun.unit(), sky: scene.Sky}
	l.sunlight = lightOf(lit(lamp, white, func(int) float64 { return 1 }))
	l.water = lightOf(lit(lamp, white, waterPart(scene.Chlorophyll)))
	// GLUE: one refractive index, at one wavelength, for every wavelength.
	l.index = Index(scene.Salinity, scene.Celsius, indexAt)
	l.along, l.across = Roughness(scene.Sea.Wind)
	l.windSin, l.windCos = math.Sincos(scene.Sea.Toward)
	l.foam = Whitecaps(scene.Sea.Wind)
	l.down = l.sun.Z
	return l, nil
}

// spread is the slope variance along a compass direction, from the
// variances along and across the wind.
func (l *lighting) spread(x, y float64) float64 {
	length := math.Hypot(x, y)
	if length == 0 {
		return l.along
	}
	along := (x*l.windCos + y*l.windSin) / length
	across := (-x*l.windSin + y*l.windCos) / length
	return l.along*along*along + l.across*across*across
}

// seen is the radiance a point of sea sends toward an eye. slopeX and
// slopeY are the slope of the waves the picture resolves there, and
// gotAlong and gotAcross the slope variance those waves account for.
func (l *lighting) seen(
	eye Vec, slopeX, slopeY, gotAlong, gotAcross float64,
) light {
	// GLUE: what [CM54] measured, less what the picture already shows,
	// is the roughness inside a pixel.
	varAlong := math.Max(l.along-gotAlong, slopeFloor)
	varAcross := math.Max(l.across-gotAcross, slopeFloor)
	var seen light

	// The sun's glint. A facet halfway between sun and eye reflects one
	// into the other; the radiance is the sun's irradiance times Fresnel
	// times the chance of that facet, over 4 cos(view) cos^4(tilt) [CM54],
	// times the share of the surface neither hidden from the eye nor
	// shadowed from the sun [S67].
	half := l.sun.plus(eye).unit()
	if half.Z > 0 {
		// GLUE: the facet's slope is measured from the pixel's own
		// resolved slope.
		dx := -half.X/half.Z - slopeX
		dy := -half.Y/half.Z - slopeY
		along := dx*l.windCos + dy*l.windSin
		across := -dx*l.windSin + dy*l.windCos
		chance := density(along, across, varAlong, varAcross)
		// GLUE: the two hidings are multiplied as if independent.
		inView := 1 / ((1 + hidden(eye.Z, l.spread(eye.X, eye.Y))) *
			(1 + hidden(l.sun.Z, l.spread(l.sun.X, l.sun.Y))))
		glint := Fresnel(l.sun.dot(half), l.index) * chance * inView /
			(glintGeometry * math.Max(eye.Z, viewFloor) *
				math.Pow(half.Z, glintTilt))
		seen = seen.plus(l.sunlight.times(glint))
	}

	// The sky, mirrored. GLUE: off the pixel's mean slope only.
	normal := Vec{-slopeX, -slopeY, 1}.unit()
	cosine := math.Max(normal.dot(eye), 0)
	mirror := Fresnel(cosine, l.index)
	if l.sky != nil {
		from := normal.times(2 * cosine).plus(eye.times(-1))
		if from.Z < 0 {
			from.Z = 0 // below the horizon is the horizon
		}
		seen = seen.plus(lightOf(l.sky(from.unit())).times(mirror))
	}

	// The water's own colour: pi times its reflectance, so over pi it is
	// radiance per unit of light falling. GLUE: the value for looking
	// straight down, at every angle.
	seen = seen.plus(l.water.times(l.down / math.Pi))

	// Whitecaps cover a fraction of everything, as a matte white of
	// [K84]'s reflectance. GLUE: an even veil, not patches.
	white := l.sunlight.times(FoamReflectance * l.down / math.Pi)
	return seen.times(1 - l.foam).plus(white.times(l.foam))
}

// rock is the radiance a point of land sends toward an eye, given how
// fast the ground rises east and north there: a matte surface, lit by
// the sun by how squarely it faces it, and by the sky.
//
// GLUE: the ground is one grey, the same in every light, and nothing
// casts a shadow on it. It is there so the shore has a shape, and it
// says nothing true about any rock.
func (l *lighting) rock(east, north float64) light {
	normal := Vec{-east, -north, 1}.unit()
	facing := math.Max(normal.dot(l.sun), 0)
	seen := l.sunlight.times(rockBack * facing / math.Pi)
	if l.sky != nil {
		seen = seen.plus(lightOf(l.sky(normal)).times(rockBack))
	}
	return seen
}

// whitewater is the radiance of surf: fresh foam, matte, lit by the sun
// and the sky [W82].
func (l *lighting) whitewater() light {
	seen := l.sunlight.times(foamFresh * l.down / math.Pi)
	if l.sky != nil {
		seen = seen.plus(lightOf(l.sky(Vec{Z: 1})).times(foamFresh))
	}
	return seen
}
