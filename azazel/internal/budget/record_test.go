package budget

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func recordExample() admissionRecord {
	owner := reservationOwner{name: "game", incarnation: "attempt-1"}
	return admissionRecord{
		version: 1, boot: "boot-1", policy: hostPolicy{loadLimit: 2.5, swapLimit: 10, diskFloor: 20},
		stop:   stopState{stopped: true, generation: 3},
		owners: []reservationOwner{owner, {name: "carrie", incarnation: "attempt-2"}},
		ledger: ownedReservationLedger{limit: 100, reservations: []ownedMemoryReservation{
			{id: "z", owner: owner, amount: 7}, {id: "a", owner: owner, amount: 8},
		}},
		predecessor: sha256.Sum256([]byte("previous")),
	}
}

func TestRecordCanonicalIndependentState(t *testing.T) {
	state := recordExample()
	beforeOwners := slices.Clone(state.owners)
	beforeReservations := slices.Clone(state.ledger.reservations)
	encoded, err := encodeAdmissionRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	permuted := recordExample()
	slices.Reverse(permuted.owners)
	slices.Reverse(permuted.ledger.reservations)
	other, err := encodeAdmissionRecord(permuted)
	if err != nil || !bytes.Equal(encoded, other) {
		t.Fatalf("noncanonical permutation: %v", err)
	}
	if !slices.Equal(state.owners, beforeOwners) || !slices.Equal(state.ledger.reservations, beforeReservations) {
		t.Fatal("encoder mutated input")
	}
	decoded, err := decodeAdmissionRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.version != 1 || decoded.boot != state.boot || decoded.policy != state.policy || decoded.stop != state.stop || decoded.predecessor != state.predecessor || decoded.ledger.limit != 100 || len(decoded.owners) != 2 || decoded.ledger.reservations[0].id != "a" {
		t.Fatalf("lost fields: %+v", decoded)
	}
	again, err := encodeAdmissionRecord(decoded)
	if err != nil || !bytes.Equal(encoded, again) {
		t.Fatalf("unstable encoding: %v", err)
	}
	clear(encoded)
	if decoded.boot != "boot-1" || decoded.owners[0].name != "carrie" {
		t.Fatal("decoder aliases bytes")
	}
	decoded.owners[0].name = "changed"
	decoded.ledger.reservations[0].amount = 99
	if !slices.Equal(state.owners, beforeOwners) || !slices.Equal(state.ledger.reservations, beforeReservations) {
		t.Fatal("decoder aliases state")
	}
}

func TestRecordInvalidState(t *testing.T) {
	cases := map[string]func(*admissionRecord){
		"version":         func(s *admissionRecord) { s.version = 2 },
		"empty-boot":      func(s *admissionRecord) { s.boot = "" },
		"invalid-token":   func(s *admissionRecord) { s.boot = "boot\n1" },
		"unicode":         func(s *admissionRecord) { s.boot = "bøot" },
		"long-token":      func(s *admissionRecord) { s.boot = strings.Repeat("b", 129) },
		"zero-load":       func(s *admissionRecord) { s.policy.loadLimit = 0 },
		"negative-load":   func(s *admissionRecord) { s.policy.loadLimit = -1 },
		"nan":             func(s *admissionRecord) { s.policy.loadLimit = math.NaN() },
		"infinite":        func(s *admissionRecord) { s.policy.loadLimit = math.Inf(1) },
		"swap":            func(s *admissionRecord) { s.policy.swapLimit = 0 },
		"disk":            func(s *admissionRecord) { s.policy.diskFloor = 0 },
		"exhausted-clear": func(s *admissionRecord) { s.stop = stopState{generation: math.MaxUint64} },
		"zero-limit":      func(s *admissionRecord) { s.ledger.limit = 0 },
		"zero-amount":     func(s *admissionRecord) { s.ledger.reservations[0].amount = 0 },
		"overflow": func(s *admissionRecord) {
			s.ledger.limit = math.MaxUint64
			s.ledger.reservations[0].amount = math.MaxUint64
		},
		"capacity":              func(s *admissionRecord) { s.ledger.limit = 14 },
		"empty-id":              func(s *admissionRecord) { s.ledger.reservations[0].id = "" },
		"duplicate-id":          func(s *admissionRecord) { s.ledger.reservations[0].id = "a" },
		"duplicate-owner":       func(s *admissionRecord) { s.owners[1] = s.owners[0] },
		"duplicate-name":        func(s *admissionRecord) { s.owners[1].name = s.owners[0].name },
		"missing-incarnation":   func(s *admissionRecord) { s.owners[0].incarnation = "" },
		"unknown-owner":         func(s *admissionRecord) { s.ledger.reservations[0].owner.name = "unknown" },
		"wrong-incarnation":     func(s *admissionRecord) { s.ledger.reservations[0].owner.incarnation = "stale" },
		"too-many-owners":       func(s *admissionRecord) { s.owners = make([]reservationOwner, 1025) },
		"too-many-reservations": func(s *admissionRecord) { s.ledger.reservations = make([]ownedMemoryReservation, 1025) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			state := recordExample()
			change(&state)
			if data, err := encodeAdmissionRecord(state); err == nil || data != nil {
				t.Fatalf("invalid state encoded: %v", err)
			}
		})
	}
}

