// Package oakland reads the supported static subset of glTF into Slayer meshes.
package oakland

import (
	"errors"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// errUnsupported identifies a static glTF feature this adapter refuses.
var errUnsupported = errors.New("unsupported static glTF")

// Limits bounds every variable-sized resource accepted from an asset.
type Limits struct {
	SourceBytes int64
	BufferBytes int64
	Vertices    int
	Indices     int
	Nodes       int
	Instances   int
	Depth       int
}

// DefaultLimits returns Oakland's documented positive source and geometry
// bounds. Callers may lower them for a narrower import boundary.
func DefaultLimits() Limits {
	return Limits{SourceBytes: 32 << 20, BufferBytes: 256 << 20,
		Vertices: 1_000_000, Indices: 3_000_000, Nodes: 100_000,
		Instances: 500_000, Depth: 256}
}

// Policy selects the scene, source boundary, and optional geometric height
// normalization for one asset read.
type Policy struct {
	Root         string
	Scene        *int
	TargetHeight float64
	HeightUnit   string
	Limits       Limits
}

// Source is one exact byte sequence in an asset's verified source closure.
type Source struct {
	path   string
	size   int64
	sha256 [32]byte
}

// Path is the canonical path relative to the policy root.
func (source Source) Path() string { return source.path }

// Size is the number of source bytes read and hashed.
func (source Source) Size() int64 { return source.size }

// SHA256 is the digest of the same bytes decoded by the adapter.
func (source Source) SHA256() [32]byte { return source.sha256 }

// Instance is one selected-scene placement of immutable triangle geometry.
type Instance struct {
	geometry     *mesh.Mesh
	materialSlot string
	placement    maths.Subspace
}

// HeightNormalization records the measured source bounds and applied scale.
type HeightNormalization struct {
	sourceMinY, sourceMaxY float64
	targetHeight           float64
	heightUnit             string
	factor, verticalOffset float64
}

// SourceBoundsY returns the original selected scene's vertical bounds in metres.
func (record HeightNormalization) SourceBoundsY() (float64, float64) {
	return record.sourceMinY, record.sourceMaxY
}

// TargetHeight returns the requested normalized height in HeightUnit.
func (record HeightNormalization) TargetHeight() float64 { return record.targetHeight }

// HeightUnit returns the unit in which TargetHeight was requested.
func (record HeightNormalization) HeightUnit() string { return record.heightUnit }

// ScaleFactor is the uniform scale applied to the source scene.
func (record HeightNormalization) ScaleFactor() float64 { return record.factor }

// VerticalOffset is the applied y translation after scaling.
func (record HeightNormalization) VerticalOffset() float64 { return record.verticalOffset }

// Geometry is the immutable Slayer mesh for this placement.
func (instance Instance) Geometry() *mesh.Mesh { return instance.geometry }

// MaterialSlot names the explicit native binding required at instantiation.
func (instance Instance) MaterialSlot() string { return instance.materialSlot }

// Placement is the remaining translation, rotation, and uniform scale.
func (instance Instance) Placement() maths.Subspace { return instance.placement }

// Asset is an immutable decoded scene and its verified source closure.
type Asset struct {
	sources    []Source
	options    [32]byte
	slots      []string
	instances  []Instance
	normalized *HeightNormalization
}

// SourceClosure returns a copy of the canonical source records.
func (asset *Asset) SourceClosure() []Source {
	if asset == nil {
		return nil
	}
	return append([]Source(nil), asset.sources...)
}

// OptionsKey identifies both the source closure and all decode options.
func (asset *Asset) OptionsKey() [32]byte {
	if asset == nil {
		return [32]byte{}
	}
	return asset.options
}

// Instances returns a copy of scene placements and immutable mesh references.
func (asset *Asset) Instances() []Instance {
	if asset == nil {
		return nil
	}
	return append([]Instance(nil), asset.instances...)
}

// MaterialSlots returns the sorted material bindings required by the selected
// scene. A material-less primitive uses the documented default slot.
func (asset *Asset) MaterialSlots() []string {
	if asset == nil {
		return nil
	}
	return append([]string(nil), asset.slots...)
}

// HeightNormalization returns the source measurement and applied transform,
// when the policy requested normalization.
func (asset *Asset) HeightNormalization() (HeightNormalization, bool) {
	if asset == nil || asset.normalized == nil {
		return HeightNormalization{}, false
	}
	return *asset.normalized, true
}
