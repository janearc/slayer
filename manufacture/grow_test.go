package manufacture

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/geom"
	"github.com/janearc/slayer/maths"
)

// standing is a place at the middle of the world, upright, of a size.
func standing(length, girth float64) place {
	return place{size: size{length: measure.Metres(length),
		girth: measure.Metres(girth)}}
}

// spruceOf is the growth and the parts of ninja's conifer, by the
// general path.
func spruceOf(t *testing.T, params ProperConiferParams) []part {
	t.Helper()
	grown, err := spruceGrowth(params.Whorls, params.PerWhorl,
		habitOf(params))
	if err != nil {
		t.Fatal(err)
	}
	parts, err := grown.parts(standing(params.Height,
		params.TrunkRadius), sow(params.Seed))
	if err != nil {
		t.Fatal(err)
	}
	return parts
}

// ninjasSlot is which of a plant's matters ninja's material number is.
var ninjasSlot = map[int]slot{barkMaterial: body, needleMaterial: leaf}

// the truth for carrying is ninja's ConiferProper: the trunk, and
// every piece of every branch with its sleeve of needles, face for
// face in ninja's order and each of ninja's material. ninja's tubes
// have no ends, so it is the walls that are compared.
func TestASpruceIsNinjasConiferProper(t *testing.T) {
	tall := ProperConiferParams{Height: 31, TrunkRadius: 0.6, Whorls: 48,
		PerWhorl: 9, CrownStart: 0.25, BranchLength: 0.18, Seed: 77}
	sparse := ProperConiferParams{Height: 8, TrunkRadius: 0.25,
		Whorls: 12, PerWhorl: 4, CrownStart: 0.1, BranchLength: 0.3,
		Seed: 9}
	for _, params := range []ProperConiferParams{
		DefaultProperConiferParams(), tall, sparse} {
		want := ConiferProper(geom.Vec3{}, params)
		var got surface
		var of []slot
		for _, one := range spruceOf(t, params) {
			round := one.shape.(tube)
			wall := round.wall(one.fine).placed(one.space)
			got = append(got, wall...)
			for range wall {
				of = append(of, one.of)
			}
		}
		if flaw := unlike(got, want); flaw != "" {
			t.Fatalf("height %v: %s", params.Height, flaw)
		}
		for index, face := range want {
			if of[index] != ninjasSlot[face.Material] {
				t.Fatalf("height %v: face %d is of matter %d, "+
					"ninja's material is %d", params.Height,
					index, of[index], face.Material)
			}
		}
	}
}

// every part of a grown thing is a sound surface on its own.
func TestEveryPartOfASpruceAndAFernIsSound(t *testing.T) {
	things := map[string][]part{
		"a spruce": spruceOf(t, DefaultProperConiferParams()),
		"a fern":   fernOf(t, 3),
	}
	for name, parts := range things {
		for index, one := range parts {
			if _, err := one.skin().meshed(); err != nil {
				t.Fatalf("%s, part %d: %v", name, index, err)
			}
			if one.shape.(tube).flat {
				continue
			}
			if flaw := unsound(one.skin()); flaw != "" {
				t.Fatalf("%s, part %d: %s", name, index, flaw)
			}
		}
	}
}

const (
	fronds   = 7
	leaflets = 16
	// what a fern's root is given: how long a frond is, and how thick
	// its stalk is at the foot
	frondLength = 0.5
	stalkGirth  = 0.004
)

// fernOf is the parts of a fern from a seed.
func fernOf(t *testing.T, number uint64) []part {
	t.Helper()
	grown, err := fernGrowth(fronds, leaflets)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := grown.parts(standing(frondLength, stalkGirth),
		sow(number))
	if err != nil {
		t.Fatal(err)
	}
	return parts
}

