package budget

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
)

// workPolicy is the host policy that the work tests share.
var workPolicy = hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}

// workOwner is the reservation owner that the work tests share.
var workOwner = reservationOwner{name: "worker", incarnation: "boot-process"}

// readySample is a host sample with every reading known and healthy.
var readySample = hostSample{load: 1, swap: 1, freeDisk: 40, loadKnown: true, swapKnown: true, diskKnown: true}

// twoEntryRecord is a sound record that holds the reservations a and b.
func twoEntryRecord() workRecord {
	return workRecord{boot: "boot", policy: workPolicy,
		ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1},
			held: []workReservation{
				{id: "a", owner: workOwner, need: workNeed{memory: 10, workers: 1}},
				{id: "b", owner: workOwner, need: workNeed{memory: 20, workers: 1, gpuSlots: 1}},
			}}}
}

// Byte offsets into the body of twoEntryRecord once encoded: the fields that
// follow the five header bytes and the boot token "boot".
const (
	offLoad       = 11
	offSwap       = 19
	offDisk       = 27
	offStopped    = 35
	offGeneration = 36
	offMemory     = 44
	offCount      = 92
	offFirst      = 94
	entryLength   = 41
	offSecond     = offFirst + entryLength
)

// workBody returns the encoded body of twoEntryRecord without its digest.
func workBody(t *testing.T) []byte {
	t.Helper()
	encoded, err := encodeWorkRecord(twoEntryRecord())
	if err != nil {
		t.Fatal(err)
	}
	return encoded[:len(encoded)-sha256.Size]
}

// sealWork appends the digest to a body, so that only the body's own faults
// can make a decode fail.
func sealWork(body []byte) []byte {
	sum := sha256.Sum256(body)
	return append(append([]byte(nil), body...), sum[:]...)
}

// TestWorkBodyOffsets checks that the offsets used below land on the fields
// they name.
func TestWorkBodyOffsets(t *testing.T) {
	body := workBody(t)
	if got := binary.BigEndian.Uint64(body[offLoad:]); got != math.Float64bits(4) {
		t.Errorf("load bits at %d: %x", offLoad, got)
	}
	if binary.BigEndian.Uint64(body[offSwap:]) != 100 || binary.BigEndian.Uint64(body[offDisk:]) != 35 {
		t.Error("swap or disk not at the expected offset")
	}
	if binary.BigEndian.Uint64(body[offMemory:]) != 100 {
		t.Error("memory limit not at the expected offset")
	}
	if binary.BigEndian.Uint16(body[offCount:]) != 2 {
		t.Error("count not at the expected offset")
	}
	if body[offFirst+2] != 'a' || body[offSecond+2] != 'b' || len(body) != offSecond+entryLength {
		t.Error("entries not at the expected offsets")
	}
}

// TestEncodeWorkRecordRefusesUnsoundRecords checks each fault the encoder
// refuses.
func TestEncodeWorkRecordRefusesUnsoundRecords(t *testing.T) {
	cases := map[string]func(*workRecord){
		"empty boot":          func(r *workRecord) { r.boot = "" },
		"boot with a space":   func(r *workRecord) { r.boot = "a b" },
		"zero load limit":     func(r *workRecord) { r.policy.loadLimit = 0 },
		"not a number load":   func(r *workRecord) { r.policy.loadLimit = math.NaN() },
		"infinite load limit": func(r *workRecord) { r.policy.loadLimit = math.Inf(1) },
		"zero swap limit":     func(r *workRecord) { r.policy.swapLimit = 0 },
		"zero disk floor":     func(r *workRecord) { r.policy.diskFloor = 0 },
		"final generation clear": func(r *workRecord) {
			r.stop = stopState{generation: lastStopGeneration}
		},
		"zero memory limit": func(r *workRecord) { r.ledger.limits.memory = 0 },
		"duplicate id":      func(r *workRecord) { r.ledger.held[1].id = "a" },
	}
	for name, mutate := range cases {
		record := twoEntryRecord()
		mutate(&record)
		if data, err := encodeWorkRecord(record); !errors.Is(err, errWorkRecord) || data != nil {
			t.Errorf("%s: got %d bytes, %v", name, len(data), err)
		}
	}
	final := twoEntryRecord()
	final.stop = stopState{stopped: true, generation: lastStopGeneration}
	if _, err := encodeWorkRecord(final); err != nil {
		t.Errorf("final stop that is in force: %v", err)
	}
}

// TestDecodeWorkRecordRefusesEveryTruncation checks that a record cut short
// at any length, with its digest made right again, is refused.
func TestDecodeWorkRecordRefusesEveryTruncation(t *testing.T) {
	body := workBody(t)
	if _, err := decodeWorkRecord(sealWork(body)); err != nil {
		t.Fatalf("whole record: %v", err)
	}
	for length := 0; length < len(body); length++ {
		if _, err := decodeWorkRecord(sealWork(body[:length])); !errors.Is(err, errWorkRecord) {
			t.Errorf("cut at %d: %v", length, err)
		}
	}
}

