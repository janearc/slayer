//go:build darwin

package metal

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

func TestNativeHostGridBudgetsWithoutAllocatingTheirOutput(t *testing.T) {
	const nativeLimit = 2_147_483_647
	for _, test := range []struct {
		width, height, per int
		fits               bool
	}{
		{nativeLimit, 1, 1, true}, {nativeLimit + 1, 1, 1, false},
		{1, nativeLimit, 1, true}, {1, nativeLimit + 1, 1, false},
		{1, 1, nativeLimit, true}, {1, 1, nativeLimit + 1, false},
		{1 << 30, 1, nativeLimit, true},
		// Four-byte outputs may occupy at most MaxInt bytes. These
		// adjacent grids lie on opposite sides of that byte budget.
		{1 << 30, (1 << 30) - 1, 2, true}, {1 << 30, 1 << 30, 2, false},
		{(1 << 30) - 1, 1 << 30, 2, true}, {1 << 30, 1 << 30, 4, false},
	} {
		if got := gridFits(test.width, test.height, test.per); got != test.fits {
			t.Fatalf("%dx%dx%d fits=%t, want %t", test.width, test.height, test.per, got, test.fits)
		}
	}
}

func TestNativeHostPlacementBudgetWithoutAllocatingItsMembers(t *testing.T) {
	const nativeLimit = 2_147_483_647
	for _, test := range []struct {
		held, added int
		fits        bool
	}{
		{0, nativeLimit, true}, {1, nativeLimit - 1, true},
		{1, nativeLimit, false}, {nativeLimit, 0, true},
		{nativeLimit, 1, false}, {nativeLimit - 1, 2, false},
	} {
		if got := instancePlacementCount(test.held, test.added); got != test.fits {
			t.Fatalf("%d held + %d added fits=%t, want %t", test.held, test.added, got, test.fits)
		}
	}
}

func TestNativeInputLimitIncludesItsLastByte(t *testing.T) {
	kernel, err := Open(fillGrid, "fill_grid")
	requireDevice(t, err)
	defer kernel.Close()
	// Zero output dimensions return before any C copy. This holds the
	// exact input endpoint with a virtual reservation and no GPU upload.
	input := reservedBytes(t, 2_147_483_647)
	got, err := kernel.Grid(0, 1, 1, 1, [][]byte{input})
	if err != nil || len(got) != 0 {
		t.Fatalf("last allowed input byte: %v, %v", got, err)
	}
}

func TestNativeSingleRowBandsRetainEveryBandOrigin(t *testing.T) {
	kernel, err := Open(where, "where")
	requireDevice(t, err)
	defer kernel.Close()
	given := binary.LittleEndian.AppendUint32(nil, math.Float32bits(7))
	out, err := kernel.Grid(3, 4, 1, 4, [][]byte{given})
	if err != nil {
		t.Fatal(err)
	}
	for row := range 4 {
		for column := range 3 {
			got := [4]float32(out[(row*3+column)*4:][:4])
			want := [4]float32{float32(column), float32(row), float32(row), 7}
			if got != want {
				t.Fatalf("pixel %d,%d: %v, want %v", column, row, got, want)
			}
		}
	}
}

func TestNativeMissingFunctionPreservesItsFullCause(t *testing.T) {
	kernel, err := Open(functions, "first_value")
	requireDevice(t, err)
	defer kernel.Close()
	if err := kernel.Function("absent_entry"); err == nil || err.Error() != "metal: no kernel named absent_entry" {
		t.Fatalf("missing function diagnostic: %v", err)
	}
}

func TestNativeSessionMaximumHostCountReachesTheDeviceLimit(t *testing.T) {
	session, err := OpenSession(twoSteps, "twice")
	requireDevice(t, err)
	defer session.Close()
	// The bridge rejects this size before calling newBufferWithLength.
	// Its device refusal must remain distinct from Go integer overflow.
	if _, err := session.Buffer(math.MaxInt / 4); err == nil ||
		!strings.Contains(err.Error(), "buffer exceeds device limit") {
		t.Fatalf("last representable host buffer count: %v", err)
	}
}
