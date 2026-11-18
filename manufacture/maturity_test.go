package manufacture

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/shape"
)

// quickTable is the kinds the plain tests report on: a can, a log, the spacecraft, the
// large observatories, the saucers and the trucks, which are made in a
// few milliseconds each. The whole table takes seconds, for its plants,
// and is reported on in the draw tier.
func quickTable() map[string]maker {
	rows := map[string]maker{"can": canRow(), "log": logRow()}
	for _, more := range []map[string]maker{spacecraftRows(), groundRows(),
		saucerRows(), truckRows()} {
		for name, row := range more {
			rows[name] = row
		}
	}
	rows["space-observatory"] = rows["spitzer"]
	return rows
}

// reportByName reports on quickTable and returns its rows by kind name.
func reportByName(t *testing.T) map[string]Maturity {
	t.Helper()
	report, err := maturityOf(quickTable)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]Maturity{}
	for _, one := range report {
		rows[one.Name] = one
	}
	return rows
}

// TestMaturityListsEveryKindOnceInOrder checks that the report has one row
// for each kind of its table, in alphabetical order.
func TestMaturityListsEveryKindOnceInOrder(t *testing.T) {
	report, err := maturityOf(quickTable)
	if err != nil {
		t.Fatal(err)
	}
	var names, want []string
	for _, one := range report {
		names = append(names, one.Name)
	}
	for name := range quickTable() {
		want = append(want, name)
	}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Fatalf("report names %v, kinds %v", names, want)
	}
}

// TestMaturityCountsTheTelescopesGeometry pins the part counts, looks and
// sizes of the three space telescopes, which were counted by hand from a
// separate probe on 15 November 2026.
func TestMaturityCountsTheTelescopesGeometry(t *testing.T) {
	rows := reportByName(t)
	for _, want := range []struct {
		name  string
		parts int
		size  [3]float64
	}{
		{"spitzer", 831, [3]float64{4.08, 4.90, 2.59}},
		{"hubble", 55, [3]float64{12.00, 16.06, 3.69}},
		{"webb", 90, [3]float64{14.16, 11.64, 21.20}},
	} {
		got := rows[want.name]
		if got.Parts != want.parts || got.Looks != 6 || got.Lights != 0 ||
			got.MeshMembers != 0 || got.MeshFaces != 0 {
			t.Errorf("%s: parts %d looks %d lights %d meshes %d faces %d",
				want.name, got.Parts, got.Looks, got.Lights, got.MeshMembers, got.MeshFaces)
		}
		for axis := range want.size {
			if math.Abs(got.Size[axis]-want.size[axis]) > 0.005 {
				t.Errorf("%s: size %v, want %v", want.name, got.Size, want.size)
			}
		}
		if got.Family != "spacecraft" || !slices.Equal(got.Details, Complexities()) {
			t.Errorf("%s: family %q details %v", want.name, got.Family, got.Details)
		}
	}
}

// TestMaturityCountsMeshesAndLights checks a kind built of meshes and a
// kind that carries lights, so those columns are not always nought.
func TestMaturityCountsMeshesAndLights(t *testing.T) {
	rows := reportByName(t)
	truck := rows["daf-xf"]
	if truck.Parts != 0 || truck.MeshMembers != 26 || truck.MeshFaces != 1488 {
		t.Errorf("truck: parts %d members %d faces %d", truck.Parts, truck.MeshMembers, truck.MeshFaces)
	}
	if truck.Family != "truck" || !slices.Equal(truck.Details, []string{"quick"}) {
		t.Errorf("a truck is of family %q, made at %v", truck.Family, truck.Details)
	}
	if saucer := rows["mcrn-larson"]; saucer.Lights != 14 || saucer.Parts != 15 {
		t.Errorf("saucer: lights %d parts %d", saucer.Lights, saucer.Parts)
	}
}