// TestDecodeWorkRecordRefusesBadFields checks each fault in a record whose
// digest is right.
func TestDecodeWorkRecordRefusesBadFields(t *testing.T) {
	put64 := func(b []byte, at int, v uint64) { binary.BigEndian.PutUint64(b[at:], v) }
	cases := map[string]func([]byte) []byte{
		"wrong marker":       func(b []byte) []byte { b[0] = 'X'; return b },
		"wrong version":      func(b []byte) []byte { b[4] = 2; return b },
		"zero boot length":   func(b []byte) []byte { b[5], b[6] = 0, 0; return b },
		"long boot length":   func(b []byte) []byte { b[5], b[6] = 0, 129; return b },
		"bad boot character": func(b []byte) []byte { b[7] = '/'; return b },
		"zero load":          func(b []byte) []byte { put64(b, offLoad, 0); return b },
		"infinite load":      func(b []byte) []byte { put64(b, offLoad, math.Float64bits(math.Inf(1))); return b },
		"zero swap":          func(b []byte) []byte { put64(b, offSwap, 0); return b },
		"zero disk":          func(b []byte) []byte { put64(b, offDisk, 0); return b },
		"stop byte two":      func(b []byte) []byte { b[offStopped] = 2; return b },
		"final generation clear": func(b []byte) []byte {
			put64(b, offGeneration, math.MaxUint64)
			return b
		},
		"zero memory limit": func(b []byte) []byte { put64(b, offMemory, 0); return b },
		"too many entries": func(b []byte) []byte {
			binary.BigEndian.PutUint16(b[offCount:], admissionEntryLimit+1)
			return b
		},
		"ids repeat":        func(b []byte) []byte { b[offSecond+2] = 'a'; return b },
		"ids fall":          func(b []byte) []byte { b[offFirst+2], b[offSecond+2] = 'b', 'a'; return b },
		"bad owner name":    func(b []byte) []byte { b[offFirst+5] = ' '; return b },
		"zero workers need": func(b []byte) []byte { binary.BigEndian.PutUint32(b[offFirst+entryLength-8:], 0); return b },
		"bytes left over":   func(b []byte) []byte { return append(b, 0) },
	}
	for name, mutate := range cases {
		body := mutate(append([]byte(nil), workBody(t)...))
		if _, err := decodeWorkRecord(sealWork(body)); !errors.Is(err, errWorkRecord) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := decodeWorkRecord(make([]byte, workRecordLimit+1)); !errors.Is(err, errWorkRecord) {
		t.Errorf("oversize data: %v", err)
	}
	if _, err := decodeWorkRecord(append(workBody(t), make([]byte, sha256.Size)...)); !errors.Is(err, errWorkRecord) {
		t.Errorf("wrong digest: %v", err)
	}
}

// TestDecodeWorkRecordKeepsFinalStop checks that a stop in force at the last
// generation decodes with its fields intact.
func TestDecodeWorkRecordKeepsFinalStop(t *testing.T) {
	record := twoEntryRecord()
	record.stop = stopState{stopped: true, generation: lastStopGeneration}
	record.predecessor[0] = 9
	encoded, err := encodeWorkRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeWorkRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.stop != record.stop || decoded.predecessor[0] != 9 || decoded.boot != "boot" ||
		decoded.ledger.held[1].need != (workNeed{memory: 20, workers: 1, gpuSlots: 1}) {
		t.Errorf("decoded %+v", decoded)
	}
}

// TestWorkReaderTake checks the bounds of take.
func TestWorkReaderTake(t *testing.T) {
	reader := workReader{data: []byte{1, 2, 3}}
	if _, err := reader.take(-1); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("negative count: %v", err)
	}
	if _, err := reader.take(4); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("count past the end: %v", err)
	}
	got, err := reader.take(3)
	if err != nil || !bytes.Equal(got, []byte{1, 2, 3}) || reader.offset != 3 {
		t.Errorf("exact count: %v %v", got, err)
	}
	if _, err := reader.uint32(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("uint32 at the end: %v", err)
	}
	if _, err := reader.uint64(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("uint64 at the end: %v", err)
	}
	if _, err := reader.token(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("token at the end: %v", err)
	}
	short := workReader{data: []byte{0, 5, 'a'}}
	if _, err := short.token(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("token cut inside its text: %v", err)
	}
	numbers := workReader{data: []byte{0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 1, 0}}
	if v, err := numbers.uint32(); err != nil || v != 7 {
		t.Errorf("uint32: %v %v", v, err)
	}
	if v, err := numbers.uint64(); err != nil || v != 256 {
		t.Errorf("uint64: %v %v", v, err)
	}
}

