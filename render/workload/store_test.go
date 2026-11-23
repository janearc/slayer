package workload

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

// identityForTest uses distinct content digests to expose missing key inputs.
func identityForTest() Identity {
	return Identity{strings.Repeat("1", 64), strings.Repeat("2", 64), strings.Repeat("3", 64), strings.Repeat("4", 64), strings.Repeat("5", 64)}
}

func TestReadOnlyStoreMissCreatesNothingAndRejectsWrites(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent-cache")
	store, err := OpenReadOnly(home, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	profile, found, err := store.Lookup(identityForTest())
	if err != nil || found || len(profile.Observations) != 0 {
		t.Fatalf("missing read-only profile = found %v, profile %+v, error %v", found, profile, err)
	}
	if _, err := os.Stat(home); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only miss created cache path: stat error %v", err)
	}
	if err := store.Record(identityForTest(), []Observation{observationForTest(0)}); err == nil {
		t.Fatal("read-only store accepted a write")
	}
}

// observationForTest is a measured interval, rather than a declared estimate.
func observationForTest(frame int) Observation {
	return Observation{Run: "render-1", Sample: fmt.Sprintf("frame-%d", frame), Frame: frame, SceneTime: time.Duration(frame) * time.Second,
		At: time.Unix(1700000000+int64(frame), 0).UTC(), Cache: "retained geometry; disk cache unknown", Wall: time.Second}
}