// a fern, in the order it is grown: its crown, then each frond's stalk
// followed by that frond's leaflets.
func TestAFernIsACrownOfFrondsOfLeaflets(t *testing.T) {
	parts := fernOf(t, 3)
	if want := 1 + fronds*(1+leaflets); len(parts) != want {
		t.Fatalf("%d parts, not a crown and %d fronds of a stalk and "+
			"%d leaflets: %d", len(parts), fronds, leaflets, want)
	}
	if parts[0].shape.(tube).flat || parts[0].of != body {
		t.Error("the crown is not a round stub of the stalk's matter")
	}
	// the crown is a stub: a small share of a frond's length, not a
	// stem as long as one
	crown := parts[0].shape.(tube)
	if stub := inScene(crown.length); stub > frondLength/10 {
		t.Errorf("the crown is %v long, under fronds of %v", stub,
			frondLength)
	}
	for frond := range fronds {
		first := 1 + frond*(1+leaflets)
		stalk := parts[first]
		if stalk.shape.(tube).flat || stalk.of != body {
			t.Errorf("frond %d: its stalk is flat or leaf", frond)
		}
		out := stalk.space.Orientation.Y
		if out.Y <= 0 || math.Hypot(out.X, out.Z) < 0.3 {
			t.Errorf("frond %d runs %v, not up and out", frond, out)
		}
		// a frond starts at the crown and lies face up: the side its
		// leaflets spread to is level
		if foot := stalk.space.Origin; foot.Len() > stalkGirth {
			t.Errorf("frond %d starts at %v, off the crown", frond,
				foot)
		}
		level := stalk.space.Orientation.X.Y
		if math.Abs(level) > close {
			t.Errorf("frond %d is twisted: its side rises %v",
				frond, level)
		}
		flaw := frondUnlike(stalk, parts[first+1:first+1+leaflets])
		if flaw != "" {
			t.Errorf("frond %d: %s", frond, flaw)
		}
	}
}

// frondUnlike is how a frond's leaflets fail to be a fern's, or
// nothing: each a blade of leaf, standing on the stalk, lying in the
// frond's own plane, in two rows with the sides taking turns, and
// shorter toward the tip.
func frondUnlike(stalk part, blades []part) string {
	axes, foot := stalk.space.Orientation, stalk.space.Origin
	reach := inScene(stalk.shape.(tube).length) * stalk.space.Scale
	last := math.Inf(1)
	for index, blade := range blades {
		if !blade.shape.(tube).flat || blade.of != leaf {
			return fmt.Sprintf("leaflet %d is not a blade of leaf",
				index)
		}
		from := blade.space.Origin.Sub(foot)
		up := from.Dot(axes.Y)
		off := from.Sub(axes.Y.Scale(up)).Len()
		if up <= 0 || up >= reach || off > 1e-9 {
			return fmt.Sprintf("leaflet %d is off the stalk", index)
		}
		lying := blade.space.Orientation
		way, face := lying.Y, lying.Z
		if !lying.X.Cross(way).Near(face, close) {
			return fmt.Sprintf("leaflet %d is left handed", index)
		}
		if math.Abs(face.Dot(axes.Y)) > 1e-9 {
			return fmt.Sprintf("leaflet %d is edge on to the frond",
				index)
		}
		side := way.Dot(axes.X)
		if (index%2 == 0) != (side > 0) || math.Abs(side) < 0.5 {
			return fmt.Sprintf("leaflet %d is on the wrong side",
				index)
		}
		long := inScene(blade.shape.(tube).length) * blade.space.Scale
		// a leaflet may be a little longer than the one below it, by
		// the seed, but not by a fifth
		if long > 1.2*last {
			return fmt.Sprintf("leaflet %d is longer than the one "+
				"below it", index)
		}
		last = long
	}
	return ""
}

func TestTheSameSeedIsTheSameFernAndAnotherIsNot(t *testing.T) {
	one, same, other := fernOf(t, 3), fernOf(t, 3), fernOf(t, 4)
	differ := 0
	for index := range one {
		if one[index] != same[index] {
			t.Fatalf("part %d changed for the same seed", index)
		}
		if one[index] != other[index] {
			differ++
		}
	}
	if differ < len(one)/2 {
		t.Errorf("only %d of %d parts differ for another seed", differ,
			len(one))
	}
}

// fronds do not share their leaflets' numbers: a part carried at a
// place grows from a seed of its own, sprouted from its carrier's.
func TestFrondsOfOneFernDiffer(t *testing.T) {
	parts := fernOf(t, 3)
	first, second := parts[2], parts[2+1+leaflets]
	if first.shape == second.shape {
		t.Error("the first leaflet of two fronds is the same size")
	}
}

