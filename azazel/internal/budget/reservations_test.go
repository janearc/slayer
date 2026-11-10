package budget

import (
	_ "embed"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"pgregory.net/rapid"
)

// authoredReservationData holds exact unsigned examples and picked test bounds.
//
//go:embed testdata/reservations.yaml
var authoredReservationData []byte

// firstReservation is defined as the first index of a nonempty ledger.
const firstReservation = 0

// reservationData decodes one identified count without float conversion.
type reservationData struct {
	ID     string `yaml:"id"`
	Amount uint64 `yaml:"amount"`
}

// value supplies the authored identity and amount without deriving either.
func (data reservationData) value() memoryReservation {
	return memoryReservation{id: data.ID, amount: byteCount(data.Amount)}
}

// reservationExample names one authored before-state and requested addition.
type reservationExample struct {
	Name     string            `yaml:"name"`
	Limit    uint64            `yaml:"limit"`
	Existing []reservationData `yaml:"existing"`
	Request  reservationData   `yaml:"request"`
}

// ledger converts exact counts and preserves the input reservation order.
func (example reservationExample) ledger() reservationLedger {
	entries := make([]memoryReservation, len(example.Existing))
	for index, entry := range example.Existing {
		entries[index] = entry.value()
	}
	return reservationLedger{
		limit: byteCount(example.Limit), reservations: entries}
}

// reservationSequenceBounds are picked test sizes, never production policy.
type reservationSequenceBounds struct {
	Ceiling       uint64 `yaml:"ceiling"`
	MinimumSteps  int    `yaml:"minimum_steps"`
	MaximumSteps  int    `yaml:"maximum_steps"`
	MinimumAmount uint64 `yaml:"minimum_amount"`
	MaximumAmount uint64 `yaml:"maximum_amount"`
}

// reservationFixtures keeps each example's positive or refusal role distinct.
type reservationFixtures struct {
	Empty         []reservationExample      `yaml:"empty"`
	Missing       []reservationExample      `yaml:"missing"`
	Kept          []reservationExample      `yaml:"kept"`
	Capacity      []reservationExample      `yaml:"capacity"`
	FullWidth     []reservationExample      `yaml:"full_width"`
	Wrapped       []reservationExample      `yaml:"wrapped"`
	Duplicates    []reservationExample      `yaml:"duplicates"`
	Invalid       []reservationExample      `yaml:"invalid"`
	Ownership     reservationExample        `yaml:"ownership"`
	SpareCapacity int                       `yaml:"spare_capacity"`
	Sequence      reservationSequenceBounds `yaml:"sequence"`
}

// reservationExamples refuses missing roles and unsafe test bounds.
func reservationExamples(t *testing.T) reservationFixtures {
	t.Helper()
	var examples reservationFixtures
	if err := yamlfile.Read("testdata/reservations.yaml", authoredReservationData,
		&examples); err != nil {
		t.Fatal(err)
	}
	for _, group := range [][]reservationExample{examples.Empty, examples.Missing,
		examples.Kept, examples.Capacity, examples.FullWidth, examples.Wrapped,
		examples.Duplicates, examples.Invalid} {
		if len(group) == noMemoryExamples {
			t.Fatal("authored reservation role is empty")
		}
	}
	bounds := examples.Sequence
	if len(examples.Ownership.Existing) == noMemoryExamples ||
		examples.SpareCapacity <= len(examples.Ownership.Existing) ||
		bounds.MinimumSteps <= noMemoryExamples ||
		bounds.MaximumSteps < bounds.MinimumSteps ||
		bounds.MinimumAmount != minimumPositiveBytes ||
		bounds.MaximumAmount < bounds.MinimumAmount ||
		bounds.MaximumAmount != bounds.Ceiling {
		t.Fatal("invalid authored ownership or property bounds")
	}
	return examples
}

// reservationWitness binds an independently specified disposition to one case.
type reservationWitness struct {
	example reservationExample
	want    reservationDisposition
}

