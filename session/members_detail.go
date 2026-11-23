package session

import (
	"fmt"
	"slices"
	"strings"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/session/placement"
)

// Where scrooge is asked for, each thing and each member of a stand is
// made at the complexity its span in the picture calls for, and no finer
// than the set names. A span is worked out from the extent of the kind,
// stood where it stands, as the set's camera sees it, before anything of
// it is made.

// The ladder of spans, in pixels across the picture, corner to corner of
// the box that holds a thing. They are picked for a start, by what each
// representation shows: under hullBelow a thing is a speck and its crown
// as one box shows as much; under clumpsBelow its leaves are far under a
// pixel and a clump of them shows as much as each; under quickBelow its
// leaves are a pixel or two.
//
// A thing of more pixels is made at the
// complexity the set gives it. They are to be measured against a draw
// made without scrooge, and moved by what that shows.
const (
	// hullBelow is the span under which a thing is made as a hull.
	hullBelow = 4.0
	// clumpsBelow is the span under which a thing is made as clumps.
	clumpsBelow = 24.0
	// quickBelow is the span under which a thing is made quick.
	quickBelow = 160.0
)

// pixelRung is a complexity and the span in pixels under which a thing is
// made at it.
type pixelRung struct {
	under float64
	level string
}

// pixelLadder is the rungs of spans, coarse to fine.
var pixelLadder = []pixelRung{{hullBelow, "hull"},
	{clumpsBelow, "clumps"}, {quickBelow, "quick"}}

// rungAt is the complexity for a thing of a span in pixels: the ladder's,
// never finer than the limit, which is the set's, and, where the thing's
// kind lacks the rung, the next finer one it has up to the limit. The
// limit must be one of the manufactory's complexities.
func rungAt(pixels float64, has []string, limit string) string {
	order := manufacture.Complexities()
	top := slices.Index(order, limit)
	want := top
	for _, rung := range pixelLadder {
		if pixels < rung.under {
			want = min(slices.Index(order, rung.level), top)
			break
		}
	}
	return finerRung(order, has, want, top)
}

// coarsestRung is the coarsest complexity a kind has, up to the limit: the
// rung of a thing out of the picture's view.
func coarsestRung(has []string, limit string) string {
	order := manufacture.Complexities()
	return finerRung(order, has, 0, slices.Index(order, limit))
}

// finerRung is the first of the complexities from one place in their
// order up to another that a kind has, or the one it is held to if it has
// none.
func finerRung(order, has []string, from, top int) string {
	for index := from; index <= top; index++ {
		if slices.Contains(has, order[index]) {
			return order[index]
		}
	}
	return order[top]
}

// memberKey is a design of a kind or recipe: what a thing's extent and
// rungs are known by.
type memberKey struct {
	kind, recipe string
	seed         uint64
}

// memberFact is what is known of a design without making it: the
// complexities its kind makes, and its extent in its own frame where the
// kind can say it.
type memberFact struct {
	has    []string
	extent maths.Box
	said   bool
}

// rungTally is how many things and members were given each complexity,
// and how many of them stood out of the picture's view.
type rungTally struct {
	made   map[string]int
	unseen int
}

// factsOf is what is known of a design, found once and kept. A kind that
// cannot say its extent is marked so, and its things are made as the set
// says.
func (stage *staging) factsOf(kind, recipe string,
	seed uint64) (memberFact, error) {
	key := memberKey{kind, recipe, seed}
	if known, found := stage.facts[key]; found {
		return known, nil
	}
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return memberFact{}, err
	}
	fact := memberFact{has: manufacture.RungsOf(*row)}
	sown := manufacture.Sow(placement.Seed(stage.given.seed, int(seed)))
	extent, err := manufacture.ExtentOf(*row, sown)
	fact.extent, fact.said = extent, err == nil
	if stage.facts == nil {
		stage.facts = map[memberKey]memberFact{}
	}
	stage.facts[key] = fact
	return fact, nil
}