// what a sprout is drawn from is a contract a kernel has to mirror:
// unit at the row, the index and the place kept for sprouts, widened
// to the whole number it came from.
func TestASproutIsDrawnFromItsRowAndIndex(t *testing.T) {
	const number, row, index = 6, 2, 5
	want := sow(uint64(unit(number, row, index, 3) * (1 << 53)))
	if got := sow(number).sprout(row, index); got != want {
		t.Errorf("sprout(%d, %d) is %v, not %v", row, index, got, want)
	}
}

func TestASproutIsASeedOfItsOwn(t *testing.T) {
	from := sow(6)
	seen := map[seed]bool{from: true}
	for row := range 5 {
		for index := range 5 {
			grown := from.sprout(row, index)
			if !grown.made() {
				t.Fatal("a sprout was never sown")
			}
			if grown != from.sprout(row, index) {
				t.Fatal("a sprout changed")
			}
			if seen[grown] {
				t.Fatalf("sprout %d, %d is a seed seen before",
					row, index)
			}
			seen[grown] = true
		}
	}
}

// a thing carried in a scaled place is that much bigger all through:
// its parts carry the scale, and so does everything they carry.
func TestWhatIsCarriedTakesItsCarriersScale(t *testing.T) {
	grown, err := fernGrowth(fronds, leaflets)
	if err != nil {
		t.Fatal(err)
	}
	plain := standing(frondLength, stalkGirth)
	doubled := plain
	doubled.space.Scale = 2
	small, _ := grown.parts(plain, sow(3))
	big, _ := grown.parts(doubled, sow(3))
	for index := range small {
		want := 2 * small[index].space.Scale
		if got := big[index].space.Scale; math.Abs(got-want) > close {
			t.Fatalf("part %d is scaled %v, not %v", index, got,
				want)
		}
		there := small[index].space.Origin.Scale(2)
		if !big[index].space.Origin.Near(there, 1e-9) {
			t.Fatalf("part %d is at %v, not %v", index,
				big[index].space.Origin, there)
		}
	}
}

