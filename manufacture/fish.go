package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// The fish: one maker for every bony fish the data gives, each species a
// row of fish.yaml with every figure's basis beside it.
//
// A fish is built as
// a fish is: its bones first, on the shared skeleton package, then a body
// lofted round the column from the species' proportions and carried by
// the bones, then its fins as webs between their rays, its eyes, and its
// skin painted by the species' rule. It is posed by turning the joints of
// its column, the way its species swims.
//
// Each species is three kinds and a school: the name alone is a fish at
// some phase of its stroke, the name with "-glide" is one gliding, nearly
// straight, and the name with "-bones" is its skeleton at rest. A species
// whose row lists schools is also a kind named with "-school-" and the
// count after it: that many fish, of a few made fish repeated. Nothing in
// the code names a species; a species more is a row more.
//
// A fish's own space is metres, y up, the snout toward plus x, the fork of
// the tail toward minus x, the middle of the length at x nought, and the
// body's axis at y nought. Shares are of the fork length, from the snout.

//go:embed fish.yaml
var fishRaw []byte

// The kinds a species is, by what follows its name.
const (
	fishGlideKind  = "-glide"
	fishBonesKind  = "-bones"
	fishSchoolKind = "-school-"
)

// Where a figure came from.
const (
	fishFromRead    = "read"    // defined: from the page its source names
	fishFromDerived = "derived" // defined: worked from figures read
	fishFromPicked  = "picked"  // defined: chosen to look right
)

// fishSchoolNames are the figures a species that schools must give.
var fishSchoolNames = []string{"school-spacing", "school-scatter",
	"school-heading", "school-variants", "school-long", "school-wide",
	"school-high"}

// fishFigureNames are the figures a species gives, in the order their
// draws are made: a name's place here is its place in the seed, so a name
// is added at the end. The figures every species gives are these less the
// school's.
var fishFigureNames = append([]string{"length", "depth", "deepest", "width",
	"head", "snout", "stalk", "stalk-width", "stalk-at", "tail-base", "eye",
	"eye-at", "eye-height", "back-line", "gape", "tail-span", "tail-sweep",
	"pectoral", "pectoral-at", "pectoral-height", "pelvic", "pelvic-at",
	"first-dorsal-start", "first-dorsal-end", "first-dorsal-front",
	"first-dorsal-back", "first-dorsal-spines", "first-dorsal-rays",
	"second-dorsal-start", "second-dorsal-end", "second-dorsal-front",
	"second-dorsal-back", "second-dorsal-spines", "second-dorsal-rays",
	"anal-start", "anal-end", "anal-front", "anal-back", "anal-spines",
	"anal-rays", "dorsal-finlets", "anal-finlets", "caudal-rays",
	"pectoral-rays", "pelvic-rays", "vertebrae", "precaudal", "keels",
	"bend-from", "amplitude", "wavelength", "glide", "stripe-height",
	"stripe-width", "spot-rows", "spot-pitch", "spot-size", "spot-height",
	"ribs-from", "fin-thickness", "bar-count", "bar-width", "bar-from"},
	fishSchoolNames...)

// fishDrawNames are the draws a fish makes that are not figures, after
// the figures' places in the seed.
var fishDrawNames = []string{"phase"}

// fishMedianFins are the fins on the body's middle line, before the tail.
var fishMedianFins = []string{"first-dorsal", "second-dorsal", "anal"}

// fishLookNames are the looks every species gives. The fin look is the
// dorsal and pectoral fins'; low-fin is the anal and pelvic fins'; spot is
// also the colour of bars.
var fishLookNames = []string{"back", "belly", "fin", "low-fin", "tail",
	"finlet", "stripe", "spot", "iris", "pupil", "bone"}

// mostSchool is the most fish a school may hold, as a stand may.
const mostSchool = 100000 // picked: the stand's own most

// fishFigure is a range a seed picks from, where it came from, and the
// page or the working that says so.
type fishFigure struct {
	Least, Most float64
	Basis       string
	Source      string
}

// fishLook is what a part of a fish is made of: its reflectance, how much
// of a mirror it is, and where that came from.
type fishLook struct {
	Colour []float64
	Gloss  float64
	Basis  string
	Source string
}

