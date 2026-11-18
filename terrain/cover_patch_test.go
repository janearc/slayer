package terrain

import (
	"errors"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
)

// rampSpec is a five by four projected raster, 100 east and 50 north, one
// metre cells, height column plus ten row.
func rampSpec() *tiffSpec {
	return newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(100, 50, 1, 1)
}

// TestRootPatchSpansTheLattice checks the root of a placed raster and the
// refusal of one that is unplaced or too narrow.
func TestRootPatchSpansTheLattice(t *testing.T) {
	root, err := openSpec(t, rampSpec()).RootPatch()
	if err != nil || root != (Patch{MaxColumn: 4, MaxRow: 3}) {
		t.Errorf("root %+v, %v", root, err)
	}
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt))
	if _, err := unplaced.RootPatch(); err == nil {
		t.Error("an unplaced raster gave a root")
	}
	narrow := openSpec(t, newTiffSpec(tiffLittle, false, 1, 4, 4, 4, rampAt).place(0, 0, 1, 1))
	if _, err := narrow.RootPatch(); err == nil {
		t.Error("a one-column raster gave a root")
	}
}

// TestPatchChildrenShareEdges checks the split of a patch into four, a narrow
// strip into two, and the cases that give none or an error.
func TestPatchChildrenShareEdges(t *testing.T) {
	got, err := (Patch{MaxColumn: 4, MaxRow: 3}).Children()
	want := []Patch{{MaxColumn: 2, MaxRow: 1, Level: 1}, {MinColumn: 2, MaxColumn: 4, MaxRow: 1, Level: 1},
		{MaxColumn: 2, MinRow: 1, MaxRow: 3, Level: 1}, {MinColumn: 2, MaxColumn: 4, MinRow: 1, MaxRow: 3, Level: 1}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("children %+v, %v", got, err)
	}
	got, err = (Patch{MaxColumn: 1, MaxRow: 4, Level: 2}).Children()
	want = []Patch{{MaxColumn: 1, MaxRow: 2, Level: 3}, {MaxColumn: 1, MinRow: 2, MaxRow: 4, Level: 3}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("strip children %+v, %v", got, err)
	}
	got, err = (Patch{MaxColumn: 1, MaxRow: 1}).Children()
	if err != nil || got != nil {
		t.Errorf("a single cell split into %+v, %v", got, err)
	}
	deep := Patch{MaxColumn: 4, MaxRow: 4, Level: strconv.IntSize - 1}
	if _, err := deep.Children(); err == nil || !strings.Contains(err.Error(), "depth") {
		t.Errorf("a patch at the deepest level split: %v", err)
	}
	for name, bad := range map[string]Patch{
		"negative column": {MinColumn: -1, MaxColumn: 2, MaxRow: 2},
		"negative row":    {MinRow: -1, MaxColumn: 2, MaxRow: 2},
		"empty columns":   {MinColumn: 2, MaxColumn: 2, MaxRow: 2},
		"empty rows":      {MaxColumn: 2, MinRow: 2, MaxRow: 2},
		"negative level":  {MaxColumn: 2, MaxRow: 2, Level: -1},
		"level too large": {MaxColumn: 2, MaxRow: 2, Level: strconv.IntSize},
	} {
		if _, err := bad.Children(); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
}

// TestPatchGridReadsNativeSamples checks the sample indices, coordinates and
// heights of a coarse read, and the accessors that describe it.
func TestPatchGridReadsNativeSamples(t *testing.T) {
	raster := openSpec(t, rampSpec())
	patch := Patch{MaxColumn: 4, MaxRow: 3}
	grid, err := raster.ReadPatch(patch, 3, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	across, down := grid.Size()
	if across != 3 || down != 3 || grid.Patch() != patch || grid.Reference() != (SourceReference{}) {
		t.Errorf("grid %d by %d, patch %+v, reference %+v", across, down, grid.Patch(), grid.Reference())
	}
	east, north, height, available := grid.At(1, 1)
	if east != 102.5 || north != 48.5 || height != 12 || !available {
		t.Errorf("middle sample %v %v %v %v", east, north, height, available)
	}
	east, north, height, available = grid.At(2, 2)
	if east != 104.5 || north != 46.5 || height != 34 || !available {
		t.Errorf("last sample %v %v %v %v", east, north, height, available)
	}
	for _, at := range [][2]int{{-1, 0}, {0, -1}, {3, 0}, {0, 3}} {
		if east, north, height, available := grid.At(at[0], at[1]); east != 0 || north != 0 || height != 0 || available {
			t.Errorf("outside at %v: %v %v %v %v", at, east, north, height, available)
		}
	}
	point := openSpec(t, rampSpec().geo([4]uint16{1025, 0, 1, 2}))
	grid, err = point.ReadPatch(patch, 3, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if east, north, _, _ := grid.At(1, 1); east != 102 || north != 49 {
		t.Errorf("a point registered grid is at %v, %v; want 102, 49", east, north)
	}
}

// TestPatchGridRowSpacingFallsBackToSpacing checks that a grid built without
// a row scale uses its column spacing for rows.
func TestPatchGridRowSpacingFallsBackToSpacing(t *testing.T) {
	grid := PatchGrid{west: 10, north: 20, spacing: 2, columns: []int{0, 1}, rows: []int{0, 1},
		heights: []float64{1, 2, 3, 4}, present: []bool{true, true, true, true}}
	if east, north, height, ok := grid.At(1, 1); east != 13 || north != 17 || height != 4 || !ok {
		t.Errorf("got %v %v %v %v", east, north, height, ok)
	}
}

// TestReadPatchRefusesWhatItCannotRead checks each refusal of a patch read.
func TestReadPatchRefusesWhatItCannotRead(t *testing.T) {
	raster := openSpec(t, rampSpec())
	root := Patch{MaxColumn: 4, MaxRow: 3}
	cases := []struct {
		name  string
		patch Patch
		side  int
		limit int64
	}{
		{"invalid patch", Patch{MaxColumn: 0, MaxRow: 3}, 3, 1 << 20},
		{"past the last column", Patch{MaxColumn: 5, MaxRow: 3}, 3, 1 << 20},
		{"past the last row", Patch{MaxColumn: 4, MaxRow: 4}, 3, 1 << 20},
		{"side of one", root, 1, 1 << 20},
		{"limit under the columns", root, 3, 23},
		{"limit under the rows", root, 3, 47},
		{"limit under the samples", root, 3, 48 + 9*9 - 1},
	}
	for _, each := range cases {
		if _, err := raster.ReadPatch(each.patch, each.side, each.limit); err == nil {
			t.Errorf("%s: no refusal", each.name)
		}
	}
	if _, err := raster.ReadPatch(root, 3, 48+9*9); err != nil {
		t.Errorf("a limit that exactly fits was refused: %v", err)
	}
	var none *Raster
	if _, err := none.ReadPatch(root, 3, 1<<20); err == nil {
		t.Error("a nil raster was read")
	}
}

// TestPatchCapacityCountsEveryArray checks the byte arithmetic on a 64-bit
// build: eight bytes per index and nine per sample.
func TestPatchCapacityCountsEveryArray(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("hand worked counts assume 8-byte indices")
	}
	cases := []struct {
		across, down int
		limit        int64
		want         int
		fails        string
	}{
		{2, 2, 68, 4, ""},
		{2, 2, 67, 0, "samples exceed"},
		{2, 2, 32, 0, "samples exceed"},
		{2, 2, 31, 0, "axes exceed"},
		{2, 2, 15, 0, "output byte budget"},
		{2, 2, -1, 0, "output byte budget"},
		{1, 2, 1000, 0, "output byte budget"},
		{2, 1, 1000, 0, "output byte budget"},
	}
	for _, each := range cases {
		got, err := patchCapacity(each.across, each.down, each.limit)
		if each.fails == "" {
			if err != nil || got != each.want {
				t.Errorf("%+v: %d, %v", each, got, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), each.fails) {
			t.Errorf("%+v: %d, %v", each, got, err)
		}
	}
}

// TestReadPatchReportsAnUnreadableTile checks a corrupt source is an error.
func TestReadPatchReportsAnUnreadableTile(t *testing.T) {
	path := rampSpec().write(t)
	raster, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	if err := os.Truncate(path, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := raster.ReadPatch(Patch{MaxColumn: 4, MaxRow: 3}, 3, 1<<20); err == nil {
		t.Error("a patch was read from a cut file")
	}
	cache, err := NewPatchCache(raster, PatchCacheBudget{Bytes: 1 << 20, Entries: 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Read(Patch{MaxColumn: 4, MaxRow: 3}, 3, 1<<20); err == nil {
		t.Error("a cached read of a cut file succeeded")
	}
	if bytes, entries, hits, misses := cache.Usage(); bytes != 0 || entries != 0 || hits != 0 || misses != 1 {
		t.Errorf("usage after a failed read: %d %d %d %d", bytes, entries, hits, misses)
	}
}

// TestNewPatchCacheRefusesBadSources checks each refusal at construction, and
// that an unchecked cache refuses to read.
func TestNewPatchCacheRefusesBadSources(t *testing.T) {
	raster := openSpec(t, rampSpec())
	closed := openSpec(t, rampSpec())
	closed.Close()
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt))
	for name, call := range map[string]func() (*PatchCache, error){
		"nil raster":       func() (*PatchCache, error) { return NewPatchCache(nil, PatchCacheBudget{}) },
		"closed raster":    func() (*PatchCache, error) { return NewPatchCache(closed, PatchCacheBudget{}) },
		"unplaced raster":  func() (*PatchCache, error) { return NewPatchCache(unplaced, PatchCacheBudget{}) },
		"negative bytes":   func() (*PatchCache, error) { return NewPatchCache(raster, PatchCacheBudget{Bytes: -1}) },
		"negative entries": func() (*PatchCache, error) { return NewPatchCache(raster, PatchCacheBudget{Entries: -1}) },
	} {
		if cache, err := call(); err == nil || cache != nil {
			t.Errorf("%s: %v %v", name, cache, err)
		}
	}
	var none *PatchCache
	if _, err := none.Read(Patch{MaxColumn: 4, MaxRow: 3}, 3, 1<<20); err == nil {
		t.Error("a nil cache read")
	}
	if _, err := (&PatchCache{}).Read(Patch{MaxColumn: 4, MaxRow: 3}, 3, 1<<20); err == nil {
		t.Error("a cache built without its constructor read")
	}
}

// meshGrid reads a three by three native grid of the ramp raster.
func meshGrid(t *testing.T, spec *tiffSpec) PatchGrid {
	t.Helper()
	grid, err := openSpec(t, spec).ReadPatch(Patch{MaxColumn: 4, MaxRow: 3}, 3, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return grid
}

// identityMap puts east on x, height on y and north on z.
func identityMap(east, north, height float64) (maths.Vec, error) {
	return maths.V(east, height, north), nil
}

// TestPatchMeshBuildsTrianglesFromAvailableCorners checks the face count and
// bounds of a mesh over a full grid, and that a missing sample removes only
// the triangles that use it.
func TestPatchMeshBuildsTrianglesFromAvailableCorners(t *testing.T) {
	grid := meshGrid(t, rampSpec())
	made, ok, err := grid.Mesh(MeshBudget{Points: 9, Faces: 8}, identityMap)
	if err != nil || !ok || made.FaceCount() != 8 {
		t.Fatalf("full mesh: %v %v %v", made, ok, err)
	}
	box := made.Bounds()
	if box.Min.X != 100.5 || box.Max.X != 104.5 || box.Min.Y != 0 || box.Max.Y != 34 ||
		box.Min.Z != 46.5 || box.Max.Z != 49.5 {
		t.Errorf("bounds %+v", box)
	}
	hole := meshGrid(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, func(column, row int) float32 {
		if column == 4 && row == 0 {
			return -9999
		}
		return rampAt(column, row)
	}).place(100, 50, 1, 1).ascii(tagNoData, "-9999"))
	made, ok, err = hole.Mesh(MeshBudget{Points: 9, Faces: 8}, identityMap)
	if err != nil || !ok || made.FaceCount() != 7 {
		t.Fatalf("mesh with a hole: %v %v %v", made, ok, err)
	}
}

// TestPatchMeshReportsAbsenceAndRefusals checks a mesh with no available
// triangle, each count refusal and each mapping failure.
func TestPatchMeshReportsAbsenceAndRefusals(t *testing.T) {
	grid := meshGrid(t, rampSpec())
	empty := grid
	empty.present = make([]bool, 9)
	if made, ok, err := empty.Mesh(MeshBudget{Points: 9, Faces: 8}, identityMap); made != nil || ok || err != nil {
		t.Errorf("an empty grid gave %v %v %v", made, ok, err)
	}
	budgets := map[string]MeshBudget{
		"no points":       {Points: 0, Faces: 8},
		"no faces":        {Points: 9, Faces: 0},
		"points short":    {Points: 8, Faces: 8},
		"faces short":     {Points: 9, Faces: 7},
		"rows over faces": {Points: 9, Faces: 3},
	}
	for name, budget := range budgets {
		if _, _, err := grid.Mesh(budget, identityMap); err == nil || !strings.Contains(err.Error(), "count budget") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, _, err := grid.Mesh(MeshBudget{Points: 9, Faces: 8}, nil); err == nil {
		t.Error("a nil mapper was accepted")
	}
	want := errors.New("no projection")
	failing := func(float64, float64, float64) (maths.Vec, error) { return maths.Vec{}, want }
	if _, _, err := grid.Mesh(MeshBudget{Points: 9, Faces: 8}, failing); !errors.Is(err, want) {
		t.Errorf("a mapper error: %v", err)
	}
	for name, point := range map[string]maths.Vec{
		"NaN x":      maths.V(math.NaN(), 0, 0),
		"infinite y": maths.V(0, math.Inf(1), 0),
		"infinite z": maths.V(0, 0, math.Inf(-1)),
	} {
		mapped := func(float64, float64, float64) (maths.Vec, error) { return point, nil }
		if _, _, err := grid.Mesh(MeshBudget{Points: 9, Faces: 8}, mapped); err == nil ||
			!strings.Contains(err.Error(), "not finite") {
			t.Errorf("%s: %v", name, err)
		}
	}
	flat := func(float64, float64, float64) (maths.Vec, error) { return maths.V(1, 1, 1), nil }
	if _, ok, err := grid.Mesh(MeshBudget{Points: 9, Faces: 8}, flat); err == nil || ok {
		t.Errorf("faces with no area: %v %v", ok, err)
	}
}

// patchWidth scores a patch by its width times ten, and sees it if asked.
func patchWidth(visible func(Patch) bool) func(Patch) (PatchView, error) {
	return func(patch Patch) (PatchView, error) {
		return PatchView{Visible: visible(patch), Score: float64(patch.MaxColumn-patch.MinColumn) * 10}, nil
	}
}

// always sees every patch.
func always(Patch) bool { return true }

// TestSelectPatchesRefinesUntilTheTargetIsMet checks the chosen patches,
// their order and the work counted for a view whose score halves with width.
func TestSelectPatchesRefinesUntilTheTargetIsMet(t *testing.T) {
	budget := SelectionBudget{Patches: 16, Evaluations: 100, Depth: 5, Target: 15}
	got, err := SelectPatches(Patch{MaxColumn: 4, MaxRow: 4}, budget, patchWidth(always))
	if err != nil || got.Limited || got.Evaluations != 21 || len(got.Patches) != 16 {
		t.Fatalf("selection %+v, %v", got, err)
	}
	first, last := got.Patches[0], got.Patches[15]
	if first != (Patch{MaxColumn: 1, MaxRow: 1, Level: 2}) || last != (Patch{MinColumn: 3, MaxColumn: 4, MinRow: 3, MaxRow: 4, Level: 2}) {
		t.Errorf("first %+v, last %+v", first, last)
	}
	for i := 1; i < len(got.Patches); i++ {
		if !patchBefore(got.Patches[i-1], got.Patches[i]) {
			t.Errorf("patches %d and %d are out of order", i-1, i)
		}
	}
}

// TestSelectPatchesDropsInvisibleChildrenAndMarksLimits checks that unseen
// children are not kept, and that each limit marks the selection limited.
func TestSelectPatchesDropsInvisibleChildrenAndMarksLimits(t *testing.T) {
	root := Patch{MaxColumn: 4, MaxRow: 4}
	west := func(patch Patch) bool { return patch.MinColumn == 0 }
	got, err := SelectPatches(root, SelectionBudget{Patches: 4, Evaluations: 100, Depth: 1, Target: 1}, patchWidth(west))
	want := []Patch{{MaxColumn: 2, MaxRow: 2, Level: 1}, {MaxColumn: 2, MinRow: 2, MaxRow: 4, Level: 1}}
	if err != nil || !got.Limited || got.Evaluations != 5 || !slices.Equal(got.Patches, want) {
		t.Errorf("west only: %+v, %v", got, err)
	}
	got, err = SelectPatches(root, SelectionBudget{Patches: 4, Evaluations: 100, Depth: 0, Target: 1}, patchWidth(always))
	if err != nil || !got.Limited || !slices.Equal(got.Patches, []Patch{root}) {
		t.Errorf("depth zero: %+v, %v", got, err)
	}
	cell := Patch{MaxColumn: 1, MaxRow: 1}
	got, err = SelectPatches(cell, SelectionBudget{Patches: 4, Evaluations: 100, Depth: 4, Target: 1}, patchWidth(always))
	if err != nil || !got.Limited || !slices.Equal(got.Patches, []Patch{cell}) {
		t.Errorf("a single cell: %+v, %v", got, err)
	}
	got, err = SelectPatches(root, SelectionBudget{Patches: 4, Evaluations: 4, Depth: 4, Target: 1}, patchWidth(always))
	if err != nil || !got.Limited || got.Evaluations != 1 || !slices.Equal(got.Patches, []Patch{root}) {
		t.Errorf("too few evaluations: %+v, %v", got, err)
	}
	got, err = SelectPatches(root, SelectionBudget{Patches: 3, Evaluations: 100, Depth: 4, Target: 1}, patchWidth(always))
	if err != nil || !got.Limited || got.Evaluations != 5 || !slices.Equal(got.Patches, []Patch{root}) {
		t.Errorf("too few patches: %+v, %v", got, err)
	}
	got, err = SelectPatches(root, SelectionBudget{Patches: 4, Evaluations: 100, Depth: 4, Target: 1}, patchWidth(func(Patch) bool { return false }))
	if err != nil || got.Limited || got.Evaluations != 1 || len(got.Patches) != 0 {
		t.Errorf("nothing seen: %+v, %v", got, err)
	}
}

// TestSelectPatchesRefusesBadInput checks each budget refusal, each view
// failure and each invalid score.
func TestSelectPatchesRefusesBadInput(t *testing.T) {
	root := Patch{MaxColumn: 4, MaxRow: 4}
	good := SelectionBudget{Patches: 4, Evaluations: 10, Depth: 2, Target: 1}
	with := func(edit func(*SelectionBudget)) SelectionBudget {
		budget := good
		edit(&budget)
		return budget
	}
	view := patchWidth(always)
	for name, budget := range map[string]SelectionBudget{
		"no patches":       with(func(b *SelectionBudget) { b.Patches = 0 }),
		"no evaluations":   with(func(b *SelectionBudget) { b.Evaluations = 0 }),
		"depth below root": with(func(b *SelectionBudget) { b.Depth = -1 }),
		"depth too large":  with(func(b *SelectionBudget) { b.Depth = strconv.IntSize }),
		"NaN target":       with(func(b *SelectionBudget) { b.Target = math.NaN() }),
		"zero target":      with(func(b *SelectionBudget) { b.Target = 0 }),
	} {
		if _, err := SelectPatches(root, budget, view); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
	if _, err := SelectPatches(Patch{}, good, view); err == nil {
		t.Error("an empty root was accepted")
	}
	if _, err := SelectPatches(root, good, nil); err == nil {
		t.Error("a nil view was accepted")
	}
	want := errors.New("no view")
	if _, err := SelectPatches(root, good, func(Patch) (PatchView, error) { return PatchView{}, want }); !errors.Is(err, want) {
		t.Errorf("a root view error: %v", err)
	}
	childFails := func(patch Patch) (PatchView, error) {
		if patch.Level > 0 {
			return PatchView{}, want
		}
		return PatchView{Visible: true, Score: 10}, nil
	}
	if _, err := SelectPatches(root, good, childFails); !errors.Is(err, want) {
		t.Errorf("a child view error: %v", err)
	}
	for name, score := range map[string]float64{"NaN": math.NaN(), "infinite": math.Inf(1), "negative": -1} {
		bad := func(Patch) (PatchView, error) { return PatchView{Visible: true, Score: score}, nil }
		if _, err := SelectPatches(root, good, bad); err == nil || !strings.Contains(err.Error(), "invalid patch view score") {
			t.Errorf("%s score: %v", name, err)
		}
	}
}

// TestPatchOrderComparesEachFieldInTurn checks the order of patches field by
// field, with rows first, and that equal patches are not before each other.
func TestPatchOrderComparesEachFieldInTurn(t *testing.T) {
	base := Patch{MinColumn: 1, MaxColumn: 3, MinRow: 1, MaxRow: 3, Level: 1}
	cases := []struct {
		name  string
		other Patch
	}{
		{"min row", Patch{MinColumn: 0, MaxColumn: 3, MinRow: 2, MaxRow: 3, Level: 1}},
		{"min column", Patch{MinColumn: 2, MaxColumn: 3, MinRow: 1, MaxRow: 3, Level: 1}},
		{"max row", Patch{MinColumn: 1, MaxColumn: 3, MinRow: 1, MaxRow: 4, Level: 1}},
		{"max column", Patch{MinColumn: 1, MaxColumn: 4, MinRow: 1, MaxRow: 3, Level: 1}},
		{"level", Patch{MinColumn: 1, MaxColumn: 3, MinRow: 1, MaxRow: 3, Level: 2}},
	}
	for _, each := range cases {
		if !patchBefore(base, each.other) || patchBefore(each.other, base) {
			t.Errorf("%s: base should come first only", each.name)
		}
	}
	if patchBefore(base, base) {
		t.Error("a patch came before itself")
	}
}

// TestProjectedSelectionRefusesMissingOrBadBounds checks a nil bounds
// function and a bounds box the view cannot assess.
func TestProjectedSelectionRefusesMissingOrBadBounds(t *testing.T) {
	view, err := footprint.New(maths.Subspace{Origin: maths.V(0, 0, 100)},
		measure.Radians(math.Pi/3), 320, 180, 1, measure.Metres(0.1))
	if err != nil {
		t.Fatal(err)
	}
	budget := SelectionBudget{Patches: 4, Evaluations: 10, Depth: 2, Target: 1}
	root := Patch{MaxColumn: 4, MaxRow: 4}
	if _, err := SelectProjectedPatches(root, budget, view, nil); err == nil {
		t.Error("nil bounds were accepted")
	}
	broken := func(Patch) (PatchGeometry, error) {
		return PatchGeometry{Bounds: maths.Box{Min: maths.V(math.NaN(), 0, 0), Max: maths.V(1, 1, 1)}}, nil
	}
	if _, err := SelectProjectedPatches(root, budget, view, broken); err == nil {
		t.Error("a NaN box was accepted")
	}
}

// TestStitchRefusesWhatItCannotJoin checks the guards of StitchTo that the
// shared-side tests do not reach.
func TestStitchRefusesWhatItCannotJoin(t *testing.T) {
	fine, coarse := seamGrids()
	same := fine
	same.patch.Level = coarse.patch.Level
	if _, _, err := same.StitchTo(coarse, 1<<10); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Errorf("equal levels: %v", err)
	}
	moved := fine
	moved.west = 5
	if _, _, err := moved.StitchTo(coarse, 1<<10); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Errorf("different placement: %v", err)
	}
	apart := fine
	apart.patch = Patch{MinColumn: 20, MaxColumn: 24, MinRow: 20, MaxRow: 24, Level: 1}
	apart.columns, apart.rows = []int{20, 22, 24}, []int{20, 22, 24}
	if _, _, err := apart.StitchTo(coarse, 1<<10); err == nil || !strings.Contains(err.Error(), "not adjacent") {
		t.Errorf("distant grids: %v", err)
	}
	wide := fine
	wide.rows = []int{0, 2, 9}
	wide.patch.MaxRow = 9
	if _, _, err := wide.StitchTo(coarse, 1<<10); err == nil || !strings.Contains(err.Error(), "does not cover") {
		t.Errorf("a fine edge longer than the coarse edge: %v", err)
	}
	infinite := coarse
	infinite.heights = slices.Clone(coarse.heights)
	for i := range infinite.heights {
		infinite.heights[i] = math.Inf(1)
	}
	if _, _, err := fine.StitchTo(infinite, 1<<10); err == nil || !strings.Contains(err.Error(), "non-finite") {
		t.Errorf("infinite coarse heights: %v", err)
	}
	if _, _, err := fine.StitchTo(PatchGrid{}, 1<<10); err == nil {
		t.Error("an empty coarse grid was accepted")
	}
}
