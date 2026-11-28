package manufacture

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Maturity is what can be counted about one kind without judging it: how
// much geometry it is made of, how many ways it can be cut, and which
// named numbers it reads, measured or picked.
//
// It says how much of a model there is and where its numbers came from.
// It does not say the model is right: no dimension here is compared with
// the thing itself.
type Maturity struct {
	// Name is the kind's name, as KindNamed takes it.
	Name string
	// Family groups kinds by what their makers build: "spacecraft",
	// "submarine", "summit instrument".
	Family string
	// Parts is the count of separate solids at quick detail.
	Parts int
	// MeshMembers and MeshFaces count the members and faces of the
	// kind's mesh groups at quick detail. A kind built of parts alone
	// has none.
	MeshMembers, MeshFaces int
	// Pieces is the count of separate constructed pieces at quick
	// detail: triangles that share a corner, directly or through others,
	// are one piece. A segmented mirror is one part and many pieces. The
	// count does not change when a round surface is cut finer; for a
	// plant it counts leaves and grows with the detail.
	Pieces int
	// Looks is the count of surfaces the kind wears, and Lights the
	// count of lights it carries.
	Looks, Lights int
	// Details are the representations the kind can be made at, in the
	// order of Complexities.
	Details []string
	// Size is the extent of the kind's own box at quick detail, in
	// metres: across, up, and deep.
	Size [3]float64
	// Measured and Picked are the names of the numbers the kind reads,
	// in order. A measured number has a pinned source; a picked one was
	// chosen by whoever its record names. See MaturityReport for which
	// reads are counted.
	Measured, Picked []string
}

// MaturityReport measures every kind, in alphabetical order.
//
// Each kind is made once at every detail it supports, from seed one. Its
// numbers are those read from research and from the manufactory's own
// books of choices while it is made, and those its row read when the
// table of kinds was built, where the row says whose they are. Numbers a
// kind takes from a file that is not a book of named numbers, as the
// lunar module and the stacks do, are not counted.
//
// The report watches every read in the process while it runs, so kinds
// made elsewhere at the same moment would be counted with it. It is for
// a report, not for a renderer's inner loop.
func MaturityReport() ([]Maturity, error) { return maturityOf(table) }

// maturityOf measures the kinds of a table. The table is built after
// the watch is set, so that reads made in building it are seen.
func maturityOf(build func() map[string]maker) ([]Maturity, error) {
	reporting.Lock()
	defer reporting.Unlock()
	book := &ledger{read: map[string]map[string]numbers.Kind{}}
	watching.Store(book)
	defer watching.Store(nil)
	defer research.Observe(func(number numbers.Number) { book.add(number) })()
	rows := build()
	names := make([]string, 0, len(rows))
	for name := range rows {
		names = append(names, name)
	}
	sort.Strings(names)
	report := make([]Maturity, 0, len(names))
	for _, name := range names {
		row := kind{name: name, how: rows[name]}
		one := Maturity{Name: name, Family: familyOf(row.how)}
		book.scope([]string{name})
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, sow(1), detail)
			if err != nil {
				continue
			}
			one.Details = append(one.Details, detail)
			if detail == "quick" {
				one.measure(made)
			}
		}
		book.scope(nil)
		if len(one.Details) == 0 {
			return nil, fmt.Errorf("manufacture: %s can be made at no detail", name)
		}
		one.Measured, one.Picked = book.names(name)
		report = append(report, one)
	}
	return report, nil
}

// measure counts the geometry of one made specimen.
func (one *Maturity) measure(made good) {
	one.Parts, one.Looks, one.Lights = len(made.parts)+len(made.native), len(made.dressed), len(made.lights)
	// Native attachments can share one material: count each shared object once.
	looks := map[*thing.Material]bool{}
	for _, part := range made.native {
		if part.Material != nil {
			looks[part.Material] = true
		}
	}
	one.Looks += len(looks)
	one.Pieces = piecesOf(made)
	for _, group := range made.groups {
		one.MeshMembers += len(group.Members())
		one.MeshFaces += group.FaceCount()
	}
	box := made.Bounds()
	one.Size = [3]float64{box.Max.X - box.Min.X, box.Max.Y - box.Min.Y,
		box.Max.Z - box.Min.Z}
}

// reporting lets one report run at a time, and watching is the ledger
// that report is filling, or nil when none runs.
var (
	reporting sync.Mutex
	watching  atomic.Pointer[ledger]
)

// ledger is the numbers read, by the kind they were read for.
type ledger struct {
	lock sync.Mutex
	// whose is the kinds the reads now happening belong to.
	whose []string
	read  map[string]map[string]numbers.Kind
}

// scope names the kinds that reads belong to from now on, and says whose
// they were before, so a caller can put that back.
func (book *ledger) scope(kinds []string) (before []string) {
	book.lock.Lock()
	defer book.lock.Unlock()
	before, book.whose = book.whose, kinds
	return before
}

// add records one number against every kind now in scope.
func (book *ledger) add(number numbers.Number) {
	book.lock.Lock()
	defer book.lock.Unlock()
	for _, name := range book.whose {
		if book.read[name] == nil {
			book.read[name] = map[string]numbers.Kind{}
		}
		book.read[name][number.Name] = number.Kind
	}
}

// names is the measured and the picked numbers read for a kind, sorted.
func (book *ledger) names(name string) (measured, picked []string) {
	book.lock.Lock()
	defer book.lock.Unlock()
	for number, from := range book.read[name] {
		if from == numbers.Measured {
			measured = append(measured, number)
		} else {
			picked = append(picked, number)
		}
	}
	sort.Strings(measured)
	sort.Strings(picked)
	return measured, picked
}

