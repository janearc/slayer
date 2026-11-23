// Package seaside stages structural studies of four public seaside piers.
package seaside

import (
	"fmt"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/pier"
	"github.com/janearc/slayer/scene/thing"
)

var (
	timber = &thing.Material{Name: "weathered deck timber", Colour: maths.RGB(.39, .27, .17)}
	piling = &thing.Material{Name: "wet timber", Colour: maths.RGB(.19, .13, .085)}
	white  = &thing.Material{Name: "painted cream", Colour: maths.RGB(.79, .79, .69)}
	iron   = &thing.Material{Name: "painted iron", Colour: maths.RGB(.105, .15, .17)}
	steel  = &thing.Material{Name: "light painted steel", Colour: maths.RGB(.66, .74, .73)}
	red    = &thing.Material{Name: "oxide red", Colour: maths.RGB(.59, .105, .065)}
	blue   = &thing.Material{Name: "marine blue", Colour: maths.RGB(.07, .24, .44)}
	gold   = &thing.Material{Name: "amber gondola", Colour: maths.RGB(.91, .40, .074)}
	glass  = &thing.Material{Name: "dark cabin windows", Colour: maths.RGB(.077, .17, .21), Gloss: .35}
)

type point struct{ east, north float64 }

// Study holds a built location and its review camera.
type Study struct {
	Cove      *arid.Cove
	From, Aim maths.Vec
}

type site struct {
	layout   func(*structures)
	from, to maths.Vec
	water    maths.Light
}

var sites = map[string]site{
	"oceanside": {oceanside, maths.V(421, 105, -250), maths.V(0, 3, -280),
		maths.RGB(.37, .12, .09)},
	"santa-monica": {santaMonica, maths.V(-480, 104, -296), maths.V(9, 13, -316),
		maths.RGB(.37, .12, .09)},
	"brighton": {brighton, maths.V(-382, 96, -243), maths.V(-3, 12, -245),
		maths.RGB(.41, .23, .13)},
	"scheveningen": {scheveningen, maths.V(345, 97, -213), maths.V(12, 19, -223),
		maths.RGB(.46, .25, .16)},
}

// NewWithWaves optionally stages the pier with a bounded wave continuation.
func NewWithWaves(name string, waterMode string) (Study, error) {
	if waterMode != "calm" && waterMode != "waves" && waterMode != "wide-waves" {
		return Study{}, fmt.Errorf("seaside: unknown water mode %q", waterMode)
	}
	site, ok := sites[name]
	if !ok {
		return Study{}, fmt.Errorf("seaside: unknown pier %q", name)
	}
	var made structures
	made.shore(name)
	site.layout(&made)
	if made.err != nil {
		return Study{}, made.err
	}
	water := &arid.Water{
		Name: name + " study water", Extinction: site.water,
		Sea:    oceantex.Sea{Wind: 0, Fetch: 7200, Depth: 14, Toward: .33},
		Extent: 32, Collar: 8, Samples: 65, Seed: 183,
	}
	if waterMode != "calm" {
		water.Sea.Wind = 4.1
		water.Extent, water.Collar, water.Samples = 680, 45, 385
		water.BoundedEdge = true
	}
	if waterMode == "wide-waves" {
		water.Extent, water.Collar, water.Samples = 3600, 180, 1024
	}
	location, err := arid.New(made.parts, water)
	return Study{location, site.from, site.to}, err
}

type structures struct {
	parts []thing.Thing
	err   error
}

// add retains geometry or remembers the first construction error.
func (s *structures) add(parts []thing.Thing, err error) {
	if s.err != nil {
		return
	}
	if err != nil {
		s.err = err
		return
	}
	s.parts = append(s.parts, parts...)
}

// deck projects a local plan onto the reusable pier constructor.
func (s *structures) deck(points []point, width, height, spacing float64,
	top, piles, rail *thing.Material) {
	line := make([]maths.Vec, 0, len(points))
	for _, point := range points {
		line = append(line, maths.V(point.east, 0, -point.north))
	}
	s.add(pier.New(line, pier.Deck{
		Width: width, Height: height, Thickness: .72, Overlap: .65,
		PileBottom: -12, PileSpacing: spacing, PileWidth: .62,
		RailHeight: 1.13, Surface: top, Pilings: piles, Rail: rail,
	}))
}

