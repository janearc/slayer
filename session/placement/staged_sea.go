package placement

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

type stagedSea struct {
	At       maths.Subspace
	Colour   maths.Light
	Author   thing.AuthorRef
	Authored bool
}

// stagedBounds conservatively surrounds the finite staged geometry. Spheres
// around local mesh bounds avoid expanding their triangles or instances.
func stagedBounds(batches []StagedBatch) (maths.Vec, maths.Vec, bool) {
	low := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for index := range batches {
		batch := &batches[index]
		minimum, maximum, ok := batch.Mesh.Bounds()
		if !ok {
			continue
		}
		centre := minimum.Add(maximum).Scale(0.5)
		radius := maximum.Sub(minimum).Len() * 0.5
		batchLow := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
		batchHigh := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
		for _, place := range batch.Places {
			point := place.Out(centre)
			scale := place.Scale
			if scale == 0 {
				scale = 1
			}
			reach := radius * scale
			batchLow = maths.V(min(batchLow.X, point.X-reach), min(batchLow.Y, point.Y-reach), min(batchLow.Z, point.Z-reach))
			batchHigh = maths.V(max(batchHigh.X, point.X+reach), max(batchHigh.Y, point.Y+reach), max(batchHigh.Z, point.Z+reach))
		}
		if !math.IsInf(batchLow.X, 0) && !math.IsInf(batchHigh.X, 0) {
			batch.Bounds, batch.HasBounds = maths.Box{Min: batchLow, Max: batchHigh}, true
			low = maths.V(min(low.X, batchLow.X), min(low.Y, batchLow.Y), min(low.Z, batchLow.Z))
			high = maths.V(max(high.X, batchHigh.X), max(high.Y, batchHigh.Y), max(high.Z, batchHigh.Z))
		}
	}
	return low, high, !math.IsInf(low.X, 0) && !math.IsInf(high.X, 0)
}

// finiteSea places a clearly labeled mean-level plane over the finite scene.
// This is not a wave or shoreline approximation and does not provide picking.
func finiteSea(sea stagedSea, low, high maths.Vec) (StagedBatch, error) {
	minX, maxX, minZ, maxZ := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, x := range []float64{low.X, high.X} {
		for _, y := range []float64{low.Y, high.Y} {
			for _, z := range []float64{low.Z, high.Z} {
				point := sea.At.In(maths.V(x, y, z))
				minX, maxX = min(minX, point.X), max(maxX, point.X)
				minZ, maxZ = min(minZ, point.Z), max(maxZ, point.Z)
			}
		}
	}
	return seaRectangle(sea, minX, maxX, minZ, maxZ, "Sea shown as a finite mean-level proxy; waves and coastline are not shown")
}

// seaAroundEye draws a square under the eye, sized by local camera height
// with a minimum half-width in scene units for water-level cameras.
func seaAroundEye(sea stagedSea, eye maths.Vec) (StagedBatch, error) {
	local := sea.At.In(eye)
	scale := sea.At.Scale
	if scale == 0 {
		scale = 1
	}
	reach := max(25/scale, math.Abs(local.Y)*4)
	return seaRectangle(sea, local.X-reach, local.X+reach, local.Z-reach, local.Z+reach,
		"Sea shown as a finite mean-level proxy around the camera; waves and coastline are not shown")
}

func seaRectangle(sea stagedSea, minX, maxX, minZ, maxZ float64, notice string) (StagedBatch, error) {
	if !displayFinite(minX) || !displayFinite(maxX) || !displayFinite(minZ) || !displayFinite(maxZ) || minX >= maxX || minZ >= maxZ {
		return StagedBatch{}, fmt.Errorf("finite sea proxy has invalid scene bounds")
	}
	mesh, err := flatProxyMesh(minX, maxX, minZ, maxZ, sea.Colour)
	if err != nil {
		return StagedBatch{}, err
	}
	mesh.Notice = notice
	return StagedBatch{Mesh: mesh, Places: []maths.Subspace{sea.At}, Author: sea.Author, Authored: sea.Authored}, nil
}

func flatProxyMesh(minX, maxX, minZ, maxZ float64, colour maths.Light) (*DisplayMesh, error) {
	a, b := maths.V(minX, 0, minZ), maths.V(maxX, 0, minZ)
	c, d := maths.V(minX, 0, maxZ), maths.V(maxX, 0, maxZ)
	builder := newDisplayBuilder()
	builder.triangle([3]maths.Vec{a, c, b}, [3]maths.Vec{}, false, maths.Local(), colour)
	builder.triangle([3]maths.Vec{b, c, d}, [3]maths.Vec{}, false, maths.Local(), colour)
	return builder.finish()
}
