package mesh

// Posed is whether the mesh carries rest points: a surface deformed from
// a shape whose paint is read where each point lay before it moved.
func (made *Mesh) Posed() bool { return len(made.rest) > nought }