func TestALimbThatCannotBeIsRefused(t *testing.T) {
	fine := mustCut(t, 5)
	taper := []joint{{ofGirth: 1}, {ofGirth: 0.5}}
	bark, err := newSleeve(taper, fine, body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newLimb(1, 1, 0, bark); err != nil {
		t.Fatalf("a limb of one piece was refused: %v", err)
	}
	nan := math.NaN()
	limbs := map[string]struct {
		pieces      int
		span, droop float64
		skins       []sleeve
	}{
		"no pieces":        {0, 1, 0, []sleeve{bark}},
		"no span":          {1, 0, 0, []sleeve{bark}},
		"a span past one":  {1, 1.5, 0, []sleeve{bark}},
		"a droop below":    {1, 1, -1, []sleeve{bark}},
		"an endless droop": {1, 1, math.Inf(1), []sleeve{bark}},
		"a droop, nan":     {1, 1, nan, []sleeve{bark}},
		"no skin":          {1, 1, 0, nil},
		"too few joints":   {2, 1, 0, []sleeve{bark}},
		"a skin not made":  {1, 1, 0, []sleeve{{}}},
	}
	for name, with := range limbs {
		_, err := newLimb(with.pieces, with.span, with.droop,
			with.skins...)
		if !errors.Is(err, errLimb) {
			t.Errorf("a limb with %s: %v, not %v", name, err,
				errLimb)
		}
	}
	sleeves := map[string][]joint{
		"one joint":           {{ofGirth: 1}},
		"a foot of nothing":   {{}, {ofGirth: 1}},
		"a share below zero":  {{ofGirth: 1}, {ofLength: -0.1}},
		"a share of no count": {{ofGirth: nan}, {ofGirth: 1}},
	}
	for name, joints := range sleeves {
		if _, err := newSleeve(joints, fine, body); !errors.Is(err,
			errSleeve) {
			t.Errorf("a sleeve with %s: %v, not %v", name, err,
				errSleeve)
		}
		if _, err := newLeaf(joints, leaf); !errors.Is(err, errSleeve) {
			t.Errorf("a leaf with %s: %v, not %v", name, err,
				errSleeve)
		}
	}
	if _, err := newSleeve(taper, cut{}, body); !errors.Is(err,
		errSleeve) {
		t.Errorf("a sleeve with no cut: %v, not %v", err, errSleeve)
	}
	// the edges: the coarsest cut there is, and a joint that is as
	// much of the girth as of the length
	even := []joint{{ofGirth: 0.5, ofLength: 0.5}, {}}
	if _, err := newSleeve(even, mustCut(t, leastAround),
		body); err != nil {
		t.Errorf("the coarsest sleeve was refused: %v", err)
	}
}

func TestAGrowthThatCannotBeIsRefused(t *testing.T) {
	if _, err := newGrowth(limb{}); !errors.Is(err, errGrowth) {
		t.Errorf("a growth with no stem: %v, not %v", err, errGrowth)
	}
	stem, err := spruceGrowth(4, 5, spruceHabit())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newGrowth(stem.stem, carried{}); !errors.Is(err,
		errGrowth) {
		t.Errorf("a growth carrying nothing anywhere: %v, not %v", err,
			errGrowth)
	}
}

// a limb droops toward the ground, piece by piece, and its pieces
// join end to end.
func TestALimbDroopsAndItsPiecesJoin(t *testing.T) {
	fine := mustCut(t, 4)
	skin, err := newSleeve([]joint{{ofGirth: 1}, {ofGirth: 0.8},
		{ofGirth: 0.6}, {ofGirth: 0.4}}, fine, body)
	if err != nil {
		t.Fatal(err)
	}
	bough, err := newLimb(3, 1, 0.5, skin)
	if err != nil {
		t.Fatal(err)
	}
	space, err := spaceAlong(maths.V(1, 2, 3), maths.V(1, 0.2, 0))
	if err != nil {
		t.Fatal(err)
	}
	space.Scale = 1.5
	on := place{space: space, size: size{length: measure.Metres(3),
		girth: measure.Metres(0.1)}}
	runs, err := bough.runs(on)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := bough.along(on.size, runs)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 3 {
		t.Fatalf("%d parts for three pieces of one skin", len(parts))
	}
	end, fall := on.space.Origin, math.Inf(1)
	for index, one := range parts {
		if !one.space.Origin.Near(end, 1e-9) {
			t.Errorf("piece %d starts at %v, the last ended at %v",
				index, one.space.Origin, end)
		}
		way := one.space.Orientation.Y
		if way.Y >= fall {
			t.Errorf("piece %d droops no lower than the last",
				index)
		}
		fall = way.Y
		reach := inScene(one.shape.(tube).length) * one.space.Scale
		if math.Abs(reach-1.5) > 1e-9 {
			t.Errorf("piece %d reaches %v, not a third of the "+
				"limb at its scale", index, reach)
		}
		end = one.space.Origin.Add(way.Scale(reach))
	}
}

// nowhere is an arrangement that cannot say where its one part goes.
type nowhere struct{ why error }

func (nowhere) rows() int      { return 1 }
func (nowhere) across(int) int { return 1 }
func (lost nowhere) at(size, int, int, seed) (place, error) {
	return place{}, lost.why
}

// sizeless is an arrangement whose one part has a length of zero, which no tube
// can have.
type sizeless struct{}

func (sizeless) rows() int      { return 1 }
func (sizeless) across(int) int { return 1 }
func (sizeless) at(on size, _, _ int, _ seed) (place, error) {
	return place{size: size{length: measure.Metres(0),
		girth: on.girth}}, nil
}

// whatever goes wrong while a thing grows comes back as the error it
// was, from however deep: nothing is grown with a piece missing.
func TestAThingThatCannotGrowSaysWhy(t *testing.T) {
	spruce, err := spruceGrowth(4, 5, spruceHabit())
	if err != nil {
		t.Fatal(err)
	}
	bough := spruce.carries[0].what
	lost := errors.New("nowhere to go")
	carrying := func(where arrangement) growth {
		grown, err := newGrowth(spruce.stem, carried{where: where,
			what: bough})
		if err != nil {
			t.Fatal(err)
		}
		return grown
	}
	// a limb pointing straight up that droops by exactly its own
	// length has no way left to go
	twoPiece, err := sleeved([]joint{{ofGirth: 1}, {ofGirth: 1},
		{ofGirth: 1}}, 4, body)
	if err != nil {
		t.Fatal(err)
	}
	straightDown, err := newLimb(2, 1, 1, twoPiece)
	if err != nil {
		t.Fatal(err)
	}
	fallen, err := newGrowth(straightDown)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		grown growth
		on    place
		want  error
	}{
		{"a stem of no length", spruce, standing(0, 0.3),
			errTubeLength},
		{"a part with nowhere to go", carrying(nowhere{lost}),
			standing(10, 0.3), lost},
		{"a part of no length", carrying(sizeless{}),
			standing(10, 0.3), errTubeLength},
		{"a limb with no way to go", fallen, standing(2, 0.1),
			errNoRun},
	}
	for _, refused := range cases {
		parts, err := refused.grown.parts(refused.on, sow(1))
		if !errors.Is(err, refused.want) || parts != nil {
			t.Errorf("%s: %d parts and %v, not none and %v",
				refused.name, len(parts), err, refused.want)
		}
	}
}

