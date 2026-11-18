package manufacture

import (
	"errors"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"slices"
)

// prepared retains one specimen's unchanged geometry and attached lights.
// Placement makes new scene objects while keeping their mesh prototypes.
type prepared struct {
	parts   []thing.Thing
	lights  []AttachedLight
	storage thing.GeometryStorage
}

// Preparation describes which work can be retained for this specimen.
// Native allocation and scratch sizes still require a renderer measurement;
// geometry payload bytes alone are not a safe scheduling reservation.
type Preparation struct {
	Geometry                thing.GeometryStorage `json:"geometry"`
	RetainLocalGeometry     bool                  `json:"retain_local_geometry"`
	IndependentPlacement    bool                  `json:"independent_placement"`
	NeedsNativeMeasurement  bool                  `json:"needs_native_measurement"`
	SharedMaterials         bool                  `json:"shared_materials"`
	NeedsFrameStateSnapshot bool                  `json:"needs_frame_state_snapshot"`
}

// Prepare constructs meshes once without changing the specimen's detail.
// The same preparation can be placed in many frames without remeshing.
func (made good) Prepare() (prepared, error) {
	parts, err := made.things(maths.Local())
	if err != nil {
		return prepared{}, err
	}
	return prepared{parts: parts, lights: made.Descriptions(), storage: thing.Storage(parts)}, nil
}

// Resources returns owned geometry and pipeline facts without rebuilding it.
// It does not claim native support, peak residency or disk-cache hits.
func (made prepared) Resources() Preparation {
	own := made.storage
	own.Unknown = slices.Clone(own.Unknown)
	valid := len(made.parts) > 0
	return Preparation{Geometry: own, RetainLocalGeometry: valid && own.MutableMeshes == 0 && len(own.Unknown) == 0,
		IndependentPlacement: valid, NeedsNativeMeasurement: true,
		SharedMaterials: valid, NeedsFrameStateSnapshot: valid}
}

// Things places the retained local geometry. Materials remain shared exactly
// as for good.Things; callers must not mutate them during rendering.
func (made prepared) Things(at maths.Subspace) ([]thing.Thing, error) {
	if len(made.parts) == 0 {
		return nil, errGood
	}
	if !shape.ValidPlacement(at) {
		return nil, errors.New("manufacture: invalid placement")
	}
	parts := slices.Clone(made.parts)
	for i := range parts {
		parts[i].Motion = thing.Still(at)
	}
	return parts, nil
}

// Lights places the retained descriptions without retaining construction forms.
func (made prepared) Lights(at maths.Subspace) ([]AttachedLight, error) {
	return PlaceLights(made.lights, at)
}
