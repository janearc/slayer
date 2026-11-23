// Package treebolt is the one bbolt file the GPU's trees are kept in
// between runs, trees.bolt in the cache home, and what keeps it bounded.
//
// The file holds two buckets. "trees" has one entry for each tree: the
// key is a sha256 of the program that grew it and of the faces it was
// grown over, and the value is the tree packed, its own sha256 first,
// which is about 588 MB for moonlake's. "used" has the time each tree
// was last used, in eight bytes, so the least recently used can go.
//
// bbolt never gives pages back to the filesystem: deleting an entry
// leaves the file as large as it was. So the bound is kept by rewriting
// the file: rewrite takes the file's lock, writes the entries it keeps
// into a new file beside it, and renames that over the old one.
//
// The lock is the same one every reader and writer takes, so none is
// inside the file while it is swapped, and one that was waiting for the
// lock finds out the file was swapped and opens the new one; see Open.
package treebolt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	bolt "go.etcd.io/bbolt"
)

// The names the file is kept under, and its permissions.
const (
	// KindName is the name the trees cache goes by in `slayer cache`,
	// for the files form and for this file alike.
	KindName = "trees"
	// File is the file's name in the cache home.
	File = "trees.bolt"
	// Bucket is the bucket the trees are kept in.
	Bucket = "trees"
	// usedBucket is the bucket the time of each tree's last use is in.
	usedBucket = "used"
	// FileMode is the file's permissions: its owner's alone.
	FileMode = 0o600
	// tempSuffix is added to the file's name for the one being written.
	tempSuffix = ".compacting"
)

// The sizes the bound is set by.
const (
	// Cap is the most the file may take on disc after a run: 8 GiB,
	// the same as the trees kind's, since the same trees are kept in one
	// or the other. It holds about thirteen of moonlake's scenes' trees
	// for one program.
	Cap int64 = 8 << 30
	// target is what a rewrite brings the live entries down to: three
	// quarters of Cap, so the next 2 GiB of new trees do not rewrite the
	// file again.
	target = Cap - Cap/4
	// Why is the line the listing gives for the cap.
	Why = "moonlake's trees are about 588 MB; this holds a dozen " +
		"scenes' for one program"
)

// The times the bound is kept by.
const (
	// heldFor is how long after its last use a tree is never removed to
	// make room, as for every other kind: a run that has just read one
	// may read it again for its next frame.
	heldFor = 10 * time.Minute
	// touchGrain is how stale a tree's last use must be before a read
	// writes it again: a use within the hour is not worth a write, and
	// the order of trees days apart does not change by it.
	touchGrain = time.Hour
	// ListWait is how long the listing waits for the file's lock, which
	// a rewrite of a large file holds for a minute or so.
	ListWait = 5 * time.Second
)

// The numbers the copying and the opening are bounded by.
const (
	// stampBytes is the bytes a last-use time takes: nanoseconds as a
	// number of eight bytes.
	stampBytes = 8
	// copyBatch is the most bytes a rewrite puts in one transaction of
	// the new file, so a large file is not held in memory.
	copyBatch = 64 << 20
	// openTries is how many times Open opens again after the file was
	// swapped while it waited. Each swap needs a rewrite, which takes
	// a cap's worth of trees to bring on, so five is never reached by
	// anything but a fault.
	openTries = 5
	// firstPlace is the first index there is.
	firstPlace = 0
)

// Bound is how large the file may be and what a rewrite brings it to,
// both in bytes; a test sets its own, so a few kilobytes are enough.
type Bound struct{ Cap, Target int64 }

// Standard is the bound the file is kept to.
func Standard() Bound { return Bound{Cap: Cap, Target: target} }

// treeEntry is one tree in the file: its key, its bytes with the key's, and
// when it was last used; the zero time where no use was ever recorded,
// as for every tree an older build kept.
type treeEntry struct {
	Key  []byte
	Size int64
	Used time.Time
}