// noted tells a running report that a book of choices gave out a number.
func noted(number *numbers.Number) {
	if book := watching.Load(); book != nil {
		book.add(*number)
	}
}

// rowOf builds a row and, while a report runs, counts the numbers the
// building reads for the kinds named. A row that reads measured numbers
// as the table is built says whose they are by building itself here.
func rowOf[row maker](build func() row, kinds ...string) row {
	book := watching.Load()
	if book == nil {
		return build()
	}
	defer book.scope(book.scope(kinds))
	return build()
}

// familyOf is the family a maker's kinds belong to.
func familyOf(how maker) string {
	switch how.(type) {
	case spacecraft, lunarModule, saucer, veneraLander,
		soho, mariner10, jimo, cassiniHuygens, newHorizons, newHorizonsVTAD, voyagerMaker:
		return "spacecraft"
	case alfaBoat:
		return "submarine"
	case boatRow:
		return "boat"
	case j58Engine:
		return "engine"
	case scaleHuman:
		return "scale reference"
	case whaleSwim, whaleStill, reptileStudy, sealKind,
		birdKind, fishRow, snakeKind, sauropodKind, adultHumanStudy,
		adultHumanReachClip, adultIrisStudy, chestMaterialStudy, nativeChestStudy:
		return "animal"
	case tissueStudy:
		return "material study"
	case hatzGargoyleKind, gargoyleRow, gargoyleCorniceRow:
		return "architectural ornament"
	case bubblesRow:
		return "water"
	case stack:
		return "launch stack"
	case groundObservatory, owlObservatory, civicObservatory, optical:
		return "observatory"
	case summitInstrument:
		return "summit instrument"
	case personalTelescope:
		return "personal telescope"
	case physicalSpruce, conifer, seasonalPlant, columnarCactus, shootKind, fern:
		return "plant"
	case kelpRow:
		return "alga"
	case yardEquipment, owlShelter, guideLauncher, scaffolding, j58Stand:
		return "site equipment"
	case truck:
		return "truck"
	case beamhead, spot, maser:
		return "light"
	case fire, pyre, log, can:
		return "fire"
	case stone:
		return "stone"
	case macrostructureMaker:
		return "macrostructure"
	case landformMaker:
		return "landform"
	case cityFamily, frontageRow, retailBlockRow:
		return "city building"
	case cityMaker, clusteredCityMaker, scavengedRow, superblockRow,
		pierRow, villageRow, streetRow, lashedStreetRow, signRow:
		return "city"
	case home, yurtRow, winnebagoRow, bivouacRow, terraceRow, condominiumRow,
		villageHouseRow:
		return "home"
	case towerRow:
		return "city building"
	case bridgeRow:
		return "city"
	}
	return "other"
}

// piecesIn counts the separate pieces of a set of triangles: two triangles
// are of one piece if they share a corner, or are joined through others
// that do. Cutting a round surface finer adds triangles and no pieces, so
// the count is of what was constructed and not of how it was cut.
func piecesIn(corners func(yield func(a, b, c maths.Vec))) int {
	// A corner is found again if it lies within a tenth of a millimetre.
	const grain = 1e4
	type place [3]int64
	index := map[place]int{}
	var parent []int
	// A chain to a root is no longer than the corners there are, so the
	// walk is bounded by them and cannot run on.
	find := func(at int) int {
		for range parent {
			if parent[at] == at {
				break
			}
			parent[at] = parent[parent[at]]
			at = parent[at]
		}
		return at
	}
	seen := func(point maths.Vec) int {
		key := place{int64(math.Round(point.X * grain)),
			int64(math.Round(point.Y * grain)), int64(math.Round(point.Z * grain))}
		at, known := index[key]
		if !known {
			at = len(parent)
			index[key] = at
			parent = append(parent, at)
		}
		return at
	}
	corners(func(a, b, c maths.Vec) {
		first := find(seen(a))
		parent[find(seen(b))] = first
		parent[find(seen(c))] = first
	})
	pieces := 0
	for at := range parent {
		if parent[at] == at {
			pieces++
		}
	}
	return pieces
}

// piecesOf is the count of separate pieces in a made specimen: each
// part's skin and each distinct mesh is counted by itself, and a mesh
// that stands in several places is counted once for each.
func piecesOf(made good) int {
	total := 0
	for _, native := range made.native {
		total += meshPieces(native.Shape)
	}
	for _, piece := range made.parts {
		skin := piece.shape.skin(piece.fine)
		total += piecesIn(func(yield func(a, b, c maths.Vec)) {
			for _, one := range skin {
				yield(one[0], one[1], one[2])
			}
		})
	}
	counted := map[shape.Shape]int{}
	for _, group := range made.groups {
		for _, member := range group.Members() {
			pieces, known := counted[member.Mesh]
			if !known {
				pieces = meshPieces(member.Mesh)
				counted[member.Mesh] = pieces
			}
			total += pieces
		}
	}
	return total
}

// faced is a mesh that gives up its triangles, as package mesh's does.
type faced interface {
	FaceCount() int
	Corners(i int) (maths.Vec, maths.Vec, maths.Vec)
}

// meshPieces is the separate pieces of one mesh. A shape that does not
// give up its triangles is one piece.
func meshPieces(form shape.Shape) int {
	mesh, open := form.(faced)
	if !open {
		return 1
	}
	return piecesIn(func(yield func(a, b, c maths.Vec)) {
		for at := range mesh.FaceCount() {
			yield(mesh.Corners(at))
		}
	})
}