// fishSpecies is one species' row: its figures and looks by name, and the
// counts of the schools it is made in, if it schools.
type fishSpecies struct {
	Figures map[string]fishFigure
	Looks   map[string]fishLook
	Schools []int
}

// fishFile is the file: every species by its kind's name, and where the
// figures were read.
type fishFile struct {
	Read    string
	Species map[string]fishSpecies
}

// fishRead is the file read and checked. fishError is why it was not.
var fishRead, fishError = readFish(fishRaw)

// readFish reads the fishes' data and checks it, so that making a fish
// afterwards cannot fail for want of a figure.
func readFish(raw []byte) (fishFile, error) {
	var read fishFile
	if err := yamlfile.Read("fish.yaml", raw, &read); err != nil {
		return fishFile{}, err
	}
	if err := read.checked(); err != nil {
		return fishFile{}, err
	}
	return read, nil
}

// checked refuses a file that says where nothing was read, or has a
// species that is not sound.
func (file fishFile) checked() error {
	if file.Read == "" || len(file.Species) == 0 {
		return fmt.Errorf("manufacture: the fishes' figures were read " +
			"nowhere, or there are no fish")
	}
	for name, species := range file.Species {
		for _, check := range []func(string) error{species.figured,
			species.looked, species.proportioned, species.schooled} {
			if err := check(name); err != nil {
				return err
			}
		}
	}
	return nil
}

// fishBased is whether a figure's basis is one the file may give, with a
// source where its basis needs one.
func fishBased(basis, source string) bool {
	switch basis {
	case fishFromRead, fishFromDerived:
		return source != ""
	case fishFromPicked:
		return true
	}
	return false
}

// figured refuses a figure nobody reads, one that is not a range, one
// with no basis, and a figure the species must give and does not.
func (species fishSpecies) figured(name string) error {
	for key, row := range species.Figures {
		finite := !math.IsInf(row.Least, 0) && !math.IsInf(row.Most, 0)
		if !slices.Contains(fishFigureNames, key) || !finite ||
			!(row.Most >= row.Least) || !fishBased(row.Basis, row.Source) {
			return fmt.Errorf("manufacture: the %s's %s, %+v, is not a "+
				"range with a name and a basis", name, key, row)
		}
	}
	for _, key := range species.wanted() {
		if _, given := species.Figures[key]; !given {
			return fmt.Errorf("manufacture: the %s gives no %s", name, key)
		}
	}
	return nil
}

// wanted is the figures the species must give: all but the school's,
// and the school's too where it schools.
func (species fishSpecies) wanted() []string {
	if len(species.Schools) > 0 {
		return fishFigureNames
	}
	return fishFigureNames[:len(fishFigureNames)-len(fishSchoolNames)]
}

// looked refuses a look that is missing, is not a colour and a gloss, or
// has no basis.
func (species fishSpecies) looked(name string) error {
	for _, key := range fishLookNames {
		look, given := species.Looks[key]
		if !given || !shares(look.Colour) || !share(look.Gloss) ||
			!fishBased(look.Basis, look.Source) {
			return fmt.Errorf("manufacture: the %s's look %q is missing or "+
				"is not a colour, a gloss and a basis", name, key)
		}
	}
	return nil
}

// proportioned refuses a body whose parts fall out of order along it, a
// column with no caudal vertebrae, and a fin outside the trunk.
func (species fishSpecies) proportioned(name string) error {
	at := func(key string) fishFigure { return species.Figures[key] }
	order := []string{"head", "deepest", "stalk-at", "tail-base"}
	for i := 1; i < len(order); i++ {
		if !(at(order[i-1]).Most < at(order[i]).Least) {
			return fmt.Errorf("manufacture: the %s's %s is not before its "+
				"%s", name, order[i-1], order[i])
		}
	}
	if !(at("tail-base").Most < 1) || !(at("precaudal").Most+
		fishFewestCaudal <= at("vertebrae").Least) ||
		!positive(at("length").Least) || !positive(at("depth").Least) {
		return fmt.Errorf("manufacture: the %s's length, depth, tail or "+
			"column cannot be drawn", name)
	}
	for _, fin := range fishMedianFins {
		if err := species.finned(name, fin); err != nil {
			return err
		}
	}
	return nil
}