// choice says which of a file's entries a rewrite leaves out: the keys,
// as strings.
type choice func(all []treeEntry) map[string]bool

// Result is what a rewrite did: the file's bytes before and after, the
// entries it kept and the entries it left out.
type Result struct {
	Before, After int64
	Kept, Dropped int
}

// stamp is a time as the file keeps it.
func stamp(at time.Time) []byte {
	return binary.BigEndian.AppendUint64(nil, uint64(at.UnixNano()))
}

// stampOf is a kept time, or the zero time where there is none.
func stampOf(kept []byte) time.Time {
	if len(kept) != stampBytes {
		return time.Time{}
	}
	return time.Unix(0, int64(binary.BigEndian.Uint64(kept)))
}

// sizeOf is a file's bytes on disc, or nothing where it is not there.
func sizeOf(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return firstPlace
	}
	return info.Size()
}

// Open opens the file, waiting wait for its lock, for a look or for a
// write. A rewrite swaps a new file in under the old one's lock, so a
// process that waited for that lock holds the old file, now unlinked,
// and a store to it would be lost; Open looks at the name again once it
// holds the lock and opens afresh when the file is not the same one.
func Open(path string, wait time.Duration, reading bool) (*bolt.DB,
	error) {
	for range openTries {
		before, _ := os.Stat(path)
		file, err := bolt.Open(path, FileMode,
			&bolt.Options{Timeout: wait, ReadOnly: reading})
		if err != nil {
			return nil, err
		}
		after, gone := os.Stat(path)
		if gone == nil && (before == nil || os.SameFile(before, after)) {
			return file, nil
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%s was replaced %d times while it was "+
		"waited for", path, openTries)
}

// Get is a copy of the tree under a key and when it was last used, or
// nil where there is none; a copy, since bbolt's own lasts only as long
// as the transaction.
func Get(tx *bolt.Tx, key []byte) ([]byte, time.Time) {
	bucket := tx.Bucket([]byte(Bucket))
	if bucket == nil {
		return nil, time.Time{}
	}
	value := bytes.Clone(bucket.Get(key))
	return value, usedOf(tx, key)
}

// usedOf is when the tree under a key was last used, or the zero time.
func usedOf(tx *bolt.Tx, key []byte) time.Time {
	if used := tx.Bucket([]byte(usedBucket)); used != nil {
		return stampOf(used.Get(key))
	}
	return time.Time{}
}

// Stale is whether a tree last used at that time wants its use written
// again at now: it was never recorded, or is older than touchGrain.
func Stale(used, now time.Time) bool {
	return used.IsZero() || now.Sub(used) > touchGrain
}

// Put keeps a tree under a key, used at now, in a write transaction,
// making the buckets when they are not there. A zero now records no use,
// which is how a tree with none is carried over.
func Put(tx *bolt.Tx, key, value []byte, now time.Time) error {
	bucket, err := tx.CreateBucketIfNotExists([]byte(Bucket))
	if err != nil {
		return err
	}
	used, err := tx.CreateBucketIfNotExists([]byte(usedBucket))
	if err != nil {
		return err
	}
	if err := bucket.Put(key, value); err != nil {
		return err
	}
	if now.IsZero() {
		return nil
	}
	return used.Put(key, stamp(now))
}

// Touch writes now as the last use of each tree of the keys the file
// still holds. It is one write transaction, so a run's reads cost one
// turn at the file however many trees it took.
func Touch(path string, wait time.Duration, keys [][]byte,
	now time.Time) error {
	file, err := Open(path, wait, false)
	if err != nil {
		return err
	}
	err = file.Update(func(writing *bolt.Tx) error {
		bucket := writing.Bucket([]byte(Bucket))
		if bucket == nil {
			return nil
		}
		return touched(writing, bucket, keys, now)
	})
	return errors.Join(err, file.Close())
}

// touched stamps each key the bucket holds.
func touched(writing *bolt.Tx, bucket *bolt.Bucket, keys [][]byte,
	now time.Time) error {
	used, err := writing.CreateBucketIfNotExists([]byte(usedBucket))
	if err != nil {
		return err
	}
	for _, key := range keys {
		if bucket.Get(key) == nil {
			continue
		}
		if err := used.Put(key, stamp(now)); err != nil {
			return err
		}
	}
	return nil
}

// Absorb puts every tree of one transaction into a write transaction,
// keeping the later of the two last uses where both have one.
func Absorb(writing, reading *bolt.Tx) error {
	from := reading.Bucket([]byte(Bucket))
	if from == nil {
		return nil
	}
	return from.ForEach(func(key, value []byte) error {
		now := usedOf(reading, key)
		if kept := usedOf(writing, key); kept.After(now) {
			now = kept
		}
		return Put(writing, bytes.Clone(key), bytes.Clone(value), now)
	})
}

// entriesOf is every tree a read transaction holds, with its bytes and
// its last use.
func entriesOf(reading *bolt.Tx) []treeEntry {
	bucket := reading.Bucket([]byte(Bucket))
	var all []treeEntry
	if bucket == nil {
		return all
	}
	_ = bucket.ForEach(func(key, value []byte) error {
		all = append(all, treeEntry{Key: bytes.Clone(key),
			Size: int64(len(key) + len(value)),
			Used: usedOf(reading, key)})
		return nil
	})
	return all
}

// Usage is what the file holds: its bytes on disc, its trees, the bytes
// of those, how many have no recorded use, and the least and most
// recent use of the rest.
type Usage struct {
	Path           string
	Disc, Live     int64
	Entries        int
	Unrecorded     int
	Oldest, Newest time.Time
}

// Look says what the file holds. A file not made yet holds nothing. It
// waits wait for the file's lock, and says so by name when it is held.
func Look(path string, wait time.Duration) (Usage, error) {
	made := Usage{Path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return made, nil
	}
	file, err := Open(path, wait, true)
	if err != nil {
		return made, fmt.Errorf("%s: %w", path, err)
	}
	err = file.View(func(reading *bolt.Tx) error {
		made.add(entriesOf(reading))
		return nil
	})
	made.Disc = sizeOf(path)
	return made, errors.Join(err, file.Close())
}

// add counts entries into the usage.
func (made *Usage) add(all []treeEntry) {
	made.Entries = len(all)
	for _, each := range all {
		made.Live += each.Size
		if each.Used.IsZero() {
			made.Unrecorded++
			continue
		}
		if made.Oldest.IsZero() || each.Used.Before(made.Oldest) {
			made.Oldest = each.Used
		}
		if each.Used.After(made.Newest) {
			made.Newest = each.Used
		}
	}
}

// evict is the choice that brings the live bytes to the bound's target
// when the file is over its cap: the least recently used go first, the
// ones with no recorded use before any other, and never a tree used
// within heldFor of now. When the file is not over, none goes.
func evict(bound Bound, now time.Time, disc int64) choice {
	return func(all []treeEntry) map[string]bool {
		drop := map[string]bool{}
		if disc <= bound.Cap {
			return drop
		}
		var live int64
		for _, each := range all {
			live += each.Size
		}
		order := slices.Clone(all)
		slices.SortStableFunc(order, func(one, other treeEntry) int {
			return one.Used.Compare(other.Used)
		})
		for _, each := range order {
			if live <= bound.Target || now.Sub(each.Used) < heldFor {
				break
			}
			drop[string(each.Key)] = true
			live -= each.Size
		}
		return drop
	}
}

// everything is the choice that leaves nothing in.
func everything(all []treeEntry) map[string]bool {
	drop := make(map[string]bool, len(all))
	for _, each := range all {
		drop[string(each.Key)] = true
	}
	return drop
}

// Trim keeps the file to its bound: when it is over the cap on disc,
// the least recently used trees go and the file is written fresh, which
// is what gives the disc back. It does nothing, and does not take the
// lock, when the file is not over.
func Trim(path string, wait time.Duration, bound Bound,
	now time.Time) (Result, error) {
	disc := sizeOf(path)
	if disc <= bound.Cap {
		return Result{Before: disc, After: disc}, nil
	}
	return rewrite(path, wait, evict(bound, now, disc))
}

// Clear removes every tree and gives the disc back, and says how many
// there were. A file not made yet has none.
func Clear(path string, wait time.Duration) (Result, error) {
	return rewrite(path, wait, everything)
}

// rewrite writes a new file of the trees choose leaves in and renames it
// over the old one, all while holding the old one's lock, so no process
// is inside the file while it is swapped. A file not made yet is left
// not made. A failed rewrite leaves the old file as it was.
func rewrite(path string, wait time.Duration, choose choice) (Result,
	error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return Result{}, nil
	}
	source, err := Open(path, wait, false)
	if err != nil {
		return Result{}, err
	}
	result, err := rewriteFrom(source, choose)
	return result, errors.Join(err, source.Close())
}