// TestMaturityGivesEachRowItsOwnMeasuredNumbers checks that a measured
// number read while the table is built is counted for the kind whose row
// read it, and for no other.
func TestMaturityGivesEachRowItsOwnMeasuredNumbers(t *testing.T) {
	rows := reportByName(t)
	// The solar cells made the telescope read the panels' and shield's numbers too.
	spitzerMeasured := []string{"spitzer-active-panel-length-centimetres",
		"spitzer-active-panel-width-centimetres", "spitzer-cell-long-centimetres",
		"spitzer-cell-short-centimetres", "spitzer-cells-per-panel",
		"spitzer-primary-diameter-centimetres", "spitzer-shield-extension-centimetres"}
	for name, want := range map[string][]string{
		"hubble":            {"hubble-primary-diameter-metres"},
		"spitzer":           spitzerMeasured,
		"space-observatory": spitzerMeasured,
		"keck-i":            {"keck-primary-metres", "keck-primary-segments"},
		"keck-ii":           {"keck-primary-metres", "keck-primary-segments"},
		"palomar": {"palomar-enclosure-diameter-metres",
			"palomar-enclosure-height-metres", "palomar-primary-inches"},
		"subaru": {"subaru-enclosure-base-metres",
			"subaru-enclosure-height-metres", "subaru-primary-effective-metres"},
		"rubin": {"rubin-primary-metres", "rubin-secondary-metres", "rubin-tertiary-metres"},
		"elt":   {"elt-primary-metres", "elt-primary-segments"},
		"webb": {"webb-primary-diameter-metres", "webb-primary-segments",
			"webb-secondary-diameter-millimetres", "webb-secondary-vertex-millimetres",
			"webb-shield-layers", "webb-shield-length-metres", "webb-shield-width-metres"},
		"mcrn-larson": nil,
	} {
		if got := rows[name].Measured; !slices.Equal(got, want) {
			t.Errorf("%s measured %v, want %v", name, got, want)
		}
	}
}

// TestMaturitySortsNumbersByWhereTheyCameFrom checks every number the
// report names against its own book: a measured one is in research and
// has a source, and a picked one is in neither.
func TestMaturitySortsNumbersByWhereTheyCameFrom(t *testing.T) {
	measured, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	rows := reportByName(t)
	picked := 0
	for _, one := range rows {
		for _, name := range one.Measured {
			number, err := measured.Get(name)
			if err != nil || number.Kind != numbers.Measured || number.From == nil {
				t.Errorf("%s: %s is listed as measured: %v", one.Name, name, err)
			}
		}
		for _, name := range one.Picked {
			picked++
			if number, err := measured.Get(name); err == nil && number.Kind == numbers.Measured {
				t.Errorf("%s: %s is listed as picked and research measures it", one.Name, name)
			}
		}
		if !slices.IsSorted(one.Measured) || !slices.IsSorted(one.Picked) {
			t.Errorf("%s: numbers are not in order", one.Name)
		}
	}
	if saucer := rows["mcrn-larson"]; len(saucer.Picked) != 43 {
		t.Errorf("the saucer reads %d picked numbers, and read 43", len(saucer.Picked))
	}
	if picked == 0 {
		t.Error("no kind reads a picked number")
	}
}

// TestMaturityIsTheSameTwice checks that a second report equals the
// first, so nothing is left over from one run to the next.
func TestMaturityIsTheSameTwice(t *testing.T) {
	first, err := maturityOf(quickTable)
	if err != nil {
		t.Fatal(err)
	}
	second, err := maturityOf(quickTable)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("two reports of the same table differ")
	}
}

// TestMaturityStopsWatchingWhenItIsDone checks that after a report no
// ledger is left set, so later reads are not recorded anywhere.
func TestMaturityStopsWatchingWhenItIsDone(t *testing.T) {
	if _, err := maturityOf(quickTable); err != nil {
		t.Fatal(err)
	}
	if watching.Load() != nil {
		t.Fatal("a ledger is still set after the report")
	}
	seen := 0
	stop := research.Observe(func(numbers.Number) { seen++ })
	defer stop()
	spacecraftValue("sides-quick")
	research.Value("hubble-primary-diameter-metres")
	if seen != 1 {
		t.Errorf("the observer saw %d reads, want only the one from research", seen)
	}
}

// TestEveryKindHasAFamily checks that no maker falls through to "other".
func TestEveryKindHasAFamily(t *testing.T) {
	for name, how := range table() {
		if family := familyOf(how); family == "other" || family == "" {
			t.Errorf("%s has no family", name)
		}
	}
	if familyOf(nil) != "other" {
		t.Error("a maker nobody knows was given a family")
	}
}

// broken is a maker that can make nothing.
type broken struct{}

// made refuses, as a kind with a fault would.
func (broken) made(seed) (good, error) { return good{}, errors.New("no such thing") }

// TestMaturityRefusesAKindThatCannotBeMade checks that a kind which fails
// at every detail stops the report by name, with no rows returned.
func TestMaturityRefusesAKindThatCannotBeMade(t *testing.T) {
	report, err := maturityOf(func() map[string]maker {
		return map[string]maker{"can": canRow(), "wreck": broken{}}
	})
	if err == nil || !strings.Contains(err.Error(), "wreck can be made at no detail") {
		t.Fatalf("error %v", err)
	}
	if report != nil {
		t.Errorf("a failed report returned %d rows", len(report))
	}
	if watching.Load() != nil {
		t.Error("a ledger is still set after a failed report")
	}
}

