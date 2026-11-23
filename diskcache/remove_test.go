package diskcache

import "testing"

// a removed entry is a miss, and the key's neighbour is still kept.
func TestRemoveTakesOneEntryOut(t *testing.T) {
	store, _ := testStore(t, "store-remove")
	gone := NewKey(store.kind).Text("row", "1")
	kept := NewKey(store.kind).Text("row", "2")
	for _, key := range []Key{gone, kept} {
		if err := store.Put(key, payload(entryBytes)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Remove(gone); err != nil {
		t.Fatal(err)
	}
	if hit, err := store.Get(gone); err != nil || hit.Found() {
		t.Errorf("a removed entry was served: %v, %v", hit, err)
	}
	if hit, err := store.Get(kept); err != nil || !hit.Found() {
		t.Errorf("its neighbour went with it: %v, %v", hit, err)
	}
}

// anti-test of TestRemoveTakesOneEntryOut: a key with no entry is no
// error, and another kind's key is refused.
func TestRemoveOfNothingIsNoErrorAndAnotherKindIsRefused(t *testing.T) {
	store, _ := testStore(t, "store-remove-none")
	if err := store.Remove(NewKey(store.kind).Text("row", "9")); err != nil {
		t.Errorf("removing an entry never kept gave %v", err)
	}
	other := NewKey(testKind(t, "store-remove-theirs"))
	if err := store.Remove(other); err == nil {
		t.Error("another kind's key was removed")
	}
}
