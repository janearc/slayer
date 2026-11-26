package manufacture

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// SuperblockKind is the name of the kind superblock. A superblock is
// never a part of a cluster, or of itself.
const SuperblockKind = "superblock"

// SuperblockHabit is the one habit a superblock has, the way a parcel is
// built over: plot by plot, each against the last, into one mass.
const SuperblockHabit = "accreted"

// superblockPartStream is the stream of draws a superblock takes the
// parts of its roof berths from, which no other use of its seed shares.
const superblockPartStream = 13

// turnsAround is how many quarter turns bring a part back to where it
// began, which is how many ways a part may stand in a berth.
const turnsAround = 4

// SuperblockAsk is what a set asks of a superblock: the size of its
// parcel in metres, its ceiling in storeys, its habit, the kinds the
// parts on its roofs are drawn from, each with a weight, the people it is
// planned for and the crowding they tolerate, and its accretion, from
// planned at nought to grown over at one, or Drawn from the seed.
//
// A size, a ceiling, a number of people or a crowding that is nought is
// left to the culture's data, and a superblock with no parts keeps no
// berths on its roofs.
type SuperblockAsk struct {
	Width, Depth float64
	Ceiling      int
	Habit        string
	Parts        map[string]float64
	Occupants    int
	Crowding     float64
	Accretion    float64
	Drawn        bool
}

// Superblocked is the superblock kind as a set asks for it. It refuses a
// row that is not the superblock, a habit that is not accreted, a part
// that is not a kind the manufactory makes, is a cluster or is a
// superblock, a weight that is not above nought, a size, a ceiling, a
// number of people or a crowding below nought, and an accretion outside
// nought to one.
func Superblocked(row Kind, ask SuperblockAsk) (Kind, error) {
	if row.name != SuperblockKind {
		return Kind{}, fmt.Errorf("manufacture: %s is not a superblock, "+
			"and takes no parcel", row.name)
	}
	if ask.Habit != "" && ask.Habit != SuperblockHabit {
		return Kind{}, fmt.Errorf("manufacture: a superblock is built %s, "+
			"not %s", SuperblockHabit, ask.Habit)
	}
	known := table()
	for name, weight := range ask.Parts {
		if _, made := known[name]; !made || name == ClusterKind ||
			name == SuperblockKind || !(weight > 0) {
			return Kind{}, fmt.Errorf("manufacture: a superblock's roofs "+
				"take %q at weight %v, which is no kind of part or no "+
				"weight", name, weight)
		}
	}
	if ask.Width < 0 || ask.Depth < 0 || ask.Ceiling < 0 ||
		ask.Occupants < 0 || ask.Crowding < 0 {
		return Kind{}, fmt.Errorf("manufacture: a superblock has a parcel, "+
			"a ceiling, people and a crowding that are not below nought, "+
			"not %v by %v, %d, %d and %v", ask.Width, ask.Depth,
			ask.Ceiling, ask.Occupants, ask.Crowding)
	}
	if !(ask.Accretion >= 0 && ask.Accretion <= 1) {
		return Kind{}, fmt.Errorf("manufacture: a superblock's accretion is "+
			"from nought to one, not %v", ask.Accretion)
	}
	return Kind{name: row.name, how: superblockRow{ask: ask}}, nil
}

// superblockRow is the kind superblock: a parcel built over, grown by
// package cities, with parts of the set's own mix standing in the berths
// it keeps on the roofs. It is made on flat ground, in space or on a
// world.
type superblockRow struct{ ask SuperblockAsk }

