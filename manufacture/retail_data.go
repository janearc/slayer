package manufacture

import (
	"bytes"
	_ "embed"
	"fmt"
	"slices"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"gopkg.in/yaml.v3"
)

//go:embed retail-units.yaml
var retailUnitsData []byte

// retailBookRead is the street-level retail data, read once from the
// embedded file, or why the file is not good.
var retailBookRead, retailBookError = readRetailBook(retailUnitsData)

// retailLevels are the detail levels a frontage is made at, coarsest
// first: a lit opening, then its counter, door and sign, then its stools
// and bowls.
var retailLevels = []string{"hull", "quick", "demo"}

// retailLevel is the place of a detail level among the levels, nought
// for none named, which is the coarsest, and minus one for a name that
// is no level.
func retailLevel(name string) int {
	if name == "" {
		return 0
	}
	return slices.Index(retailLevels, name)
}

// The sizes of a list in the data: a box, a prism, a slab, a place across
// a front, a rectangle, a step, a range, a colour.
const (
	retailBoxNumbers    = 6
	retailPrismNumbers  = 5
	retailSlabNumbers   = 7
	retailPointNumbers  = 3
	retailRectNumbers   = 4
	retailRangeNumbers  = 2
	retailColourNumbers = 3
)

// retailBook is everything the data says of street-level retail: how
// long a run is, the shell every unit has, the palettes a seed picks
// colours from, the materials, the modules that units share, the props
// that stand on the pavement, the unit types, and what a building takes
// to have shops at its foot.
type retailBook struct {
	Run       retailRun                 `yaml:"run"`
	Shell     retailShell               `yaml:"shell"`
	Palettes  map[string][]string       `yaml:"palettes"`
	Materials map[string]retailMaterial `yaml:"materials"`
	Modules   map[string]retailModule   `yaml:"modules"`
	Props     map[string]retailProp     `yaml:"props"`
	Units     map[string]retailUnit     `yaml:"units"`
	Grounds   retailGrounds             `yaml:"grounds"`
}

// retailRun is how many units a frontage has.
type retailRun struct {
	Units int `yaml:"units"`
}

// retailGrounds is what a building takes to stand on a podium of shops:
// the kinds of building that may, by weight, how deep the shops are, how
// much plain wall stands at each end, how many distinct units of each
// type a block draws from, and the look of the plain wall.
type retailGrounds struct {
	Buildings map[string]float64 `yaml:"buildings"`
	Podium    float64            `yaml:"podium"`
	Margin    float64            `yaml:"margin"`
	Pool      int                `yaml:"pool"`
	Filler    string             `yaml:"filler"`
}

// retailShell is the thicknesses every unit's walls and slabs share, in
// metres.
type retailShell struct {
	Floor     float64 `yaml:"floor"`
	Slab      float64 `yaml:"slab"`
	Wall      float64 `yaml:"wall"`
	Partition float64 `yaml:"partition"`
}

// retailMaterial is a look: its colour, what it glows, its gloss, and, for
// glass, how much light it lets through and at what index.
type retailMaterial struct {
	Colour  []float64 `yaml:"colour"`
	Glow    []float64 `yaml:"glow"`
	Gloss   float64   `yaml:"gloss"`
	Through float64   `yaml:"through"`
	Index   float64   `yaml:"index"`
}

// retailPart is one box, prism or slab, which count and step repeat, drawn
// from a detail level on. Along says how it moves when its unit is wider
// than the unit's least: right and centre carry it along, fill stretches
// it or adds to its count. Sides is how many sides a prism has, or the
// city's plant sides for none.
type retailPart struct {
	Says     string    `yaml:"says"`
	Material string    `yaml:"material"`
	Box      []float64 `yaml:"box"`
	Prism    []float64 `yaml:"prism"`
	Slab     []float64 `yaml:"slab"`
	Axis     string    `yaml:"axis"`
	Count    int       `yaml:"count"`
	Step     []float64 `yaml:"step"`
	From     string    `yaml:"from"`
	Along    string    `yaml:"along"`
	Sides    int       `yaml:"sides"`
}

// retailLight is a point of light inside or before a unit. Along says how
// it moves with a wider unit.
type retailLight struct {
	At        []float64 `yaml:"at"`
	Heat      float64   `yaml:"heat"`
	Luminance float64   `yaml:"luminance"`
	From      string    `yaml:"from"`
	Along     string    `yaml:"along"`
}

// retailScatter is a few flat things, stickers or tags, put by the seed
// in a rectangle of the wall plane.
type retailScatter struct {
	Says      string    `yaml:"says"`
	Region    []float64 `yaml:"region"`
	Z         float64   `yaml:"z"`
	Count     int       `yaml:"count"`
	Size      []float64 `yaml:"size"`
	Materials []string  `yaml:"materials"`
	From      string    `yaml:"from"`
}

