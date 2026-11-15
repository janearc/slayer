package manufacture

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/scene/thing"
)

// Webb's outside surfaces are data: webb-surfaces.yaml names, for each,
// the material it is made of and the finish it was given, and says where
// that was read. A surface's look is made from the two, so the gold of
// the mirror is gold's published optical constants and not a colour.

//go:embed webb-surfaces.yaml
var webbSurfacesRaw []byte

// webbSurface is one outside surface of Webb: which part it is, what it
// is made of, what was done to it, and what was read where.
type webbSurface struct {
	Part, Material, Finish, Says, By string
}

// webbSlot is where a named part of Webb goes among a spacecraft's
// looks, and whether Webb has such a part.
func webbSlot(part string) (slot, bool) {
	switch part {
	case "primary mirror":
		return spaceMirror, true
	case "secondary mirror":
		return spaceSecondary, true
	case "sunshield":
		return spaceFoil, true
	}
	return slot(spaceZero), false
}

// surfaceOf is the look of one surface: its material and its finish, by
// their names, made into a look by thing. An unknown name is refused,
// saying whose surface it was.
func surfaceOf(whose, material, finish string) (*thing.Material, error) {
	kind, err := materials.ByName(material)
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", whose, err)
	}
	surface, err := thing.FinishOf(finish)
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", whose, err)
	}
	look, err := thing.Made(kind, surface)
	if err != nil {
		return nil, fmt.Errorf("manufacture: %s: %w", whose, err)
	}
	return look, nil
}

// webbSurfaces dresses Webb in what it is made of: each surface the file
// names takes the place of the picked look in its slot. A part Webb does
// not have, a material or a finish slayer does not know, and a part
// named twice are each refused by name.
func webbSurfaces(raw []byte, dressed looks) error {
	var read struct{ Surfaces []webbSurface }
	if err := yamlfile.Read("webb-surfaces.yaml", raw, &read); err != nil {
		return err
	}
	seen := map[slot]bool{}
	for _, each := range read.Surfaces {
		where, known := webbSlot(each.Part)
		if !known {
			return fmt.Errorf("manufacture: webb has no surface called %q",
				each.Part)
		}
		if seen[where] {
			return fmt.Errorf("manufacture: webb's %s is named twice",
				each.Part)
		}
		seen[where] = true
		look, err := surfaceOf("webb's "+each.Part, each.Material,
			each.Finish)
		if err != nil {
			return err
		}
		// the look keeps the name its slot had: parts are told apart
		// by their looks' names, and the shield is "thermal shield" there
		look.Name = dressed[where].Name
		dressed[where] = look
	}
	return nil
}
