package manufacture

import (
	_ "embed"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// The birds: one maker for every bird, a species being a row of
// birds.yaml. A row gives the bird's sizes, each with where it was read
// or that it was picked, its bill's shape, how it carries its neck and
// legs in flight, and its plumage by region.
//
// The maker lays out a skeleton on the shared bioforms/skeleton package
// from those numbers, hangs the flight and tail feathers on the bones of
// the hand, the forearm and the tail, puts the masses of the body's
// plumage on the bones, and poses the whole by turning joints.
//
// A bird's own space is metres, y up, the bill toward plus x and its
// right toward plus z. A standing bird's toes rest on y nought, so a
// scene stands it by putting its origin on the ground or the mud.

//go:embed birds.yaml
var birdRaw []byte

// Where a figure came from.
const (
	birdReadBasis    = "read"    // defined: read on the page the row cites
	birdDerivedBasis = "derived" // defined: worked from figures read
	birdPickedBasis  = "picked"  // defined: chosen to look right
)

// The shapes a bill may have, and how a bird carries its neck and its
// legs in flight.
const (
	birdSpatula   = "spatula"   // defined: flat, widening to a spoon
	birdDagger    = "dagger"    // defined: straight, tapering to a point
	birdHooked    = "hooked"    // defined: deep, the upper tip bent down
	birdRetracted = "retracted" // defined: the neck folded back in an S
	birdExtended  = "extended"  // defined: the neck held out straight
	birdTrailed   = "trailed"   // defined: the legs held out behind
	birdTucked    = "tucked"    // defined: the feet drawn under the tail
)

// birdFigureNames are the figures every row must give. Lengths are in
// metres; head, neck and tail are shares of the length less the bill;
// depth and width of the trunk; arm of the half span outside the body;
// humerus and forearm of the arm; tibia, femur and toe of the tarsus.
var birdFigureNames = []string{"length", "wingspan", "bill", "tarsus",
	"head", "neck", "tail", "depth", "width", "arm", "humerus", "forearm",
	"tibia", "femur", "toe", "secondary", "bill-depth", "bill-width",
	"stance", "neck-bend", "bill-tilt", "cervicals", "thoracics",
	"caudals", "primaries", "secondaries", "rectrices"}

// birdRegions are the regions of plumage and bare parts every row
// colours.
var birdRegions = []string{"crown", "face", "neck", "back", "wing",
	"primaries", "secondaries", "tail", "belly", "bill", "legs", "feet",
	"eye"}

// birdFigure is a range a seed picks from, and where it came from.
type birdFigure struct {
	Least  float64
	Most   float64
	Basis  string
	Source string
}

// birdShade is a region's colour as a diffuse reflectance, the words
// its source gives for it, and where those words were read.
type birdShade struct {
	Colour [3]float64
	Words  string
	Basis  string
	Source string
}

// birdNative is where a species is found, and the words that say so.
type birdNative struct {
	Place  string
	Source string
	Quote  string
}

// birdSpecies is one row: a species and every figure of it.
type birdSpecies struct {
	Name     string
	Binomial string
	Plan     string
	Native   birdNative
	Bill     string   `yaml:"bill-shape"`
	Neck     string   `yaml:"neck-in-flight"`
	Legs     string   `yaml:"legs-in-flight"`
	Tail     birdTail `yaml:"tail-shape"`
	Figures  map[string]birdFigure
	Plumage  map[string]birdShade
}

// birdTail is the length of each pair of tail feathers, from the middle
// pair outward, as shares of the tail figure: a deep fork grows
// outward, a pair of streamers is a long first share. A row that gives
// none has a tail whose feathers are all the tail's length.
type birdTail struct {
	Lengths []float64
	Basis   string
	Source  string
}

// share is the length of the pair counted pair from the middle, from
// nought, as a share of the tail figure; past the last given, the last.
func (tail birdTail) share(pair int) float64 {
	if len(tail.Lengths) == 0 {
		return 1
	}
	return tail.Lengths[min(pair, len(tail.Lengths)-1)]
}

// longest is the share of the longest pair, or one for a plain tail.
func (tail birdTail) longest() float64 {
	if len(tail.Lengths) == 0 {
		return 1
	}
	return slices.Max(tail.Lengths)
}

// checked is why a tail's lengths cannot be used, or nil.
func (tail birdTail) checked() error {
	if len(tail.Lengths) == 0 {
		return nil
	}
	if slices.Min(tail.Lengths) <= 0 {
		return errors.New("a tail feather of no length")
	}
	figure := birdFigure{Least: 1, Most: 1, Basis: tail.Basis,
		Source: tail.Source}
	return figure.checked()
}

// birdFile is the file: the species by name, and where the figures
// were read.
type birdFile struct {
	Read    string
	Species map[string]birdSpecies
}

// birdRead is the file read and checked. birdError is why it was not.
var birdRead, birdError = readBirds(birdRaw)

var (
	errBirdFile    = errors.New("manufacture: birds.yaml")
	errBirdSpecies = errors.New("manufacture: no such bird")
)

// readBirds reads the birds' data and checks it, so that making a bird
// afterwards cannot fail for want of a figure.
func readBirds(raw []byte) (birdFile, error) {
	var read birdFile
	if err := yamlfile.Read("birds.yaml", raw, &read); err != nil {
		return birdFile{}, err
	}
	if read.Read == "" || len(read.Species) == 0 {
		return birdFile{}, fmt.Errorf("%w: no species, or read nowhere",
			errBirdFile)
	}
	for _, key := range read.keys() {
		if err := read.Species[key].checked(); err != nil {
			return birdFile{}, fmt.Errorf("%w: %s: %w", errBirdFile, key,
				err)
		}
	}
	return read, nil
}

// keys are the species' names in order.
func (file birdFile) keys() []string {
	keys := make([]string, 0, len(file.Species))
	for key := range file.Species {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// checked is why a row cannot make a bird, or nil.
func (species birdSpecies) checked() error {
	if species.Native.Source == "" || species.Native.Quote == "" {
		return errors.New("no source says where it is native")
	}
	if !slices.Contains([]string{birdSpatula, birdDagger, birdHooked},
		species.Bill) {
		return fmt.Errorf("no bill shape %q", species.Bill)
	}
	if !slices.Contains([]string{birdRetracted, birdExtended},
		species.Neck) {
		return fmt.Errorf("no neck in flight %q", species.Neck)
	}
	if !slices.Contains([]string{birdTrailed, birdTucked}, species.Legs) {
		return fmt.Errorf("no legs in flight %q", species.Legs)
	}
	if err := species.figuresChecked(); err != nil {
		return err
	}
	return species.plumageChecked()
}

// figuresChecked is why a row's figures cannot be used, or nil.
func (species birdSpecies) figuresChecked() error {
	for _, name := range birdFigureNames {
		if _, ok := species.Figures[name]; !ok {
			return fmt.Errorf("no figure %s", name)
		}
	}
	for name, figure := range species.Figures {
		if err := figure.checked(); err != nil {
			return fmt.Errorf("figure %s: %w", name, err)
		}
	}
	if err := species.Tail.checked(); err != nil {
		return fmt.Errorf("tail: %w", err)
	}
	return nil
}

// checked is why a figure cannot be used, or nil.
func (figure birdFigure) checked() error {
	switch {
	case !slices.Contains([]string{birdReadBasis, birdDerivedBasis,
		birdPickedBasis}, figure.Basis):
		return fmt.Errorf("no basis %q", figure.Basis)
	case figure.Basis != birdPickedBasis && figure.Source == "":
		return errors.New("read or derived, and no source")
	case figure.Least < 0 || figure.Most < figure.Least:
		return errors.New("the least is below nought or above the most")
	}
	return nil
}

// plumageChecked is why a row's colours cannot be used, or nil.
func (species birdSpecies) plumageChecked() error {
	regions := birdRegions
	if species.Figures["plumes"].Most > 0 {
		regions = append(slices.Clone(regions), "plumes")
	}
	for _, region := range regions {
		shade, ok := species.Plumage[region]
		if !ok {
			return fmt.Errorf("no colour for the %s", region)
		}
		if shade.Words == "" || shade.Source == "" {
			return fmt.Errorf("the %s's colour has no words read",
				region)
		}
		for _, channel := range shade.Colour {
			if channel < 0 || channel > 1 {
				return fmt.Errorf("the %s's colour is not a reflectance",
					region)
			}
		}
	}
	return nil
}

// pick is the figure named name a share of the way through its range;
// a figure the row does not give is nought.
func (species birdSpecies) pick(name string, share float64) float64 {
	figure := species.Figures[name]
	return figure.Least + (figure.Most-figure.Least)*share
}

// count is the figure named name as a whole number.
func (species birdSpecies) count(name string, share float64) int {
	return int(math.Round(species.pick(name, share)))
}

// birdSpeciesNamed is the row of the species named key.
func birdSpeciesNamed(key string) (birdSpecies, error) {
	if birdError != nil {
		return birdSpecies{}, birdError
	}
	species, ok := birdRead.Species[key]
	if !ok {
		return birdSpecies{}, fmt.Errorf("%w: %s", errBirdSpecies, key)
	}
	return species, nil
}
