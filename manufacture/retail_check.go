package manufacture

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/janearc/slayer/maths"
)

// retailMostStreams is the most props, groups of stickers or palettes a
// kit or a unit may list: the seed draws each from a stream of its own, a
// kit apart by this many.
const retailMostStreams = 16

// retailLeastSides is the fewest sides a prism has: a triangle.
const retailLeastSides = 3

// retailFit is how much a part may stand past the box a unit is drawn in,
// in metres: nothing.
const retailFit = 1e-9

// retailMoves are the ways a part, a light or a module may move with its
// unit's width, and none.
var retailMoves = []string{"", retailRight, retailCentre, retailFill}

// check is nil if the data is whole: every look a unit names is there,
// every list is the length it should be, every part lies in its unit's
// bays and storey at the least and the most width the unit is drawn at,
// there are enough unit types to draw a run in which no two of a kind
// stand together, and a podium is deep enough for every room.
func (book retailBook) check() error {
	shell := book.Shell
	if book.Run.Units < 1 || !(shell.Floor > 0 && shell.Slab > 0 &&
		shell.Wall > 0 && shell.Partition > 0) {
		return errors.New("a run needs a unit, and a shell, a thickness")
	}
	if len(book.Units) < retailRangeNumbers {
		return errors.New("a run needs two types of unit to keep like " +
			"from standing beside like")
	}
	for name, data := range book.Materials {
		if err := data.check(); err != nil {
			return fmt.Errorf("material %s: %w", name, err)
		}
	}
	if err := book.checkPalettes(); err != nil {
		return err
	}
	if err := book.checkParts(); err != nil {
		return err
	}
	var problems []error
	for _, name := range book.unitNames() {
		if err := book.checkUnit(book.Units[name]); err != nil {
			problems = append(problems, fmt.Errorf("unit %s: %w", name, err))
		}
	}
	return errors.Join(append(problems, book.checkGrounds())...)
}

// checkParts is nil if the parts of every module and every prop are in
// order.
func (book retailBook) checkParts() error {
	for name, module := range book.Modules {
		if module.Width < 0 {
			return fmt.Errorf("module %s: a width is not below nought", name)
		}
		for _, part := range module.Parts {
			if err := book.checkPart(part); err != nil {
				return fmt.Errorf("module %s: %w", name, err)
			}
		}
	}
	for name, prop := range book.Props {
		for _, part := range prop.Parts {
			if err := book.checkPart(part); err != nil {
				return fmt.Errorf("prop %s: %w", name, err)
			}
		}
	}
	return nil
}

// checkPalettes is nil if every palette has materials in it, and no
// more palettes than a unit has streams for.
func (book retailBook) checkPalettes() error {
	if len(book.Palettes) > retailMostStreams {
		return fmt.Errorf("no more than %d palettes", retailMostStreams)
	}
	for name, list := range book.Palettes {
		if len(list) == 0 {
			return fmt.Errorf("palette %s has nothing in it", name)
		}
		for _, one := range list {
			if _, known := book.Materials[one]; !known {
				return fmt.Errorf("palette %s has no material %q", name, one)
			}
		}
	}
	return nil
}

// checkGrounds is nil if a building may be given a podium: it has kinds
// to be, by weight, shops that are deep enough for their rooms, a pool
// to draw units from, and a plain wall to fill with.
func (book retailBook) checkGrounds() error {
	grounds := book.Grounds
	if len(grounds.Buildings) == 0 || grounds.Pool < 1 ||
		grounds.Margin < 0 {
		return errors.New("a podium needs buildings, a pool and a margin")
	}
	for name, weight := range grounds.Buildings {
		if !(weight > 0) {
			return fmt.Errorf("building %s has no weight", name)
		}
	}
	for name, unit := range book.Units {
		if unit.Depth > grounds.Podium {
			return fmt.Errorf("unit %s is deeper than the podium", name)
		}
	}
	if _, known := book.Materials[grounds.Filler]; !known {
		return fmt.Errorf("no material %q to fill a podium", grounds.Filler)
	}
	return nil
}

// check is nil if a look has a colour and, if it glows, a glow.
func (data retailMaterial) check() error {
	if len(data.Colour) != retailColourNumbers {
		return errors.New("a colour is three numbers")
	}
	if len(data.Glow) != 0 && len(data.Glow) != retailColourNumbers {
		return errors.New("a glow is three numbers or none")
	}
	if data.Gloss < 0 || data.Gloss > 1 || data.Through < 0 ||
		data.Through > 1 || data.Index < 0 {
		return errors.New("a gloss and a through are from nought to one, " +
			"and an index is not below nought")
	}
	return nil
}