// TestWorkTransitionProposesReservation checks a healthy host: the
// reservation joins the new record and the old record is untouched.
func TestWorkTransitionProposesReservation(t *testing.T) {
	state := twoEntryRecord()
	state.ledger.held = state.ledger.held[:1]
	before, err := encodeWorkRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	request := workAdmissionRequest{policy: workPolicy, sample: readySample,
		reservation: workReservation{id: "c", owner: workOwner, need: workNeed{memory: 30, workers: 1}}}
	transition, err := proposeWorkTransition(state, request)
	if err != nil {
		t.Fatal(err)
	}
	if transition.outcome != admissionProposed || len(transition.next.ledger.held) != 2 ||
		transition.next.ledger.held[1].id != "c" || len(state.ledger.held) != 1 {
		t.Fatalf("transition: %+v", transition)
	}
	if transition.previous != sha256.Sum256(before) || transition.next.predecessor != transition.previous {
		t.Error("predecessor is not the digest of the old encoding")
	}
	decoded, err := decodeWorkRecord(transition.encoded)
	if err != nil || len(decoded.ledger.held) != 2 {
		t.Errorf("encoded transition: %+v %v", decoded, err)
	}
}

// TestWorkTransitionRefusesUnsoundState checks that a state that cannot be
// encoded gives no transition.
func TestWorkTransitionRefusesUnsoundState(t *testing.T) {
	state := twoEntryRecord()
	state.boot = ""
	if _, err := proposeWorkTransition(state, workAdmissionRequest{policy: workPolicy, sample: readySample}); !errors.Is(err, errWorkRecord) {
		t.Errorf("got %v", err)
	}
}

// TestWorkLedgerRefusesUnsoundLedgers checks the faults that usage, release
// and reserve all refuse.
func TestWorkLedgerRefusesUnsoundLedgers(t *testing.T) {
	limits := workLimits{memory: 100, workers: 4, gpuSlots: 1}
	held := func(id reservationIdentity, need workNeed) workReservation {
		return workReservation{id: id, owner: workOwner, need: need}
	}
	good := workNeed{memory: 10, workers: 1}
	cases := map[string]workLedger{
		"zero memory limit":  {limits: workLimits{workers: 4}},
		"zero worker limit":  {limits: workLimits{memory: 100}},
		"bad id":             {limits: limits, held: []workReservation{held("a b", good)}},
		"bad owner":          {limits: limits, held: []workReservation{{id: "a", need: good}}},
		"zero need":          {limits: limits, held: []workReservation{held("a", workNeed{})}},
		"two gpu slots":      {limits: limits, held: []workReservation{held("a", workNeed{memory: 1, workers: 1, gpuSlots: 2})}},
		"duplicate id":       {limits: limits, held: []workReservation{held("a", good), held("a", good)}},
		"memory over limit":  {limits: limits, held: []workReservation{held("a", workNeed{memory: 101, workers: 1})}},
		"memory sum":         {limits: limits, held: []workReservation{held("a", workNeed{memory: 60, workers: 1}), held("b", workNeed{memory: 60, workers: 1})}},
		"workers over limit": {limits: limits, held: []workReservation{held("a", workNeed{memory: 1, workers: 5})}},
		"workers sum":        {limits: limits, held: []workReservation{held("a", workNeed{memory: 1, workers: 3}), held("b", workNeed{memory: 1, workers: 3})}},
		"gpu over limit":     {limits: workLimits{memory: 100, workers: 4}, held: []workReservation{held("a", workNeed{memory: 1, workers: 1, gpuSlots: 1})}},
		"gpu sum":            {limits: limits, held: []workReservation{held("a", workNeed{memory: 1, workers: 1, gpuSlots: 1}), held("b", workNeed{memory: 1, workers: 1, gpuSlots: 1})}},
	}
	for name, ledger := range cases {
		if _, ok := ledger.usage(); ok {
			t.Errorf("%s: usage accepted", name)
		}
		if _, status := ledger.releasing("a", workOwner); status != releaseInvalid {
			t.Errorf("%s: release gave %v", name, status)
		}
		if proposal := ledger.reserving(held("z", good)); proposal.disposition != workInvalid {
			t.Errorf("%s: reserve gave %v", name, proposal.disposition)
		}
	}
	sound := workLedger{limits: limits, held: []workReservation{held("a", workNeed{memory: 10, workers: 1, gpuSlots: 1}), held("b", workNeed{memory: 5, workers: 2})}}
	if used, ok := sound.usage(); !ok || used != (workNeed{memory: 15, workers: 3, gpuSlots: 1}) {
		t.Errorf("sound usage: %+v %v", used, ok)
	}
}

