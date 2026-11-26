package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The boats of a bay: a sampan, a steel inshore fishing boat and a
// passenger ferry, each made from a seed so that no two are alike. Every
// figure, and whether it was recalled from memory or picked to look
// right, is data: boats.yaml. Nothing in the code names a place; a set
// may float these anywhere there is water.
//
// A boat's own space is metres, y up, its waterline at y nought, the
// centreline on x nought, the bow toward minus z (the way slayer's
// spaces face, and the Alfa's), starboard toward plus x, and the middle
// of its length at z nought. A scene sits a boat on the water by putting
// its origin at the sea's height: there is no draught to take off.
//
// Each design is two kinds, by the state it is in. The kind named for
// the design is under way and shows a red light on its port side, a
// green one to starboard, a white one at its stern, and, a power vessel,
// a white one at its masthead. The kind with "-at-anchor" after its name
// is at anchor and shows one white light all round. Both carry the
// working lights of their design, a lantern, floodlights or a cabin's.
//
// The hull is three lofted solids: the underbody below the waterline,
// the topsides above it and a thin deck. The rest is boxes and rods,
// which a repeated part shares as one mesh placed many times.

//go:embed boats.yaml
var boatRaw []byte

// The designs, and what a state is called.
const (
	boatSampan   = "sampan"
	boatFishing  = "fishing-boat"
	boatFerry    = "ferry"
	boatAtAnchor = "-at-anchor"
)

// Where a figure came from.
const (
	boatRecalled = "recalled" // defined: from memory, not looked up
	boatPicked   = "picked"   // defined: chosen to look right
)

// boatDesigns are the designs, in the order the file gives them.
var boatDesigns = []string{boatSampan, boatFishing, boatFerry}

// boatNames are the ranges a design may give, in the order their
// draws are made: a name's place here is its place in the seed, so a name
// is added at the end.
var boatNames = []string{"length", "slenderness", "freeboard", "draught",
	"spring", "flare", "transom", "bilge", "rake", "camber", "forefoot",
	"bow", "wear", "canopy-length", "canopy-height", "canopy-arched",
	"canopy-tarpaulin", "outboard", "tyre-fenders", "tyres", "crates",
	"nets", "mast-height", "house-position", "house-length",
	"house-width", "house-height", "funnel-height", "boom-length",
	"floods", "window-spacing", "upper-deck", "lit-windows", "rail",
	"strakes", "hatches", "lifebuoys", "buckets", "ribs", "hoops",
	"stanchion-pitch", "cleats", "seat-pitch", "floats", "coils",
	"aerials"}

// boatHullNames are the ranges every design must give: a hull cannot be
// drawn without them.
var boatHullNames = []string{"length", "slenderness", "freeboard",
	"draught", "spring", "flare", "transom", "bilge", "rake", "camber",
	"forefoot", "bow", "wear"}

// boatLookNames are the looks the data must give, by name.
var boatLookNames = []string{"rubber", "steel", "glass", "house",
	"matting", "tarpaulin", "crate", "net", "rust", "lit window",
	"red lens", "green lens", "white lens", "warm lens", "boot topping",
	"lettering", "bamboo", "lifebuoy", "bucket", "seat"}

// boatLampNames are the lamps the data must give, by name.
var boatLampNames = []string{"port", "starboard", "stern", "masthead",
	"anchor", "lantern", "flood", "cabin"}

// boatRange is a range a seed picks from, and where it came from.
type boatRange struct {
	Least, Most float64
	Basis       string
}

// boatColour is a paint's colour, in linear light, and where it came from.
type boatColour struct {
	Name   string
	Colour []float64
	Basis  string
}

// boatDesign is one design: its ranges, the paints its hull may wear,
// its underbody, and the look of its deck by name.
type boatDesign struct {
	Ranges    map[string]boatRange
	Paints    []boatColour
	Underbody boatColour
	Deck      string
}

// boatLook is what a fitting is made of: its reflectance, the light it
// gives on its own where it gives any, and how much of a mirror it is.
type boatLook struct {
	Colour, Glow []float64
	Gloss        float64
	Basis        string
}

// boatLamp is one lamp: a tint where it has a colour of its own, or a
// heat in kelvin; the arc it shines across in degrees and how far past
// that it fades; its luminance at a metre in the kit's relative light;
// and how far from it its light starts.
type boatLamp struct {
	Tint                                 []float64
	Kelvin, Arc, Soft, Luminance, Beyond float64
	ArcBasis                             string `yaml:"arc-basis"`
	Basis                                string
}