func recordReseal(data []byte) []byte {
	body := data[:len(data)-sha256.Size]
	sum := sha256.Sum256(body)
	copy(data[len(body):], sum[:])
	return data
}

func TestRecordMalformedBytes(t *testing.T) {
	valid, err := encodeAdmissionRecord(recordExample())
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < len(valid); n++ {
		if state, err := decodeAdmissionRecord(valid[:n]); err == nil || !reflect.DeepEqual(state, admissionRecord{}) {
			t.Fatalf("accepted truncation %d", n)
		}
	}
	for n := range valid {
		bad := slices.Clone(valid)
		bad[n] ^= 0x80
		if _, err := decodeAdmissionRecord(bad); err == nil {
			t.Fatalf("accepted corruption %d", n)
		}
	}
	for _, bad := range [][]byte{append(slices.Clone(valid), 0), make([]byte, (1<<20)+1)} {
		if _, err := decodeAdmissionRecord(bad); err == nil {
			t.Fatal("accepted trailing or oversized bytes")
		}
	}
	for _, change := range []func([]byte){
		func(b []byte) { b[4] = 2 },
		func(b []byte) { b[5] = 9 },
		func(b []byte) { binary.BigEndian.PutUint16(b[6:8], math.MaxUint16) },
	} {
		bad := slices.Clone(valid)
		change(bad)
		recordReseal(bad)
		if _, err := decodeAdmissionRecord(bad); err == nil {
			t.Fatal("accepted malformed resealed bytes")
		}
	}
}

