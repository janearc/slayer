package budget

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
)

var errWorkRecord = errors.New("invalid shared work record")

const workRecordLimit = 1 << 20

type workRecord struct {
	boot        string
	policy      hostPolicy
	stop        stopState
	ledger      workLedger
	predecessor [sha256.Size]byte
}

// encodeWorkRecord returns the bytes of record, or errWorkRecord if its
// policy, stop state or ledger is not sound. Reservations are sorted by id so
// equal records give equal bytes. The bytes start with the marker AZWK and
// version 1, end with a SHA-256 digest, and may not pass 1 MiB.
func encodeWorkRecord(record workRecord) ([]byte, error) {
	if !admissionTokenValid(record.boot) || !(record.policy.loadLimit > 0) || math.IsInf(record.policy.loadLimit, 0) || record.policy.swapLimit == 0 || record.policy.diskFloor == 0 || record.stop.generation == lastStopGeneration && !record.stop.stopped {
		return nil, errWorkRecord
	}
	if _, valid := record.ledger.usage(); !valid {
		return nil, errWorkRecord
	}
	held := slices.Clone(record.ledger.held)
	slices.SortFunc(held, func(left, right workReservation) int { return strings.Compare(string(left.id), string(right.id)) })
	writer := admissionWriter{data: []byte{'A', 'Z', 'W', 'K', 1}}
	writer.token(record.boot)
	writer.u64(math.Float64bits(record.policy.loadLimit))
	writer.u64(uint64(record.policy.swapLimit))
	writer.u64(uint64(record.policy.diskFloor))
	if record.stop.stopped {
		writer.u8(1)
	} else {
		writer.u8(0)
	}
	writer.u64(record.stop.generation)
	writer.u64(uint64(record.ledger.limits.memory))
	writer.data = binary.BigEndian.AppendUint32(writer.data, record.ledger.limits.workers)
	writer.data = binary.BigEndian.AppendUint32(writer.data, record.ledger.limits.gpuSlots)
	writer.data = append(writer.data, record.predecessor[:]...)
	writer.u16(len(held))
	for _, entry := range held {
		writer.token(string(entry.id))
		writer.owner(entry.owner)
		writer.u64(uint64(entry.need.memory))
		writer.data = binary.BigEndian.AppendUint32(writer.data, entry.need.workers)
		writer.data = binary.BigEndian.AppendUint32(writer.data, entry.need.gpuSlots)
	}
	if len(writer.data) > workRecordLimit-sha256.Size {
		return nil, errWorkRecord
	}
	digest := sha256.Sum256(writer.data)
	return append(writer.data, digest[:]...), nil
}

type workReader struct {
	data   []byte
	offset int
}

// take returns the next count bytes and moves past them, or
// io.ErrUnexpectedEOF if fewer remain.
func (reader *workReader) take(count int) ([]byte, error) {
	if count < 0 || count > len(reader.data)-reader.offset {
		return nil, io.ErrUnexpectedEOF
	}
	result := reader.data[reader.offset : reader.offset+count]
	reader.offset += count
	return result, nil
}

// token reads a string with a two-byte length in front. A zero length, a
// length over the token limit, or text that is not a valid token gives
// errWorkRecord.
func (reader *workReader) token() (string, error) {
	length, err := reader.take(2)
	if err != nil {
		return "", err
	}
	size := int(binary.BigEndian.Uint16(length))
	if size == 0 || size > admissionTokenLimit {
		return "", errWorkRecord
	}
	data, err := reader.take(size)
	if err != nil {
		return "", err
	}
	value := string(data)
	if !admissionTokenValid(value) {
		return "", errWorkRecord
	}
	return value, nil
}

// uint64 reads an eight-byte big endian number.
func (reader *workReader) uint64() (uint64, error) {
	data, err := reader.take(8)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(data), nil
}

// uint32 reads a four-byte big endian number.
func (reader *workReader) uint32() (uint32, error) {
	data, err := reader.take(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(data), nil
}

// decodeWorkRecord parses data written by encodeWorkRecord. It checks the
// digest, the header, the same policy and stop rules as the encoder, that
// reservation ids strictly increase, that no bytes are left over, and that
// the ledger is sound. Every failure is errWorkRecord.
func decodeWorkRecord(data []byte) (workRecord, error) {
	if len(data) < 5+sha256.Size || len(data) > workRecordLimit {
		return workRecord{}, errWorkRecord
	}
	body, checksum := data[:len(data)-sha256.Size], data[len(data)-sha256.Size:]
	digest := sha256.Sum256(body)
	if !bytes.Equal(digest[:], checksum) {
		return workRecord{}, errWorkRecord
	}
	reader := workReader{data: body}
	header, err := reader.take(5)
	if err != nil || !bytes.Equal(header, []byte{'A', 'Z', 'W', 'K', 1}) {
		return workRecord{}, errWorkRecord
	}
	boot, err := reader.token()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	load, err := reader.uint64()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	swap, err := reader.uint64()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	disk, err := reader.uint64()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	stopBytes, err := reader.take(1)
	if err != nil || stopBytes[0] > 1 {
		return workRecord{}, errWorkRecord
	}
	generation, err := reader.uint64()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	policy := hostPolicy{loadLimit: math.Float64frombits(load), swapLimit: byteCount(swap), diskFloor: byteCount(disk)}
	stop := stopState{stopped: stopBytes[0] == 1, generation: generation}
	if !(policy.loadLimit > 0) || math.IsInf(policy.loadLimit, 0) || policy.swapLimit == 0 || policy.diskFloor == 0 || stop.generation == lastStopGeneration && !stop.stopped {
		return workRecord{}, errWorkRecord
	}
	memory, err := reader.uint64()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	workers, err := reader.uint32()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	gpu, err := reader.uint32()
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	previous, err := reader.take(sha256.Size)
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	countBytes, err := reader.take(2)
	if err != nil {
		return workRecord{}, errWorkRecord
	}
	count := int(binary.BigEndian.Uint16(countBytes))
	if count > admissionEntryLimit {
		return workRecord{}, errWorkRecord
	}
	record := workRecord{boot: boot, policy: policy, stop: stop, ledger: workLedger{limits: workLimits{memory: byteCount(memory), workers: workers, gpuSlots: gpu}}}
	copy(record.predecessor[:], previous)
	last := ""
	for index := 0; index < count; index++ {
		id, err := reader.token()
		if err != nil || id <= last {
			return workRecord{}, errWorkRecord
		}
		last = id
		name, err := reader.token()
		if err != nil {
			return workRecord{}, errWorkRecord
		}
		incarnation, err := reader.token()
		if err != nil {
			return workRecord{}, errWorkRecord
		}
		amount, err := reader.uint64()
		if err != nil {
			return workRecord{}, errWorkRecord
		}
		workers, err := reader.uint32()
		if err != nil {
			return workRecord{}, errWorkRecord
		}
		gpu, err := reader.uint32()
		if err != nil {
			return workRecord{}, errWorkRecord
		}
		record.ledger.held = append(record.ledger.held, workReservation{id: reservationIdentity(id), owner: reservationOwner{name: reservationOwnerName(name), incarnation: reservationOwnerIdentity(incarnation)}, need: workNeed{memory: byteCount(amount), workers: workers, gpuSlots: gpu}})
	}
	if reader.offset != len(body) {
		return workRecord{}, errWorkRecord
	}
	if _, valid := record.ledger.usage(); !valid {
		return workRecord{}, errWorkRecord
	}
	return record, nil
}