// boatLettering is the painted letters: the face of the Latin letters and
// digits, the names a stern may carry, and a place for Chinese, a face
// and names, empty until a face of Simplified Chinese has landed.
type boatLettering struct {
	Face    string
	Names   []string
	Basis   string
	Chinese struct {
		Face  string
		Names []string
	}
}

// boatSizes is the file: the designs, the looks, the lamps, the lettering,
// and where the figures were read.
type boatSizes struct {
	Kinds     map[string]boatDesign
	Looks     map[string]boatLook
	Lamps     map[string]boatLamp
	Lettering boatLettering
	Read      string
}

// boatRead is the file read and checked. boatError is why it was not.
var boatRead, boatError = readBoats(boatRaw)

// readBoats reads the boats' data and checks it, so that making a boat
// afterwards cannot fail for want of a figure.
func readBoats(raw []byte) (boatSizes, error) {
	var read boatSizes
	if err := yamlfile.Read("boats.yaml", raw, &read); err != nil {
		return boatSizes{}, err
	}
	if read.Read == "" {
		return boatSizes{}, fmt.Errorf("manufacture: the boats' figures " +
			"were read nowhere")
	}
	for _, check := range []func() error{read.designed, read.looked,
		read.lamped, read.lettered} {
		if err := check(); err != nil {
			return boatSizes{}, err
		}
	}
	return read, nil
}

// based is whether a basis is one the file may give.
func based(basis string) bool {
	return basis == boatRecalled || basis == boatPicked
}

// shares checks reflectance through the shared profile policy.
var shares = profilecheck.ColourShares

// designed refuses a design that is missing, or has a range that is
// not a range, a name nobody reads, a hull figure it does not give, no
// paints, or a deck, paint or underbody that cannot be made.
func (sizes boatSizes) designed() error {
	for _, name := range boatDesigns {
		design, given := sizes.Kinds[name]
		if !given {
			return fmt.Errorf("manufacture: no boat design %q", name)
		}
		if err := design.sound(name); err != nil {
			return err
		}
		if _, known := sizes.Looks[design.Deck]; !known {
			return fmt.Errorf("manufacture: the %s's deck %q is not a "+
				"look", name, design.Deck)
		}
	}
	return nil
}

// sound refuses what is wrong with one design's figures.
func (design boatDesign) sound(name string) error {
	for key, row := range design.Ranges {
		if !slices.Contains(boatNames, key) || !based(row.Basis) ||
			!(row.Least >= 0 && row.Most >= row.Least) ||
			math.IsInf(row.Most, 0) {
			return fmt.Errorf("manufacture: the %s's %s, %+v, is not a "+
				"range with a name and a basis", name, key, row)
		}
	}
	for _, key := range boatHullNames {
		if !positive(design.Ranges[key].Least) {
			return fmt.Errorf("manufacture: the %s gives no %s", name, key)
		}
	}
	paints := append([]boatColour{design.Underbody}, design.Paints...)
	for _, paint := range paints {
		if !shares(paint.Colour) || !based(paint.Basis) {
			return fmt.Errorf("manufacture: the %s's paint %q is not a "+
				"colour with a basis", name, paint.Name)
		}
	}
	if len(design.Paints) == 0 {
		return fmt.Errorf("manufacture: the %s has no paints", name)
	}
	return nil
}

// looked refuses a look that is missing or cannot be made.
func (sizes boatSizes) looked() error {
	names := append([]string{"deck wood", "deck steel"}, boatLookNames...)
	for _, name := range names {
		look, given := sizes.Looks[name]
		glows := len(look.Glow) == 0 || len(look.Glow) == alfaChannels
		if !given || !shares(look.Colour) || !share(look.Gloss) ||
			!glows || !based(look.Basis) {
			return fmt.Errorf("manufacture: the boats' look %q is missing "+
				"or is not a colour, a gloss and a basis", name)
		}
	}
	return nil
}

// lamped refuses a lamp that is missing, shines across no arc or a
// whole turn and more, has no strength, or is neither a tint nor a heat.
func (sizes boatSizes) lamped() error {
	for _, name := range boatLampNames {
		lamp, given := sizes.Lamps[name]
		coloured := len(lamp.Tint) == alfaChannels || positive(lamp.Kelvin)
		if !given || !coloured || !(lamp.Arc > 0 && lamp.Arc <= boatTurn) ||
			!(lamp.Soft >= 0) || !positive(lamp.Luminance) ||
			!positive(lamp.Beyond) || !based(lamp.Basis) ||
			!based(lamp.ArcBasis) {
			return fmt.Errorf("manufacture: the boats' lamp %q is missing "+
				"or cannot shine", name)
		}
	}
	return nil
}

