package cities

import (
	"bytes"
	"embed"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	yaml "gopkg.in/yaml.v3"
)

//go:embed cultures/*.yaml
var cultureFiles embed.FS

// commonCulture is the name of the file every culture starts from: the
// elements a culture need not say, because no culture differs on them.
const commonCulture = "common"

// scope is how often a design element is drawn anew. A district is
// everything whose seeds agree but for the last digits; a row is one
// run of bays; a bay is one house, or one unit of a building; a storey
// is one floor of a bay; a dwelling is the windows of one household in a
// storey, with the balcony before them; an opening is one window or one
// door.
type scope string

// The scopes, coarsest first.
const (
	scopeDistrict scope = "district"
	scopeRow      scope = "row"
	scopeBay      scope = "bay"
	scopeStorey   scope = "storey"
	scopeDwelling scope = "dwelling"
	scopeOpening  scope = "opening"
)

// scopes are the scopes in order, coarsest first. An index in the list is
// how many of a place's counters a draw at that scope depends on.
var scopes = []scope{scopeDistrict, scopeRow, scopeBay, scopeStorey,
	scopeDwelling, scopeOpening}

// dwellingDepth is how many counters a draw at the dwelling scope
// depends on, the place's dwelling the last of them.
const dwellingDepth = 4

// draw is how an element is chosen: a number between two, a chance, or
// one of some named options by weight.
type draw string

// The ways an element is drawn.
const (
	// drawRange is a number from Least to Most, each as likely as the
	// next; whole numbers, with both ends included, if Whole.
	drawRange draw = "range"
	// drawShare is a chance, from nought to one, that the element is so.
	drawShare draw = "share"
	// drawOptions is one of the named options, by weight.
	drawOptions draw = "options"
)

// pickedKind and recalledKind are the kinds of number a culture's data
// holds: one chosen, and one remembered from a place and not looked up.
// Nothing in a culture is measured.
const (
	pickedKind   = "picked"
	recalledKind = "recalled"
)

// option is one choice of an element drawn from options: a name, its
// weight against the others, and what a look needs of it.
type option struct {
	Name   string    `yaml:"name"`
	Weight float64   `yaml:"weight"`
	Colour []float64 `yaml:"colour"`
	Gloss  float64   `yaml:"gloss"`
}

// element is one design element as a culture's data says it: its
// distribution, the scope it is drawn at, and where the numbers came
// from. The unit, kind and by are for every number the element holds.
type element struct {
	Name    string   `yaml:"element"`
	Says    string   `yaml:"says"`
	Scope   scope    `yaml:"scope"`
	Draw    draw     `yaml:"draw"`
	Unit    string   `yaml:"unit"`
	Least   float64  `yaml:"least"`
	Most    float64  `yaml:"most"`
	Whole   bool     `yaml:"whole"`
	Share   float64  `yaml:"share"`
	Options []option `yaml:"options"`
	Kind    string   `yaml:"kind"`
	By      string   `yaml:"by"`
}

// cultureFile is one data file: a culture's name, a line on it, the
// culture it starts from, if it is a variant of one, and its elements.
type cultureFile struct {
	Culture  string    `yaml:"culture"`
	Says     string    `yaml:"says"`
	From     string    `yaml:"from"`
	Elements []element `yaml:"elements"`
}

// culture is a people's way of building houses, as separable design
// elements, each with a distribution and the scope it is drawn at. It
// holds no geometry. A drawer asks it for elements by name, so a culture
// that does not say an element a design needs is refused when asked.
type culture struct {
	Name     string
	elements map[string]element
}

// cultureNames are the cultures there is data for.
func cultureNames() []string {
	entries, err := cultureFiles.ReadDir("cultures")
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		if name != commonCulture {
			names = append(names, name)
		}
	}
	return names
}

// readCultureFile is a data file by name, or why it cannot be read. A
// key that nobody knows is refused, at its line.
func readCultureFile(name string) (cultureFile, error) {
	raw, err := cultureFiles.ReadFile("cultures/" + name + ".yaml")
	if err != nil {
		return cultureFile{}, fmt.Errorf("cities: no culture named %q, "+
			"only %v", name, cultureNames())
	}
	var file cultureFile
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return cultureFile{}, fmt.Errorf("cities: %s.yaml: %w", name, err)
	}
	for _, each := range file.Elements {
		if err := each.check(); err != nil {
			return cultureFile{}, fmt.Errorf("cities: %s.yaml: %w", name,
				err)
		}
	}
	return file, nil
}