func TestRecordResealedAntiTests(t *testing.T) {
	valid, err := encodeAdmissionRecord(recordExample())
	if err != nil {
		t.Fatal(err)
	}
	bootEnd := 8 + len("boot-1")
	ownerCount := bootEnd + 24 + 1 + 8 + 8 + 32
	ownerStart := ownerCount + 2
	firstOwnerSize := 2 + len("carrie") + 2 + len("attempt-2")
	secondOwnerSize := 2 + len("game") + 2 + len("attempt-1")
	reservationCount := ownerStart + firstOwnerSize + secondOwnerSize
	reservationStart := reservationCount + 2
	reservationSize := 2 + 1 + 2 + len("game") + 2 + len("attempt-1") + 8
	cases := map[string]func([]byte) []byte{
		"bad-boolean":           func(b []byte) []byte { b[bootEnd+24] = 2; return b },
		"nan-policy":            func(b []byte) []byte { binary.BigEndian.PutUint64(b[bootEnd:], math.Float64bits(math.NaN())); return b },
		"owner-count":           func(b []byte) []byte { binary.BigEndian.PutUint16(b[ownerCount:], math.MaxUint16); return b },
		"reservation-count":     func(b []byte) []byte { binary.BigEndian.PutUint16(b[reservationCount:], math.MaxUint16); return b },
		"owner-token-length":    func(b []byte) []byte { binary.BigEndian.PutUint16(b[ownerStart:], math.MaxUint16); return b },
		"owner-token-control":   func(b []byte) []byte { b[ownerStart+2] = 0; return b },
		"duplicate-reservation": func(b []byte) []byte { b[reservationStart+reservationSize+2] = 'a'; return b },
		"unknown-owner":         func(b []byte) []byte { b[reservationStart+5] = 'x'; return b },
		"unsorted-reservations": func(b []byte) []byte {
			first := slices.Clone(b[reservationStart : reservationStart+reservationSize])
			copy(b[reservationStart:], b[reservationStart+reservationSize:reservationStart+2*reservationSize])
			copy(b[reservationStart+reservationSize:], first)
			return b
		},
		"unsorted-owners": func(b []byte) []byte {
			owners := slices.Clone(b[ownerStart:reservationCount])
			copy(b[ownerStart:], owners[firstOwnerSize:])
			copy(b[ownerStart+secondOwnerSize:], owners[:firstOwnerSize])
			return b
		},
		"trailing-body": func(b []byte) []byte { return append(b[:len(b)-32], make([]byte, 33)...) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			bad := recordReseal(mutate(slices.Clone(valid)))
			if state, err := decodeAdmissionRecord(bad); err == nil || !reflect.DeepEqual(state, admissionRecord{}) {
				t.Fatal("accepted resealed invalid state")
			}
		})
	}
	intent := admissionIntent{version: 1, phase: intentPrepared, boot: "boot-1", owner: recordExample().owners[0], predecessor: recordExample().predecessor, stateDigest: sha256.Sum256(valid)}
	state := recordExample()
	intent.next = &state
	encoded, err := encodeAdmissionIntent(intent)
	if err != nil {
		t.Fatal(err)
	}
	lengthOffset := 7 + 2 + len(intent.boot) + 2 + len(intent.owner.name) + 2 + len(intent.owner.incarnation) + 64
	for _, change := range []func([]byte){
		func(b []byte) { b[6] = 0 },
		func(b []byte) { binary.BigEndian.PutUint64(b[lengthOffset:], math.MaxUint64) },
		func(b []byte) { b[lengthOffset-1] ^= 1 },
		func(b []byte) { b[lengthOffset-33] ^= 1 },
	} {
		bad := slices.Clone(encoded)
		change(bad)
		recordReseal(bad)
		if got, err := decodeAdmissionIntent(bad); err == nil || !reflect.DeepEqual(got, admissionIntent{}) {
			t.Fatal("accepted resealed invalid intent")
		}
	}
	if _, err := decodeAdmissionIntent(make([]byte, (1<<20)+1)); err == nil {
		t.Fatal("accepted oversized intent")
	}
}

func TestRecordBoundaryAndGoldenBody(t *testing.T) {
	state := admissionRecord{version: 1, boot: "b", policy: hostPolicy{loadLimit: 1, swapLimit: 1, diskFloor: 1}, ledger: ownedReservationLedger{limit: 1}}
	data, err := encodeAdmissionRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{'A', 'Z', 'A', 'S', 1, 1, 0, 1, 'b', 0x3f, 0xf0, 0, 0, 0, 0, 0, 0}
	want = append(want, 0, 0, 0, 0, 0, 0, 0, 1)
	want = append(want, 0, 0, 0, 0, 0, 0, 0, 1)
	want = append(want, make([]byte, 9)...)
	want = append(want, 0, 0, 0, 0, 0, 0, 0, 1)
	want = append(want, make([]byte, 36)...)
	sum := sha256.Sum256(want)
	want = append(want, sum[:]...)
	if !bytes.Equal(data, want) {
		t.Fatalf("wire body changed: %x", data)
	}
	if _, err := decodeAdmissionRecord(want); err != nil {
		t.Fatal(err)
	}
	state.boot = strings.Repeat("b", 128)
	state.stop = stopState{stopped: true, generation: math.MaxUint64}
	state.ledger.limit = math.MaxUint64
	for j := 0; j < 1024; j++ {
		owner := reservationOwner{name: reservationOwnerName(fmt.Sprintf("owner-%04d", j)), incarnation: "i"}
		state.owners = append(state.owners, owner)
		state.ledger.reservations = append(state.ledger.reservations, ownedMemoryReservation{id: reservationIdentity(fmt.Sprintf("id-%04d", j)), owner: owner, amount: 1})
	}
	data, err = encodeAdmissionRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeAdmissionRecord(data); err != nil {
		t.Fatal(err)
	}
	state.ledger.reservations = state.ledger.reservations[:1]
	state.ledger.reservations[0].amount = math.MaxUint64
	if _, err := encodeAdmissionRecord(state); err != nil {
		t.Fatalf("full-width exact fit: %v", err)
	}
}

