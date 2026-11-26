//go:build cgo

package workspace

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session"
	"github.com/janearc/slayer/session/placement"
)

// StageSnapshot stages a buffer and detaches shared display data, including
// the authored identities used to select supported things in the editor.
func StageSnapshot(ctx context.Context, path, source, kind string) (stagewire.Snapshot, error) {
	wanted, err := stagingJob(ctx, path, source, kind)
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	owner, err := session.Open(ctx, wanted, session.Stamp{})
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = owner.Close()
		}
	}()
	shot, err := owner.WorkingViewShared(0)
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return stagewire.Snapshot{}, err
	}
	seen := make(map[*placement.DisplayMesh]*placement.DisplayMesh)
	mentioned := make(map[string]bool)
	var notices []string
	for i := range shot.World.Batches {
		batch := &shot.World.Batches[i]
		mesh, found := seen[batch.Mesh]
		if !found {
			mesh = batch.Mesh.Detached()
			seen[batch.Mesh] = mesh
			if mesh.Notice != "" && !mentioned[mesh.Notice] {
				notices = append(notices, mesh.Notice)
				mentioned[mesh.Notice] = true
			}
		}
		batch.Mesh = mesh
	}
	if shot.World.Volumes+shot.World.Fields+shot.World.Lamps+shot.World.OmittedSeas > 0 {
		notices = append(notices, fmt.Sprintf("Not shown: %d volumes, %d fields, %d lights, %d seas", shot.World.Volumes, shot.World.Fields, shot.World.Lamps, shot.World.OmittedSeas))
	}
	if shot.RigProxy {
		notices = append(notices, "Physical lens shown as a pinhole: depth of field, motion blur and shutter exposure are not shown")
	}
	notices = append(notices, "Select supported authored things and groups; generated members and proxies are read-only")
	scene := placement.DisplayScene{Eye: shot.Eye, Lens: shot.Lens, Width: shot.Width, Height: shot.Height, Batches: shot.World.Batches, Origins: shot.Origins}
	scene.Focus = sharedFocus(scene.Batches, shot.Eye)
	if err := owner.Close(); err != nil {
		return stagewire.Snapshot{}, err
	}
	closed = true
	return stagewire.Snapshot{Version: 5, Scene: scene, Notice: strings.Join(notices, " · ")}, nil
}

// sharedFocus uses conservative transformed bounding spheres for navigation;
// the staged eye and lens themselves remain unchanged.
func sharedFocus(batches []placement.StagedBatch, eye maths.Subspace) maths.Vec {
	low := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, batch := range batches {
		if batch.HasBounds {
			minimum, maximum := batch.Bounds.Min, batch.Bounds.Max
			low = maths.V(min(low.X, minimum.X), min(low.Y, minimum.Y), min(low.Z, minimum.Z))
			high = maths.V(max(high.X, maximum.X), max(high.Y, maximum.Y), max(high.Z, maximum.Z))
			continue
		}
		minPoint, maxPoint, ok := batch.Mesh.Bounds()
		if !ok {
			continue
		}
		centre := minPoint.Add(maxPoint).Scale(0.5)
		radius := maxPoint.Sub(minPoint).Len() * 0.5
		for _, place := range batch.Places {
			point := place.Out(centre)
			scale := place.Scale
			if scale == 0 {
				scale = 1
			}
			reach := radius * scale
			low = maths.V(min(low.X, point.X-reach), min(low.Y, point.Y-reach), min(low.Z, point.Z-reach))
			high = maths.V(max(high.X, point.X+reach), max(high.Y, point.Y+reach), max(high.Z, point.Z+reach))
		}
	}
	distance := 1.0
	if !math.IsInf(low.X, 0) {
		centre := low.Add(high).Scale(0.5)
		distance = max(1, centre.Sub(eye.Origin).Dot(eye.Forward()))
	}
	return eye.Origin.Add(eye.Forward().Scale(distance))
}

// stagingJob reads the current unsaved source into a set or job request.
func stagingJob(ctx context.Context, path, source, kind string) (job.Job, error) {
	if err := ctx.Err(); err != nil {
		return job.Job{}, err
	}
	var wanted job.Job
	var err error
	switch kind {
	case "set", "legacy-set":
		wanted, err = job.StillFrom(path, []byte(source))
	case "job":
		wanted, err = job.ReadSource(path, []byte(source))
	default:
		return job.Job{}, fmt.Errorf("working view does not yet stage %s documents", kind)
	}
	return wanted, err
}

// stageWorking stages a buffer without saving it or starting a render.
func stageWorking(ctx context.Context, path, source, kind string) (*stagedView, error) {
	wanted, err := stagingJob(ctx, path, source, kind)
	if err != nil {
		return nil, err
	}
	owner, err := session.Open(ctx, wanted, session.Stamp{})
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = owner.Close()
		}
	}()
	shot, err := owner.WorkingView(0)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	low, high, ok := shot.World.Mesh.Bounds()
	distance := 1.0
	if ok {
		centre := low.Add(high).Scale(0.5)
		distance = max(1, centre.Sub(shot.Eye.Origin).Dot(shot.Eye.Forward()))
	}
	scene := &placement.DisplayScene{
		Eye: shot.Eye, Focus: shot.Eye.Origin.Add(shot.Eye.Forward().Scale(distance)),
		Lens: shot.Lens, Width: shot.Width, Height: shot.Height,
		Objects: []placement.DisplayObject{{ID: "scene.world", Name: "staged world", Mesh: shot.World.Mesh, At: maths.Local()}},
	}
	var notices []string
	if shot.RigProxy {
		notices = append(notices, "Physical lens shown as a pinhole: depth of field, motion blur and shutter exposure are not shown")
	}
	if shot.World.Mesh.Notice != "" {
		notices = append(notices, shot.World.Mesh.Notice)
	}
	if shot.World.Volumes+shot.World.Fields+shot.World.Lamps > 0 {
		notices = append(notices, fmt.Sprintf("Not shown: %d volumes, %d fields, %d lights", shot.World.Volumes, shot.World.Fields, shot.World.Lamps))
	}
	notices = append(notices, "Staged geometry is not individually editable yet")
	keep = true
	return &stagedView{scene: scene, notice: strings.Join(notices, " · "), owner: owner}, nil
}
