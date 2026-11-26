package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// TowerSpec is what a set says of a tower recipe: how irregular its
// towers are, or that each draws its own, and the features it pins. See
// cities.Tower for what irregularity means. A zero field is not pinned.
type TowerSpec struct {
	// Irregularity is from nought to one; Drawn takes it from the
	// skyline's distribution instead.
	Irregularity float64
	Drawn        bool
	// Plan is the pinned plan, one of TowerPlans.
	Plan string
	// Shafts is the pinned number of shafts.
	Shafts int
	// Taper, Twist, Setbacks, Spire and LitCrown pin a feature on or off
	// for every part of the tower that could have it.
	Taper, Twist, Setbacks, Spire, LitCrown *bool
	// Stack and Nest pin a stack of flared modules and a slim core carried
	// on over the body, on or off.
	Stack, Nest *bool
	// Span is the pinned join between shafts, one of TowerSpans.
	Span string
	// Room is the width of the block the tower stands in, in metres, or
	// nought for no limit.
	Room float64
	// Skin is the pinned skin, one of TowerSkins; Style how granite is
	// windowed, punched or ribbon; Colour the name of the stone, tint or
	// pastel; Use office or home for every segment.
	Skin, Style, Colour, Use string
	// Cyber and Planted pin the cyberpunk layer and the planting on or
	// off, and Spiral a tower of blocks that turn as they rise.
	Cyber, Planted, Spiral *bool
	// Gargoyles pins the gargoyles under the tower's ledges on or off.
	Gargoyles *bool
	// Signs pins the signs a tower carries on, glowing words on its roofs
	// and walls; a tower is given none unless it is pinned on.
	Signs *bool
	// Lettering is the script of the signs, the trades each must carry,
	// and the files of the fonts; see WithScript. It is nothing for Latin
	// capitals of the trades the culture draws.
	Lettering cities.Lettering
	// Retail pins shops at the foot of the tower on or off: a podium of
	// units from the retail data on each wall that fronts open ground.
	Retail *bool
	// Storeys and Segments pin how many storeys a shaft has and how many
	// segments it is cut into; Width is the width of the plan at its
	// foot in metres. Each is nought if not pinned.
	Storeys, Segments int
	Width             float64
	// TreeProfiles, when supplied, is the complete checked seasonal tree book.
	TreeProfiles *PlantProfiles
}

// TowerSkins are the skins a set may pin a tower to.
func TowerSkins() []string {
	var names []string
	for _, skin := range cities.SkinNames() {
		names = append(names, string(skin))
	}
	return names
}

// TowerUses are the uses a set may pin the segments of a tower to.
func TowerUses() []string {
	var names []string
	for _, use := range cities.UseNames() {
		names = append(names, string(use))
	}
	return names
}

// TowerPlans are the plans a set may pin a tower to.
func TowerPlans() []string {
	var names []string
	for _, plan := range cities.PlanNames() {
		names = append(names, string(plan))
	}
	return names
}

// TowerSpans are the joins a set may pin between shafts.
func TowerSpans() []string {
	return []string{string(cities.SpanNone), string(cities.SpanSkybridge),
		string(cities.SpanDeck)}
}

// towerRow is the kind tower: a tower drawn by package cities from the
// seed, at the irregularity and with the pins a recipe gives it, and made
// a good here, which is where a material becomes a slot. Without a recipe
// it is a plain block.
type towerRow struct{ spec TowerSpec }

// Tower is the kind tower with a recipe's irregularity and pins. It
// draws a tower once to check the pins, and refuses those it cannot keep.
func Tower(spec TowerSpec) (Kind, error) {
	row := towerRow{spec: spec}
	tower, err := row.drawn(sow(0))
	if err != nil {
		return Kind{}, err
	}
	if spec.TreeProfiles != nil {
		for _, species := range tower.TreeSpecies() {
			if _, err := seasonalTree(spec.TreeProfiles, species); err != nil {
				return Kind{}, err
			}
		}
	}
	return Kind{name: "tower", how: row}, nil
}

// drawn is the tower of a seed, before it is made a good, with a storey
// at its foot left open for shops if its recipe pins retail on and it is a
// tower that can have them (see towerfeet.go).
func (row towerRow) drawn(from seed) (*cities.Tower, error) {
	spec := row.spec
	tower, err := row.grown(from)
	if err != nil || spec.Retail == nil || !*spec.Retail {
		return tower, err
	}
	_, err = tower.Footed(cityValue("storey-max"))
	return tower, err
}

