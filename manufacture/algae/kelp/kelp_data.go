package kelp

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
)

// The figures of giant kelp are data: kelp.yaml. Each carries a basis
// that begins with cited, derived or picked, and a cited one names a
// source the file lists. The names a figure may have are kelpNames; a
// name's place in that list is its place in a seed, so a name is added
// at the end.

//go:embed kelp.yaml
var kelpRaw []byte

// kelpNames are the figures a plant is made from, in seed order.
var kelpNames = []string{"depth", "drop", "stipes", "stipe-foot-radius",
	"stipe-tip-radius", "segment", "fan", "spread", "node-spacing",
	"blade-length", "blade-width", "blade-thickness", "ruffle-height",
	"ruffle-waves", "blade-lift", "float-length",
	"holdfast-radius", "holdfast-lobes", "canopy-reach", "variants",
	"spacing", "plants", "blade-curve"}

// kelpPhysics are the constants the file must give.
var kelpPhysics = []string{"gravity", "lean-reference", "lean-share",
	"follow", "tip-width", "float-aspect", "holdfast-cone", "blade-stream",
	"blade-buoy"}

// kelpLooks are the looks the file must give.
var kelpLooks = []string{"stipe", "blade", "float"}

// kelpLook is a reflectance in linear light, and where it came from.
type kelpLook struct {
	Colour []float64
	Basis  string
}

// kelpCondition is a named motion of the water: the current at the
// plant in metres a second, and the swell's height from trough to
// crest in metres and its period in seconds.
type kelpCondition struct {
	Current     float64
	SwellHeight float64 `yaml:"swell-height"`
	SwellPeriod float64 `yaml:"swell-period"`
	Basis       string
}

// kelpBook is the file.
type kelpBook struct {
	Species    string
	Read       string
	Sources    map[string]string
	Figures    map[string]profilecheck.Figure
	Looks      map[string]kelpLook
	Physics    map[string]profilecheck.Constant
	Conditions map[string]kelpCondition
}

// readKelp reads the file and checks every figure, so that making a
// plant cannot fail for want of one.
func readKelp(raw []byte) (kelpBook, error) {
	var book kelpBook
	if err := yamlfile.Read("kelp.yaml", raw, &book); err != nil {
		return kelpBook{}, err
	}
	if strings.TrimSpace(book.Species) == "" {
		return kelpBook{}, fmt.Errorf("manufacture: kelp has no species")
	}
	if book.Read == "" {
		return kelpBook{}, fmt.Errorf("manufacture: kelp's figures " +
			"were read nowhere")
	}
	for _, check := range []func() error{book.figured, book.constant,
		book.looked, book.conditioned} {
		if err := check(); err != nil {
			return kelpBook{}, err
		}
	}
	return book, nil
}

// figured refuses a figure that is missing, unknown, not a range or
// without a basis.
func (book kelpBook) figured() error {
	for _, name := range kelpNames {
		row, given := book.Figures[name]
		if !given || !(row.Least >= 0 && row.Most >= row.Least) ||
			!positive(row.Most) || !profilecheck.Based(book.Sources, row.Basis) {
			return fmt.Errorf("manufacture: kelp's %s, %+v, is not a range "+
				"with a basis", name, row)
		}
	}
	for name := range book.Figures {
		if !slices.Contains(kelpNames, name) {
			return fmt.Errorf("manufacture: kelp's %s is read by nothing",
				name)
		}
	}
	for _, name := range []string{"depth", "segment", "node-spacing", "blade-length"} {
		if !positive(book.Figures[name].Least) {
			return fmt.Errorf("manufacture: kelp's %s must stay above zero", name)
		}
	}
	if book.Figures["drop"].Most >= book.Figures["depth"].Least {
		return fmt.Errorf("manufacture: kelp's drop must stay below its depth")
	}
	variants := book.Figures["variants"]
	if variants.Least < 1 || variants.Least != math.Trunc(variants.Least) || variants.Most != variants.Least || variants.Most > 64 {
		return fmt.Errorf("manufacture: kelp needs one to 64 whole blade variants")
	}
	return nil
}

// constant refuses a constant that is missing or without a basis.
func (book kelpBook) constant() error {
	for _, name := range kelpPhysics {
		row, given := book.Physics[name]
		if !given || !positive(row.Value) || !profilecheck.Based(book.Sources, row.Basis) {
			return fmt.Errorf("manufacture: kelp's %s, %+v, is not a "+
				"number with a basis", name, row)
		}
	}
	return nil
}

// looked refuses a look that is missing or is not three shares.
func (book kelpBook) looked() error {
	for _, name := range kelpLooks {
		look, given := book.Looks[name]
		if !given || !profilecheck.ColourShares(look.Colour) || !profilecheck.Based(book.Sources, look.Basis) {
			return fmt.Errorf("manufacture: kelp's look %q is not a "+
				"colour with a basis", name)
		}
	}
	return nil
}

// conditioned refuses a condition that has no basis, a negative
// current, or a swell with a height and no period.
func (book kelpBook) conditioned() error {
	if len(book.Conditions) == 0 {
		return fmt.Errorf("manufacture: kelp has no water conditions")
	}
	for name, water := range book.Conditions {
		if !(water.Current >= 0) || math.IsInf(water.Current, 0) ||
			!(water.SwellHeight >= 0) || math.IsInf(water.SwellHeight, 0) ||
			!positive(water.SwellPeriod) || !profilecheck.Based(book.Sources, water.Basis) {
			return fmt.Errorf("manufacture: kelp's condition %q, %+v, is "+
				"not a water", name, water)
		}
	}
	return nil
}

// pick is a figure from a plant's seed: a share of the way across the
// range, the share the seed draws for the name at an index.
func (book kelpBook) pick(name string, from branching.Seed, index int) float64 {
	row := book.Figures[name]
	place := slices.Index(kelpNames, name)
	return row.Least + (row.Most-row.Least)*from.Draw(0, place, index)
}

// count is a pick of a whole number: each from the least to the most is
// as likely as the next.
func (book kelpBook) count(name string, from branching.Seed, index int) int {
	row := book.Figures[name]
	place := slices.Index(kelpNames, name)
	return maths.CountBetween(row.Least, row.Most, from.Draw(0, place, index))
}

// constantOf is a constant by name.
func (book kelpBook) constantOf(name string) float64 {
	return book.Physics[name].Value
}

// Embedded returns an independent copy of the default species book.
func Embedded() []byte { return append([]byte(nil), kelpRaw...) }

func positive(n float64) bool { return n > 0 && !math.IsInf(n, 0) }
