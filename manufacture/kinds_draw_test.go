//go:build draw

package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// one number in, a whole thing out: every kind, from forty seeds,
// makes a thing slayer's mesh takes, already in its own looks.
func TestEveryKindMakesASoundThingFromAnySeed(t *testing.T) {
	for _, name := range solids {
		for number := range uint64(40) {
			made := mustMake(t, name, number)
			things, err := made.things(maths.Subspace{})
			if err != nil {
				t.Fatalf("%s, seed %d: %v", name, number, err)
			}
			if len(things) == 0 || things[0].Material == nil {
				t.Fatalf("%s, seed %d: %d things", name, number,
					len(things))
			}
		}
	}
}