// check requires no proposal on refusal and an intact prefix on success.
func (witness reservationWitness) check(t *testing.T) {
	ledger := witness.example.ledger()
	before := slices.Clone(ledger.reservations)
	request := witness.example.Request.value()
	got := ledger.reserving(request)
	if got.disposition != witness.want {
		t.Fatalf("disposition %v, want %v", got.disposition, witness.want)
	}
	if !slices.Equal(ledger.reservations, before) {
		t.Fatal("input ledger changed")
	}
	if witness.want != reservationProposed {
		if got.next != nil {
			t.Fatal("refusal returned a proposed ledger")
		}
		return
	}
	want := append(slices.Clone(before), request)
	if got.next == nil || got.next.limit != ledger.limit ||
		!slices.Equal(got.next.reservations, want) {
		t.Fatalf("proposal %+v, want ordered entries %+v", got.next, want)
	}
}

// checkReservations refuses missing examples instead of dropping a witness.
func checkReservations(t *testing.T, cases []reservationExample,
	want reservationDisposition,
) {
	t.Helper()
	if len(cases) == noMemoryExamples {
		t.Fatal("no authored reservation examples")
	}
	for _, example := range cases {
		witness := reservationWitness{example: example, want: want}
		t.Run(example.Name, witness.check)
	}
}

// TestReservationAddsToAnEmptyLedger requires one identified positive request.
func TestReservationAddsToAnEmptyLedger(t *testing.T) {
	checkReservations(t, reservationExamples(t).Empty, reservationProposed)
}

// TestReservationRejectsMissingInput is the paired omitted-input anti-test.
func TestReservationRejectsMissingInput(t *testing.T) {
	checkReservations(t, reservationExamples(t).Missing, reservationInvalid)
}

// TestReservationPreservesExistingOrder includes a multi-entry exact fit.
func TestReservationPreservesExistingOrder(t *testing.T) {
	checkReservations(t, reservationExamples(t).Kept, reservationProposed)
}

// TestReservationCannotSpendPromisedBytes refuses an overcommitted addition.
func TestReservationCannotSpendPromisedBytes(t *testing.T) {
	checkReservations(t, reservationExamples(t).Capacity, reservationCapacityHeld)
}

// TestReservationAllowsFullWidthExactFits includes the entire unsigned range.
func TestReservationAllowsFullWidthExactFits(t *testing.T) {
	checkReservations(t, reservationExamples(t).FullWidth, reservationProposed)
}

// TestReservationRejectsWrappedAddition is the paired overflow anti-test.
func TestReservationRejectsWrappedAddition(t *testing.T) {
	checkReservations(t, reservationExamples(t).Wrapped, reservationCapacityHeld)
}

// TestReservationRefusesDuplicateRequests forbids both replacement spellings.
func TestReservationRefusesDuplicateRequests(t *testing.T) {
	checkReservations(t, reservationExamples(t).Duplicates, reservationDuplicate)
}

// TestReservationRefusesCorruptLedgers distinguishes invalid prior state.
func TestReservationRefusesCorruptLedgers(t *testing.T) {
	checkReservations(t, reservationExamples(t).Invalid, reservationInvalid)
}

// TestReservationReturnsIndependentStorage permits later caller mutations.
func TestReservationReturnsIndependentStorage(t *testing.T) {
	example := reservationExamples(t).Ownership
	ledger := example.ledger()
	before := slices.Clone(ledger.reservations)
	got := ledger.reserving(example.Request.value())
	if got.disposition != reservationProposed || got.next == nil {
		t.Fatalf("ownership proposal: %+v", got)
	}
	proposed := slices.Clone(got.next.reservations)
	got.next.reservations[firstReservation].id = "changed proposal"
	if !slices.Equal(ledger.reservations, before) {
		t.Fatal("proposal mutation changed the input")
	}
	got.next.reservations[firstReservation] = proposed[firstReservation]
	ledger.reservations[firstReservation].id = "changed input"
	if !slices.Equal(got.next.reservations, proposed) {
		t.Fatal("input mutation changed the proposal")
	}
}

// TestReservationDoesNotUseSpareInputCapacity checks hidden append aliasing.
func TestReservationDoesNotUseSpareInputCapacity(t *testing.T) {
	examples := reservationExamples(t)
	ledger := examples.Ownership.ledger()
	storage := make([]memoryReservation, examples.SpareCapacity)
	for index := range storage {
		storage[index] = memoryReservation{
			id: "spare", amount: byteCount(minimumPositiveBytes)}
	}
	copy(storage, ledger.reservations)
	before := slices.Clone(storage)
	ledger.reservations = storage[:len(ledger.reservations)]
	got := ledger.reserving(examples.Ownership.Request.value())
	if got.disposition != reservationProposed || got.next == nil {
		t.Fatalf("spare-capacity proposal: %+v", got)
	}
	if !slices.Equal(storage, before) {
		t.Fatal("proposal overwrote the input's spare capacity")
	}
}