// rewriteFrom is rewrite of a file that is open and locked.
func rewriteFrom(source *bolt.DB, choose choice) (Result, error) {
	path := source.Path()
	temp := path + tempSuffix
	_ = os.Remove(temp)
	reading, err := source.Begin(false)
	if err != nil {
		return Result{}, err
	}
	defer reading.Rollback()
	all := entriesOf(reading)
	drop := choose(all)
	result := Result{Before: sizeOf(path), Dropped: len(drop),
		Kept: len(all) - len(drop)}
	if err := copiedKept(reading, temp, drop); err != nil {
		return result, errors.Join(err, os.Remove(temp))
	}
	if err := os.Rename(temp, path); err != nil {
		return result, errors.Join(err, os.Remove(temp))
	}
	result.After = sizeOf(path)
	return result, nil
}

// copiedKept writes the trees not dropped into a new file at temp.
func copiedKept(reading *bolt.Tx, temp string, drop map[string]bool) error {
	made, err := bolt.Open(temp, FileMode, &bolt.Options{})
	if err != nil {
		return err
	}
	grouped := &batch{made: made}
	if bucket := reading.Bucket([]byte(Bucket)); bucket != nil {
		err = bucket.ForEach(func(key, value []byte) error {
			if drop[string(key)] {
				return nil
			}
			return grouped.put(key, value, usedOf(reading, key))
		})
	}
	return errors.Join(err, grouped.finish(), made.Close())
}