func TestRecordIntentBinding(t *testing.T) {
	state := recordExample()
	encoded, err := encodeAdmissionRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	base := admissionIntent{version: 1, phase: intentObserving, boot: state.boot, owner: state.owners[0], predecessor: state.predecessor}
	for _, prepared := range []bool{false, true} {
		intent := base
		if prepared {
			intent.phase = intentPrepared
			intent.next = &state
			intent.stateDigest = sha256.Sum256(encoded)
		}
		data, err := encodeAdmissionIntent(intent)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeAdmissionIntent(data)
		if err != nil || got.version != intent.version || got.phase != intent.phase || got.boot != intent.boot || got.owner != intent.owner || got.predecessor != intent.predecessor || got.stateDigest != intent.stateDigest {
			t.Fatalf("intent round trip: %+v %v", got, err)
		}
		again, err := encodeAdmissionIntent(got)
		if err != nil || !bytes.Equal(data, again) {
			t.Fatalf("intent unstable: %v", err)
		}
		for n := 0; n < len(data); n++ {
			if _, err := decodeAdmissionIntent(data[:n]); err == nil {
				t.Fatalf("intent truncation %d", n)
			}
		}
		for n := range data {
			bad := slices.Clone(data)
			bad[n] ^= 1
			if _, err := decodeAdmissionIntent(bad); err == nil {
				t.Fatalf("intent corruption %d", n)
			}
		}
		if _, err := decodeAdmissionRecord(data); err == nil {
			t.Fatal("intent accepted as state")
		}
		if _, err := decodeAdmissionIntent(append(slices.Clone(data), 0)); err == nil {
			t.Fatal("intent trailing bytes")
		}
		clear(data)
		if got.boot != "boot-1" {
			t.Fatal("intent aliases source")
		}
		if prepared {
			got.next.ledger.reservations[0].amount = 99
			if state.ledger.reservations[1].amount != 8 {
				t.Fatal("intent aliases state")
			}
		}
	}
	if _, err := decodeAdmissionIntent(encoded); err == nil {
		t.Fatal("state accepted as intent")
	}
	cases := map[string]func(*admissionIntent){
		"version":               func(i *admissionIntent) { i.version = 0 },
		"phase":                 func(i *admissionIntent) { i.phase = 42 },
		"boot":                  func(i *admissionIntent) { i.boot = "" },
		"owner":                 func(i *admissionIntent) { i.owner.incarnation = "" },
		"predecessor":           func(i *admissionIntent) { i.predecessor = [32]byte{} },
		"observing-next":        func(i *admissionIntent) { i.next = &state },
		"observing-digest":      func(i *admissionIntent) { i.stateDigest[0] = 1 },
		"prepared-missing":      func(i *admissionIntent) { i.phase = intentPrepared },
		"prepared-wrong-digest": func(i *admissionIntent) { i.phase = intentPrepared; i.next = &state },
		"prepared-wrong-boot": func(i *admissionIntent) {
			i.phase = intentPrepared
			i.next = &state
			i.stateDigest = sha256.Sum256(encoded)
			i.boot = "other"
		},
		"prepared-wrong-predecessor": func(i *admissionIntent) {
			i.phase = intentPrepared
			i.next = &state
			i.stateDigest = sha256.Sum256(encoded)
			i.predecessor[0] ^= 1
		},
		"prepared-wrong-owner": func(i *admissionIntent) {
			i.phase = intentPrepared
			i.next = &state
			i.stateDigest = sha256.Sum256(encoded)
			i.owner.incarnation = "stale"
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			intent := base
			change(&intent)
			if data, err := encodeAdmissionIntent(intent); err == nil || data != nil {
				t.Fatalf("invalid intent encoded: %v", err)
			}
		})
	}
}