// unscaled is whether a thing of a name is made at the size its kind says:
// the named giants, close trees and rigs are grown or magnified as they
// are stocked, so their extent is not their kind's.
func (stage *staging) unscaled(name string) bool {
	taller, branched := stage.growth(name)
	return taller == 1 && !branched && !strings.HasPrefix(name, "rig ")
}

// memberDetail is the complexity a thing, or a member of a stand, is made
// at, standing at a place: the set's where scrooge is not asked for, where
// the set's is not one of the manufactory's, where the thing burns, or
// where its kind cannot say its extent; otherwise the rung its span in
// pixels gives it, and for a thing out of the picture's view the coarsest
// its kind has, since it may still cast a shadow into the picture.
//
// This is the one place the choice is made, for things and for every stand
// path that stands a thing through putRecipeFacing.
//
// A scrooge.Node whose Detail gave a thing's next finer rung would carry
// this as a promise. Its Detail gives the parts of a thing, not the same
// thing finer, and its Aggregate has no span in pixels, so the choice is
// made here instead.
func (stage *staging) memberDetail(kind, recipe, detail string, seed uint64,
	name string, at maths.Subspace, burning bool) (string, error) {
	chosen, err := stage.rungChosen(kind, recipe, detail, seed, name, at,
		burning)
	if err != nil {
		return "", err
	}
	if chosen.counted {
		stage.counted(chosen.rung, chosen.unseen)
	}
	return chosen.rung, nil
}

// rungChoice is the complexity a thing is made at, and what the tally of
// complexities is told of it: counted where scrooge judged the thing, and
// unseen where it stood out of the picture's view.
type rungChoice struct {
	rung    string
	counted bool
	unseen  bool
}

// rungChosen is memberDetail's choice without its count, so a thing's
// complexity can be known ahead of the thing without the thing being
// counted twice. It fills the camera and the facts the stage keeps, which
// come out the same whenever they are first asked for.
func (stage *staging) rungChosen(kind, recipe, detail string, seed uint64,
	name string, at maths.Subspace, burning bool) (rungChoice, error) {
	if !scrooge.Asked() || burning ||
		!slices.Contains(manufacture.Complexities(), detail) ||
		!stage.unscaled(name) {
		return rungChoice{rung: detail}, nil
	}
	eye, err := stage.seen()
	if err != nil {
		return rungChoice{}, err
	}
	fact, err := stage.factsOf(kind, recipe, seed)
	if err != nil {
		return rungChoice{}, err
	}
	if !fact.said {
		return rungChoice{rung: detail, counted: true}, nil
	}
	seen := eye.Cover(placedBox(fact.extent, at))
	rung := coarsestRung(fact.has, detail)
	if seen.Visible {
		rung = rungAt(seen.Pixels, fact.has, detail)
	}
	return rungChoice{rung: rung, counted: true, unseen: !seen.Visible}, nil
}

// counted counts a thing made at a complexity, and whether it was out of
// the view.
func (stage *staging) counted(rung string, unseen bool) {
	if stage.rungs.made == nil {
		stage.rungs.made = map[string]int{}
	}
	stage.rungs.made[rung]++
	if unseen {
		stage.rungs.unseen++
	}
}

// rungNote says, among the session's notes, how many things and members
// were made at each complexity. It says nothing where none were counted.
func (stage *staging) rungNote() {
	if len(stage.rungs.made) == 0 {
		return
	}
	var parts []string
	for _, rung := range manufacture.Complexities() {
		parts = append(parts, fmt.Sprintf("%s %d", rung,
			stage.rungs.made[rung]))
	}
	stage.given.cached = append(stage.given.cached, cacheNote{told: true,
		what: fmt.Sprintf("scrooge: things and members made at %s, of "+
			"which %d out of view", strings.Join(parts, ", "),
			stage.rungs.unseen)})
}
