package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// Rendered is the mass drawn: its form meshed at the level, where it
// sits in the building, and what its role is made of. A mass is an
// element, so a design draws its masses as it draws all the rest.
func (mass mass) Rendered(level Detail) (Rendered, error) {
	mesh, err := mass.Form.Mesh(level)
	if err != nil {
		return Rendered{}, fmt.Errorf("cities: mass %s: %w", mass.Name, err)
	}
	mesh.Name = mass.Name
	material, err := material(mass.Role)
	if err != nil {
		return Rendered{}, fmt.Errorf("cities: mass %s: %w", mass.Name, err)
	}
	return Rendered{Mesh: mesh, Place: mass.Place, Material: material}, nil
}

// Rendered is everything the design is made of, drawn at a level: each
// mass, then each placed element, where it stands on the building. It
// is the design levelled by a pass whose leveller is the one level.
func (design Design) Rendered(level Detail) ([]Rendered, error) {
	return design.Levelled(Pass{Leveller: Constant(level), Finest: level},
		standingAt(maths.Vec{}))
}

// Levelled is everything the design is made of, each part drawn at the
// level a pass gives it, the design standing at a place in the world the
// pass's leveller sees. The masses are drawn at the pass's finest level.
func (design Design) Levelled(pass Pass, place maths.Subspace) ([]Rendered,
	error) {
	if pass.Leveller == nil {
		return nil, errPass
	}
	var drawn []Rendered
	for _, mass := range design.Massing {
		one, err := mass.Rendered(pass.Finest)
		if err != nil {
			return nil, err
		}
		drawn = append(drawn, one)
	}
	flat, err := pass.levelled(design.Elements, place, pass.Finest)
	if err != nil {
		return nil, fmt.Errorf("cities: %s: %w", design.Name, err)
	}
	for _, each := range flat {
		one, err := each.Element.Rendered(each.Level)
		if err != nil {
			return nil, fmt.Errorf("cities: %s: %w", design.Name, err)
		}
		one.Place = one.Place.Within(each.Place)
		drawn = append(drawn, one)
	}
	return drawn, nil
}
