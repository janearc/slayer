package diskcache

import (
	"bytes"
	"strings"
	"testing"
)

// testRecord is a record a test entry is written under.
const testRecord = "kind entry\na 1\n"

func TestDecodedIsWhatEncodedWrote(t *testing.T) {
	read, err := decoded(encoded(testRecord, started, payload(entryBytes)),
		testRecord)
	if err != nil || !bytes.Equal(read.payload, payload(entryBytes)) ||
		!read.written.Equal(started) {
		t.Fatalf("read %v, %v", read, err)
	}
}

// anti-test of TestDecodedIsWhatEncodedWrote
func TestDecodedRefusesEveryMalformedHeader(t *testing.T) {
	whole := string(encoded(testRecord, started, payload(entryBytes)))
	for _, each := range []struct{ name, from, to, says string }{
		{"no record word", "record ", "rekord ", "ends early"},
		{"no break after the count", "record 15\n", "record 15", "record"},
		{"a record count that is not", "record 15", "record x5", "record"},
		{"a negative record count", "record 15", "record -1", "record"},
		{"a record past the end", "record 15", "record 99999", "early"},
		{"no written word", "written ", "writen ", "ends early"},
		{"a time that is not", "written 2026", "written 20x6", "time"},
		{"no payload word", "payload ", "paylode ", "ends early"},
		{"a payload count that is not", "payload 300", "payload 3x0",
			"payload"},
		{"a payload of another length", "payload 300", "payload 301",
			"ends early"},
	} {
		spoilt := strings.Replace(whole, each.from, each.to, 1)
		_, err := decoded([]byte(spoilt), testRecord)
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("%s: %v; want it to say %q", each.name, err, each.says)
		}
	}
}

// anti-test of TestDecodedIsWhatEncodedWrote
func TestDecodedRefusesAHeaderCutShort(t *testing.T) {
	whole := encoded(testRecord, started, nil)
	for cut := len(entryFirst); cut < len(whole); cut++ {
		if _, err := decoded(whole[:cut], testRecord); err == nil {
			t.Fatalf("an entry cut at %d of %d was read", cut, len(whole))
		}
	}
}