// batch is the new file's writes, grouped into transactions of at most
// copyBatch bytes. The bytes it is given stay valid because the old
// file's read transaction is open the whole time.
type batch struct {
	made    *bolt.DB
	writing *bolt.Tx
	pending int64
}

// put adds a tree to the batch, committing it when it is large enough.
func (grouped *batch) put(key, value []byte, used time.Time) error {
	if grouped.writing == nil {
		writing, err := grouped.made.Begin(true)
		if err != nil {
			return err
		}
		grouped.writing = writing
	}
	if err := Put(grouped.writing, key, value, used); err != nil {
		return err
	}
	grouped.pending += int64(len(key) + len(value))
	if grouped.pending < copyBatch {
		return nil
	}
	return grouped.commit()
}

// commit ends the batch's transaction, if it has one.
func (grouped *batch) commit() error {
	writing := grouped.writing
	grouped.writing, grouped.pending = nil, firstPlace
	if writing == nil {
		return nil
	}
	return writing.Commit()
}

// finish commits what is left, and makes the buckets when nothing was
// put, so the new file is a tree file all the same.
func (grouped *batch) finish() error {
	if grouped.writing == nil {
		writing, err := grouped.made.Begin(true)
		if err != nil {
			return err
		}
		grouped.writing = writing
	}
	if _, err := grouped.writing.CreateBucketIfNotExists(
		[]byte(Bucket)); err != nil {
		return errors.Join(err, grouped.writing.Rollback())
	}
	return grouped.commit()
}