// fishFewestCaudal is the fewest caudal vertebrae a column may have: the
// last carries the tail's own skeleton and the one before bends to it.
const fishFewestCaudal = 2 // defined: see above

// finned refuses a median fin that has rays and a base that does not lie
// on the trunk between the head and the tail stalk, front to back.
func (species fishSpecies) finned(name, fin string) error {
	at := func(key string) fishFigure { return species.Figures[fin+key] }
	if at("-spines").Most+at("-rays").Most == 0 {
		return nil
	}
	head := species.Figures["head"].Most
	stalk := species.Figures["stalk-at"].Least
	if at("-spines").Least+at("-rays").Least < webFewestRays ||
		!(head < at("-start").Least && at("-start").Most < at("-end").Least &&
			at("-end").Most < stalk) || !positive(at("-front").Least) ||
		!positive(at("-back").Least) {
		return fmt.Errorf("manufacture: the %s's %s fin is not on its "+
			"trunk, or has too few rays to be a web", name, fin)
	}
	return nil
}

// webFewestRays is the fewest rays a fin's web is stretched between, the
// skeleton package's own fewest.
const webFewestRays = 3 // defined: skeleton's webFewest

// schooled refuses a school of no fish or of more than a stand holds.
func (species fishSpecies) schooled(name string) error {
	for _, count := range species.Schools {
		if count < 1 || count > mostSchool {
			return fmt.Errorf("manufacture: the %s's school of %d is not "+
				"one to %d fish", name, count, mostSchool)
		}
	}
	return nil
}

// fishRow is a kind of fish: its species, and whether it is its bones
// alone, a fish gliding, or a fish at a phase of its stroke.
type fishRow struct {
	species string
	bones   bool
	glide   bool
}

// fishRows supplies each fish by name. A file that cannot be read stops
// the build, as the other embedded data files do.
func fishRows() map[string]maker {
	if fishError != nil {
		panic(fishError)
	}
	rows := map[string]maker{}
	for name, species := range fishRead.Species {
		rows[name] = fishRow{species: name}
		rows[name+fishGlideKind] = fishRow{species: name, glide: true}
		rows[name+fishBonesKind] = fishRow{species: name, bones: true}
		for _, count := range species.Schools {
			rows[name+fishSchoolKind+strconv.Itoa(count)] =
				fishSchoolRow{species: name, count: count}
		}
	}
	return rows
}

// fishPlan is one fish of a species before it is built: the species'
// name and row, and the seed that picks every figure the row leaves open.
type fishPlan struct {
	name    string
	species fishSpecies
	from    seed
}

// planFish is the plan of a fish of the species named, from a seed.
func planFish(name string, from seed) fishPlan {
	return fishPlan{name: name, species: fishRead.Species[name], from: from}
}

// share is the seed's draw for a name, a figure's or one of the other
// draws a fish makes.
func (plan fishPlan) share(name string) float64 {
	index := slices.Index(fishFigureNames, name)
	if index < 0 {
		index = len(fishFigureNames) + slices.Index(fishDrawNames, name)
	}
	return plan.from.draw(picking, index, 0)
}

// figure is a figure of the fish: a share of the way across its range,
// the share the seed draws for the name.
func (plan fishPlan) figure(name string) float64 {
	row := plan.species.Figures[name]
	return span{row.Least, row.Most}.pick(plan.share(name))
}

// whole is a figure that is a count: each from the least to the most is
// as likely as the next.
func (plan fishPlan) whole(name string) int {
	row := plan.species.Figures[name]
	return span{row.Least, row.Most}.count(plan.share(name))
}

// length is the fish's fork length, in metres.
func (plan fishPlan) length() float64 { return plan.figure("length") }

// made is the fish of the row from a seed: its bones at rest, or its
// body, fins and eyes posed. It refuses a seed never sown.
func (row fishRow) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	body, err := buildFish(planFish(row.species, from))
	if err != nil {
		return good{}, err
	}
	if row.bones {
		return body.bonesGood()
	}
	pose := body.stroke(body.plan.figure("amplitude"))
	if row.glide {
		pose = body.stroke(body.plan.figure("glide"))
	}
	return body.fleshGood(pose, true)
}
