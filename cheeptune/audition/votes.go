package audition

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// votesFile is the record of votes, inside the library folder.
const votesFile = "votes.json"

// The three things a vote may say. Defined 2026-11-12: thumbs up keeps, thumbs down tosses, and a vote may be taken
// back.
const (
	Keep  = "keep"
	Toss  = "toss"
	Clear = "clear"
)

// Vote is one tune's vote and when it was cast.
type Vote struct {
	Vote string `json:"vote"`
	At   string `json:"at"`
}

// Record is every vote and the order the kept tunes were picked in.
type Record struct {
	Votes map[string]Vote `json:"votes"`
	Queue []string        `json:"queue"`
}

// ledger keeps the record in a file in the folder. A vote moves nothing
// else on disk. A tune is known when its score is in the folder at the
// moment of the vote, so tunes added while the server runs can be voted on.
type ledger struct {
	mu  sync.Mutex
	dir string
	now func() time.Time
}

// newLedger is a ledger for the tunes in a folder.
func newLedger(dir string, now func() time.Time) *ledger {
	return &ledger{dir: dir, now: now}
}

// isTune says the folder holds a score of that name now. The name must be
// a tune's name, so nothing with a separator or a dot path is looked for.
func (l *ledger) isTune(name string) bool {
	if validName(name) != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(l.dir, name+scoreSuffix))
	return err == nil && info.Mode().IsRegular()
}

// inRecord says the record holds a vote or a queue place for the name.
func inRecord(rec Record, name string) bool {
	_, voted := rec.Votes[name]
	return voted || slices.Contains(rec.Queue, name)
}

// read is the record on disk; a folder with no votes yet has an empty one.
func (l *ledger) read() (Record, error) {
	rec := Record{Votes: map[string]Vote{}, Queue: []string{}}
	b, err := os.ReadFile(filepath.Join(l.dir, votesFile))
	if errors.Is(err, fs.ErrNotExist) {
		return rec, nil
	}
	if err != nil {
		return rec, err
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		return rec, fmt.Errorf("%s is not a record of votes: %w", votesFile,
			err)
	}
	if rec.Votes == nil {
		rec.Votes = map[string]Vote{}
	}
	if rec.Queue == nil {
		rec.Queue = []string{}
	}
	return rec, nil
}

// Record is the record as it stands.
func (l *ledger) Record() (Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.read()
}

// cast applies one vote and writes the record. An unknown tune or an
// unknown vote is refused by name and nothing is written. A vote may be
// cleared for any name the record holds, so a tune taken out of the folder
// does not leave its vote stuck.
func (l *ledger) cast(name, vote string) (Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, err := l.read()
	if err != nil {
		return rec, err
	}
	if !l.isTune(name) && !(vote == Clear && inRecord(rec, name)) {
		return Record{}, fmt.Errorf("no tune %q in this folder", name)
	}
	rec.Queue = slices.DeleteFunc(rec.Queue, func(q string) bool {
		return q == name
	})
	switch vote {
	case Keep:
		rec.Queue = append(rec.Queue, name)
		rec.Votes[name] = Vote{Vote: Keep, At: l.stamp()}
	case Toss:
		rec.Votes[name] = Vote{Vote: Toss, At: l.stamp()}
	case Clear:
		delete(rec.Votes, name)
	default:
		return rec, fmt.Errorf("vote %q is not keep, toss or clear", vote)
	}
	return rec, l.write(rec)
}

// stamp is the time a vote is cast, in the form JSON dates are.
func (l *ledger) stamp() string { return l.now().UTC().Format(time.RFC3339) }

// write replaces the record atomically: a temporary file in the same
// folder, then a rename, so a reader never sees half a record.
func (l *ledger) write(rec Record) error {
	b, err := json.MarshalIndent(rec, "", "\t")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(l.dir, ".votes-*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(l.dir, votesFile))
}
