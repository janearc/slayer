package swatches

import (
	"reflect"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// a paint gives back no more light than reaches it: every paint, every world's
// palette and pair, the giants' stripes, the airs and the aurorae are each a
// share of every lamp's light from zero to one. the lamps and the lights bodies
// give off are light, and may be more.
func TestEveryPaintIsAShareOfTheLight(t *testing.T) {
	paints := map[string]swatch.Swatch{"chalk": Chalk, "clay": Clay,
		"paint": Paint, "slate": Slate, "sea": Sea, "moss": Moss,
		"cream": Cream, "rust": Rust, "ochre": Ochre, "storm": Storm,
		"ice": Ice, "jupiter's oval": JupiterOval,
		"saturn's cream": SaturnCream, "saturn's belt": SaturnBelt,
		"saturn's ochre": SaturnOchre, "saturn's pale": SaturnPale,
		"uranus's cyan": UranusCyan, "uranus's deep": UranusDeep,
		"uranus's pale": UranusPale, "uranus's ring": UranusRing,
		"neptune's blue": NeptuneBlue, "neptune's deep": NeptuneDeep,
		"neptune's pale": NeptunePale, "neptune's dark": NeptuneDark,
		"neptune's spot": NeptuneSpot, "neptune's cloud": NeptuneCloud,
		"earth's air": EarthAir, "venus's air": VenusAir,
		"mars's air": MarsAir, "titan's air": TitanAir,
		"aurora purple": AuroraPurple, "aurora green": AuroraGreen,
		"aurora red": AuroraRed, "aurora violet": AuroraViolet,
		"aurora blue": AuroraBlue, "aurora mauve": AuroraMauve}
	for name, pair := range map[string]Pair{"venus": Venus, "titan": Titan,
		"europa": Europa, "ganymede": Ganymede} {
		paints[name+"'s pale"], paints[name+"'s dark"] = pair.Pale, pair.Dark
	}
	for i, s := range Terran {
		paints["terran "+string(rune('a'+i))] = s
	}
	for name, palette := range palettes() {
		v := reflect.ValueOf(palette)
		for i := range v.NumField() {
			paints[name+"'s "+v.Type().Field(i).Name] =
				v.Field(i).Interface().(swatch.Swatch)
		}
	}
	for name, s := range paints {
		l := Light(s)
		for _, share := range []float64{l.R, l.G, l.B} {
			if share < -noise || share > 1+noise {
				t.Errorf("%s gives back %v", name, l)
				break
			}
		}
	}
}

// palettes is every world's palette, by name.
func palettes() map[string]Palette {
	return map[string]Palette{"mercury": Mercury, "moon": Moon,
		"earth": Earth, "mars": Mars, "io": Io, "pluto": Pluto,
		"charon": Charon}
}
