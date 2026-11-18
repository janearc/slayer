package queue

import (
	"errors"
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// TestLedgerKeepsAgentKeysDistinct checks retries against an independent model.
func TestLedgerKeepsAgentKeysDistinct(t *testing.T) {
	rapid.Check(t, func(property *rapid.T) {
		store, template := fixture(t)
		sequences := map[string]uint64{}
		cancelled := map[string]bool{}
		for step := 0; step < 12; step++ {
			request := template
			request.Agent = fmt.Sprint(rapid.IntRange(0, 2).Draw(property, "agent"))
			request.Key = fmt.Sprint(rapid.IntRange(0, 2).Draw(property, "key"))
			identity := request.Agent + "/" + request.Key
			previous := sequences[identity]
			operation := rapid.IntRange(0, 2).Draw(property, "operation")
			if previous != 0 && operation == 1 {
				if err := store.Cancel(t.Context(), previous, request.Agent); err != nil {
					property.Fatal(err)
				}
				cancelled[identity] = true
				continue
			}
			if previous != 0 && operation == 2 {
				request.Args = []string{"different"}
				if _, err := store.Submit(t.Context(), request); !errors.Is(err, ErrConflict) {
					property.Fatal(err)
				}
				continue
			}
			job, err := store.Submit(t.Context(), request)
			if err != nil {
				property.Fatal(err)
			}
			if previous == 0 {
				previous = uint64(len(sequences) + 1)
				sequences[identity] = previous
			}
			if job.Sequence != previous {
				property.Fatalf("identity %s changed sequence", identity)
			}
		}
		snapshot, err := store.Status(t.Context())
		if err != nil || len(snapshot.Jobs) != len(sequences) {
			property.Fatal(snapshot, err)
		}
		for _, job := range snapshot.Jobs {
			identity := job.Request.Agent + "/" + job.Request.Key
			want := Queued
			if cancelled[identity] {
				want = Cancelled
			}
			if job.Sequence != sequences[identity] || job.State != want {
				property.Fatal(job)
			}
		}
	})
}