// made is the superblock at the quick detail level.
func (row superblockRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the superblock at the named detail level. It returns errGood
// when the seed is not made.
func (row superblockRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	ask, err := row.asked(detail)
	if err != nil {
		return good{}, err
	}
	block, err := cities.NewSuperblock(ask, cities.Seed(from.number))
	if err != nil {
		return good{}, err
	}
	drawn, err := block.Design().Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	whole, err := goodOfRendered(drawn)
	if err != nil || len(block.Berths) == 0 {
		return whole, err
	}
	return row.berthed(whole, block.Berths, from, detail)
}

// asked is the ask package cities takes at a detail level: the culture's
// own parcel, ceiling, people and crowding, where the set says none, the
// set's accretion, and berths where the set has parts.
func (row superblockRow) asked(detail string) (cities.SuperblockAsk,
	error) {
	ask, err := cities.DefaultSuperblock()
	if err != nil {
		return ask, err
	}
	ask.Width = given(row.ask.Width, ask.Width)
	ask.Depth = given(row.ask.Depth, ask.Depth)
	ask.Ceiling = int(given(float64(row.ask.Ceiling), float64(ask.Ceiling)))
	ask.Occupants = int(given(float64(row.ask.Occupants),
		float64(ask.Occupants)))
	ask.Crowding = given(row.ask.Crowding, ask.Crowding)
	ask.Accretion = cities.Accretion{Value: row.ask.Accretion,
		Drawn: row.ask.Drawn}
	ask.Berths = len(row.ask.Parts) > 0
	ask.Level = cities.Detail(detail)
	return ask, nil
}

// given is a number the set gave, or the data's where it gave nought.
func given(set, data float64) float64 {
	if set > 0 {
		return set
	}
	return data
}

// berthed is the superblock with a part of the set's mix in each berth
// there is one that fits, each part drawn from the seed in the order the
// berths are in. A berth that no part drawn fits is left empty.
func (row superblockRow) berthed(whole good, berths []cities.Volume,
	from seed, detail string) (good, error) {
	plan, err := cities.Scavenged()
	if err != nil {
		return good{}, err
	}
	rng := rand.New(rand.NewPCG(from.number, superblockPartStream))
	goods := []good{whole}
	for _, berth := range berths {
		for range plan.DrawTries {
			part, fits, err := row.partFor(berth, rng, detail)
			if err != nil {
				return good{}, err
			}
			if fits {
				goods = append(goods, part)
				break
			}
		}
	}
	return gather(goods...)
}

// partFor is a part drawn from the mix and stood in a berth, and whether
// it fits: its box, turned to the quarter that suits the berth, no longer,
// wider or taller than the berth, and standing on its floor at its middle.
func (row superblockRow) partFor(berth cities.Volume, rng *rand.Rand,
	detail string) (good, bool, error) {
	name, err := drawKind(ClusterAsk{Parts: row.ask.Parts}, rng)
	if err != nil {
		return good{}, false, err
	}
	var part scavengedPart
	if len(row.ask.Parts) == 0 {
		made, makeErr := madeDefaultPart(name, rng.Uint64(), detail)
		if makeErr != nil {
			return good{}, false, makeErr
		}
		part, err = turnedPart(name, made, maths.Identity)
	} else {
		part, err = partOf(name, rng.Uint64(), maths.Identity, detail)
	}
	if err != nil {
		return good{}, false, err
	}
	size := part.piece.Box.Max.Sub(part.piece.Box.Min)
	room := maths.V(berth.High[0]-berth.Low[0], berth.High[1]-berth.Low[1],
		berth.High[2]-berth.Low[2])
	for turn := range turnsAround {
		wide, deep := size.X, size.Z
		if turn%2 == 1 {
			wide, deep = deep, wide
		}
		if wide > room.X || deep > room.Z || size.Y > room.Y {
			continue
		}
		return row.stood(part, berth, turn)
	}
	return good{}, false, nil
}

// stood is a part turned a number of quarters about the vertical and
// set in a berth: its middle over the berth's, its foot on the berth's
// floor.
func (row superblockRow) stood(part scavengedPart, berth cities.Volume,
	quarters int) (good, bool, error) {
	turn := maths.Rotation(maths.YAxis, quarterTurn*float64(quarters))
	middle := turn.Out(part.piece.Box.Min.Add(part.piece.Box.Max).Scale(0.5))
	at := maths.V((berth.Low[0]+berth.High[0])/2-middle.X,
		berth.Low[1]-part.piece.Box.Min.Y,
		(berth.Low[2]+berth.High[2])/2-middle.Z)
	placed, err := part.turned.placed(maths.NewSubspace(at, turn, 1))
	return placed, err == nil, err
}

// quarterTurn is a quarter of a whole turn, in radians.
const quarterTurn = math.Pi / 2

// SuperblockAccretion is how far a superblock of a seed is grown over,
// from nought to one: the accretion the ask gives, or the one the seed
// draws if the ask leaves it to the seed. It does not grow the block. The
// seed is the number the block is made from.
func SuperblockAccretion(ask SuperblockAsk, number uint64) (float64, error) {
	return cities.AccretionOf(cities.Accretion{Value: ask.Accretion,
		Drawn: ask.Drawn}, cities.Seed(number))
}