// TestReservationZeroProposalGrantsNothing covers the omitted-result boundary.
func TestReservationZeroProposalGrantsNothing(t *testing.T) {
	zero := reservationProposal{}
	if zero.disposition == reservationProposed || zero.next != nil {
		t.Fatal("zero result claims a reservation proposal")
	}
}

// reservationCapacityProperty owns an independent arbitrary-precision oracle.
type reservationCapacityProperty struct{}

// check spans full-width existing and requested counts without wrapped sums.
func (reservationCapacityProperty) check(trial *rapid.T) {
	limit := rapid.Uint64().Draw(trial, "limit")
	held := rapid.Uint64().Draw(trial, "held")
	amount := rapid.Uint64().Draw(trial, "requested")
	ledger := reservationLedger{limit: byteCount(limit)}
	if held != noRequestedBytes {
		ledger.reservations = []memoryReservation{
			{id: "held", amount: byteCount(held)}}
	}
	heldValue, amountValue := new(big.Int).SetUint64(held),
		new(big.Int).SetUint64(amount)
	total := new(big.Int).Add(heldValue, amountValue)
	want := reservationCapacityHeld
	if limit == noRequestedBytes || amount == noRequestedBytes || held > limit {
		want = reservationInvalid
	} else if total.Cmp(new(big.Int).SetUint64(limit)) <= equalIntegerComparison {
		want = reservationProposed
	}
	request := memoryReservation{id: "new", amount: byteCount(amount)}
	got := ledger.reserving(request)
	hasProposal := want == reservationProposed
	if got.disposition != want || (got.next != nil) != hasProposal {
		trial.Fatalf("limit=%d held=%d request=%d: %+v, want %v",
			limit, held, amount, got, want)
	}
}

// TestReservationMatchesAnUnboundedSum checks the independent sum oracle.
func TestReservationMatchesAnUnboundedSum(t *testing.T) {
	rapid.Check(t, reservationCapacityProperty{}.check)
}

// reservationSequenceProperty keeps picked sequence sizes outside the source.
type reservationSequenceProperty struct{ bounds reservationSequenceBounds }

// check verifies every proposed prefix and each no-proposal capacity refusal.
func (property reservationSequenceProperty) check(trial *rapid.T) {
	ledger := reservationLedger{limit: byteCount(property.bounds.Ceiling)}
	total := new(big.Int)
	ceiling := new(big.Int).SetUint64(property.bounds.Ceiling)
	steps := rapid.IntRange(property.bounds.MinimumSteps,
		property.bounds.MaximumSteps).Draw(trial, "steps")
	for index := range steps {
		amount := rapid.Uint64Range(property.bounds.MinimumAmount,
			property.bounds.MaximumAmount).Draw(trial, "amount")
		request := memoryReservation{
			id: fmt.Sprintf("step-%d", index), amount: byteCount(amount)}
		before := slices.Clone(ledger.reservations)
		nextTotal := new(big.Int).Add(total, new(big.Int).SetUint64(amount))
		fits := nextTotal.Cmp(ceiling) <= equalIntegerComparison
		got := ledger.reserving(request)
		if !slices.Equal(ledger.reservations, before) {
			trial.Fatal("addition changed the input ledger")
		}
		if !fits {
			if got.disposition != reservationCapacityHeld || got.next != nil {
				trial.Fatalf("overcommitted sequence proposal: %+v", got)
			}
			continue
		}
		want := append(slices.Clone(before), request)
		if got.disposition != reservationProposed || got.next == nil ||
			got.next.limit != ledger.limit ||
			!slices.Equal(got.next.reservations, want) {
			trial.Fatalf("sequence proposal: %+v, want %+v", got, want)
		}
		ledger, total = *got.next, nextTotal
	}
}

// TestReservationSequencesPreserveEarlierClaims pairs arbitrary additions.
func TestReservationSequencesPreserveEarlierClaims(t *testing.T) {
	bounds := reservationExamples(t).Sequence
	property := reservationSequenceProperty{bounds: bounds}
	rapid.Check(t, property.check)
}
