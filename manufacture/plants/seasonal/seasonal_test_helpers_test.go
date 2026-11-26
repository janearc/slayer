package seasonal

import (
	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths/noise"
	"os"
)

// Family-only tests need deterministic variation; root integration witnesses
// independently verify the production seed stream and exact geometry.
type testSeed uint64

func (s testSeed) Draw(a, b, c int) float64 {
	return noise.Stirred(uint64(s) + uint64(a+1)*127 + uint64(b+1)*8191 + uint64(c+1)*65537)
}
func (s testSeed) Swing(a, b, c int) float64 { return 2*s.Draw(a, b, c) - 1 }
func (s testSeed) Sprout(a, b int) branching.Seed {
	return testSeed(uint64(s.Draw(a, b, 3) * (1 << 53)))
}
func (s testSeed) OnPiece(piece int) branching.Seed { return s.Sprout(piece, 0) }
func sow(n uint64) branching.Seed                   { return testSeed(n) }

var testChoices = func() numbers.Numbers {
	raw, err := os.ReadFile("../../plant-choices.yaml")
	if err != nil {
		panic(err)
	}
	book, err := numbers.Read("plant-choices.yaml", raw)
	if err != nil {
		panic(err)
	}
	return book
}()

func plantValue(name string) float64 {
	value, err := testChoices.Get(name)
	if err != nil {
		panic(err)
	}
	return value.Value
}

var testPolicy = &Policy{Value: plantValue, Sow: sow}
var plantProfiles = func() map[string]Profile {
	book, err := Read("profiles.yaml", raw, testPolicy)
	if err != nil {
		panic(err)
	}
	return book.profiles
}()
