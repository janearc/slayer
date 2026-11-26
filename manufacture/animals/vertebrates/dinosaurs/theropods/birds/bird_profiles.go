// Package birds builds profile-driven birds from shared skeletons and plumage.
package birds

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sort"

	"github.com/janearc/slayer/manufacture/profilecheck"
	yaml "gopkg.in/yaml.v3"
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

// Book is an immutable, checked book of bird species.
type Book struct{ file birdFile }

// Read checks a bird book without changing other books.
func Read(file string, raw []byte) (*Book, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil, fmt.Errorf("%s: empty or oversized bird book", file)
	}
	if err := profilecheck.SpeciesNumbersPresent(file, raw, []profilecheck.Section{
		{Name: "figures", Kind: reflect.TypeOf(birdFigure{})}, {Name: "plumage", Kind: reflect.TypeOf(birdShade{})},
	}); err != nil {
		return nil, err
	}
	var read birdFile
	strict := yaml.NewDecoder(bytes.NewReader(raw))
	strict.KnownFields(true)
	if err := strict.Decode(&read); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if read.Read == "" || len(read.Species) == 0 || len(read.Species) > 256 {
		return nil, fmt.Errorf("%s: invalid bird species count or provenance", file)
	}
	for _, key := range read.keys() {
		if key == "" {
			return nil, fmt.Errorf("%s: empty bird species key", file)
		}
		row := read.Species[key]
		if err := row.checked(); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, key, err)
		}
		if err := birdProfileBounds(row); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, key, err)
		}
	}
	return &Book{file: read}, nil
}

