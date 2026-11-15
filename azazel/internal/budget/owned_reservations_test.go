package budget

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

func reservationTestOwner(name, incarnation string) reservationOwner {
	return reservationOwner{name: reservationOwnerName(name), incarnation: reservationOwnerIdentity(incarnation)}
}

func TestReservationOwnedAddition(t *testing.T) {
	owner := reservationTestOwner("composite", "attempt-1")
	other := reservationTestOwner("render", "attempt-2")
	first := ownedMemoryReservation{id: "first", owner: owner, amount: 4}
	request := ownedMemoryReservation{id: "second", owner: other, amount: 6}
	storage := make([]ownedMemoryReservation, 3)
	storage[0] = first
	storage[1] = ownedMemoryReservation{id: "unused", owner: owner, amount: 1}
	ledger := ownedReservationLedger{limit: 10, reservations: storage[:1]}
	before := slices.Clone(storage)
	got := ledger.reserving(request)
	if got.disposition != reservationProposed || got.next == nil {
		t.Fatalf("addition: %+v", got)
	}
	if got.next.limit != 10 || !slices.Equal(got.next.reservations, []ownedMemoryReservation{first, request}) {
		t.Fatalf("next ledger: %+v", got.next)
	}
	if !slices.Equal(storage, before) {
		t.Fatal("addition changed input or spare capacity")
	}
	got.next.reservations[0].owner.name = "changed"
	if !slices.Equal(storage, before) {
		t.Fatal("proposal aliases input")
	}
	ledger.reservations[0].id = "changed-input"
	if got.next.reservations[0].id != first.id {
		t.Fatal("input aliases proposal")
	}
}

func TestReservationOwnedRefusals(t *testing.T) {
	owner := reservationTestOwner("composite", "attempt-1")
	claim := ownedMemoryReservation{id: "first", owner: owner, amount: 4}
	valid := ownedReservationLedger{limit: 10, reservations: []ownedMemoryReservation{claim}}
	cases := []struct {
		name    string
		ledger  ownedReservationLedger
		request ownedMemoryReservation
		want    reservationDisposition
	}{
		{"missing-name", valid, ownedMemoryReservation{id: "next", owner: reservationTestOwner("", "attempt-2"), amount: 1}, reservationInvalid},
		{"missing-incarnation", valid, ownedMemoryReservation{id: "next", owner: reservationTestOwner("render", ""), amount: 1}, reservationInvalid},
		{"missing-id", valid, ownedMemoryReservation{owner: owner, amount: 1}, reservationInvalid},
		{"missing-amount", valid, ownedMemoryReservation{id: "next", owner: owner}, reservationInvalid},
		{"missing-limit", ownedReservationLedger{}, claim, reservationInvalid},
		{"duplicate-same-owner", valid, claim, reservationDuplicate},
		{"duplicate-other-owner", valid, ownedMemoryReservation{id: claim.id, owner: reservationTestOwner("render", "attempt-2"), amount: 1}, reservationDuplicate},
		{"capacity", valid, ownedMemoryReservation{id: "next", owner: owner, amount: 7}, reservationCapacityHeld},
		{"wrap", ownedReservationLedger{limit: math.MaxUint64, reservations: []ownedMemoryReservation{{id: "first", owner: owner, amount: math.MaxUint64}}}, ownedMemoryReservation{id: "next", owner: owner, amount: 1}, reservationCapacityHeld},
	}
	for _, example := range cases {
		t.Run(example.name, func(t *testing.T) {
			before := slices.Clone(example.ledger.reservations)
			got := example.ledger.reserving(example.request)
			if got.disposition != example.want || got.next != nil {
				t.Fatalf("proposal %+v, want refusal %v", got, example.want)
			}
			if !slices.Equal(example.ledger.reservations, before) {
				t.Fatal("refusal changed ledger")
			}
		})
	}
	full := ownedReservationLedger{limit: math.MaxUint64}.reserving(ownedMemoryReservation{id: "full", owner: owner, amount: math.MaxUint64})
	if full.disposition != reservationProposed || full.next == nil || full.next.reservations[0].amount != math.MaxUint64 {
		t.Fatalf("full-width exact fit: %+v", full)
	}
}

