package budget

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"slices"
	"strings"
)

const (
	admissionRecordVersion = 1
	admissionRecordLimit   = 1 << 20
	admissionEntryLimit    = 1024
	admissionTokenLimit    = 128
	admissionStateKind     = 1
	admissionIntentKind    = 2
)

var errAdmissionRecord = errors.New("invalid admission record")

type admissionRecord struct {
	version     uint8
	boot        string
	policy      hostPolicy
	stop        stopState
	owners      []reservationOwner
	ledger      ownedReservationLedger
	predecessor [sha256.Size]byte
}

type admissionIntentPhase uint8

const (
	intentObserving admissionIntentPhase = 1
	intentPrepared  admissionIntentPhase = 2
)

type admissionIntent struct {
	version     uint8
	phase       admissionIntentPhase
	boot        string
	owner       reservationOwner
	predecessor [sha256.Size]byte
	stateDigest [sha256.Size]byte
	next        *admissionRecord
}

// admissionTokenValid is true for a token of 1 to 128 bytes made only of
// ASCII letters, digits, hyphen, underscore and dot.
func admissionTokenValid(token string) bool {
	if len(token) == 0 || len(token) > admissionTokenLimit {
		return false
	}
	for i := range token {
		c := token[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

// admissionOwnerValid is true when both the name and the incarnation of owner
// are valid tokens.
func admissionOwnerValid(owner reservationOwner) bool {
	return admissionTokenValid(string(owner.name)) && admissionTokenValid(string(owner.incarnation))
}

// valid is true when the record can be encoded as it is: the current version,
// a valid boot token, a policy with a positive finite load limit and a
// nonzero swap limit and disk floor, no stop state at the last generation
// while running, at most 1024 owners and reservations, no repeated owner
// name, every reservation held by a listed owner with the same incarnation,
// and a sound ledger.
func (state admissionRecord) valid() bool {
	if state.version != admissionRecordVersion || !admissionTokenValid(state.boot) ||
		!(state.policy.loadLimit > 0) || math.IsInf(state.policy.loadLimit, 0) ||
		state.policy.swapLimit == 0 || state.policy.diskFloor == 0 ||
		state.stop.generation == lastStopGeneration && !state.stop.stopped ||
		len(state.owners) > admissionEntryLimit || len(state.ledger.reservations) > admissionEntryLimit {
		return false
	}
	owners := make(map[reservationOwnerName]reservationOwnerIdentity, len(state.owners))
	for _, owner := range state.owners {
		if !admissionOwnerValid(owner) {
			return false
		}
		if _, duplicate := owners[owner.name]; duplicate {
			return false
		}
		owners[owner.name] = owner.incarnation
	}
	for _, held := range state.ledger.reservations {
		if !admissionTokenValid(string(held.id)) || owners[held.owner.name] != held.owner.incarnation || !admissionOwnerValid(held.owner) {
			return false
		}
	}
	_, valid := state.ledger.validatedUsage()
	return valid
}

type admissionWriter struct{ data []byte }

// u8 appends one byte.
func (w *admissionWriter) u8(n uint8) { w.data = append(w.data, n) }

// u16 appends n as two bytes, big endian; only the low 16 bits of n are kept.
func (w *admissionWriter) u16(n int) { w.data = binary.BigEndian.AppendUint16(w.data, uint16(n)) }

// u64 appends n as eight bytes, big endian.
func (w *admissionWriter) u64(n uint64) { w.data = binary.BigEndian.AppendUint64(w.data, n) }

// token appends the length of s as two bytes, then the bytes of s.
func (w *admissionWriter) token(s string) { w.u16(len(s)); w.data = append(w.data, s...) }

// owner appends the name then the incarnation of o, each as a token.
func (w *admissionWriter) owner(o reservationOwner) {
	w.token(string(o.name))
	w.token(string(o.incarnation))
}

// seal returns the data with its SHA-256 digest added on the end. It fails if
// the data plus the digest would pass the record size limit of 1 MiB.
func (w *admissionWriter) seal() ([]byte, error) {
	if len(w.data) > admissionRecordLimit-sha256.Size {
		return nil, errAdmissionRecord
	}
	sum := sha256.Sum256(w.data)
	return append(w.data, sum[:]...), nil
}

// encodeAdmissionRecord returns the bytes of state, or an error if state is
// not valid. Owners and reservations are sorted by name and id first, so
// equal states give equal bytes. The bytes start with the marker AZAS, the
// version and the state kind, and end with a SHA-256 digest.
func encodeAdmissionRecord(state admissionRecord) ([]byte, error) {
	if !state.valid() {
		return nil, errAdmissionRecord
	}
	owners := slices.Clone(state.owners)
	slices.SortFunc(owners, func(a, b reservationOwner) int { return strings.Compare(string(a.name), string(b.name)) })
	reservations := slices.Clone(state.ledger.reservations)
	slices.SortFunc(reservations, func(a, b ownedMemoryReservation) int { return strings.Compare(string(a.id), string(b.id)) })
	w := admissionWriter{data: []byte{'A', 'Z', 'A', 'S', state.version, admissionStateKind}}
	w.token(state.boot)
	w.u64(math.Float64bits(state.policy.loadLimit))
	w.u64(uint64(state.policy.swapLimit))
	w.u64(uint64(state.policy.diskFloor))
	w.u8(0)
	if state.stop.stopped {
		w.data[len(w.data)-1] = 1
	}
	w.u64(state.stop.generation)
	w.u64(uint64(state.ledger.limit))
	w.data = append(w.data, state.predecessor[:]...)
	w.u16(len(owners))
	for _, owner := range owners {
		w.owner(owner)
	}
	w.u16(len(reservations))
	for _, held := range reservations {
		w.token(string(held.id))
		w.owner(held.owner)
		w.u64(uint64(held.amount))
	}
	return w.seal()
}

// stateBytes returns the encoded next state that the intent carries, checking
// the intent as it goes. An observing intent must carry no state and no
// digest, and gives nil bytes. A prepared intent must carry a next state with
// the same boot and predecessor that lists the intent's owner and encodes to
// bytes with the stored digest. Anything else is an error.
func (intent admissionIntent) stateBytes() ([]byte, error) {
	if intent.version != admissionRecordVersion || !admissionTokenValid(intent.boot) || !admissionOwnerValid(intent.owner) || intent.predecessor == [sha256.Size]byte{} {
		return nil, errAdmissionRecord
	}
	switch intent.phase {
	case intentObserving:
		if intent.next != nil || intent.stateDigest != [sha256.Size]byte{} {
			return nil, errAdmissionRecord
		}
		return nil, nil
	case intentPrepared:
		if intent.next == nil || intent.next.boot != intent.boot || intent.next.predecessor != intent.predecessor {
			return nil, errAdmissionRecord
		}
		found := false
		for _, owner := range intent.next.owners {
			if owner == intent.owner {
				found = true
				break
			}
		}
		if !found {
			return nil, errAdmissionRecord
		}
		encoded, err := encodeAdmissionRecord(*intent.next)
		if err != nil || sha256.Sum256(encoded) != intent.stateDigest {
			return nil, errAdmissionRecord
		}
		return encoded, nil
	default:
		return nil, errAdmissionRecord
	}
}

// encodeAdmissionIntent returns the bytes of intent: the phase, boot, owner,
// predecessor and digest, then the length and bytes of the next state, sealed
// with a digest. It fails if stateBytes does.
func encodeAdmissionIntent(intent admissionIntent) ([]byte, error) {
	state, err := intent.stateBytes()
	if err != nil {
		return nil, err
	}
	w := admissionWriter{data: []byte{'A', 'Z', 'A', 'S', intent.version, admissionIntentKind}}
	w.u8(uint8(intent.phase))
	w.token(intent.boot)
	w.owner(intent.owner)
	w.data = append(w.data, intent.predecessor[:]...)
	w.data = append(w.data, intent.stateDigest[:]...)
	stateLength := uint64(len(state))
	w.u64(stateLength)
	w.data = append(w.data, state...)
	return w.seal()
}

type admissionReader struct {
	data     []byte
	position int
	bad      bool
	build    bool
}

// take returns the next n bytes and moves past them. If the reader is already
// bad, or fewer than n bytes remain, it marks the reader bad and returns nil.
func (r *admissionReader) take(n int) []byte {
	if r.bad || n < 0 || n > len(r.data)-r.position {
		r.bad = true
		return nil
	}
	result := r.data[r.position : r.position+n]
	r.position += n
	return result
}

// u8 reads one byte, or returns 0 if the read failed.
func (r *admissionReader) u8() uint8 {
	b := r.take(1)
	if b == nil {
		return 0
	}
	return b[0]
}

// u16 reads a two-byte big endian number, or returns 0 if the read failed.
func (r *admissionReader) u16() int {
	b := r.take(2)
	if b == nil {
		return 0
	}
	return int(binary.BigEndian.Uint16(b))
}

// u64 reads an eight-byte big endian number, or returns 0 if the read failed.
func (r *admissionReader) u64() uint64 {
	b := r.take(8)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// token reads a string with a two-byte length in front. A length of zero, or
// above the token limit, marks the reader bad. The text is returned only when
// the reader is building; a scanning reader returns an empty string.
func (r *admissionReader) token() string {
	n := r.u16()
	if n == 0 || n > admissionTokenLimit {
		r.bad = true
		return ""
	}
	b := r.take(n)
	if r.build {
		return string(b)
	}
	return ""
}

// owner reads a name token then an incarnation token.
func (r *admissionReader) owner() reservationOwner {
	return reservationOwner{name: reservationOwnerName(r.token()), incarnation: reservationOwnerIdentity(r.token())}
}

// count reads an entry count. It marks the reader bad and returns 0 if the
// count is above the entry limit, or if that many entries of at least minimum
// bytes each could not fit in the data left.
func (r *admissionReader) count(minimum int) int {
	n := r.u16()
	if n > admissionEntryLimit || n > (len(r.data)-r.position)/minimum {
		r.bad = true
		return 0
	}
	return n
}

// admissionBody checks the size of data, the AZAS marker, the version, the
// kind byte and the trailing SHA-256 digest. It returns the bytes between the
// 6-byte header and the digest, or an error if any check fails.
func admissionBody(data []byte, kind uint8) ([]byte, error) {
	if len(data) < 6+sha256.Size || len(data) > admissionRecordLimit || !bytes.Equal(data[:4], []byte("AZAS")) || data[4] != admissionRecordVersion || data[5] != kind {
		return nil, errAdmissionRecord
	}
	body := data[:len(data)-sha256.Size]
	sum := sha256.Sum256(body)
	if !bytes.Equal(sum[:], data[len(body):]) {
		return nil, errAdmissionRecord
	}
	return body[6:], nil
}

// readAdmissionRecord reads one record body from r and leaves it to the
// caller to check r.bad. A scanning reader only moves along the data; a
// building reader also fills in the owners and reservations. A stop byte
// above 1 marks r bad.
func readAdmissionRecord(r *admissionReader) admissionRecord {
	state := admissionRecord{version: admissionRecordVersion}
	state.boot = r.token()
	state.policy = hostPolicy{loadLimit: math.Float64frombits(r.u64()), swapLimit: byteCount(r.u64()), diskFloor: byteCount(r.u64())}
	stopped := r.u8()
	if stopped > 1 {
		r.bad = true
	}
	state.stop = stopState{stopped: stopped == 1, generation: r.u64()}
	state.ledger.limit = byteCount(r.u64())
	copy(state.predecessor[:], r.take(sha256.Size))
	n := r.count(6)
	if r.build && !r.bad {
		state.owners = make([]reservationOwner, n)
	}
	for j := 0; j < n && !r.bad; j++ {
		owner := r.owner()
		if r.build {
			state.owners[j] = owner
		}
	}
	n = r.count(17)
	if r.build && !r.bad {
		state.ledger.reservations = make([]ownedMemoryReservation, n)
	}
	for j := 0; j < n && !r.bad; j++ {
		held := ownedMemoryReservation{id: reservationIdentity(r.token()), owner: r.owner(), amount: byteCount(r.u64())}
		if r.build {
			state.ledger.reservations[j] = held
		}
	}
	return state
}

// decodeAdmissionRecord parses data as a state record. It reads the body once
// to check its shape and that every byte is used, reads it again to build the
// record, then encodes the result and requires the same bytes as data. Only
// the canonical encoding of a valid record is accepted.
func decodeAdmissionRecord(data []byte) (admissionRecord, error) {
	body, err := admissionBody(data, admissionStateKind)
	if err != nil {
		return admissionRecord{}, err
	}
	scan := admissionReader{data: body}
	readAdmissionRecord(&scan)
	if scan.bad || scan.position != len(body) {
		return admissionRecord{}, errAdmissionRecord
	}
	r := admissionReader{data: body, build: true}
	state := readAdmissionRecord(&r)
	canonical, err := encodeAdmissionRecord(state)
	if r.bad || err != nil || !bytes.Equal(canonical, data) {
		return admissionRecord{}, errAdmissionRecord
	}
	return state, nil
}

// readAdmissionIntent reads an intent body from r and returns it with the
// bytes of the state record that follows. A length over 1 MiB, or past the
// end of the data, marks r bad. The next field of the intent is not filled in
// here.
func readAdmissionIntent(r *admissionReader) (admissionIntent, []byte) {
	intent := admissionIntent{version: admissionRecordVersion, phase: admissionIntentPhase(r.u8())}
	intent.boot = r.token()
	intent.owner = r.owner()
	copy(intent.predecessor[:], r.take(sha256.Size))
	copy(intent.stateDigest[:], r.take(sha256.Size))
	n := r.u64()
	if n > admissionRecordLimit || n > uint64(len(r.data)-r.position) {
		r.bad = true
		return intent, nil
	}
	return intent, r.take(int(n))
}

// decodeAdmissionIntent parses data as an intent. It checks that the body,
// and any state record inside it, is well formed, decodes that record into
// next, and requires that encoding the result gives back data exactly.
func decodeAdmissionIntent(data []byte) (admissionIntent, error) {
	body, err := admissionBody(data, admissionIntentKind)
	if err != nil {
		return admissionIntent{}, err
	}
	scan := admissionReader{data: body}
	_, nested := readAdmissionIntent(&scan)
	if scan.bad || scan.position != len(body) {
		return admissionIntent{}, errAdmissionRecord
	}
	if len(nested) != 0 {
		stateBody, err := admissionBody(nested, admissionStateKind)
		if err != nil {
			return admissionIntent{}, err
		}
		stateScan := admissionReader{data: stateBody}
		readAdmissionRecord(&stateScan)
		if stateScan.bad || stateScan.position != len(stateBody) {
			return admissionIntent{}, errAdmissionRecord
		}
	}
	r := admissionReader{data: body, build: true}
	intent, nested := readAdmissionIntent(&r)
	if len(nested) != 0 {
		state, err := decodeAdmissionRecord(nested)
		if err != nil {
			return admissionIntent{}, err
		}
		intent.next = &state
	}
	canonical, err := encodeAdmissionIntent(intent)
	if r.bad || err != nil || !bytes.Equal(canonical, data) {
		return admissionIntent{}, errAdmissionRecord
	}
	return intent, nil
}
