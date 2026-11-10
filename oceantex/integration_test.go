package oceantex

import (
	"math"
	"testing"

	"github.com/janearc/slayer/titan"
)

func TestReferenceOceanRefusesUnboundedAllocationsBeforeRendering(t *testing.T) {
	for _, row := range []struct {
		n      int
		extent float64
	}{
		{1 << 30, 1}, {2, math.Inf(1)}, {2, math.NaN()},
	} {
		if _, err := NewPatch(Sea{}, row.n, row.extent, 1); err == nil {
			t.Fatal("unsafe wave grid accepted", row)
		}
	}
	if _, err := View(Scene{}, Camera{Width: 1 << 30, Height: 1 << 30,
		Eye: Subspace{Origin: Vec{Z: 1}}}); err == nil {
		t.Fatal("unsafe image accepted")
	}
	if _, err := NewShore(&titan.Ground{}, 1, 0, 0, 0, 1e300, 1e-300); err == nil {
		t.Fatal("unsafe shore accepted")
	}
}

func TestReferenceSourceLedgerRejectsAmbiguityAndNonfiniteValues(t *testing.T) {
	for _, raw := range []string{
		"a:\n  value: .nan\n",
		"a:\n  value: 1\n  values: [2]\n",
	} {
		raw += "  from: test fixture\n  url: https://example.test/fixture\n  fetched: 2026-11-09\n  read: none\n  kind: glue\n"
		if _, err := parseSources(raw); err == nil {
			t.Fatal("invalid ledger accepted")
		}
	}
}
