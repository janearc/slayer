package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// MakeRadioRig separates the existing dish before material flattening.
// Its pier stays fixed; the yoke yaws and the bowl tilts about its seat.
func MakeRadioRig(name string) (*GroundRig, error) {
	row, ok := summitRows()[name].(summitInstrument)
	if !ok || row.Mount != "radio" || len(row.Array) > 0 {
		return nil, fmt.Errorf("no radio rig for %q", name)
	}
	a := &instrumentAssembly{fine: cut{around: int(groundValue("sides-quick"))}, object: good{dressed: row.looks()}}
	row.radioMount(a)
	mount := len(a.object.parts)
	row.radio(a)
	if a.trouble != nil {
		return nil, a.trouble
	}
	r := &GroundRig{Pivot: row.frame().Origin}
	for i, parts := range [][]part{a.object.parts[:1], a.object.parts[1:mount], a.object.parts[mount:]} {
		group := good{parts: parts, dressed: a.object.dressed}
		made, err := group.Things(maths.Local())
		if err != nil {
			return nil, err
		}
		*([]*[]thing.Thing{&r.Fixed, &r.Azimuth, &r.Optical}[i]) = made
	}
	return r, nil
}