// TestWorkLedgerReleaseRefusesBadRequests checks that a bad id or owner is
// invalid even on a sound ledger.
func TestWorkLedgerReleaseRefusesBadRequests(t *testing.T) {
	ledger := workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}
	if _, status := ledger.releasing("a b", workOwner); status != releaseInvalid {
		t.Errorf("bad id: %v", status)
	}
	if _, status := ledger.releasing("a", reservationOwner{}); status != releaseInvalid {
		t.Errorf("bad owner: %v", status)
	}
	other := reservationOwner{name: "other", incarnation: "boot-process"}
	held := workLedger{limits: ledger.limits, held: []workReservation{{id: "a", owner: workOwner, need: workNeed{memory: 1, workers: 1}}}}
	if _, status := held.releasing("a", other); status != releaseOwnerMismatch {
		t.Errorf("other owner: %v", status)
	}
}

// TestWorkLedgerReserveHoldsAtEntryLimit checks that a ledger already at the
// entry limit refuses another reservation even when capacity remains.
func TestWorkLedgerReserveHoldsAtEntryLimit(t *testing.T) {
	ledger := workLedger{limits: workLimits{memory: 1 << 20, workers: 1 << 20, gpuSlots: 1}}
	for i := 0; i < admissionEntryLimit; i++ {
		id := reservationIdentity("r" + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676)))
		ledger.held = append(ledger.held, workReservation{id: id, owner: workOwner, need: workNeed{memory: 1, workers: 1}})
	}
	if _, ok := ledger.usage(); !ok {
		t.Fatal("full ledger is not sound")
	}
	proposal := ledger.reserving(workReservation{id: "extra", owner: workOwner, need: workNeed{memory: 1, workers: 1}})
	if proposal.disposition != workInvalid || proposal.next != nil {
		t.Errorf("got %+v", proposal)
	}
}

// rawAdmissionState builds a sealed state record body by hand with the given
// owners in the given order and no reservations, so that its shape is right
// even when its contents are not canonical.
func rawAdmissionState(owners []reservationOwner, trailing []byte) []byte {
	w := admissionWriter{data: []byte{'A', 'Z', 'A', 'S', admissionRecordVersion, admissionStateKind}}
	w.token("boot-1")
	w.u64(math.Float64bits(2.5))
	w.u64(10)
	w.u64(20)
	w.u8(0)
	w.u64(0)
	w.u64(100)
	w.data = append(w.data, make([]byte, sha256.Size)...)
	w.u16(len(owners))
	for _, owner := range owners {
		w.owner(owner)
	}
	w.u16(0)
	w.data = append(w.data, trailing...)
	sum := sha256.Sum256(w.data)
	return append(w.data, sum[:]...)
}

// rawAdmissionIntent builds a sealed intent whose carried state bytes are
// whatever the caller supplies.
func rawAdmissionIntent(nested []byte) []byte {
	w := admissionWriter{data: []byte{'A', 'Z', 'A', 'S', admissionRecordVersion, admissionIntentKind}}
	w.u8(uint8(intentPrepared))
	w.token("boot-1")
	w.owner(reservationOwner{name: "game", incarnation: "attempt-1"})
	w.data = append(w.data, make([]byte, 2*sha256.Size)...)
	w.u64(uint64(len(nested)))
	w.data = append(w.data, nested...)
	sum := sha256.Sum256(w.data)
	return append(w.data, sum[:]...)
}

// TestDecodeAdmissionIntentRefusesBadNestedState checks that an intent whose
// carried state is not a sound record is refused at each stage.
func TestDecodeAdmissionIntentRefusesBadNestedState(t *testing.T) {
	first := reservationOwner{name: "carrie", incarnation: "attempt-2"}
	second := reservationOwner{name: "game", incarnation: "attempt-1"}
	cases := map[string][]byte{
		"not a state record":   []byte("not a record"),
		"bytes after the body": rawAdmissionState([]reservationOwner{first, second}, []byte{0}),
		"owners out of order":  rawAdmissionState([]reservationOwner{second, first}, nil),
	}
	for name, nested := range cases {
		if _, err := decodeAdmissionIntent(rawAdmissionIntent(nested)); !errors.Is(err, errAdmissionRecord) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestWorkLedgerReleasesLaterEntry checks that release skips earlier
// entries with other ids and removes only the one named.
func TestWorkLedgerReleasesLaterEntry(t *testing.T) {
	ledger := twoEntryRecord().ledger
	released, status := ledger.releasing("b", workOwner)
	if status != releaseProposed || len(released.held) != 1 || released.held[0].id != "a" {
		t.Errorf("got %v %+v", status, released)
	}
	if len(ledger.held) != 2 {
		t.Error("original ledger changed")
	}
}
