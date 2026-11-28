package placement

import (
	"reflect"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

type surfaceDisplay int

const (
	unclassifiedSurface surfaceDisplay = iota
	finiteSurface
	groupSurface
	meanSeaProxy
	finitePlaneProxy
	omittedSurface
)

// surfaceCoverage records how the staged view treats producer shapes.
// The finite entries describe geometry only. They do not promise the
// renderer's materials or that a particular instance is editable.
var surfaceCoverage = map[reflect.Type]surfaceDisplay{
	reflect.TypeFor[*mesh.Mesh]():                     finiteSurface,
	reflect.TypeFor[*shape.Mesh]():                    finiteSurface,
	reflect.TypeFor[shape.Sphere]():                   finiteSurface,
	reflect.TypeFor[shape.Tube]():                     finiteSurface,
	reflect.TypeFor[*shape.HeightField]():             finiteSurface,
	reflect.TypeFor[*shape.MeshGroup]():               groupSurface,
	reflect.TypeFor[*arid.Sea]():                      meanSeaProxy,
	reflect.TypeFor[shape.Plane]():                    finitePlaneProxy,
	reflect.TypeFor[*shape.Plane]():                   finitePlaneProxy,
	reflect.TypeFor[globe.Globe]():                    omittedSurface,
	reflect.TypeFor[*arid.InfiniteMeanContinuation](): omittedSurface,
	reflect.TypeFor[*arid.FlatMeanContinuation]():     omittedSurface,
}
