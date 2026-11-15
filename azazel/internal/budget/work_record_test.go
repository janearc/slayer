package budget

import (
	"bytes"
	"reflect"
	"testing"
)

func TestWorkRecordCanonicalRoundTrip(t *testing.T) {
	owner := reservationOwner{name: "worker", incarnation: "boot-process"}
	record := workRecord{boot: "boot", policy: hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}, ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}, held: []workReservation{{id: "render", owner: owner, need: workNeed{memory: 20, workers: 2, gpuSlots: 1}}, {id: "composite", owner: owner, need: workNeed{memory: 30, workers: 1}}}}}
	encoded, err := encodeWorkRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeWorkRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	encodedAgain, err := encodeWorkRecord(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, encodedAgain) || decoded.ledger.held[0].id != "composite" {
		t.Fatal("record is not canonical")
	}
	if reflect.DeepEqual(record.ledger.held, decoded.ledger.held) {
		t.Fatal("unsorted input was not canonicalized")
	}
}

func TestWorkRecordRefusesCorruptionAndTruncation(t *testing.T) {
	record := workRecord{boot: "boot", policy: hostPolicy{loadLimit: 4, swapLimit: 100, diskFloor: 35}, ledger: workLedger{limits: workLimits{memory: 100, workers: 4, gpuSlots: 1}}}
	encoded, err := encodeWorkRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{nil, encoded[:len(encoded)-1], append([]byte(nil), encoded...)} {
		if len(data) == len(encoded) {
			data[len(data)/2] ^= 1
		}
		if _, err := decodeWorkRecord(data); err == nil {
			t.Fatal("invalid record accepted")
		}
	}
}