// retailPropUse is a prop a unit may have: how likely, and over what
// range of x and z the seed stands it.
type retailPropUse struct {
	Prop   string    `yaml:"prop"`
	Chance float64   `yaml:"chance"`
	X      []float64 `yaml:"x"`
	Z      []float64 `yaml:"z"`
	From   string    `yaml:"from"`
}

// retailProp is a thing that stands on the pavement, its parts in its
// own frame, at its foot.
type retailProp struct {
	Says  string       `yaml:"says"`
	Parts []retailPart `yaml:"parts"`
}

// retailUse is a module put in a unit: where, how it moves with a wider
// unit, and how wide it is made to span at the unit's least width. A
// module that spans more than its own width has its filling parts grown
// by the difference.
type retailUse struct {
	Module string    `yaml:"module"`
	At     []float64 `yaml:"at"`
	Along  string    `yaml:"along"`
	Span   float64   `yaml:"span"`
	From   string    `yaml:"from"`
}

// retailModule is a set of parts that more than one unit has: a door, an
// awning, a frame for a sign. Its frame has its start at its lower left
// corner on the wall plane.
type retailModule struct {
	Says  string       `yaml:"says"`
	Width float64      `yaml:"width"`
	Parts []retailPart `yaml:"parts"`
}

// retailKit is what a unit is dressed with: parts, modules, lights,
// stickers and props. A unit has one, and a variant of an option has one.
type retailKit struct {
	Parts   []retailPart    `yaml:"parts"`
	Uses    []retailUse     `yaml:"uses"`
	Lights  []retailLight   `yaml:"lights"`
	Scatter []retailScatter `yaml:"scatter"`
	Props   []retailPropUse `yaml:"props"`
}

// retailVariant is one way an option may go, as likely as its weight.
type retailVariant struct {
	Says   string    `yaml:"says"`
	Weight float64   `yaml:"weight"`
	Kit    retailKit `yaml:",inline"`
}

// retailOption is a choice the seed makes for a unit: with the chance
// given it takes one of the variants. A variant with nothing in it is a
// way to have none. Wide is the fewest bays a unit has to have to be
// offered the option, nought for any.
type retailOption struct {
	Says     string          `yaml:"says"`
	Chance   float64         `yaml:"chance"`
	Wide     int             `yaml:"wide"`
	Variants []retailVariant `yaml:"variants"`
}

// retailUnit is one type of shopfront. It is Bays wide at least and
// BaysMost at most, the seed drawing which; its opening stretches with the
// width if Stretch is set.
type retailUnit struct {
	Says     string         `yaml:"says"`
	Bays     int            `yaml:"bays"`
	BaysMost int            `yaml:"bays-most"`
	Weight   float64        `yaml:"weight"`
	Depth    float64        `yaml:"depth"`
	Stretch  bool           `yaml:"stretch"`
	Opening  []float64      `yaml:"opening"`
	Wall     string         `yaml:"wall"`
	Floor    string         `yaml:"floor"`
	Ceiling  string         `yaml:"ceiling"`
	Inner    string         `yaml:"inner"`
	Sides    string         `yaml:"sides"`
	Sign     []float64      `yaml:"sign"`
	Kit      retailKit      `yaml:",inline"`
	Options  []retailOption `yaml:"options"`
}

// readRetailBook reads the data and checks it, so that a unit that does
// not fit its bays, or names a look there is not, makes no frontage.
func readRetailBook(raw []byte) (retailBook, error) {
	var book retailBook
	reading := yaml.NewDecoder(bytes.NewReader(raw))
	reading.KnownFields(true)
	if err := reading.Decode(&book); err != nil {
		return retailBook{}, fmt.Errorf("manufacture: retail units: %w", err)
	}
	if err := book.check(); err != nil {
		return retailBook{}, fmt.Errorf("manufacture: retail units: %w", err)
	}
	return book, nil
}

// unitNames are the unit types, in the order their names sort.
func (book retailBook) unitNames() []string {
	names := make([]string, 0, len(book.Units))
	for name := range book.Units {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// paletteNames are the palettes, in the order their names sort.
func (book retailBook) paletteNames() []string {
	names := make([]string, 0, len(book.Palettes))
	for name := range book.Palettes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// slots are the numbers of the materials, in the order their names
// sort, so that a material has the same slot in every frontage.
func (book retailBook) slots() map[string]slot {
	names := make([]string, 0, len(book.Materials))
	for name := range book.Materials {
		names = append(names, name)
	}
	sort.Strings(names)
	slots := map[string]slot{}
	for index, name := range names {
		slots[name] = slot(index)
	}
	return slots
}

// look is a material as slayer's things have it.
func (book retailBook) look(name string) *thing.Material {
	data := book.Materials[name]
	look := &thing.Material{Name: "frontage " + name, Gloss: data.Gloss,
		Through: data.Through, Index: data.Index,
		Colour: maths.RGB(data.Colour[0], data.Colour[1], data.Colour[2])}
	if len(data.Glow) == retailColourNumbers {
		look.Glow = maths.RGB(data.Glow[0], data.Glow[1], data.Glow[2])
	}
	return look
}