// wheel makes an observation wheel at a site-specific dock height.
func (s *structures) wheel(base maths.Vec, yaw, radius, hub float64, count int,
	frame, rim, cabin *thing.Material) {
	s.add(pier.NewWheel(pier.Wheel{
		Base: base, Yaw: yaw, Phase: .11, Radius: radius, HubHeight: hub,
		Gondolas: count, Frame: frame, Rim: rim, Gondola: cabin, Windows: glass,
	}))
}

// oceanside stages the timber pier with its pre-fire end pavilion.
func oceanside(s *structures) {
	const height = 7.7
	s.deck([]point{{0, -30}, {0, 70}}, 12.5, height, 8, timber, steel, white)
	s.deck([]point{{0, 70}, {0, 290}, {0, 547}}, 9, height, 7.6, timber, piling, white)
	s.deck([]point{{0, 547}, {0, 595.6}}, 25, height, 6.7, timber, piling, white)
	s.house(maths.V(0, height, -577), 15, 3.8, 15, white, blue)
	s.house(maths.V(-2.9, height, -326), 3, 2.7, 6.3, white, blue)
	s.house(maths.V(5, height, -17), 10, 5, 16, white, red)
}

// santaMonica places the widened amusement deck beside the long promenade.
func santaMonica(s *structures) {
	const height = 10.3
	s.deck([]point{{0, -24}, {0, 162}, {0, 452}, {1, 625}}, 20, height, 9, timber, piling, iron)
	s.deck([]point{{19, 145}, {19, 339}}, 46, height, 9, timber, piling, iron)
	s.deck([]point{{1, 625}, {1, 650.7}}, 26, height, 8, timber, piling, iron)
	s.wheel(maths.V(25, height, -284), -1.28, 12.954, 16.8, 20, white, blue, gold)
	s.house(maths.V(21, height, -153), 24, 7, 41, white, red)
	s.house(maths.V(25, height, -335), 27, 5, 16, white, blue)
	s.house(maths.V(1, height, -616), 14, 5, 26, white, red)
	s.coaster(height)
}

// brighton stages the Palace Pier with widening sections and gabled roofs.
func brighton(s *structures) {
	const height = 8.2
	s.deck([]point{{0, -35}, {0, 76}, {0, 150}}, 20, height, 9, white, iron, white)
	s.deck([]point{{0, 150}, {0, 320}, {0, 432}}, 28, height, 9, white, iron, white)
	s.deck([]point{{0, 432}, {0, 524.9}}, 45, height, 8, white, iron, white)
	s.house(maths.V(0, height, -233), 24, 7, 71, white, iron)
	s.house(maths.V(0, height, -345), 22, 7, 48, white, red)
	s.house(maths.V(0, height, -481), 31, 8, 28, white, iron)
	for _, side := range []float64{-1, 1} {
		s.house(maths.V(side*6, height, -23), 7, 5, 10, white, red)
	}
}

// scheveningen uses two promenade levels and distinct terminal platforms.
func scheveningen(s *structures) {
	s.deck([]point{{0, -20}, {0, 278}, {0, 367}}, 21, 6, 12, white, iron, steel)
	s.deck([]point{{0, -20}, {0, 278}, {0, 363}}, 18, 12, 12, white, iron, steel)
	s.deck([]point{{0, 304}, {44, 337}, {51, 380}}, 15, 8, 9, white, iron, steel)
	s.deck([]point{{51, 375}, {51, 413}}, 31, 8, 9, white, iron, steel)
	s.deck([]point{{0, 320}, {-40, 354}, {-46, 386}}, 18, 8, 9, white, iron, steel)
	s.house(maths.V(-46, 8, -375), 27, 6, 27, steel, blue)
	s.house(maths.V(0, 12, -258), 13, 4, 63, white, iron)
	s.wheel(maths.V(51, 8, -392), 1.24, 19, 23, 36, steel, white, white)
}
