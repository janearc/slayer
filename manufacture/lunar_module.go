package manufacture

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The Apollo lunar module: an eight-sided descent stage in its gold
// blanket, standing on four legs, with the ascent stage on top. Its
// sizes are data, lunar-module.yaml, with where each was read.

//go:embed lunar-module.yaml
var lunarModuleRaw []byte

// lunarSizes is the lunar module's sizes, in metres.
type lunarSizes struct {
	Overall float64
	Descent struct{ Across, Height float64 }
	Ascent  struct{ Width, Depth, Height float64 }
	Gear    struct {
		Span, Pad, Strut float64
		PadThick         float64 `yaml:"pad-thick"`
	}
	Blanket float64
	Read    string
	// dressed is what it wears, made once when the file is read and held
	// to be sound, so that building it afterwards has nothing to refuse.
	dressed looks
}

// The lunar module's own looks.
const (
	lunarBlanket slot = iota
	lunarCabin
	lunarGear

	lunarSides = 8    // defined: the descent stage is an eight-sided prism
	lunarLegs  = 4    // defined: it stands on four legs
	lunarSheen = 0.35 // picked: a foil blanket is a wrinkled mirror; how wide its highlight spreads
)

var lunarRead, lunarError = readLunar(lunarModuleRaw)

// readLunar reads the lunar module's sizes and checks them, so that
// making it afterwards cannot fail: every size above nought, the two
// stages no taller together than the whole, the pads further apart than
// the descent stage is wide, and somewhere the sizes were read.
func readLunar(raw []byte) (lunarSizes, error) {
	var read lunarSizes
	if err := yamlfile.Read("lunar-module.yaml", raw, &read); err != nil {
		return lunarSizes{}, err
	}
	for name, size := range map[string]float64{"overall": read.Overall,
		"descent across": read.Descent.Across, "descent height": read.Descent.Height,
		"ascent width": read.Ascent.Width, "ascent depth": read.Ascent.Depth,
		"ascent height": read.Ascent.Height, "gear span": read.Gear.Span,
		"pad": read.Gear.Pad, "pad thickness": read.Gear.PadThick,
		"strut": read.Gear.Strut, "blanket": read.Blanket} {
		if !sound(size) {
			return lunarSizes{}, fmt.Errorf("manufacture: the lunar module's "+
				"%s is %v", name, size)
		}
	}
	if read.stands() < read.Gear.PadThick {
		return lunarSizes{}, fmt.Errorf("manufacture: the lunar module's stages "+
			"are %v tall together and it is %v overall: no room under them",
			read.Descent.Height+read.Ascent.Height, read.Overall)
	}
	if read.Gear.Span-read.Gear.Pad <= read.Descent.Across || read.Read == "" {
		return lunarSizes{}, fmt.Errorf("manufacture: the lunar module's pads, "+
			"%v apart, are inside its descent stage, %v across, or its sizes "+
			"were read nowhere", read.Gear.Span, read.Descent.Across)
	}
	dressed, err := read.looks()
	if err != nil {
		return lunarSizes{}, err
	}
	read.dressed = dressed
	return read, nil
}

// stands is how high the descent stage's underside is off the ground:
// what is left of the whole height under the two stages.
func (size lunarSizes) stands() float64 {
	return size.Overall - size.Descent.Height - size.Ascent.Height
}

// lunarModule is the maker of the lunar module.
type lunarModule struct{}

// lunarRows supplies the lunar module, by name. A sizes file that
// cannot be read stops the build, as the other embedded data files do.
func lunarRows() map[string]maker {
	if lunarError != nil {
		panic(lunarError)
	}
	return map[string]maker{"apollo-lunar-module": lunarModule{}}
}

// lunarLooks is what the lunar module wears. The descent stage's
// blanket is Kapton over aluminium, which is why such a blanket is
// gold: thing's own film over thing's own metal. The cabin and the gear
// are bare aluminium, the cabin frosted. Every finish is picked.
func (size lunarSizes) looks() (looks, error) {
	cabin, err := surfaceOf("the lunar module's ascent stage", "aluminium", "frosted")
	if err != nil {
		return nil, err
	}
	gear, err := surfaceOf("the lunar module's gear", "aluminium", "brushed")
	if err != nil {
		return nil, err
	}
	cabin.Name, gear.Name = "ascent stage", "landing gear"
	blanket := &thing.Material{Name: "descent stage", Gloss: spaceOne,
		Satin: lunarSheen, Metal: thing.Aluminium(),
		Film: thing.Kapton(size.Blanket)}
	return looks{lunarBlanket: blanket, lunarCabin: cabin, lunarGear: gear}, nil
}

// extent is the lunar module's box: as wide as its pads' ends are
// apart, and as tall as it is overall.
func (size lunarSizes) extent() maths.Box {
	out := size.Gear.Span * spaceHalf
	return maths.SegmentBox(maths.V(-out, spaceZero, -out),
		maths.V(out, size.Overall, out), spaceZero)
}

// eightSided is an eight-sided prism standing on its own zero: so wide
// across its flats along x, so deep across them along z, and so high,
// with a flat toward each of the four ways a leg goes.
func eightSided(wide, deep, high float64) instrumentPlate {
	corner := spaceHalf / math.Cos(math.Pi/lunarSides)
	var outline []maths.Vec
	for side := range lunarSides {
		// from north through east, a corner either side of each flat
		turn := fullTurn * (float64(side) + spaceHalf) / lunarSides
		outline = append(outline, maths.V(wide*corner*math.Sin(turn),
			spaceZero, -deep*corner*math.Cos(turn)))
	}
	return instrumentPlate{outline: outline, thickness: high}
}

// made keeps quick as the lunar module's default detail.
func (row lunarModule) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt builds the lunar module: the descent stage, an eight-sided
// prism with a flat toward each leg; the ascent stage, another on it,
// of its own width, depth and height; and four legs, each a strut from
// the descent stage's upper edge out and down to a round pad on the
// ground.
func (row lunarModule) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	size := lunarRead
	// each lunar module made has its own copy of each look
	dressed := looks{}
	for where, look := range size.dressed {
		own := *look
		dressed[where] = &own
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(spacecraftValue("sides-" + detail))},
		object: good{dressed: dressed}}
	foot := size.stands()
	assembly.add(eightSided(size.Descent.Across, size.Descent.Across,
		size.Descent.Height), maths.NewSubspace(
		maths.V(spaceZero, foot, spaceZero), maths.Identity, spaceOne),
		lunarBlanket)
	deck := foot + size.Descent.Height
	assembly.add(eightSided(size.Ascent.Width, size.Ascent.Depth,
		size.Ascent.Height), maths.NewSubspace(
		maths.V(spaceZero, deck, spaceZero), maths.Identity, spaceOne),
		lunarCabin)
	reach := (size.Gear.Span - size.Gear.Pad) * spaceHalf
	for leg := range lunarLegs {
		turn := fullTurn * float64(leg) / lunarLegs
		out := maths.V(math.Sin(turn), spaceZero, -math.Cos(turn))
		pad := out.Scale(reach)
		assembly.rod(pad, pad.Add(maths.Up.Scale(size.Gear.PadThick)),
			size.Gear.Pad*spaceHalf, lunarGear)
		top := out.Scale(size.Descent.Across * spaceHalf).Add(
			maths.Up.Scale(deck))
		assembly.rod(pad.Add(maths.Up.Scale(size.Gear.PadThick)), top,
			size.Gear.Strut, lunarGear)
	}
	object, err := assembly.finish(size.extent())
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}
