package cities

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/janearc/slayer/scene/thing"
)

// A superblock is one parcel of a city built as one mass, planned at one
// end and grown over at the other. Its accretion, from nought to one, says
// where between the two it stands: how many decades of additions have
// been laid over the plan.
//
// Planned, it is a perimeter block: a podium of shops with parking over
// them, standing on the parcel less a pavement round it, a car ramp up its
// back, and on the podium a slab of flats along each side round a court,
// with a stair and lift core at each corner and between long runs.
//
// The slabs' storeys come from their people: the occupants the block is
// planned for over the floor a storey of slab holds at the crowding they
// tolerate, rounded up. Nobody picks a height; a block whose people need
// more storeys than its ceiling is refused, and says how many do not fit.
//
// The slabs' faces are households, as a condominium's are: each draws its
// curtains or blind, its light, a balcony with what it keeps there, an
// air-conditioning unit, a replaced window. Along the court face an access
// deck stands every few storeys, where no household has a balcony.
//
// Grown over, it is what Kowloon Walled City was: the court cut into plots and
// narrow lanes by the seed, every plot built full against its neighbours,
// floors added on the slabs' roofs at heights that match nothing below,
// rooms carried out over the air, walkways across the lanes.
//
// What fuses it is hung on it: cages and boxes on the walls, a few signs
// that glow, cables across the lanes and pipes along them, roofs railed and
// used as yards, and on each roof water tanks, antennas and shacks strapped
// down with the scavenged bands. Each of these comes by a share that the
// accretion gates, so that at nought there are none and at one the most.
//
// Between the two, a planned block is partly overgrown. A lane that reaches
// the court's edge through no other is a light well, and is reported, not
// hidden. Every number is in the superblock culture, picked; the parcel,
// the ceiling, the people, their crowding, the accretion and the seed are
// the set's.

// superblockCulture is the culture a superblock is drawn from.
const superblockCulture = "superblock"

// superblockStream is the stream of draws a superblock takes its order
// of building from, which no other use of its seed shares.
const superblockStream = 11

// meeting is how far apart two faces may be and still be one face, in
// metres: the slack of the arithmetic, not a choice about buildings.
const meeting = 1e-6

// The four sides of a plot, each named by where its wall faces.
const (
	frontSide = iota
	eastSide
	backSide
	westSide
	sidesAround
)

// What a plot of a superblock is built as.
const (
	// useFlats is a run of a planned slab, its faces households.
	useFlats = "flats"
	// useCore is a stair and lift core of the planned block.
	useCore = "core"
	// useInfill is a building grown on a plot of the court.
	useInfill = "infill"
	// useTopping is floors added on a slab's roof.
	useTopping = "topping"
)

// rect is a stretch of the parcel's ground: from X0 to X1 along x and
// from Z0 to Z1 along z, in metres, the parcel's middle at the origin.
type rect struct{ X0, Z0, X1, Z1 float64 }

// Volume is a box of air a part of the superblock fills, with its kind,
// the plot it belongs to (or minus one), and its low and high corners as
// x, y and z in metres. No two volumes of a superblock meet.
type Volume struct {
	Kind      string
	Plot      int
	Low, High [3]float64
}

// Accretion is how far a superblock is grown over, from planned at nought
// to accreted at one, or the word that it is drawn from its seed.
type Accretion struct {
	// Value is the accretion, from nought to one, if not Drawn.
	Value float64
	// Drawn says the block draws its accretion from its seed, from the
	// culture's distribution.
	Drawn bool
}

// SuperblockAsk is what a set says of a superblock: the width of the
// parcel along x and its depth along z in metres, the ceiling in storeys,
// whether roofs keep berths for parts of the set's own mix, the people it
// is planned for and the crowding they tolerate, its accretion, and the
// detail its faces are drawn at.
type SuperblockAsk struct {
	Width, Depth float64
	Ceiling      int
	Berths       bool
	Occupants    int
	Crowding     float64
	Accretion    Accretion
	Level        Detail
}

// plot is one plot of the superblock and the building on it: its ground,
// what it is built as, the side of a slab or core that faces out of the
// block, the height of its foot and of its top, how much taller its ground
// storey is than the rest, its storeys, and when it was built.
type plot struct {
	Lot     rect
	Use     string
	Out     int
	Base    float64
	Height  float64
	Ground  float64
	Storeys int
	Order   int
}