// lettered refuses lettering with no face, no names or no basis, and a
// Chinese face that is named with no names to set in it.
func (sizes boatSizes) lettered() error {
	words := sizes.Lettering
	if words.Face == "" || len(words.Names) == 0 || !based(words.Basis) ||
		(words.Chinese.Face != "" && len(words.Chinese.Names) == 0) {
		return fmt.Errorf("manufacture: the boats' lettering needs a face, "+
			"names and a basis, and a Chinese face needs Chinese names: %+v",
			words)
	}
	return nil
}

// boatTurn is a whole turn in degrees: the arc of a lamp that shines
// every way.
const boatTurn = 360 // defined

// pick is a figure of the design from a seed: a share of the way across
// its range, the share the seed draws for the name.
func (design boatDesign) pick(name string, from seed) float64 {
	row := design.Ranges[name]
	index := slices.Index(boatNames, name)
	return row.Least + (row.Most-row.Least)*from.draw(picking, index, 0)
}

// count is a pick of a whole number: each from the least to the most is
// as likely as the next.
func (design boatDesign) count(name string, from seed) int {
	row := design.Ranges[name]
	index := slices.Index(boatNames, name)
	return span{row.Least, row.Most}.count(from.draw(picking, index, 0))
}

// chance is whether the seed draws a share below the design's figure
// for the name, which is then the chance it has: a draw of its own,
// not the pick's.
func (design boatDesign) chance(name string, from seed) bool {
	index := slices.Index(boatNames, name)
	return from.draw(picking, index, 1) < design.pick(name, from)
}

// boatRow is a kind of boat: a design, and whether it rides at anchor.
type boatRow struct {
	design   string
	anchored bool
}

// boatRows supplies each boat by name, a design under way and at
// anchor. A file that cannot be read stops the build, as the other
// embedded data files do.
func boatRows() map[string]maker {
	if boatError != nil {
		panic(boatError)
	}
	rows := map[string]maker{}
	for _, name := range boatDesigns {
		rows[name] = boatRow{design: name}
		rows[name+boatAtAnchor] = boatRow{design: name, anchored: true}
	}
	return rows
}

// made keeps quick as the boat's default detail.
func (row boatRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the boat at a detail: its hull, its fittings and its lights.
// It refuses a seed never sown.
func (row boatRow) madeAt(from seed, detail string) (good, error) {
	rank := slices.Index(Complexities(), detail)
	if !from.made() || rank < 0 {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	craft := row.craft(from, rank)
	fine, err := cutInto(boatRods[rank])
	if err != nil {
		return good{}, err
	}
	craft.assembly = &instrumentAssembly{fine: fine,
		object: good{dressed: craft.dressed()}}
	craft.build()
	bounds := partBounds(craft.assembly.object.parts)
	if err := craft.assembly.shareRepeats(); err != nil {
		return good{}, err
	}
	made, err := craft.assembly.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return craft.lit(*made)
}

// Detail's corners round a rod, by the rank of the detail, from the
// silhouette up: chosen, enough for a mast to read as round at 2K.
var boatRods = [...]int{6, 8, 10, 14}

// Buoyant is what the water needs of a kind that floats, by its name,
// and whether it floats: the Alfa's, as Floating gives it, or a boat's.
// A boat's waterline is its origin, so it has no waterline to take off
// and no wet look to put where it is wet.
func Buoyant(name string) (Afloat, bool) {
	for _, design := range boatDesigns {
		if name == design || name == design+boatAtAnchor {
			return Afloat{Hull: "hull paint"}, boatError == nil
		}
	}
	return Floating(name)
}

// BoatDraught is the most water, in metres, that a boat of a kind draws
// under its waterline, which is the most of its design's range for the
// draught, and whether the kind is a boat. A boat stood in less water than
// that may have a seed whose keel is under the bed.
func BoatDraught(name string) (float64, bool) {
	for _, design := range boatDesigns {
		if name == design || name == design+boatAtAnchor {
			return boatRead.Kinds[design].Ranges["draught"].Most,
				boatError == nil
		}
	}
	return 0, false
}

// material is the look the file gives a name, as slayer's material.
func (sizes boatSizes) material(name string) *thing.Material {
	look := sizes.Looks[name]
	made := &thing.Material{Name: name, Gloss: look.Gloss,
		Colour: maths.RGB(look.Colour[0], look.Colour[1], look.Colour[2])}
	if len(look.Glow) == alfaChannels {
		made.Glow = maths.RGB(look.Glow[0], look.Glow[1], look.Glow[2])
	}
	return made
}
