package stage

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// imports binds checked geometry to native materials and puts it in the world.
func imports(world *thing.World, said set.World, worn map[string]*thing.Material, shift maths.Vec) error {
	for _, part := range said.Imports() {
		bindings := map[string]*thing.Material{}
		for slot, name := range part.Bindings() {
			bindings[slot] = worn[name]
		}
		pieces, err := part.Asset().Instantiate(bindings)
		if err != nil {
			return fmt.Errorf("import %s: %w", part.Name(), err)
		}
		placement := part.Placement(said.Unit())
		placement.Origin = moved(placement.Origin, shift)
		for _, piece := range pieces {
			piece.Motion = thing.Still(piece.At(0).Within(placement))
			stored := world.Add(piece)
			world.RecordAuthor(stored, thing.AuthorRef{Collection: "scene.world.imports", Name: part.Name()})
		}
	}
	return nil
}