func TestReleaseRequiresMatchingOwnerAndIdentity(t *testing.T) {
	owner := reservationTestOwner("composite", "attempt-1")
	claim := ownedMemoryReservation{id: "first", owner: owner, amount: 4}
	ledger := ownedReservationLedger{limit: 10, reservations: []ownedMemoryReservation{claim}}
	cases := []struct {
		name    string
		request reservationRelease
		want    releaseDisposition
	}{
		{"missing-id", reservationRelease{owner: owner}, releaseInvalid},
		{"missing-name", reservationRelease{id: claim.id, owner: reservationTestOwner("", "attempt-1")}, releaseInvalid},
		{"missing-incarnation", reservationRelease{id: claim.id, owner: reservationTestOwner("composite", "")}, releaseInvalid},
		{"unknown-id", reservationRelease{id: "absent", owner: owner}, releaseNotFound},
		{"different-name", reservationRelease{id: claim.id, owner: reservationTestOwner("render", "attempt-1")}, releaseOwnerMismatch},
		{"stale-incarnation", reservationRelease{id: claim.id, owner: reservationTestOwner("composite", "attempt-0")}, releaseOwnerMismatch},
	}
	for _, example := range cases {
		t.Run(example.name, func(t *testing.T) {
			before := slices.Clone(ledger.reservations)
			got := ledger.releasing(example.request)
			if got.disposition != example.want || got.next != nil {
				t.Fatalf("release %+v, want refusal %v", got, example.want)
			}
			if !slices.Equal(ledger.reservations, before) {
				t.Fatal("refusal changed ledger")
			}
		})
	}
}

func TestReleasePreservesOrderAndIndependentStorage(t *testing.T) {
	owner := reservationTestOwner("render", "attempt-1")
	claims := []ownedMemoryReservation{
		{id: "first", owner: owner, amount: 2},
		{id: "middle", owner: owner, amount: 3},
		{id: "last", owner: owner, amount: 5},
	}
	for index, claim := range claims {
		ledger := ownedReservationLedger{limit: 10, reservations: slices.Clone(claims)}
		got := ledger.releasing(reservationRelease{id: claim.id, owner: owner})
		want := append(slices.Clone(claims[:index]), claims[index+1:]...)
		if got.disposition != releaseProposed || got.next == nil || got.next.limit != ledger.limit || !slices.Equal(got.next.reservations, want) {
			t.Fatalf("release %s: %+v", claim.id, got)
		}
		if !slices.Equal(ledger.reservations, claims) {
			t.Fatal("release changed input")
		}
		got.next.reservations[0].id = "changed"
		if !slices.Equal(ledger.reservations, claims) {
			t.Fatal("release proposal aliases input")
		}
		got.next.reservations[0] = want[0]
		ledger.reservations[0].owner.name = "changed-input"
		if !slices.Equal(got.next.reservations, want) {
			t.Fatal("release input aliases proposal")
		}
	}
	ledger := ownedReservationLedger{limit: 10, reservations: claims[:1]}
	request := reservationRelease{id: claims[0].id, owner: owner}
	got := ledger.releasing(request)
	if got.disposition != releaseProposed || got.next == nil || len(got.next.reservations) != 0 || got.next.limit != ledger.limit {
		t.Fatalf("last release: %+v", got)
	}
	again := got.next.releasing(request)
	if again.disposition != releaseNotFound || again.next != nil {
		t.Fatalf("repeated release: %+v", again)
	}
	replacement := got.next.reserving(ownedMemoryReservation{id: "replacement", owner: owner, amount: 10})
	if replacement.disposition != reservationProposed {
		t.Fatalf("released capacity cannot be reused: %+v", replacement)
	}
}

func TestReservationAndReleaseRejectWholeCorruptLedger(t *testing.T) {
	owner := reservationTestOwner("render", "attempt-1")
	claim := ownedMemoryReservation{id: "target", owner: owner, amount: 1}
	badEntries := [][]ownedMemoryReservation{
		{claim, {id: "bad", amount: 1}},
		{claim, {id: "bad", owner: reservationTestOwner("render", ""), amount: 1}},
		{claim, {owner: owner, amount: 1}},
		{claim, {id: "bad", owner: owner}},
		{claim, {id: claim.id, owner: reservationTestOwner("composite", "other"), amount: 1}},
		{claim, {id: "bad", owner: owner, amount: 10}},
		{claim, {id: "bad", owner: owner, amount: math.MaxUint64}},
	}
	for _, entries := range badEntries {
		ledger := ownedReservationLedger{limit: 10, reservations: entries}
		before := slices.Clone(entries)
		addition := ledger.reserving(ownedMemoryReservation{id: "new", owner: owner, amount: 1})
		release := ledger.releasing(reservationRelease{id: claim.id, owner: owner})
		if addition.disposition != reservationInvalid || addition.next != nil || release.disposition != releaseInvalid || release.next != nil {
			t.Fatalf("corrupt ledger accepted: addition %+v, release %+v", addition, release)
		}
		if !reflect.DeepEqual(entries, before) {
			t.Fatal("corrupt ledger repaired or mutated")
		}
	}
	missingLimit := ownedReservationLedger{}.releasing(reservationRelease{id: claim.id, owner: owner})
	if missingLimit.disposition != releaseInvalid || missingLimit.next != nil {
		t.Fatalf("missing limit: %+v", missingLimit)
	}
}

func TestReservationOwnedAndReleaseZeroResultsAreUnknown(t *testing.T) {
	var addition ownedReservationProposal
	var release releaseProposal
	if addition.disposition != reservationUnknown || addition.next != nil || release.disposition != releaseUnknown || release.next != nil {
		t.Fatal("zero result proposes a change")
	}
}