// grown is the tower of a seed as its pins draw it, and nothing more.
func (row towerRow) grown(from seed) (*cities.Tower, error) {
	spec := row.spec
	return cities.NewTower(cities.Seed(from.number),
		cities.Irregularity{Value: spec.Irregularity, Drawn: spec.Drawn},
		cities.Pins{Plan: cities.Plan(spec.Plan), Shafts: spec.Shafts,
			Taper: spec.Taper, Twist: spec.Twist, Setbacks: spec.Setbacks,
			Spire: spec.Spire, LitCrown: spec.LitCrown, Stack: spec.Stack,
			Nest: spec.Nest, Room: spec.Room,
			Span: cities.SpanKind(spec.Span), Skin: cities.Skin(spec.Skin),
			Style: spec.Style, Colour: spec.Colour,
			Use: cities.Use(spec.Use), Cyber: spec.Cyber,
			Planted: spec.Planted, Spiral: spec.Spiral,
			Storeys: spec.Storeys, Segments: spec.Segments,
			Width: spec.Width})
}

// made is the tower at the quick detail level.
func (row towerRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the tower at the named detail level, at time nought, which
// is where a lit crown's band starts. It returns errGood when the seed is
// not made.
func (row towerRow) madeAt(from seed, detail string) (good, error) {
	return row.madeLevelled(from, constantPass(detail), maths.Subspace{})
}

// madeLevelled is the tower drawn by a pass, standing at a place in the
// world the pass sees: each segment, or each band of its storeys, at the
// level the pass gives it, and its trees at the pass's finest.
func (row towerRow) madeLevelled(from seed, pass cities.Pass,
	place maths.Subspace) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	tower, err := row.drawn(from)
	if err != nil {
		return good{}, err
	}
	design, err := tower.Design(pass.Finest, 0)
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Levelled(pass, place)
	if err != nil {
		return good{}, err
	}
	made, err := goodOfRendered(drawn)
	if err != nil {
		return good{}, err
	}
	made, err = row.gargoyled(made, tower, from, pass, place)
	if err != nil {
		return good{}, err
	}
	made, err = signed(made, row.spec, tower, from, string(pass.Finest))
	if err != nil {
		return good{}, err
	}
	made, err = retailed(made, row.spec, tower, from, string(pass.Finest))
	if err != nil {
		return good{}, err
	}
	return treed(made, tower.Trees(), row.spec.TreeProfiles, func(tree cities.Tree) string {
		return treeLevel(pass, place, tree)
	})
}

// treeLevel is the detail a tower's tree is grown at: the demo level when
// the pass's finest is the demo level and the pass gives the tree that,
// where it stands, and the detail in clumps otherwise.
func treeLevel(pass cities.Pass, place maths.Subspace,
	tree cities.Tree) string {
	at := maths.Subspace{Origin: place.Out(tree.At)}
	if pass.Finest == "demo" && pass.Leveller(at, tree.Height) == "demo" {
		return "demo"
	}
	return treeDetail
}

// treeSalt is mixed into the seed of every tree a tower holds, so that its
// trees are not the seeds the catalogue's own plants are grown from.
const treeSalt = 0x544f574552545245

// treeDetail is the detail a planted tower's trees are grown at, short of
// the demo level: foliage in clumps, because a tower holds hundreds of
// trees and the scene keeps only so many faces.
const treeDetail = "clumps"

// treed is the good with the trees of a planted tower grown on it: each
// by the kind the tower names in the catalogue, from the seed it names,
// made as tall as the tower asks and stood where it says, at the detail
// it is given. A tree of a kind, seed and detail is grown once and placed
// wherever it is wanted.
func treed(made good, trees []cities.Tree, profiles *PlantProfiles,
	detailOf func(cities.Tree) string) (good, error) {
	grown := map[string]good{}
	all := []good{made}
	for _, tree := range trees {
		detail := detailOf(tree)
		key := fmt.Sprintf("%s %d %s", tree.Kind, tree.Seed, detail)
		specimen, known := grown[key]
		if !known {
			row, err := seasonalTree(profiles, tree.Kind)
			if err != nil {
				return good{}, err
			}
			specimen, err = MakeWithComplexity(row, sow(treeSalt^tree.Seed),
				detail)
			if err != nil {
				return good{}, err
			}
			grown[key] = specimen
		}
		high := specimen.Bounds().Max.Y - specimen.Bounds().Min.Y
		placed, err := specimen.placed(maths.NewSubspace(tree.At,
			maths.Identity, tree.Height/high))
		if err != nil {
			return good{}, err
		}
		all = append(all, placed)
	}
	if len(all) == 1 {
		return made, nil
	}
	return gather(all...)
}
