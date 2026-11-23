package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// BubblesKind is a stream of air bubbles rising through water.
//
// The bubbles are air as it is: each is a clear body of the index of
// air, bounded by a surface, that sits in a sea whose own index is the
// set's. The tracer bends a look into the bubble and out again and
// mirrors at its rim by the two indices, so nothing here paints it.
//
// The stream's origin is where it is born: a set puts the thing where
// it wants it, on an animal's mouth or under a breaking wave, and turns
// it by heading. y is up.
//
// Sizes, the spread and the height are data, bubbles.yaml, each with
// its basis. A bubble's rise is sqrt(2 sigma / (rho d) + g d / 2) for a
// diameter d (a wave-analogy form, recalled and not read), and its
// shape is oblate by the figure in the file.
//
// The stream moves with time (MakeAtTime): each bubble rises at its
// own speed and begins again at the origin when it has climbed the
// stream's height.
const BubblesKind = "bubble-stream"

//go:embed bubbles.yaml
var bubblesRaw []byte

// bubblesNames are the figures the file must give, in seed order.
var bubblesNames = []string{"count", "radius", "height", "spread", "flare"}

// bubblesPhysics are the constants the file must give.
var bubblesPhysics = []string{"surface-tension", "density", "gravity",
	"shape-coefficient", "shape-power", "air-index"}

// the numbers of a stream's making
const (
	bubblesClasses = 4    // picked: shapes a stream's bubbles are cut into
	bubblesSides   = 12   // picked: flat sides round a bubble
	bubblesRings   = 9    // picked: rings from pole to pole
	bubblesPole    = 0.05 // picked: the radius at a pole, over the widest
	bubblesRank    = 1    // picked: above water's, so a point in both is air
)

// bubblesBook is the file.
type bubblesBook struct {
	Read    string
	Sources map[string]string
	Figures map[string]kelpFigure
	Physics map[string]kelpConstant
}

// bubblesRead is the file read and checked, and bubblesError why not.
var bubblesRead, bubblesError = readBubbles(bubblesRaw)

// readBubbles reads the file and checks each figure and constant.
func readBubbles(raw []byte) (bubblesBook, error) {
	var book bubblesBook
	if err := yamlfile.Read("bubbles.yaml", raw, &book); err != nil {
		return bubblesBook{}, err
	}
	basis := kelpBook{Sources: book.Sources}
	for _, name := range bubblesNames {
		row, given := book.Figures[name]
		if !given || !(row.Least >= 0 && row.Most >= row.Least) ||
			!positive(row.Most) || !basis.based(row.Basis) {
			return bubblesBook{}, fmt.Errorf("manufacture: the bubbles' "+
				"%s, %+v, is not a range with a basis", name, row)
		}
	}
	for _, name := range bubblesPhysics {
		row, given := book.Physics[name]
		if !given || !positive(row.Value) || !basis.based(row.Basis) {
			return bubblesBook{}, fmt.Errorf("manufacture: the bubbles' "+
				"%s, %+v, is not a number with a basis", name, row)
		}
	}
	return book, nil
}

// pick is a figure from a seed at an index.
func (book bubblesBook) pick(name string, from seed, index int) float64 {
	row := book.Figures[name]
	place := slices.Index(bubblesNames, name)
	return row.Least + (row.Most-row.Least)*from.draw(picking, place, index)
}

// number is a constant by name.
func (book bubblesBook) number(name string) float64 {
	return book.Physics[name].Value
}

// rise is how fast a bubble of a radius climbs, in metres a second.
func (book bubblesBook) rise(radius float64) float64 {
	diameter := 2 * radius
	tension, density := book.number("surface-tension"), book.number("density")
	return math.Sqrt(2*tension/(density*diameter) +
		book.number("gravity")*diameter*kelpHalf)
}