func TestASpruceOrAFernThatCannotBeIsRefused(t *testing.T) {
	if _, err := spruceGrowth(0, 5, spruceHabit()); !errors.Is(err,
		errWhorls) {
		t.Errorf("a spruce of no whorls: %v, not %v", err, errWhorls)
	}
	if _, err := fernGrowth(0, 9); !errors.Is(err, errRing) {
		t.Errorf("a fern of no fronds: %v, not %v", err, errRing)
	}
	if _, err := fernGrowth(5, 0); !errors.Is(err, errWhorls) {
		t.Errorf("a fern of no leaflets: %v, not %v", err, errWhorls)
	}
}

// a space already lying along the way it is asked to lie along has
// nothing to turn toward, and is left as it is.
func TestLyingAlongItsOwnRunLeavesASpaceAlone(t *testing.T) {
	space, err := spaceAlong(maths.V(1, 1, 1), maths.V(2, 3, -1))
	if err != nil {
		t.Fatal(err)
	}
	if got := lyingAlong(space, space.Orientation.Y); got != space {
		t.Errorf("the space was turned: %v", got.Orientation)
	}
	turned := lyingAlong(space, maths.V(0, 0, 1))
	if turned.Orientation.X.Dot(maths.V(0, 0, 1)) <= 0 {
		t.Errorf("its x is %v, not toward the way asked",
			turned.Orientation.X)
	}
}

// what is carried on one piece of a stem is not what is carried on
// the next: each piece's parts are placed and grown from the seed
// with the piece stirred in. found by review: a stem of two pieces
// grew the same branches on both, to four places. the first piece
// still draws from the seed itself, which is what keeps a spruce
// ninja's.
func TestPiecesOfOneStemCarryDifferentParts(t *testing.T) {
	stem, err := sleeved([]joint{{ofGirth: 1}, {ofGirth: 0.8},
		{ofGirth: 0.6}}, 4, body)
	if err != nil {
		t.Fatal(err)
	}
	twig, err := sleeved([]joint{{ofGirth: 1}, {}}, 4, body)
	if err != nil {
		t.Fatal(err)
	}
	up := mustWhorls(t, 1, 3, spruceHabit())
	grown, err := stemCarrying(2, 0, []sleeve{stem}, up, 1, 0,
		[]sleeve{twig})
	if err != nil {
		t.Fatal(err)
	}
	parts, err := grown.parts(standing(4, 0.2), sow(9))
	if err != nil {
		t.Fatal(err)
	}
	// two pieces of stem, then three twigs on each
	if len(parts) != 8 {
		t.Fatalf("%d parts", len(parts))
	}
	for twig := range 3 {
		low, high := parts[2+twig], parts[5+twig]
		if low.shape == high.shape {
			t.Errorf("twig %d is one size on both pieces", twig)
		}
		if low.space.Orientation == high.space.Orientation {
			t.Errorf("twig %d points the same way on both", twig)
		}
	}
	if got, want := sow(9).onPiece(0), sow(9); got != want {
		t.Errorf("the first piece's seed is %v, not the seed", got)
	}
	if sow(9).onPiece(1) == sow(9).onPiece(2) {
		t.Error("two pieces share a seed")
	}
}