func TestWorkloadPreservesColdPeaksAndUnknownsAcrossReopen(t *testing.T) {
	home := t.TempDir()
	store, err := Open(home, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	identity := identityForTest()
	if _, found, err := store.Lookup(identity); err != nil || found {
		t.Fatalf("missing database: %v %v", found, err)
	}
	cold, warm := observationForTest(0), observationForTest(1)
	cold.Cache = "cold process; persistent cache unknown"
	physical, peak, rss, zero := uint64(20<<30), uint64(23<<30), uint64(16<<30), uint64(0)
	cold.Physical, cold.PhysicalPeak, cold.GroupRSS = &physical, &peak, &rss
	warm.Physical, warm.Scratch = &zero, &zero
	if err := store.Record(identity, []Observation{cold}); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(identity, []Observation{warm, warm}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(home, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	profile, found, err := reopened.Lookup(identity)
	if err != nil || !found || len(profile.Observations) != 2 {
		t.Fatalf("reload: %+v, %v, %v", profile, found, err)
	}
	peaks := profile.Peaks()
	if *peaks.Physical != physical || *peaks.PhysicalPeak != peak || *peaks.GroupRSS != rss || peaks.Native != nil || peaks.Scratch == nil || *peaks.Scratch != 0 {
		t.Fatalf("distinct peaks or unknowns lost: %+v", peaks)
	}
	// Returned data must remain valid after another transaction/remap, and must
	// not let a caller mutate persisted observations or an aggregate's inputs.
	*peaks.Physical = 7
	*profile.Observations[0].Physical = 9
	if err := store.Record(identity, []Observation{observationForTest(2)}); err != nil {
		t.Fatal(err)
	}
	profile, _, err = store.Lookup(identity)
	if err != nil || *profile.Peaks().Physical != physical {
		t.Fatalf("transaction memory escaped: %v", err)
	}
}

func TestWorkloadInvalidatesEveryDependency(t *testing.T) {
	store, err := Open(t.TempDir(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	identity := identityForTest()
	if err := store.Record(identity, []Observation{observationForTest(0)}); err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		changed := identity
		fields := []*string{&changed.Source, &changed.Build, &changed.Device, &changed.Render, &changed.Probe}
		*fields[i] = strings.Repeat("a", 64)
		if _, found, err := store.Lookup(changed); err != nil || found {
			t.Fatalf("dependency %d reused stale evidence: %v %v", i, found, err)
		}
	}
	for _, value := range []string{"", "partial", strings.Repeat("A", 64), strings.Repeat("g", 64)} {
		bad := identity
		bad.Source = value
		if _, err := bad.Key(); err == nil {
			t.Fatalf("accepted source %q", value)
		}
	}
}

func TestWorkloadIndependentWritersDoNotLoseObservations(t *testing.T) {
	home := t.TempDir()
	identity := identityForTest()
	var pending sync.WaitGroup
	errorsOut := make(chan error, 12)
	for i := range 12 {
		pending.Add(1)
		go func() {
			defer pending.Done()
			store, err := Open(home, 5*time.Second)
			if err == nil {
				err = store.Record(identity, []Observation{observationForTest(i)})
			}
			errorsOut <- err
		}()
	}
	pending.Wait()
	close(errorsOut)
	for err := range errorsOut {
		if err != nil {
			t.Fatal(err)
		}
	}
	store, err := Open(home, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	profile, found, err := store.Lookup(identity)
	if err != nil || !found || len(profile.Observations) != 12 {
		t.Fatalf("lost concurrent samples: %d, %v", len(profile.Observations), err)
	}
}

func TestWorkloadAttemptRetriesAreIdempotentAndConflictsRollBack(t *testing.T) {
	store, err := Open(t.TempDir(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	identity := identityForTest()
	sample := observationForTest(0)
	if err := store.Record(identity, []Observation{sample, sample}); err != nil {
		t.Fatal(err)
	}
	changed := sample
	changed.Wall = 2 * time.Second
	if err := store.Record(identity, []Observation{observationForTest(1), changed}); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed sample was not refused: %v", err)
	}
	profile, _, err := store.Lookup(identity)
	if err != nil || len(profile.Observations) != 1 || profile.Observations[0].Wall != sample.Wall {
		t.Fatalf("conflicting batch changed stored evidence: %+v %v", profile, err)
	}
	changed.Run = "render-2"
	if err := store.Record(identity, []Observation{changed}); err != nil {
		t.Fatal(err)
	}
	profile, _, err = store.Lookup(identity)
	if err != nil || len(profile.Observations) != 2 {
		t.Fatalf("independent attempt lost: %+v %v", profile, err)
	}
}

func TestWorkloadRejectsInvalidEvidenceAtomically(t *testing.T) {
	store, err := Open(t.TempDir(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	identity := identityForTest()
	for _, change := range []func(*Observation){
		func(o *Observation) { o.Run = "" }, func(o *Observation) { o.Frame = -1 },
		func(o *Observation) { o.Sample = "" },
		func(o *Observation) { o.SceneTime = -1 }, func(o *Observation) { o.At = time.Time{} },
		func(o *Observation) { o.Cache = "" }, func(o *Observation) { o.Wall = 0 },
		func(o *Observation) { n := -time.Second; o.GPU = &n },
		func(o *Observation) { o.Stages = map[string]time.Duration{"build": -1} },
		func(o *Observation) { now, peak := uint64(3), uint64(2); o.Physical, o.PhysicalPeak = &now, &peak },
	} {
		bad := observationForTest(1)
		change(&bad)
		if err := store.Record(identity, []Observation{observationForTest(0), bad}); err == nil {
			t.Fatal("accepted invalid evidence")
		}
		if _, found, err := store.Lookup(identity); found || err != nil {
			t.Fatalf("partial batch committed: %v %v", found, err)
		}
	}
	if _, err := Open("relative", time.Second); err == nil {
		t.Fatal("guessed cache home")
	}
	if _, err := Open(t.TempDir(), 0); err == nil {
		t.Fatal("unbounded lock wait")
	}
}

func TestWorkloadCorruptionNeverReturnsPartialEvidence(t *testing.T) {
	for _, poison := range []string{"identity", "digest", "measurement"} {
		t.Run(poison, func(t *testing.T) {
			home := t.TempDir()
			store, err := Open(home, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			identity := identityForTest()
			if err := store.Record(identity, []Observation{observationForTest(0)}); err != nil {
				t.Fatal(err)
			}
			file, err := bolt.Open(filepath.Join(home, fileName), 0600, nil)
			if err != nil {
				t.Fatal(err)
			}
			key, _ := identity.Key()
			err = file.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte(bucketName)).Bucket([]byte(key))
				switch poison {
				case "identity":
					return bucket.Put([]byte(identityName), []byte("{}"))
				case "digest":
					return bucket.Put([]byte("wrong digest"), []byte("{}"))
				default:
					raw := []byte(`{"run":"bad"}`)
					digest := sha256.Sum256(raw)
					kept, err := json.Marshal(envelope{fmt.Sprintf("%x", digest), raw})
					if err != nil {
						return err
					}
					return bucket.Put(digest[:], kept)
				}
			})
			if err = errors.Join(err, file.Close()); err != nil {
				t.Fatal(err)
			}
			profile, found, err := store.Lookup(identity)
			if !errors.Is(err, ErrCorrupt) || found || len(profile.Observations) != 0 {
				t.Fatalf("served corrupt evidence: %+v %v %v", profile, found, err)
			}
		})
	}
}