func birdProfileBounds(row birdSpecies) error {
	// These are numerical margins, not biological limits. Joint positions
	// combine several proportions, so keep each away from zero relative to
	// the largest coordinate the authored measures can produce.
	const minBirdShare = 1e-3
	const minBirdLengthRatio = 1e-4
	if row.Name == "" || row.Binomial == "" {
		return fmt.Errorf("bird needs name and binomial")
	}
	for name, f := range row.Figures {
		if !slices.Contains(birdFigureNames, name) && !slices.Contains([]string{"plumes", "plume-length", "spatula", "web"}, name) {
			return fmt.Errorf("unknown figure %q", name)
		}
		if math.IsNaN(f.Least) || math.IsNaN(f.Most) || math.IsInf(f.Least, 0) || math.IsInf(f.Most, 0) || f.Most > 1000 {
			return fmt.Errorf("figure %s is not bounded", name)
		}
	}
	for _, name := range []string{"cervicals", "thoracics", "caudals", "primaries", "secondaries", "rectrices"} {
		f := row.Figures[name]
		if f.Least < 1 || f.Most > 64 || f.Least != math.Trunc(f.Least) || f.Most != math.Trunc(f.Most) {
			return fmt.Errorf("invalid count %s", name)
		}
	}
	if f, ok := row.Figures["plumes"]; ok && (f.Most > 64 || f.Least != math.Trunc(f.Least) || f.Most != math.Trunc(f.Most)) {
		return fmt.Errorf("invalid plume count")
	}
	for _, name := range []string{"length", "wingspan", "bill", "tarsus", "head", "neck", "tail", "depth", "width", "arm", "humerus", "forearm", "tibia", "femur", "toe", "secondary", "bill-depth", "bill-width"} {
		if row.Figures[name].Least <= 0 {
			return fmt.Errorf("figure %s must be positive", name)
		}
	}
	if row.Bill == birdSpatula && row.Figures["spatula"].Least <= 0 {
		return fmt.Errorf("spatula bill needs a positive spatula width")
	}
	if row.Figures["plumes"].Most > 0 && row.Figures["plume-length"].Least <= 0 {
		return fmt.Errorf("plumes need a positive length")
	}
	if row.Figures["web"].Most > 1 || len(row.Tail.Lengths) > 64 {
		return fmt.Errorf("invalid web or tail")
	}
	for _, x := range row.Tail.Lengths {
		if math.IsNaN(x) || math.IsInf(x, 0) || x < minBirdShare || x > 10 {
			return fmt.Errorf("invalid tail length")
		}
	}
	for name, shade := range row.Plumage {
		if !slices.Contains(birdRegions, name) && name != "plumes" {
			return fmt.Errorf("unknown plumage region %q", name)
		}
		for _, c := range shade.Colour {
			if math.IsNaN(c) || math.IsInf(c, 0) || c < 0 || c > 1 {
				return fmt.Errorf("invalid %s colour", name)
			}
		}
	}
	// These worst-case inequalities keep each interpolated seeded animal valid.
	f := row.Figures
	largestOf := func(values ...float64) float64 {
		largest := 0.0
		for _, value := range values {
			largest = math.Max(largest, value)
		}
		return largest
	}
	largest := largestOf(f["length"].Most*largestOf(1, f["depth"].Most, f["width"].Most),
		f["wingspan"].Most, f["tarsus"].Most*largestOf(1, f["tibia"].Most, f["femur"].Most, f["toe"].Most),
		f["bill"].Most*largestOf(1, f["bill-depth"].Most, f["bill-width"].Most, f["spatula"].Most), f["plume-length"].Most)
	for _, name := range []string{"length", "wingspan", "bill", "tarsus"} {
		if f[name].Least < minBirdLengthRatio*largest {
			return fmt.Errorf("figure %s is too small relative to the bird", name)
		}
	}
	for _, name := range []string{"head", "neck", "tail", "depth", "width", "arm", "humerus", "forearm", "tibia", "femur", "toe", "secondary", "bill-depth", "bill-width"} {
		if f[name].Least < minBirdShare {
			return fmt.Errorf("figure %s is too small for stable geometry", name)
		}
	}
	if row.Bill == birdSpatula && f["spatula"].Least < minBirdShare ||
		f["plumes"].Most > 0 && f["plume-length"].Least < minBirdLengthRatio*largest {
		return fmt.Errorf("bill or plume is too small for stable geometry")
	}
	if f["length"].Least-f["bill"].Most < minBirdLengthRatio*largest ||
		f["head"].Most+f["neck"].Most+f["tail"].Most*row.Tail.longest() > 1-minBirdShare ||
		f["arm"].Most > 1-minBirdShare || f["humerus"].Most+f["forearm"].Most > 1-minBirdShare ||
		f["wingspan"].Least-birdShoulderOut*f["width"].Most*f["length"].Most < minBirdLengthRatio*largest {
		return fmt.Errorf("bird proportions leave no body or wing")
	}
	for _, share := range []float64{0, .5, 1} {
		measures, err := measured(row, share)
		if err != nil {
			return err
		}
		// A positive authored size can still round a derived joint onto
		// its parent. Check the actual skeleton at the same size witnesses.
		if _, err := newBirdBody(row, measures, nil); err != nil {
			return fmt.Errorf("bird skeleton at size %g: %w", share, err)
		}
	}
	return nil
}

// Embedded returns an independent copy of the authored profile book.
func Embedded() []byte { return append([]byte(nil), birdRaw...) }

// Species lists the checked profile names in order.
func (book *Book) Species() []string { return book.file.keys() }

// Specimen pairs one checked bird profile with its stance.
type Specimen struct {
	profile birdSpecies
	stance  string
}

// New selects one species and stance from the checked book.
func (book *Book) New(species, stance string) (Specimen, error) {
	if book == nil {
		return Specimen{}, fmt.Errorf("no bird book")
	}
	if !slices.Contains(birdStances, stance) {
		return Specimen{}, fmt.Errorf("unknown bird stance %q", stance)
	}
	row, ok := book.file.Species[species]
	if !ok {
		return Specimen{}, fmt.Errorf("no bird species %q", species)
	}
	return Specimen{profile: row, stance: stance}, nil
}
