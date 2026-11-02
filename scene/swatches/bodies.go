package swatches

import (
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Palette is a world's paint for each biome, by name. A body's ground
// is painted from one; scene/world's Look takes it as lights.
type Palette struct {
	Deep, Sea, Ice, Beach, Desert, Grass, Forest, Jungle swatch.Swatch
	Tundra, Rock, Snow, Lava, Ash                        swatch.Swatch
}

// Pair is two paints for a ground with no biomes to speak of, where
// the two only keep the tiles apart. in every pair here Pale is the darker
// of the two as typed: the names are the template's, and which is drawn
// where is left as typed.
type Pair struct {
	Pale, Dark swatch.Swatch
}

// The grounds. Each number is the linear light scene/bodies/template typed
// before these had names; the numbers were chosen to look right and
// have not been checked against a source.
var (
	// Mercury is bare grey rock, a hair warm.
	Mercury = Palette{
		Deep:   srgb.FromLight(0.18, 0.17, 0.16),
		Sea:    srgb.FromLight(0.24, 0.23, 0.22),
		Ice:    srgb.FromLight(0.5, 0.5, 0.5),
		Beach:  srgb.FromLight(0.36, 0.35, 0.33),
		Desert: srgb.FromLight(0.42, 0.40, 0.38),
		Grass:  srgb.FromLight(0.38, 0.37, 0.35),
		Forest: srgb.FromLight(0.33, 0.32, 0.31),
		Jungle: srgb.FromLight(0.30, 0.29, 0.28),
		Tundra: srgb.FromLight(0.45, 0.44, 0.43),
		Rock:   srgb.FromLight(0.28, 0.27, 0.26),
		Snow:   srgb.FromLight(0.55, 0.55, 0.55),
		Lava:   srgb.FromLight(0.6, 0.3, 0.1),
		Ash:    srgb.FromLight(0.2, 0.19, 0.18),
	}
	// Moon is the moon's greys, the maria dark.
	Moon = Palette{
		Deep:   srgb.FromLight(0.14, 0.14, 0.14),
		Sea:    srgb.FromLight(0.2, 0.2, 0.2),
		Ice:    srgb.FromLight(0.5, 0.5, 0.5),
		Beach:  srgb.FromLight(0.4, 0.4, 0.4),
		Desert: srgb.FromLight(0.45, 0.45, 0.44),
		Grass:  srgb.FromLight(0.42, 0.42, 0.42),
		Forest: srgb.FromLight(0.4, 0.4, 0.4),
		Jungle: srgb.FromLight(0.38, 0.38, 0.38),
		Tundra: srgb.FromLight(0.47, 0.47, 0.47),
		Rock:   srgb.FromLight(0.33, 0.33, 0.33),
		Snow:   srgb.FromLight(0.58, 0.58, 0.58),
		Lava:   srgb.FromLight(0.6, 0.3, 0.1),
		Ash:    srgb.FromLight(0.2, 0.2, 0.2),
	}
	// Earth is the terran world's paints as the earth template typed
	// them, in linear light; Terran above is miami's hex set, which
	// these were read from and rounded.
	Earth = Palette{
		Deep:   srgb.FromLight(0.0028, 0.018, 0.085),
		Sea:    srgb.FromLight(0.013, 0.11, 0.32),
		Ice:    srgb.FromLight(0.67, 0.79, 0.87),
		Beach:  srgb.FromLight(0.69, 0.55, 0.26),
		Desert: srgb.FromLight(0.64, 0.37, 0.11),
		Grass:  srgb.FromLight(0.21, 0.39, 0.058),
		Forest: srgb.FromLight(0.045, 0.19, 0.040),
		Jungle: srgb.FromLight(0.020, 0.13, 0.025),
		Tundra: srgb.FromLight(0.26, 0.32, 0.20),
		Rock:   srgb.FromLight(0.19, 0.16, 0.12),
		Snow:   srgb.FromLight(0.89, 0.92, 0.94),
		Lava:   srgb.FromLight(1.0, 0.1, 0.0),
		Ash:    srgb.FromLight(0.045, 0.030, 0.023),
	}
	// Mars is rust and dust with white caps.
	Mars = Palette{
		Deep:   srgb.FromLight(0.25, 0.12, 0.06),
		Sea:    srgb.FromLight(0.35, 0.17, 0.08),
		Ice:    srgb.FromLight(0.85, 0.85, 0.85),
		Beach:  srgb.FromLight(0.55, 0.3, 0.15),
		Desert: srgb.FromLight(0.6, 0.32, 0.14),
		Grass:  srgb.FromLight(0.5, 0.26, 0.12),
		Forest: srgb.FromLight(0.45, 0.24, 0.11),
		Jungle: srgb.FromLight(0.42, 0.22, 0.1),
		Tundra: srgb.FromLight(0.52, 0.3, 0.18),
		Rock:   srgb.FromLight(0.32, 0.2, 0.12),
		Snow:   srgb.FromLight(0.9, 0.9, 0.9),
		Lava:   srgb.FromLight(0.9, 0.3, 0.05),
		Ash:    srgb.FromLight(0.2, 0.12, 0.08),
	}
	// Io is sulphur: yellows and a white frost.
	Io = Palette{
		Deep:   srgb.FromLight(0.6, 0.45, 0.1),
		Sea:    srgb.FromLight(0.7, 0.55, 0.15),
		Ice:    srgb.FromLight(0.9, 0.9, 0.8),
		Beach:  srgb.FromLight(0.85, 0.7, 0.2),
		Desert: srgb.FromLight(0.8, 0.65, 0.2),
		Grass:  srgb.FromLight(0.75, 0.6, 0.15),
		Forest: srgb.FromLight(0.6, 0.5, 0.15),
		Jungle: srgb.FromLight(0.55, 0.45, 0.12),
		Tundra: srgb.FromLight(0.85, 0.8, 0.6),
		Rock:   srgb.FromLight(0.4, 0.3, 0.1),
		Snow:   srgb.FromLight(0.95, 0.95, 0.9),
		Lava:   srgb.FromLight(0.9, 0.3, 0.05),
		Ash:    srgb.FromLight(0.25, 0.18, 0.08),
	}
	// Pluto is red tholin and nitrogen ice.
	Pluto = Palette{
		Deep:   srgb.FromLight(0.3, 0.18, 0.12),
		Sea:    srgb.FromLight(0.4, 0.25, 0.16),
		Ice:    srgb.FromLight(0.9, 0.88, 0.85),
		Beach:  srgb.FromLight(0.6, 0.45, 0.35),
		Desert: srgb.FromLight(0.55, 0.35, 0.25),
		Grass:  srgb.FromLight(0.5, 0.32, 0.22),
		Forest: srgb.FromLight(0.45, 0.28, 0.2),
		Jungle: srgb.FromLight(0.42, 0.26, 0.18),
		Tundra: srgb.FromLight(0.7, 0.62, 0.55),
		Rock:   srgb.FromLight(0.35, 0.22, 0.15),
		Snow:   srgb.FromLight(0.92, 0.9, 0.88),
		Lava:   srgb.FromLight(0.9, 0.3, 0.05),
		Ash:    srgb.FromLight(0.25, 0.15, 0.1),
	}
	// Charon is grey, its snow slot the dark red of its cap.
	Charon = Palette{
		Deep:   srgb.FromLight(0.3, 0.3, 0.3),
		Sea:    srgb.FromLight(0.38, 0.38, 0.38),
		Ice:    srgb.FromLight(0.7, 0.7, 0.7),
		Beach:  srgb.FromLight(0.5, 0.5, 0.5),
		Desert: srgb.FromLight(0.52, 0.5, 0.48),
		Grass:  srgb.FromLight(0.48, 0.47, 0.46),
		Forest: srgb.FromLight(0.45, 0.44, 0.43),
		Jungle: srgb.FromLight(0.42, 0.41, 0.4),
		Tundra: srgb.FromLight(0.55, 0.52, 0.5),
		Rock:   srgb.FromLight(0.4, 0.38, 0.36),
		Snow:   srgb.FromLight(0.45, 0.25, 0.2),
		Lava:   srgb.FromLight(0.9, 0.3, 0.05),
		Ash:    srgb.FromLight(0.25, 0.25, 0.25),
	}
	// Venus's ground, under its clouds, in two browns.
	Venus = Pair{srgb.FromLight(0.45, 0.35, 0.25),
		srgb.FromLight(0.5, 0.4, 0.3)}
	// Titan's ground, in two oranges.
	Titan = Pair{srgb.FromLight(0.75, 0.55, 0.2),
		srgb.FromLight(0.8, 0.6, 0.25)}
	// Europa's ice, in two near-whites.
	Europa = Pair{srgb.FromLight(0.8, 0.78, 0.7),
		srgb.FromLight(0.86, 0.84, 0.78)}
	// Ganymede's and callisto's dark ground, in two greys.
	Ganymede = Pair{srgb.FromLight(0.28, 0.25, 0.22),
		srgb.FromLight(0.36, 0.33, 0.3)}
)

// The giants' stripes, spots and rings. Jupiter's are Cream, Rust,
// Ochre and Storm above.
var (
	// JupiterOval is the white ovals' cloud.
	JupiterOval = srgb.FromLight(0.9, 0.9, 0.9)
	// Saturn's stripes, most of them a little paler than jupiter's.
	SaturnCream = srgb.FromLight(0.85, 0.78, 0.62)
	SaturnBelt  = srgb.FromLight(0.72, 0.62, 0.45)
	SaturnOchre = srgb.FromLight(0.78, 0.68, 0.5)
	SaturnPale  = srgb.FromLight(0.88, 0.82, 0.66)
	// Uranus's cyans, nearly featureless.
	UranusCyan = srgb.FromLight(0.62, 0.82, 0.86)
	UranusDeep = srgb.FromLight(0.58, 0.78, 0.83)
	UranusPale = srgb.FromLight(0.64, 0.84, 0.88)
	// UranusRing is its thin rings, a mid grey here. the real rings
	// are far darker; this was chosen to show, not checked.
	UranusRing = srgb.FromLight(0.3, 0.3, 0.3)
	// Neptune's blues, its dark spot and its white streaks.
	NeptuneBlue  = srgb.FromLight(0.18, 0.32, 0.75)
	NeptuneDeep  = srgb.FromLight(0.14, 0.26, 0.65)
	NeptunePale  = srgb.FromLight(0.2, 0.35, 0.8)
	NeptuneDark  = srgb.FromLight(0.12, 0.24, 0.62)
	NeptuneSpot  = srgb.FromLight(0.06, 0.12, 0.4)
	NeptuneCloud = srgb.FromLight(0.9, 0.92, 0.95)
)

// The airs: the colour of what each scatters, until an air scatters by its
// own physics.
var (
	EarthAir = srgb.FromLight(0.5, 0.7, 1.0)
	VenusAir = srgb.FromLight(0.92, 0.85, 0.65)
	MarsAir  = srgb.FromLight(0.9, 0.7, 0.55)
	TitanAir = srgb.FromLight(0.9, 0.7, 0.3)
)

// The aurorae's colours, for the earth's and a giant's; which is drawn at
// what height is the aurora's texture's to say.
var (
	AuroraPurple = srgb.FromLight(0.6, 0.3, 1)
	AuroraGreen  = srgb.FromLight(0.3, 1, 0.35)
	AuroraRed    = srgb.FromLight(1, 0.25, 0.2)
	AuroraViolet = srgb.FromLight(0.5, 0.4, 1)
	AuroraBlue   = srgb.FromLight(0.35, 0.55, 1)
	AuroraMauve  = srgb.FromLight(0.7, 0.4, 1)
)

// The lights the bodies give off, at the strength the template types.
var (
	// City is a city's light at night against the sun's, a few
	// thousandths, warm.
	City = srgb.FromLight(0.002, 0.0015, 0.0008)
	// IoPlume is a plume's dust and gas.
	IoPlume = srgb.FromLight(0.82, 0.78, 0.7)
	// JetBlue is a quasar's jets' glow.
	JetBlue = srgb.FromLight(0.5, 0.6, 1)
	// PulsarWhite and PulsarBeam are a neutron star and its beams,
	// blue-white and brighter than white.
	PulsarWhite = srgb.FromLight(2, 2.2, 3)
	PulsarBeam  = srgb.FromLight(0.9, 0.95, 1.4)
	// MagnetarBlue is a magnetar's dim blue; FlareWhite its fireball.
	MagnetarBlue = srgb.FromLight(0.4, 0.5, 0.9)
	FlareWhite   = srgb.FromLight(1.5, 1.2, 1)
	// SunWhite is the sun's surface, white a little warm, three times
	// white.
	SunWhite = srgb.FromLight(3, 2.9, 2.6)
	// TailGlow is a hot jupiter's escaping air glowing faintly.
	TailGlow = srgb.FromLight(0.04, 0.03, 0.015)
	// A comet: the coma's dust, the green of carbon it gives off, the
	// dust tail's cream, the ion tail's dark blue and its glow.
	ComaDust        = srgb.FromLight(0.95, 0.95, 0.9)
	ComaDustCool    = srgb.FromLight(0.9, 0.95, 0.9)
	ComaGreen       = srgb.FromLight(1.2, 1.8, 1.3)
	ComaGreenBright = srgb.FromLight(1.2, 2, 1.3)
	DustTail        = srgb.FromLight(1, 0.95, 0.82)
	DustTailPale    = srgb.FromLight(1, 0.95, 0.85)
	IonTail         = srgb.FromLight(0.1, 0.1, 0.2)
	IonGlow         = srgb.FromLight(0.15, 0.3, 0.6)
)