// loadCulture is the culture of a name: the common elements, over them
// those of the culture it starts from if it says one, and over those its
// own. It refuses the common file as a culture, a name there is no data
// for, a culture that starts from a variant, and an element written twice.
func loadCulture(name string) (*culture, error) {
	if name == commonCulture {
		return nil, fmt.Errorf("cities: %q is the elements every culture "+
			"starts from, not a culture", name)
	}
	own, err := readCultureFile(name)
	if err != nil {
		return nil, err
	}
	files := []string{commonCulture, name}
	if own.From != "" {
		files = []string{commonCulture, own.From, name}
	}
	culture := &culture{Name: name, elements: map[string]element{}}
	for _, from := range files {
		if err := culture.layer(from, name); err != nil {
			return nil, err
		}
	}
	return culture, nil
}

// layer lays the elements of one file over the culture's, refusing an
// element written twice in it, and a file a culture starts from that
// itself starts from another, which is a chain nobody can read.
func (culture *culture) layer(from, name string) error {
	file, err := readCultureFile(from)
	if err != nil {
		return err
	}
	if from != name && file.From != "" {
		return fmt.Errorf("cities: %s starts from %s, which starts from %s",
			name, from, file.From)
	}
	seen := map[string]bool{}
	for _, each := range file.Elements {
		if seen[each.Name] {
			return fmt.Errorf("cities: %s.yaml says %s twice", from,
				each.Name)
		}
		seen[each.Name], culture.elements[each.Name] = true, each
	}
	return nil
}

// check refuses an element with no name, a scope or draw that is
// not one there is, no unit, no line on what it is, a kind that is not
// picked or recalled, or no word on who picked or recalled it.
func (each element) check() error {
	if each.Name == "" || each.Says == "" || each.Unit == "" ||
		each.By == "" ||
		(each.Kind != pickedKind && each.Kind != recalledKind) {
		return fmt.Errorf("element %q needs a name, what it says, a unit, "+
			"kind %s or %s and who chose it", each.Name, pickedKind,
			recalledKind)
	}
	if !slices.Contains(scopes, each.Scope) {
		return fmt.Errorf("element %s has scope %q, and the scopes are %v",
			each.Name, each.Scope, scopes)
	}
	switch each.Draw {
	case drawRange:
		if each.Least > each.Most {
			return fmt.Errorf("element %s ranges from %v to %v",
				each.Name, each.Least, each.Most)
		}
	case drawShare:
		if each.Share < 0 || each.Share > 1 {
			return fmt.Errorf("element %s is a chance of %v, not from "+
				"nought to one", each.Name, each.Share)
		}
	case drawOptions:
		return each.checkOptions()
	default:
		return fmt.Errorf("element %s is drawn %q, and the ways are %v",
			each.Name, each.Draw, []draw{drawRange, drawShare, drawOptions})
	}
	return nil
}

// checkOptions refuses options of none, of no weight, or whose colour
// is not three shares of the light.
func (each element) checkOptions() error {
	if len(each.Options) == 0 {
		return fmt.Errorf("element %s has no options", each.Name)
	}
	for _, one := range each.Options {
		if !(one.Weight > 0) || one.Name == "" {
			return fmt.Errorf("element %s: option %q needs a name and a "+
				"weight above nought", each.Name, one.Name)
		}
		if len(one.Colour) != 0 && !colourShares(one.Colour) {
			return fmt.Errorf("element %s: option %s has a colour that is "+
				"not three shares of the light", each.Name, one.Name)
		}
	}
	return nil
}

// colourShares is whether a colour is three channels, each a share of
// the light from nought to one.
func colourShares(colour []float64) bool {
	const channels = 3
	if len(colour) != channels {
		return false
	}
	for _, channel := range colour {
		if !(channel >= 0 && channel <= 1) {
			return false
		}
	}
	return true
}

// place says which bay, storey and opening a draw is for. An element is
// drawn from the part of the place its scope reaches: a bay's element
// the same for every storey of the bay.
type place struct {
	Row, Bay, Storey, Dwelling, Opening int
}

// counters are the place's counters from the coarsest to the finest.
func (at place) counters() []int {
	return []int{at.Row, at.Bay, at.Storey, at.Dwelling, at.Opening}
}

