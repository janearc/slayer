package manufacture

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// The figures of giant kelp are data: kelp.yaml. Each carries a basis
// that begins with cited, derived or picked, and a cited one names a
// source the file lists. The names a figure may have are kelpNames; a
// name's place in that list is its place in a seed, so a name is added
// at the end.

//go:embed kelp.yaml
var kelpRaw []byte

// the words a basis may begin with
const (
	kelpCited   = "cited"    // defined: a published page was read
	kelpDerived = "derived"  // defined: worked from other figures
	kelpPicked  = "picked"   // defined: chosen to look right
	kelpRecall  = "recalled" // defined: from memory, not read
)

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

// kelpFigure is a range a seed picks from, and where it came from.
type kelpFigure struct {
	Least, Most float64
	Basis       string
}

// kelpLook is a reflectance in linear light, and where it came from.
type kelpLook struct {
	Colour []float64
	Basis  string
}

// kelpConstant is a number that is not a range.
type kelpConstant struct {
	Value float64
	Unit  string
	Basis string
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
	Read       string
	Sources    map[string]string
	Figures    map[string]kelpFigure
	Looks      map[string]kelpLook
	Physics    map[string]kelpConstant
	Conditions map[string]kelpCondition
}

// kelpRead is the file read and checked, and kelpError why it was not.
var kelpRead, kelpError = readKelp(kelpRaw)

// readKelp reads the file and checks every figure, so that making a
// plant cannot fail for want of one.
func readKelp(raw []byte) (kelpBook, error) {
	var book kelpBook
	if err := yamlfile.Read("kelp.yaml", raw, &book); err != nil {
		return kelpBook{}, err
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

// based is whether a basis begins with a word the file may use, and a
// cited one names a source the file lists.
func (book kelpBook) based(basis string) bool {
	word, rest, _ := strings.Cut(basis, ":")
	switch word {
	case kelpPicked, kelpDerived, kelpRecall:
		return true
	case kelpCited:
		for name := range book.Sources {
			if strings.Contains(rest, name) {
				return true
			}
		}
	}
	return false
}

// figured refuses a figure that is missing, unknown, not a range or
// without a basis.
func (book kelpBook) figured() error {
	for _, name := range kelpNames {
		row, given := book.Figures[name]
		if !given || !(row.Least >= 0 && row.Most >= row.Least) ||
			!positive(row.Most) || !book.based(row.Basis) {
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
	return nil
}

// constant refuses a constant that is missing or without a basis.
func (book kelpBook) constant() error {
	for _, name := range kelpPhysics {
		row, given := book.Physics[name]
		if !given || !positive(row.Value) || !book.based(row.Basis) {
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
		if !given || !shares(look.Colour) || !book.based(look.Basis) {
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
		if !(water.Current >= 0) || !(water.SwellHeight >= 0) ||
			!positive(water.SwellPeriod) || !book.based(water.Basis) {
			return fmt.Errorf("manufacture: kelp's condition %q, %+v, is "+
				"not a water", name, water)
		}
	}
	return nil
}

// pick is a figure from a plant's seed: a share of the way across the
// range, the share the seed draws for the name at an index.
func (book kelpBook) pick(name string, from seed, index int) float64 {
	row := book.Figures[name]
	place := slices.Index(kelpNames, name)
	return row.Least + (row.Most-row.Least)*from.draw(picking, place, index)
}

// count is a pick of a whole number: each from the least to the most is
// as likely as the next.
func (book kelpBook) count(name string, from seed, index int) int {
	row := book.Figures[name]
	place := slices.Index(kelpNames, name)
	return span{row.Least, row.Most}.count(from.draw(picking, place, index))
}

// constantOf is a constant by name.
func (book kelpBook) constantOf(name string) float64 {
	return book.Physics[name].Value
}
