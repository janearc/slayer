package thing

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// Diffuse describes a plain reflecting material. As with Material literals,
// the caller supplies linear reflectance and owns any later mutation.
func Diffuse(name string, colour maths.Light) *Material {
	return &Material{Name: name, Colour: colour}
}

// Polished adds the existing Fresnel controls to a plain material. Colour,
// gloss and index are supplied by the caller, as with Material literals.
func Polished(name string, colour maths.Light,
	gloss, index float64,
) *Material {
	return &Material{Name: name, Colour: colour,
		Gloss: gloss, Index: index}
}

// NewVolume describes a stationary medium in world coordinates. A caller can
// set Motion before adding it to a World when local placement is needed.
func NewVolume(field medium.Medium, material *Material) Volume {
	return Volume{Medium: field, Material: material}
}