// drawer draws a culture's elements for one seed, and keeps the first
// thing that went wrong. After that it draws nothing and gives nought,
// so a run of draws is checked once at the end.
type drawer struct {
	culture  *culture
	seed     Seed
	district uint64
	trouble  error
	// differed is whether the culture said, when the colours of a row
	// were last drawn, that no two neighbours share one.
	differed bool
	// era is the era of the bay being drawn, or nothing. An element is
	// looked for under its name and a hyphen and the era first, so that
	// each era says its own storeys, roofs and windows.
	era string
	// level is the detail the faces drawn are made at: the hull, the
	// quick level if it is not given, or the demo level.
	level Detail
	// homes says the windows drawn belong to households: a dwelling's
	// windows are lit, curtained and framed together, by its choices.
	homes bool
	// storeyBase is how many storeys stand under the tier being drawn, so
	// that its storeys are drawn as those that follow, and onDecks the
	// storeys of a building that stand on a deck, by bay and storey.
	storeyBase int
	onDecks    map[place]bool
}

// has is whether the culture says an element of a name, in any way.
func (drawing *drawer) has(name string) bool {
	_, known := drawing.culture.elements[name]
	return known
}

// newDrawer is a drawer of the culture of a name, for a seed.
func newDrawer(name string, seed Seed) (*drawer, error) {
	culture, err := loadCulture(name)
	if err != nil {
		return nil, err
	}
	drawing := &drawer{culture: culture, seed: seed}
	drawing.district = uint64(drawing.fixed("rows-per-district"))
	if drawing.trouble == nil && drawing.district < 1 {
		drawing.trouble = fmt.Errorf("cities: a district holds at least " +
			"one row")
	}
	return drawing, drawing.trouble
}

// found is the element of a name drawn the way given, or nothing and a
// note of why.
func (drawing *drawer) found(name string, how draw) (element, bool) {
	if drawing.trouble != nil {
		return element{}, false
	}
	if drawing.era != "" {
		own, said := drawing.culture.elements[name+"-"+drawing.era]
		if said && own.Draw == how {
			return own, true
		}
	}
	each, known := drawing.culture.elements[name]
	if !known || each.Draw != how {
		drawing.trouble = fmt.Errorf("cities: culture %s has no element "+
			"%s drawn as a %s", drawing.culture.Name, name, how)
		return element{}, false
	}
	return each, true
}

// share is the next share from nought up to but not one, from the stream
// of an element at a place. Draws are the same for the same seed, element
// and place, and the same across the places a scope does not tell apart.
//
// A draw can be made at a coarser scope than the element's own, and
// with a salt that makes its stream another's: a district draws the
// palette it uses from the palette's options, once per colour.
func (drawing *drawer) share(each element, scope scope, salt string,
	at place) float64 {
	hash := fnv.New64a()
	hash.Write([]byte(each.Name + salt))
	depth := slices.Index(scopes, scope)
	for index, counter := range at.counters()[:depth] {
		// A draw that is not a dwelling's, at the first dwelling, is
		// hashed as it was before dwellings were a scope.
		if index == dwellingDepth-1 && counter == 0 &&
			scope != scopeDwelling {
			continue
		}
		fmt.Fprint(hash, ",", counter)
	}
	seed := uint64(drawing.seed)
	if scope == scopeDistrict {
		seed /= max(drawing.district, 1)
	}
	return rand.New(rand.NewPCG(seed, hash.Sum64())).Float64()
}

// fixed is the value of an element that is one number, which is a range
// whose ends are the same. It is how a culture says a size that does not
// vary.
func (drawing *drawer) fixed(name string) float64 {
	each, found := drawing.found(name, drawRange)
	if found && each.Least != each.Most {
		drawing.trouble = fmt.Errorf("cities: %s is asked for as a fixed "+
			"number, and it ranges from %v to %v", name, each.Least,
			each.Most)
	}
	return each.Least
}

// number is an element drawn as a range, at a place.
func (drawing *drawer) number(name string, at place) float64 {
	each, found := drawing.found(name, drawRange)
	if !found {
		return 0
	}
	return each.Least + (each.Most-each.Least)*
		drawing.share(each, each.Scope, "", at)
}

// count is an element drawn as a range of whole numbers, at a place,
// each from the least to the most as likely as the next.
func (drawing *drawer) count(name string, at place) int {
	each, found := drawing.found(name, drawRange)
	if !found {
		return 0
	}
	if !each.Whole || each.Least != math.Trunc(each.Least) ||
		each.Most != math.Trunc(each.Most) {
		drawing.trouble = fmt.Errorf("cities: %s is asked for as a count, "+
			"and is not a range of whole numbers", name)
		return 0
	}
	wide := each.Most - each.Least + 1
	return int(each.Least + math.Min(math.Floor(
		drawing.share(each, each.Scope, "", at)*wide), wide-1))
}

// chance is whether an element drawn as a share is so, at a place.
func (drawing *drawer) chance(name string, at place) bool {
	each, found := drawing.found(name, drawShare)
	return found && drawing.share(each, each.Scope, "", at) < each.Share
}

