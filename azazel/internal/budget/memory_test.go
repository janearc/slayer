package budget

import (
	_ "embed"
	"math"
	"math/big"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"pgregory.net/rapid"
)

// authoredMemoryData stores reviewed examples separate from the working path.
//
//go:embed testdata/memory.yaml
var authoredMemoryData []byte

const (
	// noMemoryExamples is defined as the cardinality of an empty case group.
	noMemoryExamples = 0
	// noRequestedBytes is defined as an empty request.
	// It is independent of production's byte count.
	noRequestedBytes uint64 = 0
	// minimumPositiveBytes is defined as one byte, the least positive count.
	minimumPositiveBytes uint64 = 1
	// equalIntegerComparison is defined by big.Int.Cmp for equal integers.
	equalIntegerComparison = 0
)

// memoryCase decodes an authored ledger without a floating-point conversion.
type memoryCase struct {
	Name      string `yaml:"name"`
	Limit     uint64 `yaml:"limit"`
	Reserved  uint64 `yaml:"reserved"`
	Requested uint64 `yaml:"requested"`
}

// brokenMemoryRange records the original generated broken-ledger bounds.
type brokenMemoryRange struct {
	MinimumLimit   uint64 `yaml:"minimum_limit"`
	MaximumLimit   uint64 `yaml:"maximum_limit"`
	Overcommit     uint64 `yaml:"overcommit"`
	MinimumRequest uint64 `yaml:"minimum_request"`
	MaximumRequest uint64 `yaml:"maximum_request"`
}

// memoryFixtures keeps each original example group and property range named.
type memoryFixtures struct {
	UnreservedCapacity []memoryCase      `yaml:"unreserved_capacity"`
	KeptReservations   []memoryCase      `yaml:"kept_reservations"`
	ExactFits          []memoryCase      `yaml:"exact_fits"`
	OneByteOver        []memoryCase      `yaml:"one_byte_over"`
	FullWidthRequests  []memoryCase      `yaml:"full_width_requests"`
	WrappedOvercommits []memoryCase      `yaml:"wrapped_overcommits"`
	MissingOrBroken    []memoryCase      `yaml:"missing_or_broken"`
	BrokenLedgerRange  brokenMemoryRange `yaml:"broken_ledger_range"`
}

// memoryExamples refuses missing or unsafe generator bounds during setup.
func memoryExamples(t *testing.T) memoryFixtures {
	t.Helper()
	var examples memoryFixtures
	if err := yamlfile.Read("testdata/memory.yaml", authoredMemoryData,
		&examples); err != nil {
		t.Fatal(err)
	}
	bounds := examples.BrokenLedgerRange
	if bounds.MinimumLimit != noRequestedBytes ||
		bounds.Overcommit == noRequestedBytes ||
		bounds.MaximumLimit != math.MaxUint64-bounds.Overcommit ||
		bounds.MinimumRequest != minimumPositiveBytes ||
		bounds.MaximumRequest != math.MaxUint64 {
		t.Fatalf("invalid authored broken-ledger range: %+v", bounds)
	}
	return examples
}

// memoryWitness binds the expected decision to one named subtest.
type memoryWitness struct {
	example  memoryCase
	expected bool
}

// check holds the predicate to an authored decision without deriving it again.
func (witness memoryWitness) check(t *testing.T) {
	budget := memoryBudget{limit: byteCount(witness.example.Limit),
		reserved: byteCount(witness.example.Reserved)}
	request := memoryRequest{amount: byteCount(witness.example.Requested)}
	if got := budget.admits(request); got != witness.expected {
		t.Fatalf("limit %d, reserved %d, request %d: admitted %t, want %t",
			witness.example.Limit, witness.example.Reserved,
			witness.example.Requested, got, witness.expected)
	}
}

// checkMemory refuses an empty group instead of silently dropping witnesses.
func checkMemory(t *testing.T, cases []memoryCase, expected bool) {
	t.Helper()
	if len(cases) == noMemoryExamples {
		t.Fatal("authored memory example group is empty")
	}
	for _, example := range cases {
		witness := memoryWitness{example: example, expected: expected}
		t.Run(example.Name, witness.check)
	}
}

// TestMemoryAdmitsUnreservedCapacity checks room left after reservations.
func TestMemoryAdmitsUnreservedCapacity(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.UnreservedCapacity, true)
}

// TestMemoryKeepsReservations is the anti-test for unreserved capacity.
func TestMemoryKeepsReservations(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.KeptReservations, false)
}

// TestMemoryAdmitsExactFits keeps equality on the allowed side.
func TestMemoryAdmitsExactFits(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.ExactFits, true)
}

// TestMemoryRejectsOneByteOver is the anti-test for the equality boundary.
func TestMemoryRejectsOneByteOver(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.OneByteOver, false)
}

// TestMemoryAdmitsFullWidthRequests covers counts above signed range.
func TestMemoryAdmitsFullWidthRequests(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.FullWidthRequests, true)
}

// TestMemoryNeverWrapsAnOvercommit catches adding unsigned counts first.
func TestMemoryNeverWrapsAnOvercommit(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.WrappedOvercommits, false)
}

// memoryLedgerProperty owns the independent unbounded addition oracle.
type memoryLedgerProperty struct{}

// check compares full-width counts with an arbitrary-precision sum.
func (memoryLedgerProperty) check(t *rapid.T) {
	limit := rapid.Uint64().Draw(t, "limit")
	reserved := rapid.Uint64().Draw(t, "reserved")
	requested := rapid.Uint64().Draw(t, "requested")
	reservedValue := new(big.Int).SetUint64(reserved)
	requestedValue := new(big.Int).SetUint64(requested)
	total := new(big.Int).Add(reservedValue, requestedValue)
	limitValue := new(big.Int).SetUint64(limit)
	expected := requested > noRequestedBytes &&
		total.Cmp(limitValue) <= equalIntegerComparison
	budget := memoryBudget{limit: byteCount(limit), reserved: byteCount(reserved)}
	request := memoryRequest{amount: byteCount(requested)}
	if got := budget.admits(request); got != expected {
		t.Fatalf("limit %d, reserved %d, request %d: admitted %t, want %t",
			limit, reserved, requested, got, expected)
	}
}

// TestMemoryMatchesAnUnboundedLedger uses addition that cannot overflow.
func TestMemoryMatchesAnUnboundedLedger(t *testing.T) {
	rapid.Check(t, memoryLedgerProperty{}.check)
}

// brokenMemoryProperty retains the original overcommitted-ledger generator.
type brokenMemoryProperty struct {
	bounds brokenMemoryRange
}

// check generates a positive request against an already overcommitted ledger.
func (property brokenMemoryProperty) check(t *rapid.T) {
	limit := rapid.Uint64Range(property.bounds.MinimumLimit,
		property.bounds.MaximumLimit).Draw(t, "limit")
	reserved := limit + property.bounds.Overcommit
	requested := rapid.Uint64Range(property.bounds.MinimumRequest,
		property.bounds.MaximumRequest).Draw(t, "requested")
	budget := memoryBudget{limit: byteCount(limit), reserved: byteCount(reserved)}
	if budget.admits(memoryRequest{amount: byteCount(requested)}) {
		t.Fatal("an already broken budget admitted more memory")
	}
}

// TestMemoryRefusesMissingAndBrokenCeilings is the ledger anti-test.
func TestMemoryRefusesMissingAndBrokenCeilings(t *testing.T) {
	examples := memoryExamples(t)
	checkMemory(t, examples.MissingOrBroken, false)
	property := brokenMemoryProperty{bounds: examples.BrokenLedgerRange}
	rapid.Check(t, property.check)
}