// Superblock is a parcel built as one mass. Plots are the slabs and cores
// first, then the court's buildings, then the floors added on roofs. The
// alleys are the lanes round the court and the gaps the court's cuts left,
// Sealed those that reach the court's edge through no other (light wells),
// Yards the plots of the court left open, and Fixtures everything hung or
// stood on the buildings.
type Superblock struct {
	Ask       SuperblockAsk
	Accretion float64
	Parcel    rect
	Podium    rect
	Court     rect
	Plots     []plot
	Alleys    []rect
	Sealed    []rect
	Yards     []rect
	Fixtures  []Volume
	Berths    []Volume
	// Storeys are the storeys the slabs rise to, from their people, and
	// PerPerson the floor a person is given, in square metres.
	Storeys   int
	PerPerson float64
	// Deck is the height of the podium's top, Ceiling the highest a
	// building stands, and Allowance the most that anything on a roof
	// stands over its building.
	Deck      float64
	Ceiling   float64
	Allowance float64
	parts     []placedElement
}

// accretion is a superblock as it is being grown: the drawer that says
// every number and the one that draws households, the stream that says
// the order, the materials of each plot, the court's plots not yet built,
// what is already solid, and the parts drawn so far.
type accretion struct {
	drawing *drawer
	homes   *drawer
	ask     SuperblockAsk
	grown   float64
	order   *rand.Rand
	straps  Straps
	lashing *thing.Material
	walls   []*thing.Material
	lots    []rect
	solids  []Volume
	parts   []placedElement
	out     Superblock
}

// DefaultSuperblock is the ask a set gets that says nothing of its
// parcel, its ceiling, its people or its accretion: the culture's own, a
// small parcel, no berths, and an accretion drawn from the seed.
func DefaultSuperblock() (SuperblockAsk, error) {
	drawing, err := newDrawer(superblockCulture, 0)
	if err != nil {
		return SuperblockAsk{}, err
	}
	ask := SuperblockAsk{Width: drawing.fixed("default-width"),
		Depth:     drawing.fixed("default-depth"),
		Ceiling:   drawing.count("default-ceiling", place{}),
		Occupants: drawing.count("default-occupants", place{}),
		Crowding:  drawing.fixed("default-crowding"),
		Accretion: Accretion{Drawn: true}}
	return ask, drawing.trouble
}

// NewSuperblock grows a superblock from a seed. It refuses a parcel with
// no room for a podium, slabs and a court of two plots each way, a ceiling
// below one storey, people who do not fit under it, an accretion outside
// nought to one, and anything the cultures or the lashing data cannot say.
func NewSuperblock(ask SuperblockAsk, seed Seed) (Superblock, error) {
	growing, err := newAccretion(ask, seed)
	if err != nil {
		return Superblock{}, err
	}
	if err := growing.sound(); err != nil {
		return Superblock{}, err
	}
	growing.grow()
	if err := growing.trouble(); err != nil {
		return Superblock{}, err
	}
	return growing.out, nil
}

// newAccretion is a superblock ready to grow: its drawers, the lashing,
// and the stream of its order.
func newAccretion(ask SuperblockAsk, seed Seed) (*accretion, error) {
	drawing, err := newDrawer(superblockCulture, seed)
	if err != nil {
		return nil, err
	}
	homes, err := newDrawer(condominiumCulture, seed)
	if err != nil {
		return nil, err
	}
	homes.level, homes.homes = ask.Level, ask.Level == demoLevel
	plan, err := Scavenged()
	if err != nil {
		return nil, err
	}
	lashing, err := material(roleLashing)
	if err != nil {
		return nil, err
	}
	return &accretion{drawing: drawing, homes: homes, ask: ask,
		straps: plan.Band, lashing: lashing, order: rand.New(
			rand.NewPCG(uint64(seed), superblockStream))}, nil
}

// trouble is the first thing that went wrong in either drawer.
func (growing *accretion) trouble() error {
	if growing.drawing.trouble != nil {
		return growing.drawing.trouble
	}
	return growing.homes.trouble
}