// options are the options of an element, in the order written.
func (drawing *drawer) options(name string) []option {
	each, _ := drawing.found(name, drawOptions)
	return each.Options
}

// pick is one of the options given, by weight, from the stream of the
// element of a name at a place. An option named in skip is not chosen
// unless every option is skipped.
func (drawing *drawer) pick(name string, from []option, at place,
	skip ...string) option {
	return drawing.choose(name, "", "", from, at, skip)
}

// choose is pick at a scope and with a salt that are not the element's,
// where the scope is given. It gives nothing if there are no options.
func (drawing *drawer) choose(name string, scope scope, salt string,
	from []option, at place, skip []string) option {
	each, found := drawing.found(name, drawOptions)
	if scope == "" {
		scope = each.Scope
	}
	if !found || len(from) == 0 {
		return option{}
	}
	var total float64
	for _, one := range from {
		if !slices.Contains(skip, one.Name) {
			total += one.Weight
		}
	}
	if total == 0 {
		skip = nil
		total = weightOf(from)
	}
	left := drawing.share(each, scope, salt, at) * total
	for _, one := range from {
		if slices.Contains(skip, one.Name) {
			continue
		}
		if left -= one.Weight; left < 0 {
			return one
		}
	}
	return from[len(from)-1]
}

// weightOf is the weight of all the options together.
func weightOf(from []option) float64 {
	var total float64
	for _, one := range from {
		total += one.Weight
	}
	return total
}

// subset is that many of the element's options, drawn by weight without
// putting one back, at a place. It is how a district uses some of a
// palette and not all.
func (drawing *drawer) subset(name string, size int, at place) []option {
	left := slices.Clone(drawing.options(name))
	var chosen []option
	for len(chosen) < size && len(left) > 0 {
		one := drawing.choose(name, scopeDistrict,
			fmt.Sprint(len(chosen)), left, at, nil)
		chosen = append(chosen, one)
		left = slices.DeleteFunc(left, func(other option) bool {
			return other.Name == one.Name
		})
	}
	return chosen
}

// namesOf are the names of some options.
func namesOf(options []option) []string {
	names := make([]string, len(options))
	for index, one := range options {
		names[index] = one.Name
	}
	return names
}

// colourOf is an option's colour, as the light it reflects.
func colourOf(one option) maths.Light {
	return maths.RGB(one.Colour[0], one.Colour[1], one.Colour[2])
}

// paint is a material of a role in the colour and gloss of an option,
// named so that every ask for the same gets the same material.
func paint(role string, one option) *thing.Material {
	return &thing.Material{Name: fmt.Sprintf("cities %s %s", role, one.Name),
		Colour: colourOf(one), Gloss: one.Gloss}
}

// wall is the material of a wall: the colour of a palette option, as
// the cladding's tint takes it, in the cladding's gloss.
func wall(palette, cladding option) *thing.Material {
	tint := cladding.Colour
	return &thing.Material{Name: fmt.Sprintf("cities wall %s %s",
		palette.Name, cladding.Name),
		Colour: maths.RGB(palette.Colour[0]*tint[0],
			palette.Colour[1]*tint[1], palette.Colour[2]*tint[2]),
		Gloss: cladding.Gloss}
}

// heat is the colour temperature of the light in a window at a place,
// held to the culture's steps.
func (drawing *drawer) heat(at place) float64 {
	if drawing.homes {
		at.Opening = 0
	}
	step := drawing.fixed("light-heat-step")
	return math.Round(drawing.number("light-heat", at)/step) * step
}

// glass is the glass of a window at a place: dark, or alight. The share
// of windows alight, and the colour temperature of each one's light,
// are the culture's, and the temperature is held to steps so that the
// windows come in a few materials and not a thousand.
func (drawing *drawer) glass(at place) (*thing.Material, error) {
	dark, err := material(roleGlazing)
	if err != nil {
		return nil, err
	}
	alight := drawing.chance("lit-share", at)
	if drawing.homes {
		alight = drawing.chance("household-lit", at)
	}
	if !alight {
		return dark, drawing.trouble
	}
	return drawing.litGlass(dark, at), drawing.trouble
}

// litGlass is glass alight at a place: the dark glass given, glowing at
// the heat the culture draws for the place.
func (drawing *drawer) litGlass(dark *thing.Material,
	at place) *thing.Material {
	heat := drawing.heat(at)
	lit := *dark
	lit.Name = fmt.Sprintf("cities lit window %.0fK", heat)
	lit.Glow = swatches.Light(swatch.Planckian(heat)).Scale(
		drawing.fixed("window-glow"))
	return &lit
}
