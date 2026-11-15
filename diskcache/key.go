package diskcache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Key is what an entry was made from: its kind and its parts, each a
// name and a value, kept in the order said. A key is never changed in
// place; each part said is a new key.
type Key struct {
	kind  Kind
	lines []string
	names map[string]bool
	err   error
}

// NewKey is a key of a kind with no parts yet.
func NewKey(kind Kind) Key {
	return Key{kind: kind, names: map[string]bool{}}
}

// Text is the key with a part whose value is a word or a number, said
// as it is. A value with a line break is refused, since it would be two
// lines of the record.
func (key Key) Text(name, value string) Key {
	if strings.ContainsAny(value, "\n\r") {
		return key.failed(fmt.Errorf("diskcache: part %s holds a line "+
			"break; give it as Bytes", name))
	}
	return key.with(name, value)
}

// Bytes is the key with a part made of bytes, kept in the record as
// their sha256.
func (key Key) Bytes(name string, value []byte) Key {
	sum := sha256.Sum256(value)
	return key.with(name, hex.EncodeToString(sum[:]))
}

// Digest is the key with a part that is a sha256 already worked out, in
// hex, as a pinned source's or a job's. Anything else is refused.
func (key Key) Digest(name, sum string) Key {
	if _, err := hex.DecodeString(sum); err != nil ||
		len(sum) != digestLength {
		return key.failed(fmt.Errorf("diskcache: part %s is %q, not a "+
			"sha256 in hex", name, sum))
	}
	return key.with(name, sum)
}

// with is the key with one more part, refused where the name is said
// already or is not one word.
func (key Key) with(name, value string) Key {
	if name == "" || strings.ContainsAny(name, " \t\n\r") {
		return key.failed(fmt.Errorf("diskcache: a part's name is one "+
			"word, not %q", name))
	}
	if key.names[name] {
		return key.failed(fmt.Errorf("diskcache: part %s is said twice",
			name))
	}
	made := Key{kind: key.kind, err: key.err,
		lines: append(key.lines[:len(key.lines):len(key.lines)],
			name+" "+value),
		names: make(map[string]bool, len(key.names)+1)}
	for each := range key.names {
		made.names[each] = true
	}
	made.names[name] = true
	return made
}

// failed is the key with a refusal kept, the first one said.
func (key Key) failed(err error) Key {
	if key.err == nil {
		key.err = err
	}
	return key
}

// Err is the first refusal any part met, or nil: a key that has one is
// neither read nor written.
func (key Key) Err() error { return key.err }

// Record is the key as lines a person can read: the kind, then each
// part as said. It is what the name is the sha256 of.
func (key Key) Record() string {
	var lines strings.Builder
	lines.WriteString("kind " + key.kind.name + "\n")
	for _, line := range key.lines {
		lines.WriteString(line + "\n")
	}
	return lines.String()
}

// Name is the sha256 of the record, in hex: the entry's file name.
func (key Key) Name() string {
	sum := sha256.Sum256([]byte(key.Record()))
	return hex.EncodeToString(sum[:])
}

// Kind is the kind the key is of.
func (key Key) Kind() Kind { return key.kind }

// Built is the key with the build as its part "build": the sha256 of
// the running program. Where that cannot be known the key is refused,
// since a key without its build would serve one program's work to
// another.
func (key Key) Built() Key {
	program, err := build()
	if err != nil {
		return key.failed(err)
	}
	return key.Digest("build", program)
}

// Build is the sha256 of the running program, in hex, worked out once a
// process: a part every key of something the code computes takes, so a
// changed program misses. It is the program's bytes, not its version
// stamp, since two builds of one stamp can differ.
func Build() (string, error) { return build() }

// build is Build, once.
var build = sync.OnceValues(func() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("diskcache: the running program cannot be "+
			"found (%w), so nothing it computes can be keyed", err)
	}
	return FileDigest(path)
})

// FileDigest is the sha256 of a file's bytes, in hex, read as a stream.
func FileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", errors.Join(fmt.Errorf("diskcache: reading %s", path),
			err)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
