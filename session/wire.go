package session

import (
	"errors"

	"github.com/janearc/slayer/render/colour"
)

// errWireOnGPU is the refusal of a wire picture anywhere but the cpu.
// The gpu has no wire pass yet, and a wire drawn as light would be a
// lie, so it is refused by name and never drawn some other way.
var errWireOnGPU = errors.New("hillside: film mode wire is drawn on the " +
	"cpu only; the gpu has no wire pass yet. Say renderer: cpu")

// wireRefusal refuses a wire picture asked of the gpu, or of a
// composite pass, which is the gpu's; anything else passes.
func wireRefusal(given asked) error {
	if given.wire && (given.onGPU || given.composite != nil) {
		return errWireOnGPU
	}
	return nil
}

// wireTone develops a wire picture: exposure one, no shoulder, no
// bloom, so an edge's light of one is white and nothing spills from it.
// Defined: trace.Wire's light is made for this tone and no other.
var wireTone = colour.Tone{}
