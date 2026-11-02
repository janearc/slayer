// Package swatches provides the named paint box used by the original scenes.
// Each paint is a libtheme swatch describing daylight reflectance, with each
// channel at most one. Lamp swatches describe emitted light and may exceed one.
// Light converts either to the linear RGB values used by the renderer.
package swatches

import (
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// The paints.
var (
	// Chalk is the floor: nearly white, a little warm.
	Chalk = srgb.FromLight(0.80, 0.80, 0.78)
	// Clay is the first ball, a flowerpot that comes out Maui.
	Clay = srgb.FromLight(0.70, 0.30, 0.20)
	// Paint is white paint, which gives back nine tenths of every light.
	Paint = srgb.FromLight(0.90, 0.90, 0.90)
	// Slate is chalk's dark partner on a checkered floor, dark enough that
	// the squares read even in two colours a cell.
	Slate = srgb.FromLight(0.22, 0.22, 0.24)
	// Sea is a world's water, a deep blue that stays blue in shadow.
	Sea = srgb.FromLight(0.10, 0.28, 0.55)
	// Moss is a world's land, a green that reads against the sea in two
	// colours a cell.
	Moss = srgb.FromLight(0.34, 0.52, 0.20)
)

// The lamps.
var (
	// Daylight is libtheme's white, d65.
	Daylight = swatch.White
	// Red is the screen's red lamp at full: as hard a red as light gets
	// here, with no green or blue in it at all.
	Red = srgb.FromLight(1, 0, 0)
	// Green is the screen's green lamp at full, with no red or blue.
	Green = srgb.FromLight(0, 1, 0)
	// Blue is the screen's blue lamp at full, with no red or green.
	Blue = srgb.FromLight(0, 0, 1)
	// Horizon is a pale sky low down, where the air is thickest.
	Horizon = srgb.FromLight(0.85, 0.88, 0.92)
	// Zenith is a clear sky straight up.
	Zenith = srgb.FromLight(0.30, 0.45, 0.80)
	// Tungsten is a studio lamp's light: a black body at 3200 kelvin, as
	// libtheme works it out, so it is derived and never typed.
	Tungsten = swatch.Planckian(3200)
)

// The gas giant's paints: the bands of a jovian world, and its storm.
var (
	// Cream is the pale zone of a gas giant, ammonia cloud in sunlight.
	Cream = srgb.FromLight(0.88, 0.80, 0.62)
	// Rust is a dark belt, the deeper warmer gas showing through.
	Rust = srgb.FromLight(0.52, 0.30, 0.17)
	// Ochre is a belt between the two, dusty and brown.
	Ochre = srgb.FromLight(0.66, 0.46, 0.26)
	// Storm is the heart of a great red spot.
	Storm = srgb.FromLight(0.62, 0.18, 0.10)
	// Ice is a ring's ice, nearly white with the faintest warmth.
	Ice = srgb.FromLight(0.86, 0.84, 0.80)
)

// Terran is a blue and green world's thirteen classes, in biome order:
// deep sea, shallow sea, ice, beach, desert, grass, forest, jungle,
// tundra, rock, snow, lava, ash. The first eleven are miami's terran
// look, as hex; lava is its molten world's sea, and ash its crust.
var Terran = [13]swatch.Swatch{
	hex(0x0a, 0x24, 0x52), hex(0x1d, 0x5c, 0x9a), hex(0xd6, 0xe6, 0xf0),
	hex(0xd8, 0xc4, 0x8c), hex(0xd0, 0xa4, 0x5e), hex(0x7d, 0xa7, 0x44),
	hex(0x3b, 0x78, 0x38), hex(0x27, 0x66, 0x2c), hex(0x8c, 0x9a, 0x7c),
	hex(0x7a, 0x6e, 0x62), hex(0xf2, 0xf5, 0xf8), hex(0xff, 0x5a, 0x0a),
	hex(0x3a, 0x30, 0x2a),
}

// hex is a swatch from the three bytes of a hex colour.
func hex(r, g, b uint8) swatch.Swatch { return srgb.RGB8(r, g, b).Swatch() }

// Light is a swatch as slayer's light: amounts of the three srgb lamps.
func Light(s swatch.Swatch) maths.Light {
	r, g, b := srgb.Light(s)
	return maths.RGB(r, g, b)
}