// TestLedgerCountsOnlyWhatIsInScope checks the ledger by itself: a read
// outside any scope is dropped, a read inside one is kept for each kind
// named, and an inner scope gives way to the outer one again.
func TestLedgerCountsOnlyWhatIsInScope(t *testing.T) {
	book := &ledger{read: map[string]map[string]numbers.Kind{}}
	number := func(name string, from numbers.Kind) numbers.Number {
		return numbers.Number{Name: name, Kind: from}
	}
	book.add(number("stray", numbers.Measured))
	outer := book.scope([]string{"a", "b"})
	book.add(number("zeta", numbers.Measured))
	book.add(number("alpha", numbers.Measured))
	inner := book.scope([]string{"c"})
	book.add(number("only-c", numbers.Picked))
	book.scope(inner)
	book.add(number("shared", numbers.Picked))
	book.scope(outer)
	book.add(number("late", numbers.Measured))
	if outer != nil || !slices.Equal(inner, []string{"a", "b"}) {
		t.Errorf("scopes before were %v and %v", outer, inner)
	}
	for _, name := range []string{"a", "b"} {
		measured, picked := book.names(name)
		if !slices.Equal(measured, []string{"alpha", "zeta"}) || !slices.Equal(picked, []string{"shared"}) {
			t.Errorf("%s: measured %v picked %v", name, measured, picked)
		}
	}
	if measured, picked := book.names("c"); measured != nil || !slices.Equal(picked, []string{"only-c"}) {
		t.Errorf("c: measured %v picked %v", measured, picked)
	}
	if measured, picked := book.names("nobody"); measured != nil || picked != nil {
		t.Errorf("a kind with no reads has %v and %v", measured, picked)
	}
}

// TestRowOfBuildsWithoutAReport checks that outside a report rowOf only
// builds, and that a book's lookup still answers.
func TestRowOfBuildsWithoutAReport(t *testing.T) {
	if watching.Load() != nil {
		t.Fatal("a report is running")
	}
	built := 0
	row := rowOf(func() broken { built++; return broken{} }, "anything")
	if built != 1 {
		t.Errorf("the row was built %d times", built)
	}
	if _, err := row.made(sow(1)); err == nil {
		t.Error("rowOf did not return the row it built")
	}
	if got := spacecraftValue("sides-quick"); got <= 0 {
		t.Errorf("a lookup outside a report gave %v", got)
	}
}

// TestMaturityMakesEachKindFromSeedOne checks the seed the report says it
// uses, on a kind that differs by seed: a log from seed one is three
// parts and 3.19 m long, and from seed two is five parts and 3.00 m.
func TestMaturityMakesEachKindFromSeedOne(t *testing.T) {
	log := reportByName(t)["log"]
	if log.Parts != 3 || math.Abs(log.Size[0]-3.1857) > 0.0005 {
		t.Errorf("the log has %d parts and is %.4f m long", log.Parts, log.Size[0])
	}
	if log.Family != "fire" {
		t.Errorf("the log is of family %q", log.Family)
	}
}

// triangles gives piecesIn the triangles of a list, three points each.
func triangles(list ...[3]maths.Vec) func(func(a, b, c maths.Vec)) {
	return func(yield func(a, b, c maths.Vec)) {
		for _, one := range list {
			yield(one[0], one[1], one[2])
		}
	}
}

// TestPiecesInJoinsTrianglesThatShareACorner checks the count on shapes
// small enough to count by eye.
func TestPiecesInJoinsTrianglesThatShareACorner(t *testing.T) {
	at := func(x, y float64) maths.Vec { return maths.V(x, y, 0) }
	one := [3]maths.Vec{at(0, 0), at(1, 0), at(0, 1)}
	touching := [3]maths.Vec{at(1, 0), at(2, 0), at(2, 1)}
	apart := [3]maths.Vec{at(5, 5), at(6, 5), at(5, 6)}
	bridge := [3]maths.Vec{at(2, 1), at(5, 5), at(9, 9)}
	hair := [3]maths.Vec{at(0.00004, 0), at(-3, 0), at(-3, 1)}
	near := [3]maths.Vec{at(0.0004, 0), at(-3, -2), at(-3, -1)}
	for _, c := range []struct {
		name string
		list [][3]maths.Vec
		want int
	}{
		{"nothing", nil, 0},
		{"one triangle", [][3]maths.Vec{one}, 1},
		{"two that share a corner", [][3]maths.Vec{one, touching}, 1},
		{"two apart", [][3]maths.Vec{one, apart}, 2},
		{"two apart, joined by a third given last", [][3]maths.Vec{one, touching, apart, bridge}, 1},
		{"the same triangle twice", [][3]maths.Vec{one, one}, 1},
		{"a corner a twenty-fifth of a millimetre off is the same corner", [][3]maths.Vec{one, hair}, 1},
		{"a corner four tenths of a millimetre off is another", [][3]maths.Vec{one, near}, 2},
	} {
		if got := piecesIn(triangles(c.list...)); got != c.want {
			t.Errorf("%s: %d pieces, want %d", c.name, got, c.want)
		}
	}
}

