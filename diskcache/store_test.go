package diskcache

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPutThenGetIsByteForByte(t *testing.T) {
	store, _ := testStore(t, "store-exact")
	key := NewKey(store.kind).Text("width", "640")
	want := payload(entryBytes)
	if err := store.Put(key, want); err != nil {
		t.Fatal(err)
	}
	hit, err := store.Get(key)
	if err != nil || !hit.Found() || !bytes.Equal(hit.Payload(), want) ||
		!hit.Written().Equal(started) || hit.Why() != "" {
		t.Fatalf("hit %v, %v", hit, err)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetMissesAKeyOneByteAway(t *testing.T) {
	store, _ := testStore(t, "store-near")
	nearKey := NewKey(store.kind).Bytes("set", []byte("abc"))
	if err := store.Put(nearKey,
		payload(entryBytes)); err != nil {
		t.Fatal(err)
	}
	awayKey := NewKey(store.kind).Bytes("set", []byte("abd"))
	hit, err := store.Get(awayKey)
	if err != nil || hit.Found() || hit.Why() != "no entry" ||
		hit.Payload() != nil {
		t.Fatalf("a key one byte away was served: %v, %v", hit, err)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetRefusesAnotherKindsKey(t *testing.T) {
	store, _ := testStore(t, "store-mine")
	other := NewKey(testKind(t, "store-theirs"))
	if _, err := store.Get(other); err == nil {
		t.Fatal("another kind's key was read")
	}
	if err := store.Put(other, nil); err == nil {
		t.Fatal("another kind's key was written")
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetAndPutRefuseAKeyWithARefusal(t *testing.T) {
	store, _ := testStore(t, "store-refused")
	key := NewKey(store.kind).Text("a", "1").Text("a", "1")
	if _, err := store.Get(key); err == nil {
		t.Fatal("a refused key was read")
	}
	if err := store.Put(key, nil); err == nil {
		t.Fatal("a refused key was written")
	}
}

// spoiled puts an entry, changes its file as told, and is the Get after.
func spoiled(t *testing.T, name string, change func([]byte) []byte) Hit {
	t.Helper()
	store, _ := testStore(t, name)
	key := NewKey(store.kind).Text("a", "1")
	if err := store.Put(key, payload(entryBytes)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.path(key))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path(key), change(raw), fileMode); err != nil {
		t.Fatal(err)
	}
	hit, err := store.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	return hit
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetMissesAPayloadChangedOnDisk(t *testing.T) {
	hit := spoiled(t, "store-changed", func(raw []byte) []byte {
		raw[len(raw)-1] ^= 1
		return raw
	})
	if hit.Found() || !strings.Contains(hit.Why(), "sha256") {
		t.Fatalf("a changed payload was served: %v", hit)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetMissesATornEntry(t *testing.T) {
	hit := spoiled(t, "store-torn", func(raw []byte) []byte {
		return raw[:len(raw)-1]
	})
	if hit.Found() || hit.Why() != errTorn.Error() {
		t.Fatalf("a torn entry was served: %v", hit)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetMissesAnotherVersion(t *testing.T) {
	hit := spoiled(t, "store-version", func(raw []byte) []byte {
		return append([]byte("x"), raw...)
	})
	if hit.Found() || !strings.Contains(hit.Why(), "version") {
		t.Fatalf("another version was served: %v", hit)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetMissesAnotherKeysEntryUnderItsName(t *testing.T) {
	store, _ := testStore(t, "store-collided")
	mine, theirs := NewKey(store.kind).Text("a", "1"),
		NewKey(store.kind).Text("a", "2")
	if err := store.Put(theirs, payload(entryBytes)); err != nil {
		t.Fatal(err)
	}
	mineFolder := filepath.Dir(store.path(mine))
	if err := os.MkdirAll(mineFolder, folderMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(store.path(theirs), store.path(mine)); err != nil {
		t.Fatal(err)
	}
	hit, err := store.Get(mine)
	if err != nil || hit.Found() || !strings.Contains(hit.Why(), "another key") {
		t.Fatalf("another key's entry was served: %v, %v", hit, err)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestGetSaysAnUnreadableEntryIsAMiss(t *testing.T) {
	store, _ := testStore(t, "store-unreadable")
	key := NewKey(store.kind).Text("a", "1")
	if err := os.MkdirAll(store.path(key), folderMode); err != nil {
		t.Fatal(err)
	}
	hit, err := store.Get(key)
	if err != nil || hit.Found() || !strings.HasPrefix(hit.Why(), "unreadable") {
		t.Fatalf("a folder was served as an entry: %v, %v", hit, err)
	}
}

func TestPutLeavesNoPartBehind(t *testing.T) {
	store, _ := testStore(t, "store-parts")
	if err := store.Put(NewKey(store.kind).Text("a", "1"), nil); err != nil {
		t.Fatal(err)
	}
	_, parts, err := store.listed()
	if err != nil || len(parts) != nothing {
		t.Fatalf("parts %v, %v", parts, err)
	}
}

// anti-test of TestPutThenGetIsByteForByte
func TestPutSaysWhenItCannotWrite(t *testing.T) {
	store, _ := testStore(t, "store-blocked")
	key := NewKey(store.kind).Text("a", "1")
	fan := filepath.Dir(store.path(key))
	if err := os.WriteFile(fan, nil, fileMode); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(key, nil); err == nil {
		t.Fatal("a put where a file stands in for its folder succeeded")
	}
}

// anti-test of TestPutLeavesNoPartBehind
func TestPutSaysWhenItCannotRename(t *testing.T) {
	store, _ := testStore(t, "store-unrenamed")
	key := NewKey(store.kind).Text("a", "1")
	fullPath := filepath.Join(store.path(key), "full")
	if err := os.MkdirAll(fullPath,
		folderMode); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(key, nil); err == nil {
		t.Fatal("a rename onto a full folder succeeded")
	}
	_, parts, _ := store.listed()
	if len(parts) != nothing {
		t.Fatalf("a failed put left %v", parts)
	}
}

func TestManyWritersOfOneKeyAgree(t *testing.T) {
	store, _ := testStore(t, "store-writers")
	key := NewKey(store.kind).Text("a", "1")
	want := payload(entryBytes)
	var group sync.WaitGroup
	for range entryBytes {
		group.Go(func() {
			if err := store.Put(key, want); err != nil {
				t.Error(err)
			}
			hit, err := store.Get(key)
			if err != nil || (hit.Found() && !bytes.Equal(hit.Payload(), want)) {
				t.Errorf("a reader saw %v, %v", hit, err)
			}
		})
	}
	group.Wait()
	hit, _ := store.Get(key)
	if !hit.Found() || !bytes.Equal(hit.Payload(), want) {
		t.Fatalf("after the writers: %v", hit)
	}
}

// anti-test of TestOpenMakesTheKindsFolderUnderTheHome
func TestOpenRefusesARelativeHome(t *testing.T) {
	if _, err := Open("cache", testKind(t, "store-relative")); err == nil {
		t.Fatal("a relative home was opened")
	}
}

// anti-test of TestOpenMakesTheKindsFolderUnderTheHome
func TestOpenRefusesAnUndeclaredKind(t *testing.T) {
	if _, err := Open(t.TempDir(), Kind{name: "never-declared"}); err == nil {
		t.Fatal("an undeclared kind was opened")
	}
}

func TestOpenMakesTheKindsFolderUnderTheHome(t *testing.T) {
	home := t.TempDir()
	store, err := Open(home, testKind(t, "store-opened"))
	if err != nil || store.Folder() != filepath.Join(home, "store-opened") {
		t.Fatalf("store %v, %v", store, err)
	}
	if info, err := os.Stat(store.Folder()); err != nil || !info.IsDir() {
		t.Fatalf("no folder: %v", err)
	}
}

// anti-test of TestOpenMakesTheKindsFolderUnderTheHome
func TestOpenSaysWhenTheFolderCannotBeMade(t *testing.T) {
	home := t.TempDir()
	kind := testKind(t, "store-unmade")
	unmade := filepath.Join(home, kind.Name())
	if err := os.WriteFile(unmade, nil,
		fileMode); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(home, kind); err == nil {
		t.Fatal("a file where the folder goes was opened")
	}
}

// anti-test of TestManyWritersOfOneKeyAgree
func TestManyWritersOfDifferentPayloadsNeverMix(t *testing.T) {
	store, _ := testStore(t, "store-mixed")
	key := NewKey(store.kind).Text("a", "1")
	ones, twos := payload(entryBytes), bytes.Repeat([]byte{1}, entryBytes)
	var group sync.WaitGroup
	for i := range entryBytes {
		group.Go(func() {
			want := ones
			if i%2 == 1 {
				want = twos
			}
			if err := store.Put(key, want); err != nil {
				t.Error(err)
			}
			hit, err := store.Get(key)
			if err != nil || (!hit.Found() && hit.Why() != "no entry") ||
				(hit.Found() && !bytes.Equal(hit.Payload(), ones) &&
					!bytes.Equal(hit.Payload(), twos)) {
				t.Errorf("a reader saw %v, %v", hit, err)
			}
		})
	}
	group.Wait()
}
