package molluscs

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Spec supplies an authored soft body, its material and a deformation.
// Rest is in the author's coordinates; Scale converts those to scene units.
// Pose maps an author-coordinate point to its posed scene-coordinate point.
// A species supplies its own anatomy: this does not prescribe a shell or limbs.
type Spec struct {
	Rest  *shape.Mesh
	Scale float64
	Pose  func(maths.Vec) maths.Vec
	Skin  *thing.Material
}

// Mollusc is a posed soft body with corresponding author-coordinate anchors.
type Mollusc struct {
	body        thing.Thing
	rest, posed []maths.Vec
}

func molluscFinite(p maths.Vec) bool {
	return !math.IsNaN(p.X) && !math.IsInf(p.X, 0) &&
		!math.IsNaN(p.Y) && !math.IsInf(p.Y, 0) &&
		!math.IsNaN(p.Z) && !math.IsInf(p.Z, 0)
}

// NewMollusc makes a smooth deformable body and retains its rest coordinates
// for attaching details. The input mesh may be released or changed afterwards.
func NewMollusc(spec Spec) (*Mollusc, error) {
	if spec.Rest == nil || spec.Pose == nil || spec.Skin == nil ||
		math.IsNaN(spec.Scale) || math.IsInf(spec.Scale, 0) || spec.Scale <= 0 || len(spec.Rest.Vertices) == 0 {
		return nil, fmt.Errorf("mollusc: invalid body, scale, pose or skin")
	}
	for _, face := range spec.Rest.Faces {
		for _, index := range face {
			if index < 0 || index >= len(spec.Rest.Vertices) {
				return nil, fmt.Errorf("mollusc: face names a missing point")
			}
		}
	}
	source := slices.Clone(spec.Rest.Vertices)
	rest, posed := make([]maths.Vec, len(source)), make([]maths.Vec, len(source))
	for i, p := range source {
		rest[i], posed[i] = p.Scale(spec.Scale), spec.Pose(p)
		if !molluscFinite(rest[i]) || !molluscFinite(posed[i]) {
			return nil, fmt.Errorf("mollusc: nonfinite body point %d", i)
		}
	}
	skin, err := mesh.NewWithRestPoints(posed, rest, spec.Rest.Faces, geometry.SmoothNormals(posed, spec.Rest.Faces), true)
	if err != nil {
		return nil, fmt.Errorf("mollusc: %w", err)
	}
	return &Mollusc{body: thing.Thing{Shape: skin, Material: spec.Skin, Motion: thing.Still(maths.Local())}, rest: source, posed: posed}, nil
}

// Body is the soft surface, in its local space.
func (m *Mollusc) Body() thing.Thing { return m.body }

// Nearest is the posed position of the rest vertex nearest to an authored
// attachment point. It keeps a species' eyes or other details with its skin.
func (m *Mollusc) Nearest(at maths.Vec) maths.Vec {
	best, index := math.Inf(1), 0
	for i, p := range m.rest {
		if d := p.Sub(at).Len(); d < best {
			best, index = d, i
		}
	}
	return m.posed[index]
}