func TestRecordRapidCanonical(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		state := recordExample()
		state.stop = stopState{stopped: rapid.Bool().Draw(t, "stopped"), generation: rapid.Uint64Range(0, math.MaxUint64-1).Draw(t, "generation")}
		state.ledger.limit = math.MaxUint64
		n := rapid.IntRange(0, 40).Draw(t, "count")
		state.ledger.reservations = nil
		for j := 0; j < n; j++ {
			state.ledger.reservations = append(state.ledger.reservations, ownedMemoryReservation{id: reservationIdentity(fmt.Sprintf("id-%03d", j)), owner: state.owners[j%2], amount: byteCount(rapid.Uint64Range(1, 1000).Draw(t, "amount"))})
		}
		first, err := encodeAdmissionRecord(state)
		if err != nil {
			t.Fatal(err)
		}
		slices.Reverse(state.owners)
		slices.Reverse(state.ledger.reservations)
		second, err := encodeAdmissionRecord(state)
		if err != nil || !bytes.Equal(first, second) {
			t.Fatal("permutation changed encoding")
		}
		decoded, err := decodeAdmissionRecord(first)
		if err != nil {
			t.Fatal(err)
		}
		third, err := encodeAdmissionRecord(decoded)
		if err != nil || !bytes.Equal(first, third) {
			t.Fatal("round trip changed bytes")
		}
		intent := admissionIntent{version: 1, phase: intentPrepared, boot: state.boot, owner: state.owners[0], predecessor: state.predecessor, stateDigest: sha256.Sum256(first), next: &state}
		encoded, err := encodeAdmissionIntent(intent)
		if err != nil {
			t.Fatal(err)
		}
		decodedIntent, err := decodeAdmissionIntent(encoded)
		if err != nil {
			t.Fatal(err)
		}
		again, err := encodeAdmissionIntent(decodedIntent)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatal("intent round trip changed bytes")
		}
	})
}

func FuzzRecordDecode(f *testing.F) {
	seed, err := encodeAdmissionRecord(recordExample())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte{})
	intent, err := encodeAdmissionIntent(admissionIntent{version: 1, phase: intentObserving, boot: "boot-1", owner: reservationOwner{name: "game", incarnation: "attempt-1"}, predecessor: sha256.Sum256(seed)})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(intent)
	state := recordExample()
	prepared, err := encodeAdmissionIntent(admissionIntent{version: 1, phase: intentPrepared, boot: state.boot, owner: state.owners[0], predecessor: state.predecessor, stateDigest: sha256.Sum256(seed), next: &state})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(prepared)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) >= 6+sha256.Size && len(data) <= 1<<20 {
			resealed := recordReseal(slices.Clone(data))
			if state, err := decodeAdmissionRecord(resealed); err == nil {
				encoded, err := encodeAdmissionRecord(state)
				if err != nil || !bytes.Equal(resealed, encoded) {
					t.Fatal("accepted resealed noncanonical state")
				}
			}
			if intent, err := decodeAdmissionIntent(resealed); err == nil {
				encoded, err := encodeAdmissionIntent(intent)
				if err != nil || !bytes.Equal(resealed, encoded) {
					t.Fatal("accepted resealed noncanonical intent")
				}
			}
		}
		state, err := decodeAdmissionRecord(data)
		if err == nil {
			encoded, err := encodeAdmissionRecord(state)
			if err != nil || !bytes.Equal(data, encoded) {
				t.Fatal("accepted noncanonical state")
			}
		} else if !reflect.DeepEqual(state, admissionRecord{}) {
			t.Fatal("partial state on failure")
		}
		i, err := decodeAdmissionIntent(data)
		if err == nil {
			encoded, err := encodeAdmissionIntent(i)
			if err != nil || !bytes.Equal(data, encoded) {
				t.Fatal("accepted noncanonical intent")
			}
		} else if !reflect.DeepEqual(i, admissionIntent{}) {
			t.Fatal("partial intent on failure")
		}
	})
}