// TestMaturityCountsPiecesAndNotCuts checks the pieces of kinds whose
// mirrors are segmented, counted on 15 November 2026, and that every
// kind of the quick table but the trucks has the same pieces at every
// detail: cutting finer makes no hardware.
func TestMaturityCountsPiecesAndNotCuts(t *testing.T) {
	rows := reportByName(t)
	for name, want := range map[string]int{"elt": 972, "keck-i": 205, "webb": 107,
		"hubble": 55, "rubin": 300, "daf-xf": 26, "log": 3} {
		if got := rows[name].Pieces; got != want {
			t.Errorf("%s: %d pieces, want %d", name, got, want)
		}
	}
	if elt := rows["elt"]; elt.Pieces <= elt.Parts {
		t.Errorf("the segmented ELT has %d pieces in %d parts", elt.Pieces, elt.Parts)
	}
	for name, how := range quickTable() {
		row := kind{name: name, how: how}
		for _, detail := range rows[name].Details {
			made, err := MakeWithComplexity(row, sow(1), detail)
			if err != nil {
				t.Fatal(err)
			}
			if got := piecesOf(made); got != rows[name].Pieces {
				t.Errorf("%s at %s has %d pieces, and %d at quick", name, detail, got, rows[name].Pieces)
			}
		}
	}
}

// TestMeshPiecesOfAShapeWithNoTriangles checks that a shape which cannot
// give up its triangles is counted as one piece.
func TestMeshPiecesOfAShapeWithNoTriangles(t *testing.T) {
	if got := meshPieces(shape.Sphere{Radius: 1}); got != 1 {
		t.Errorf("a sphere is %d pieces", got)
	}
}

// TestPiecesInTellsCornersApartOnEveryAxis checks that two triangles whose
// nearest corners are four tenths of a millimetre apart are two pieces
// whichever way the gap runs, and one piece when the gap is closed.
func TestPiecesInTellsCornersApartOnEveryAxis(t *testing.T) {
	base := [3]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	for axis, gap := range []maths.Vec{maths.V(0.0004, 0, 0), maths.V(0, 0.0004, 0), maths.V(0, 0, 0.0004)} {
		other := [3]maths.Vec{gap, maths.V(-3, -2, -5), maths.V(-3, -1, -5)}
		if got := piecesIn(triangles(base, other)); got != 2 {
			t.Errorf("axis %d: a gap of 0.4 mm gave %d pieces", axis, got)
		}
		closed := [3]maths.Vec{gap.Scale(0.1), maths.V(-3, -2, -5), maths.V(-3, -1, -5)}
		if got := piecesIn(triangles(base, closed)); got != 1 {
			t.Errorf("axis %d: a gap of 0.04 mm gave %d pieces", axis, got)
		}
	}
}

// loose is a form whose skin is the triangles it holds, for a test.
type loose surface

// skin is the triangles as they are, whatever the cut.
func (form loose) skin(cut) surface { return surface(form) }

// TestPiecesOfJoinsAPartThroughAnyCorner checks that a part's triangles
// are joined whichever of a triangle's three corners is the shared one,
// and that two parts are counted apart even where they touch.
func TestPiecesOfJoinsAPartThroughAnyCorner(t *testing.T) {
	a, b, c := maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)
	far := func(n float64) (maths.Vec, maths.Vec) { return maths.V(9, n, 0), maths.V(9, n, 1) }
	for corner := range 3 {
		p, q := far(float64(corner))
		joined := face{p, q, p}
		joined[corner] = a
		others := face{maths.V(5, 5, 5), maths.V(6, 5, 5), maths.V(5, 6, 5)}
		others[corner] = b
		made := good{parts: []part{{shape: loose{{a, b, c}, joined, others}}}}
		if got := piecesOf(made); got != 1 {
			t.Errorf("triangles joined through corner %d are %d pieces", corner, got)
		}
	}
	two := good{parts: []part{
		{shape: loose{{a, b, c}}},
		{shape: loose{{a, b, c}}}}}
	if got := piecesOf(two); got != 2 {
		t.Errorf("two parts in one place are %d pieces", got)
	}
}
