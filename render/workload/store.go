package workload

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
)

// ErrCorrupt means a stored profile failed its identity or payload checks.
// The caller must treat it as unavailable evidence and retain the diagnostic.
var ErrCorrupt = errors.New("workload: corrupt profile")

// ErrConflict refuses changed evidence for an already recorded attempt/sample.
var ErrConflict = errors.New("workload: conflicting attempt sample")

// envelope checks the payload independently of its stable sample identity.
type envelope struct {
	SHA256  string          `json:"sha256"`
	Payload json.RawMessage `json:"payload"`
}

// storeNames are versioned separately from the renderer's geometry tree cache.
const (
	fileName     = "workload-profiles.bolt"
	bucketName   = "scene-profiles-v1"
	identityName = "identity"
)

// Store opens a database only during a lookup or write, allowing independent
// processes to cooperate without holding its exclusive lock through GPU work.
// Its immutable path and wait make it safe for concurrent calls.
type Store struct {
	path     string
	wait     time.Duration
	readOnly bool
}

// Open selects the explicit absolute cache home, refusing guessed locations.
// Wait bounds bbolt file-lock acquisition; no resource cap is imposed here.
func Open(home string, wait time.Duration) (*Store, error) {
	if !filepath.IsAbs(home) || wait <= 0 {
		return nil, errors.New("workload: absolute cache home and positive lock wait required")
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(home, fileName), wait: wait}, nil
}

// OpenReadOnly creates no directories and refuses writes through the returned
// store. A missing database is a normal Lookup miss.
func OpenReadOnly(home string, wait time.Duration) (*Store, error) {
	if !filepath.IsAbs(home) || wait <= 0 {
		return nil, errors.New("workload: absolute cache home and positive lock wait required")
	}
	return &Store{path: filepath.Join(home, fileName), wait: wait, readOnly: true}, nil
}

// Record atomically adds measured samples, preserving previous observations.
// Repeating identical evidence is idempotent; changing the same attempt/sample
// returns ErrConflict and rolls back the batch. The caller owns queue lease
// fencing: this store grants no authority to publish an obsolete attempt.
// Validation and serialization occur before acquiring the writer; no rendering
// happens inside a transaction.
func (store *Store) Record(identity Identity, samples []Observation) error {
	if store.readOnly {
		return errors.New("workload: read-only store cannot record observations")
	}
	key, err := identity.Key()
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		return errors.New("workload: no measured samples to record")
	}
	encoded := make([][]byte, len(samples))
	slots := make([][sha256.Size]byte, len(samples))
	for i, sample := range samples {
		if err := sample.validate(); err != nil {
			return err
		}
		sample.At = sample.At.UTC()
		raw, err := json.Marshal(sample)
		if err != nil {
			return err
		}
		encoded[i], err = json.Marshal(envelope{fmt.Sprintf("%x", sha256.Sum256(raw)), raw})
		if err != nil {
			return err
		}
		slots[i] = sample.slot()
	}
	metadata, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	file, err := bolt.Open(store.path, 0600, &bolt.Options{Timeout: store.wait})
	if err != nil {
		return err
	}
	err = file.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		bucket := root.Bucket([]byte(key))
		if bucket == nil {
			bucket, err = root.CreateBucket([]byte(key))
			if err != nil {
				return err
			}
		} else if !bytes.Equal(bucket.Get([]byte(identityName)), metadata) {
			return ErrCorrupt
		}
		if err := bucket.Put([]byte(identityName), metadata); err != nil {
			return err
		}
		for i, raw := range encoded {
			key := slots[i][:]
			if old := bucket.Get(key); old != nil && !bytes.Equal(old, raw) {
				return ErrConflict
			}
			if err := bucket.Put(key, raw); err != nil {
				return err
			}
		}
		return nil
	})
	return errors.Join(err, file.Close())
}

// Lookup copies database values before closing the read transaction, then
// checks payload digests and decodes them. Unknown identities return a miss;
// malformed records return an error, never a partial scheduling profile.
func (store *Store) Lookup(identity Identity) (Profile, bool, error) {
	key, err := identity.Key()
	if err != nil {
		return Profile{}, false, err
	}
	file, err := bolt.Open(store.path, 0600, &bolt.Options{ReadOnly: true, Timeout: store.wait})
	if errors.Is(err, os.ErrNotExist) {
		return Profile{}, false, nil
	}
	if err != nil {
		return Profile{}, false, err
	}
	var metadata []byte
	var payloads []struct{ key, value []byte }
	err = file.View(func(tx *bolt.Tx) error {
		root := tx.Bucket([]byte(bucketName))
		if root == nil {
			return nil
		}
		bucket := root.Bucket([]byte(key))
		if bucket == nil {
			if root.Get([]byte(key)) != nil {
				return ErrCorrupt
			}
			return nil
		}
		metadata = bytes.Clone(bucket.Get([]byte(identityName)))
		if metadata == nil {
			return ErrCorrupt
		}
		return bucket.ForEach(func(name, raw []byte) error {
			if string(name) == identityName {
				return nil
			}
			if raw == nil || len(name) != sha256.Size {
				return ErrCorrupt
			}
			payloads = append(payloads, struct{ key, value []byte }{bytes.Clone(name), bytes.Clone(raw)})
			return nil
		})
	})
	if err = errors.Join(err, file.Close()); err != nil {
		return Profile{}, false, err
	}
	if metadata == nil {
		return Profile{}, false, nil
	}
	var profile Profile
	if err := json.Unmarshal(metadata, &profile.Identity); err != nil || profile.Identity != identity || len(payloads) == 0 {
		return Profile{}, false, ErrCorrupt
	}
	for _, record := range payloads {
		var kept envelope
		if err := json.Unmarshal(record.value, &kept); err != nil || fmt.Sprintf("%x", sha256.Sum256(kept.Payload)) != kept.SHA256 {
			return Profile{}, false, ErrCorrupt
		}
		var sample Observation
		if err := json.Unmarshal(kept.Payload, &sample); err != nil {
			return Profile{}, false, fmt.Errorf("%w: %v", ErrCorrupt, err)
		}
		if err := sample.validate(); err != nil {
			return Profile{}, false, fmt.Errorf("%w: %v", ErrCorrupt, err)
		}
		slot := sample.slot()
		if !bytes.Equal(slot[:], record.key) {
			return Profile{}, false, ErrCorrupt
		}
		profile.Observations = append(profile.Observations, sample)
	}
	sort.Slice(profile.Observations, func(i, j int) bool {
		a, b := profile.Observations[i], profile.Observations[j]
		if !a.At.Equal(b.At) {
			return a.At.Before(b.At)
		}
		if a.Run != b.Run {
			return a.Run < b.Run
		}
		return a.Sample < b.Sample
	})
	return profile, true, nil
}