// hasMaterial is whether a name is a material of the data, or a palette
// of it.
func (book retailBook) hasMaterial(name string) bool {
	if palette, named := strings.CutPrefix(name, retailPalettePrefix); named {
		_, known := book.Palettes[palette]
		return known
	}
	_, known := book.Materials[name]
	return known
}

// checkPart is nil if a part is one box, one prism or one slab, of a look
// there is, from a level there is, moving a way there is.
func (book retailBook) checkPart(part retailPart) error {
	if !book.hasMaterial(part.Material) {
		return fmt.Errorf("part %q has no material %q", part.Says,
			part.Material)
	}
	if retailLevel(part.From) < 0 {
		return fmt.Errorf("part %q: no level %q", part.Says, part.From)
	}
	if err := part.checkShape(); err != nil {
		return err
	}
	if part.Step != nil && len(part.Step) != retailPointNumbers {
		return fmt.Errorf("part %q: a step is three numbers", part.Says)
	}
	if !slices.Contains([]string{"", "x", "y", "z"}, part.Axis) {
		return fmt.Errorf("part %q: axis is x, y or z", part.Says)
	}
	if part.Count < 0 || (part.Sides != 0 && part.Sides < retailLeastSides) {
		return fmt.Errorf("part %q: a count is not below nought, and a "+
			"prism has no sides or three or more", part.Says)
	}
	if !slices.Contains(retailMoves, part.Along) {
		return fmt.Errorf("part %q: along is %v", part.Says, retailMoves)
	}
	return nil
}

// checkShape is nil if a part is one shape, the right length of list,
// and, if it is a slab, thick.
func (part retailPart) checkShape() error {
	shapes := 0
	for _, one := range []struct {
		list []float64
		size int
	}{
		{part.Box, retailBoxNumbers}, {part.Prism, retailPrismNumbers},
		{part.Slab, retailSlabNumbers}} {
		if one.list == nil {
			continue
		}
		shapes++
		if len(one.list) != one.size {
			return fmt.Errorf("part %q: a list of %d numbers, not %d",
				part.Says, len(one.list), one.size)
		}
	}
	if shapes != 1 {
		return fmt.Errorf("part %q is one box, one prism or one slab",
			part.Says)
	}
	if part.Slab != nil && !(part.Slab[retailSlabNumbers-1] > 0) {
		return fmt.Errorf("part %q: a slab has a thickness", part.Says)
	}
	return nil
}

// mostBays is the most bays a unit is drawn at.
func (unit retailUnit) mostBays() int {
	return max(unit.Bays, unit.BaysMost)
}

// checkUnit is nil if a unit's shell, kit and options are in order.
func (book retailBook) checkUnit(unit retailUnit) error {
	if unit.Bays < 1 || !(unit.Weight > 0) || unit.Depth < 0 ||
		(unit.BaysMost != 0 && unit.BaysMost < unit.Bays) {
		return errors.New("bays, weight and depth are not below one, " +
			"nought and nought, and most bays is not below bays")
	}
	if err := book.checkShell(unit); err != nil {
		return err
	}
	pitch := cityValue("bay-pitch")
	most := float64(unit.mostBays()-unit.Bays) * pitch
	if err := book.checkKit(unit, unit.Kit, 0, most); err != nil {
		return err
	}
	if len(unit.Options) > retailMostOptions {
		return fmt.Errorf("no more than %d options", retailMostOptions)
	}
	for _, option := range unit.Options {
		least := float64(max(option.Wide-unit.Bays, 0)) * pitch
		if err := book.checkOption(unit, option, least, most); err != nil {
			return err
		}
	}
	return nil
}

// checkOption is nil if an option has a chance, and variants of a weight
// each, whose kits hold at the widths the option is offered at.
func (book retailBook) checkOption(unit retailUnit, option retailOption,
	least, most float64) error {
	if option.Chance < 0 || option.Chance > 1 || option.Wide < 0 ||
		len(option.Variants) == 0 || least > most {
		return fmt.Errorf("option %q is ill drawn", option.Says)
	}
	for _, variant := range option.Variants {
		if !(variant.Weight > 0) {
			return fmt.Errorf("option %q: a variant has a weight", option.Says)
		}
		if err := book.checkKit(unit, variant.Kit, least, most); err != nil {
			return fmt.Errorf("option %q: %w", option.Says, err)
		}
	}
	return nil
}

// checkKit is nil if a kit's parts and modules lie in the unit at the
// least and the most extra width it is dressed at, and its lights,
// stickers and props are in order.
func (book retailBook) checkKit(unit retailUnit, kit retailKit, least,
	most float64) error {
	var problems []error
	for _, grown := range []float64{least, most} {
		for _, part := range kit.Parts {
			if err := book.checkPart(part); err != nil {
				problems = append(problems, err)
				continue
			}
			problems = append(problems, unit.holds(part.slid(grown),
				maths.Vec{}, grown))
		}
		for _, use := range kit.Uses {
			problems = append(problems, book.checkUse(unit, use, grown))
		}
	}
	return errors.Join(append(problems, book.checkLists(kit))...)
}

