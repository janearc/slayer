package pages

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// realQueue is a queue made by the queue package, holding one job that
// writes into a render folder, its database's path and its bytes.
func realQueue(t *testing.T, root string) (queue.Store, string, string) {
	t.Helper()
	queueRoot := filepath.Join(t.TempDir(), "queue")
	store := queue.At(queueRoot)
	request := jobFor(1, root, pair[0], 2).Request
	request.TimeoutSeconds = 60 // picked: any timeout the queue takes
	err := store.Init(t.Context(), queue.Config{Game: "/usr/bin/true"})
	if err == nil {
		_, err = store.Submit(t.Context(), request)
	}
	if err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(queueRoot, "queue.db")
	return store, database, read(t, database)
}

// TestAPublisherTickNeverWritesTheStore: looks at a real queue's ledger,
// and the ticks that follow, leave its database's bytes as they were.
func TestAPublisherTickNeverWritesTheStore(t *testing.T) {
	root := renders(t)
	stills(t, root, pair[0], 0, 1)
	store, database, before := realQueue(t, root)
	coordinator := New(settingsFor(root, nil, &clock{at: fixtureNoon}),
		io.Discard)
	for range 2 { // picked: a first look and a later one
		coordinator.look(t.Context(), store)
	}
	if read(t, database) != before {
		t.Fatal("a publisher's look wrote the store")
	}
}

// TestAWriteToTheStoreIsSeen is the anti-test: a pause, which does write
// the store, changes its bytes, so the comparison above can see a write.
func TestAWriteToTheStoreIsSeen(t *testing.T) {
	store, database, before := realQueue(t, renders(t))
	if err := store.Pause(t.Context(), "an anti-test's pause"); err != nil {
		t.Fatal(err)
	}
	if read(t, database) == before {
		t.Fatal("a write left the store's bytes as they were")
	}
}
