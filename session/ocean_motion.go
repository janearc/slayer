package session

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// sceneClockOcean is whether a set explicitly asks the ocean to follow the
// job's absolute scene time.
func sceneClockOcean(read set.Set) bool {
	said, hasWorld := read.World()
	if !hasWorld {
		return false
	}
	surface, hasOcean := said.OceanSurface()
	return hasOcean && surface.Motion() == "scene-clock"
}

// worldAt is the still framer's immutable world snapshot at one absolute
// scene time. Snapshot oceans keep the original world pointer and its bytes.
func (still stillFramer) worldAt(at time.Duration) (*thing.World, error) {
	world := still.world
	if still.given.live != nil && len(still.given.live.deformations) > 0 {
		var err error
		world, _, err = still.given.live.frameAt(at)
		if err != nil {
			return nil, err
		}
	}
	return sceneClockOceanWorldAt(world, at)
}

// sceneClockOceanWorldAt applies the same immutable field snapshot to either
// the still world-only or retained CPU film path.
func sceneClockOceanWorldAt(source *thing.World, at time.Duration) (*thing.World, error) {
	if source == nil || source.OceanSurface == nil ||
		source.OceanSurface.WaveMotion == nil ||
		!source.OceanSurface.WaveMotion.Config().SceneClock {
		return source, nil
	}
	ocean := source.OceanSurface
	motion := ocean.WaveMotion.Config()
	var wake *arid.SurfaceWake
	if authored := motion.SurfaceWake; authored != nil {
		converted := arid.SurfaceWake(*authored)
		wake = &converted
	}
	field, err := arid.NewHeightField(arid.HeightFieldConfig{
		Sea: oceantex.Sea{Wind: motion.WindMetresPerSecond,
			Fetch: motion.FetchMetres, Depth: motion.DepthMetres,
			Toward: motion.TowardRadians},
		Extent: motion.ExtentMetres, Samples: motion.Samples,
		Seed: motion.Seed, Seconds: at.Seconds(),
		SpectralShadingNormals: motion.ShadingNormal ==
			arid.SpectralSlopeNormals,
		SurfaceWake: wake,
	})
	if err == nil {
		field, err = smoothedField(field, motion.ShadingNormal)
	}
	if err != nil {
		return nil, fmt.Errorf("hillside: scene-clock ocean at %s: %w", at, err)
	}
	var boundaryShape shape.Shape = field
	if motion.FlatMean {
		if motion.InfiniteMeanPlane {
			boundaryShape, err = arid.NewInfiniteMeanContinuation(field,
				motion.MeanLevelMetres, motion.CollarWidthMetres)
		} else {
			boundaryShape, err = arid.NewFlatMeanContinuation(field,
				motion.MeanLevelMetres, motion.CollarWidthMetres)
		}
		if err != nil {
			return nil, fmt.Errorf("hillside: scene-clock ocean at %s: %w", at, err)
		}
	}

	if ocean.Boundary == nil {
		return nil, fmt.Errorf("hillside: scene-clock ocean boundary is absent from the world")
	}
	boundaryIndex := -1
	for index, each := range source.Things {
		if each == ocean.Boundary {
			if boundaryIndex >= 0 {
				return nil, fmt.Errorf("hillside: scene-clock ocean boundary occurs more than once")
			}
			boundaryIndex = index
		}
	}
	if boundaryIndex < 0 {
		return nil, fmt.Errorf("hillside: scene-clock ocean boundary is absent from the world")
	}

	// Clone scene containers and their Things so this frame's field stays
	// immutable after another frame is prepared. Materials, lamps, media and
	// all non-ocean shapes remain the exact authored objects.
	snapshot := *source
	snapshot.Things = make([]*thing.Thing, len(source.Things))
	for index, each := range source.Things {
		if each == nil {
			continue
		}
		placed := *each
		snapshot.Things[index] = &placed
	}
	snapshot.Things[boundaryIndex].Shape = boundaryShape
	metadata := *ocean
	metadata.Boundary = snapshot.Things[boundaryIndex]
	snapshot.OceanSurface = &metadata
	return &snapshot, nil
}

// smoothedField gives each frame's field the bilinear vertex-slope shading
// normal its staged set named, as staging does for the first. Any other
// mode, spectral or none, is already in the field and passes unchanged.
func smoothedField(field *shape.HeightField,
	mode string) (*shape.HeightField, error) {
	if mode != arid.SmoothNormalBilinearVertexSlopes {
		return field, nil
	}
	return field.WithApproximateSmoothNormals()
}