// sound refuses an ask that cannot make a superblock, and resolves its
// accretion.
func (growing *accretion) sound() error {
	drawing, ask := growing.drawing, growing.ask
	least := 2 * (drawing.fixed("apron-width") + drawing.fixed(
		"slab-depth") + drawing.fixed("lane-width") +
		drawing.fixed("plot-least"))
	if !(ask.Width >= least && ask.Depth >= least) || ask.Ceiling < 1 {
		return fmt.Errorf("cities: a superblock needs a parcel at least "+
			"%v m each way and a ceiling of a storey, not %v by %v m "+
			"and %d", least, ask.Width, ask.Depth, ask.Ceiling)
	}
	grown, err := growing.resolved()
	growing.grown = grown
	if err != nil {
		return err
	}
	return drawing.trouble
}

// resolved is the accretion as a number from nought to one.
func (growing *accretion) resolved() (float64, error) {
	return growing.ask.Accretion.number(growing.drawing)
}

// number is the accretion as a number from nought to one: the value
// given, or the one the drawer's seed draws where it is Drawn.
func (given Accretion) number(drawing *drawer) (float64, error) {
	if given.Drawn {
		return drawing.number("accretion-draw", place{}), nil
	}
	if !(given.Value >= 0 && given.Value <= 1) {
		return 0, fmt.Errorf("cities: a superblock's accretion is from "+
			"nought to one, not %v", given.Value)
	}
	return given.Value, nil
}

// AccretionOf is the accretion a superblock grown from a seed has, from
// nought to one, without growing it: the value asked, or the one the seed
// draws. It is what a weathering that follows the accretion reads.
func AccretionOf(given Accretion, seed Seed) (float64, error) {
	drawing, err := newDrawer(superblockCulture, seed)
	if err != nil {
		return 0, err
	}
	grown, err := given.number(drawing)
	if err != nil {
		return 0, err
	}
	return grown, drawing.trouble
}

// grow does the work, in the order a superblock is made: lay out the
// plan, cut the court, raise what the accretion builds on it and on the
// roofs, draw every building, then hang what is hung.
func (growing *accretion) grow() {
	across, along := growing.ask.Width/half, growing.ask.Depth/half
	growing.out.Ask = growing.ask
	growing.out.Accretion = growing.grown
	growing.out.Parcel = rect{-across, -along, across, along}
	growing.plan()
	if growing.trouble() != nil {
		return
	}
	growing.cut()
	growing.raise()
	growing.top()
	growing.build()
	growing.hang()
	growing.out.parts = growing.parts
}

// Design is the superblock as the parts it is drawn from, to render.
func (block Superblock) Design() Design {
	return Design{Name: "superblock", Elements: block.parts}
}

// Volumes are every volume the superblock fills: the podium, the
// buildings, and what is hung or stood on them, but not the berths, which
// are left empty for parts a set brings.
func (block Superblock) Volumes() []Volume {
	podium := block.Podium
	volumes := []Volume{{Kind: "podium", Plot: -1,
		Low:  [3]float64{podium.X0, 0, podium.Z0},
		High: [3]float64{podium.X1, block.Deck, podium.Z1}}}
	for index, plot := range block.Plots {
		volumes = append(volumes, plot.volume(index))
	}
	return append(volumes, block.Fixtures...)
}

// volume is the box of air a plot's building fills, from its foot to its
// top.
func (plot plot) volume(index int) Volume {
	return Volume{Kind: "building", Plot: index,
		Low:  [3]float64{plot.Lot.X0, plot.Base, plot.Lot.Z0},
		High: [3]float64{plot.Lot.X1, plot.Height, plot.Lot.Z1}}
}

// housing is whether a plot is built for people to live in: a slab's
// flats, a building of the court, or floors added on a roof, but not a
// core.
func (plot plot) housing() bool {
	return plot.Use != useCore
}

// Facts are what the superblock is, physically, that its density follows
// from: the floor of every storey built for people, the people that floor
// sleeps at the floor a person is given, and the parcel's ground.
func (block Superblock) Facts() Facts {
	facts := Facts{PitchArea: block.Parcel.area()}
	for _, plot := range block.Plots {
		if plot.housing() {
			facts.FloorArea += plot.Lot.area() * float64(plot.Storeys)
		}
	}
	facts.Sleeps = int(math.Floor(facts.FloorArea / block.PerPerson))
	return facts
}

// Density is the superblock's design occupancy, people per square metre
// of its parcel, derived from its facts: the planned slabs and everything
// the accretion added for people to live in.
func (block Superblock) Density() (float64, error) {
	return DensityOf(block.Facts())
}
