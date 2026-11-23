package terrain

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// policyBook reads a book of picked numbers, one per name and value given.
func policyBook(t *testing.T, values map[string]string) numbers.Numbers {
	t.Helper()
	var text strings.Builder
	text.WriteString("numbers:\n")
	for name, value := range values {
		fmt.Fprintf(&text, "  - name: %s\n    value: %s\n    says: a test count\n    kind: picked\n    by: test\n", name, value)
	}
	written := text.String()
	book, err := numbers.Read("test.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	return book
}

// usePolicy makes the package read the given book for the rest of the test.
func usePolicy(t *testing.T, book numbers.Numbers, err error) {
	t.Helper()
	old := memoryPolicy
	memoryPolicy = func() (numbers.Numbers, error) { return book, err }
	t.Cleanup(func() { memoryPolicy = old })
}

// goodPolicy is a complete resource book with a changeable value per name.
func goodPolicy(t *testing.T, change map[string]string) numbers.Numbers {
	t.Helper()
	values := map[string]string{"window-bytes": "1000", "retained-bytes": "2000",
		"tile-bytes": "300", "metadata-bytes": "400", "retained-tiles": "5"}
	for name, value := range change {
		values[name] = value
	}
	return policyBook(t, values)
}

// TestDefaultsComeFromThePolicyBook checks each default against the value
// written in the book it is read from.
func TestDefaultsComeFromThePolicyBook(t *testing.T) {
	usePolicy(t, goodPolicy(t, nil), nil)
	budget, err := defaultRasterBudget()
	want := RasterBudget{CacheBytes: 2000, CacheTiles: 5, TileBytes: 300, MetadataBytes: 400}
	if err != nil || budget != want {
		t.Fatalf("budget %+v, %v; want %+v", budget, err, want)
	}
	window, err := defaultWindowBytes()
	if err != nil || window != 1000 {
		t.Fatalf("window bytes %d, %v", window, err)
	}
}

// TestDefaultsRefuseABrokenPolicyBook checks that a missing, fractional or
// negative count, and a book that cannot be read, each make the defaults fail.
func TestDefaultsRefuseABrokenPolicyBook(t *testing.T) {
	raster := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(100, 50, 1, 1))
	cases := map[string]struct {
		book numbers.Numbers
		err  error
		want string
	}{
		"unreadable":    {nil, fmt.Errorf("book is gone"), "book is gone"},
		"fractional":    {goodPolicy(t, map[string]string{"tile-bytes": "300.5"}), nil, "invalid resource policy"},
		"negative":      {goodPolicy(t, map[string]string{"retained-bytes": "-1"}), nil, "invalid resource policy"},
		"too large":     {goodPolicy(t, map[string]string{"metadata-bytes": "1e30"}), nil, "invalid resource policy"},
		"tiles missing": {policyBook(t, map[string]string{"window-bytes": "1", "retained-bytes": "1", "tile-bytes": "8", "metadata-bytes": "8"}), nil, "retained-tiles"},
		"tile too small for a sample": {goodPolicy(t, map[string]string{"tile-bytes": "3"}), nil,
			"invalid raster byte budget"},
	}
	for name, each := range cases {
		usePolicy(t, each.book, each.err)
		if _, err := defaultRasterBudget(); err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: default budget error %v, want %q", name, err, each.want)
		}
		if _, err := Open("missing.tif"); err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: Open error %v, want %q", name, err, each.want)
		}
	}
	usePolicy(t, goodPolicy(t, map[string]string{"window-bytes": "-5"}), nil)
	if _, err := defaultWindowBytes(); err == nil {
		t.Error("a negative window limit was accepted")
	}
	usePolicy(t, nil, fmt.Errorf("book is gone"))
	if _, err := defaultWindowBytes(); err == nil {
		t.Error("an unreadable book gave a window limit")
	}
	usePolicy(t, policyBook(t, map[string]string{"other": "1"}), nil)
	if _, err := defaultWindowBytes(); err == nil {
		t.Error("a book without a window limit gave one")
	}
	if _, err := raster.Window(100, 50, 2, 2, 1); err == nil {
		t.Error("a window was cut without a window limit")
	}
}

// TestBudgetValidityIsCheckedFieldByField checks each way a budget is refused.
func TestBudgetValidityIsCheckedFieldByField(t *testing.T) {
	good := RasterBudget{CacheBytes: 0, CacheTiles: 0, TileBytes: 4, MetadataBytes: 8}
	if err := good.valid(); err != nil {
		t.Fatalf("the smallest budget was refused: %v", err)
	}
	for name, edit := range map[string]func(*RasterBudget){
		"negative cache bytes": func(b *RasterBudget) { b.CacheBytes = -1 },
		"negative tiles":       func(b *RasterBudget) { b.CacheTiles = -1 },
		"tile under a sample":  func(b *RasterBudget) { b.TileBytes = 3 },
		"metadata under count": func(b *RasterBudget) { b.MetadataBytes = 7 },
	} {
		budget := good
		edit(&budget)
		if err := budget.valid(); err == nil {
			t.Errorf("%s was accepted", name)
		}
		if _, err := OpenWithBudget("missing.tif", budget); err == nil ||
			!strings.Contains(err.Error(), "invalid raster byte budget") {
			t.Errorf("%s: open error %v", name, err)
		}
	}
}
