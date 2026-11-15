package diskcache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// entry is a file's contents read back: when it was written, and its
// payload.
type entry struct {
	written time.Time
	payload []byte
}

// encoded is a whole entry's bytes: its first line, the record and its
// length, when it was written, the payload's length and sha256, and the
// payload.
func encoded(record string, now time.Time, payload []byte) []byte {
	sum := sha256.Sum256(payload)
	var made bytes.Buffer
	made.WriteString(entryFirst)
	fmt.Fprintf(&made, "%s %d\n%s", recordWord, len(record), record)
	fmt.Fprintf(&made, "%s %s\n", writtenWord,
		now.UTC().Format(writtenLayout))
	fmt.Fprintf(&made, "%s %d %s\n", payloadWord, len(payload),
		hex.EncodeToString(sum[:]))
	made.Write(payload)
	return made.Bytes()
}

// errTorn is an entry that ends before its header says.
var errTorn = errors.New("the entry ends early")

// decoded is an entry read from its bytes and checked: its first line,
// its record against the one wanted, its payload's length and sha256.
func decoded(raw []byte, record string) (entry, error) {
	rest, found := bytes.CutPrefix(raw, []byte(entryFirst))
	if !found {
		return entry{}, errors.New("not an entry of this version")
	}
	length, rest, err := counted(rest, recordWord)
	if err != nil {
		return entry{}, err
	}
	if length > len(rest) {
		return entry{}, errTorn
	}
	if string(rest[:length]) != record {
		return entry{}, errors.New("another key's entry under this name")
	}
	return payloadOf(rest[length:])
}

// payloadOf is the written line, the payload line and the payload,
// checked.
func payloadOf(rest []byte) (entry, error) {
	line, rest, found := bytes.Cut(rest, []byte("\n"))
	stamp, said := bytes.CutPrefix(line, []byte(writtenWord+" "))
	if !found || !said {
		return entry{}, errTorn
	}
	when, err := time.Parse(writtenLayout, string(stamp))
	if err != nil {
		return entry{}, fmt.Errorf("its time does not read: %w", err)
	}
	length, rest, err := counted(rest, payloadWord)
	if err != nil {
		return entry{}, err
	}
	sum, payload, found := bytes.Cut(rest, []byte("\n"))
	if !found || len(payload) != length {
		return entry{}, errTorn
	}
	made := sha256.Sum256(payload)
	if string(sum) != hex.EncodeToString(made[:]) {
		return entry{}, errors.New("its payload's sha256 does not match")
	}
	return entry{written: when, payload: payload}, nil
}

// counted is a header line's word and its count, and what follows the
// count's space or line break.
func counted(rest []byte, word string) (int, []byte, error) {
	rest, found := bytes.CutPrefix(rest, []byte(word+" "))
	if !found {
		return nothing, nil, errTorn
	}
	end := bytes.IndexAny(rest, " \n")
	if end < nothing {
		return nothing, nil, errTorn
	}
	count, err := strconv.Atoi(string(rest[:end]))
	if err != nil || count < nothing {
		return nothing, nil, fmt.Errorf("its %s length does not read",
			word)
	}
	return count, rest[end+1:], nil
}