// checkUse is nil if a use names a module there is, a place, a way to move
// and a level, and every part of the module lies in the unit.
func (book retailBook) checkUse(unit retailUnit, use retailUse,
	extra float64) error {
	module, known := book.Modules[use.Module]
	if !known || (use.At != nil && len(use.At) != retailPointNumbers) ||
		!slices.Contains(retailMoves, use.Along) || use.Span < 0 ||
		retailLevel(use.From) < 0 {
		return fmt.Errorf("use of module %q is unknown or ill drawn",
			use.Module)
	}
	grow := 0.0
	if use.Span > 0 {
		grow = use.Span - module.Width
	}
	if use.Along == retailFill {
		grow += extra
	}
	at := retailPoint(use.At)
	at.X += retailSlide(use.Along, extra)
	for _, part := range module.Parts {
		if err := unit.holds(part.slid(grow), at, extra); err != nil {
			return fmt.Errorf("module %q: %w", use.Module, err)
		}
	}
	return nil
}

// checkShell is nil if a unit's opening lies in its front and the looks
// of its walls are there.
func (book retailBook) checkShell(unit retailUnit) error {
	names := []string{unit.Wall}
	if len(unit.Opening) != 0 {
		names = append(names, unit.Floor, unit.Ceiling, unit.Inner, unit.Sides)
		wide, tall := unit.size()
		open := unit.Opening
		if len(open) != retailRectNumbers || !(open[0] >= 0 &&
			open[0] < open[2] && open[2] <= wide && open[1] >= 0 &&
			open[1] < open[3] && open[3] <= tall) {
			return errors.New("an opening is a rectangle in the front")
		}
		least := book.Shell.Wall + 2*book.Shell.Partition
		if unit.Depth <= least {
			return errors.New("an opening needs a room behind it")
		}
	}
	for _, name := range names {
		if !book.hasMaterial(name) {
			return fmt.Errorf("no material %q for the shell", name)
		}
	}
	return nil
}

// checkLists is nil if a kit's lights, scatters, props and sign are in
// order.
func (book retailBook) checkLists(kit retailKit) error {
	for _, one := range kit.Lights {
		if len(one.At) != retailPointNumbers || !(one.Heat > 0 &&
			one.Luminance > 0) || retailLevel(one.From) < 0 ||
			!slices.Contains(retailMoves, one.Along) {
			return errors.New("a light has a place, a heat, a luminance " +
				"and a level, and moves a way there is")
		}
	}
	if len(kit.Scatter) > retailMostStreams ||
		len(kit.Props) > retailMostStreams {
		return fmt.Errorf("no more than %d scatters or props",
			retailMostStreams)
	}
	for _, group := range kit.Scatter {
		if err := book.checkScatter(group); err != nil {
			return err
		}
	}
	for _, use := range kit.Props {
		_, known := book.Props[use.Prop]
		if !known || len(use.X) != retailRangeNumbers ||
			len(use.Z) != retailRangeNumbers || retailLevel(use.From) < 0 ||
			use.Chance < 0 || use.Chance > 1 {
			return fmt.Errorf("prop %q is unknown or ill drawn", use.Prop)
		}
	}
	return nil
}

// checkScatter is nil if a group of stickers has a region, a size and
// looks, all there.
func (book retailBook) checkScatter(group retailScatter) error {
	if len(group.Region) != retailRectNumbers ||
		len(group.Size) != retailRangeNumbers || group.Count < 0 ||
		len(group.Materials) == 0 || retailLevel(group.From) < 0 {
		return fmt.Errorf("scatter %q is ill drawn", group.Says)
	}
	for _, name := range group.Materials {
		if !book.hasMaterial(name) {
			return fmt.Errorf("scatter %q has no material %q", group.Says, name)
		}
	}
	return nil
}

// size is a unit's least width and its height: its bays, and a storey.
func (unit retailUnit) size() (float64, float64) {
	return float64(unit.Bays) * cityValue("bay-pitch"), cityValue("storey-max")
}

// holds is nil if every step of a part, put at an origin, lies in the
// unit's width, which is as much wider than the least as the extra says,
// and its storey, and in front of its back wall.
func (unit retailUnit) holds(part retailPart, origin maths.Vec,
	extra float64) error {
	wide, tall := unit.size()
	wide += extra
	for _, corner := range part.corners() {
		low := corner.Add(origin)
		if low.X < -retailFit || low.X > wide+retailFit || low.Y < -retailFit ||
			low.Y > tall+retailFit || low.Z < -unit.Depth-retailFit {
			return fmt.Errorf("part %q stands outside its unit", part.Says)
		}
	}
	return nil
}