// aspect is a bubble's height over its width, from its Eotvos number.
func (book bubblesBook) aspect(radius float64) float64 {
	diameter := 2 * radius
	eotvos := book.number("density") * book.number("gravity") *
		diameter * diameter / book.number("surface-tension")
	return 1 / (1 + book.number("shape-coefficient")*
		math.Pow(eotvos, book.number("shape-power")))
}

// bubblesKit is the shared meshes: one oblate bubble of a radius of one
// for each class of size, its aspect that of the class's middle radius.
var bubblesKit = sync.OnceValues(func() ([]*mesh.Mesh, error) {
	book := bubblesRead
	least, most := book.Figures["radius"].Least, book.Figures["radius"].Most
	var kit []*mesh.Mesh
	for class := range bubblesClasses {
		middle := least + (most-least)*(float64(class)+kelpHalf)/bubblesClasses
		made, err := bubbleMesh(book.aspect(middle))
		if err != nil {
			return nil, err
		}
		kit = append(kit, made)
	}
	return kit, nil
})

// bubbleMesh is a bubble of radius one and a height over its width, as
// a flat mesh, its axis y.
func bubbleMesh(aspect float64) (*mesh.Mesh, error) {
	rings := make([][]maths.Vec, bubblesRings)
	for step := range rings {
		along := 2*float64(step)/float64(bubblesRings-1) - 1
		radius := math.Max(bubblesPole, math.Sqrt(1-along*along))
		rings[step] = (tube{}).ring(radius, along*aspect, cut{around: bubblesSides})
	}
	return (alfaLoft{rings: rings}).skin(cut{}).meshed()
}

// bubblesRow is the kind.
type bubblesRow struct{}

// bubblesRows is the kind, by name. A file that cannot be read stops the
// build.
func bubblesRows() map[string]maker {
	if bubblesError != nil {
		panic(bubblesError)
	}
	return map[string]maker{BubblesKind: bubblesRow{}}
}

// made is the stream at time nought.
func (row bubblesRow) made(from seed) (good, error) {
	return row.madeAtTime(from, "", 0)
}

// madeAtTime is the stream at a time. The detail is not read.
func (bubblesRow) madeAtTime(from seed, _ string,
	at time.Duration) (good, error) {
	kit, err := bubblesKit()
	if err != nil {
		return good{}, err
	}
	book := bubblesRead
	count := int(book.pick("count", from, 0))
	height := book.pick("height", from, 0)
	members := make([]shape.MeshMember, 0, count)
	for index := range count {
		members = append(members, bubbleAt(book, kit, from, index, height,
			at.Seconds()))
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		return good{}, err
	}
	air := &thing.Material{Name: "air bubble", Colour: maths.Grey(1),
		Gloss: 1, Through: 1, Rank: bubblesRank,
		Index: book.number("air-index")}
	return good{dressed: looks{body: air},
		groups:     map[slot]*shape.MeshGroup{body: group},
		groupParts: map[slot]int{body: len(members)}}, nil
}

// bubbleAt is one bubble placed at a time: its radius, its class, how
// far up the stream it has climbed and how far out it has wandered.
func bubbleAt(book bubblesBook, kit []*mesh.Mesh, from seed, index int,
	height, clock float64) shape.MeshMember {
	radius := book.pick("radius", from, index)
	share := (radius - book.Figures["radius"].Least) /
		(book.Figures["radius"].Most - book.Figures["radius"].Least)
	class := min(bubblesClasses-1, int(share*bubblesClasses))
	climbed := from.draw(growing, index, 0) + book.rise(radius)*clock/height
	up := height * (climbed - math.Floor(climbed))
	turn := fullTurn * from.draw(growing, index, 1)
	wander := book.pick("spread", from, index)*
		math.Sqrt(from.draw(growing, index, 2)) +
		book.pick("flare", from, index)*up
	at := maths.V(math.Cos(turn)*wander, up, math.Sin(turn)*wander)
	return shape.MeshMember{Mesh: kit[class],
		Space: maths.NewSubspace(at, maths.Identity, radius)}
}
