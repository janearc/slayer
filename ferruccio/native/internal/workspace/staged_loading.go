//go:build cgo

package workspace

import (
	"context"
	"fmt"
	"math"
	"sort"
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
	return StageSnapshotWithCamera(ctx, path, source, kind, stagewire.Camera{})
}

// StageSnapshotWithCamera selects an authored camera for this view only.
func StageSnapshotWithCamera(ctx context.Context, path, source, kind string, camera stagewire.Camera) (stagewire.Snapshot, error) {
	return StageDetailSnapshot(ctx, path, source, kind, camera, "")
}

// StageDetailSnapshot optionally stages one authored piece before detaching its display data.
func StageDetailSnapshot(ctx context.Context, path, source, kind string, camera stagewire.Camera, detail string) (stagewire.Snapshot, error) {
	wanted, err := stagingJob(ctx, path, source, kind)
	if err == nil && camera.Frame != 0 && (camera.Frame < 1 || wanted.Clock().Still() || detail != "") {
		return stagewire.Snapshot{}, fmt.Errorf("working frame selection requires a film job and no scene detail")
	}
	if err == nil && camera.Name != "" {
		wanted, err = wanted.With(job.CameraName(camera.Name))
	}
	if err == nil && camera.Wide != 0 {
		wanted, err = wanted.With(job.CameraWide(camera.Wide))
	}
	if err == nil && camera.PathRecipe != "" {
		wanted, err = wanted.With(job.StillCamera(camera.PathRecipe))
	}
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	owner, err := openSnapshotSession(ctx, wanted, detail)
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = owner.Close()
		}
	}()
	times := owner.Times()
	frame := max(1, camera.Frame)
	if frame > len(times) {
		return stagewire.Snapshot{}, fmt.Errorf("film frame %d is outside the producer's %d frames", frame, len(times))
	}
	shot, err := owner.WorkingViewShared(frame - 1)
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return stagewire.Snapshot{}, err
	}
	snapshot, err := workingSnapshot(shot, camera, detail)
	if err != nil {
		return stagewire.Snapshot{}, err
	}
	if err := owner.Close(); err != nil {
		return stagewire.Snapshot{}, err
	}
	closed = true
	if !wanted.Clock().Still() {
		snapshot.Frame = &stagewire.FilmFrame{Number: frame, Count: len(times), Time: times[frame-1]}
		snapshot.Notice = fmt.Sprintf(
			"Working view shows film frame %d of %d at scene time %s · %s",
			frame, len(times), times[frame-1], snapshot.Notice)
	}
	return snapshot, nil
}

func workingSnapshot(shot session.SharedWorkingView, camera stagewire.Camera, detail string) (stagewire.Snapshot, error) {
	seen := make(map[*placement.DisplayMesh]*placement.DisplayMesh)
	mentioned := make(map[string]bool)
	notices := shot.World.MaterialLimits.Notices()
	if detail != "" {
		name := detail[strings.IndexByte(detail, '/')+1:]
		notices = append(notices, fmt.Sprintf("Scene detail %q · selected geometry, terrain and placement dependencies · other scene content omitted · full renders use the complete scene", name))
	}
	if detail == "" && camera.Name != "" {
		notices = append(notices, fmt.Sprintf("Job camera %q is used for the working view and as the default render camera.", camera.Name))
	}
	if detail == "" && camera.PathRecipe != "" {
		notices = append(notices, "The job camera path sets the working view and default render framing.")
	}
	if camera.Wide != 0 {
		notices = append(notices, fmt.Sprintf("The job's still uses a horizontal angle of %g°.", camera.Wide))
	}
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
	omittedLights := shot.World.Lamps - len(shot.Lights)
	omittedVolumes := shot.World.Volumes - len(shot.VolumeBounds)
	if shot.World.GiantAtmospheres > 0 {
		notices = append(notices, omittedGiantNotice(shot.World.GiantAtmospheres))
	}
	if len(shot.GiantBoundaries) > 0 {
		notices = append(notices, giantBoundaryNotice)
	}
	if len(shot.World.GlobeOutlines) > 0 {
		notices = append(notices, globeOutlineNotice)
	}
	if omitted := shot.World.GlobeOutlinesOmitted; omitted > 0 {
		notices = append(notices, fmt.Sprintf("Additional globe outlines omitted due to the working-view limit (%d)", omitted))
	}
	if omittedVolumes+shot.World.Fields+omittedLights+shot.World.OmittedSeas > 0 {
		notices = append(notices, fmt.Sprintf("Not shown: %d volume bounds, %d fields, %d light markers, %d seas", omittedVolumes, shot.World.Fields, omittedLights, shot.World.OmittedSeas))
	}
	if len(shot.VolumeBounds) > 0 {
		notices = append(notices, volumeBoundsNotice)
	}
	if len(shot.Lights) > 0 {
		notices = append(notices, lightMarkerNotice)
	}
	if shot.World.OmittedPlanes > 0 {
		notices = append(notices, fmt.Sprintf("Infinite planes omitted (%d) due to material or coordinate limits in the working view", shot.World.OmittedPlanes))
	}
	kinds := make([]string, 0, len(shot.World.UnsupportedSurfaces))
	for kind := range shot.World.UnsupportedSurfaces {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		notices = append(notices, fmt.Sprintf("Unsupported surfaces omitted: %s (%d)", kind, shot.World.UnsupportedSurfaces[kind]))
	}
	if shot.SkyGradient != nil {
		notices = append(notices, workingSkyNotice)
	} else if shot.World.Sky {
		notices = append(notices, omittedWorkingSkyNotice)
	}
	if shot.World.Haze || shot.World.Glow {
		notices = append(notices, "Atmospheric haze and glow are not shown")
	}
	if shot.RigProxy {
		notices = append(notices, "Physical lens shown as a pinhole: depth of field, motion blur and shutter exposure are not shown")
	}
	if len(shot.World.Batches) == 0 {
		notices = append([]string{"No drawable surfaces in the working view"}, notices...)
	} else {
		notices = append(notices, "Select supported authored things and groups; generated members and proxies are read-only")
	}
	scene := placement.DisplayScene{
		Eye: shot.Eye, Lens: shot.Lens, Width: shot.Width, Height: shot.Height,
		Batches: shot.World.Batches, Origins: shot.Origins, Lights: shot.Lights,
		VolumeBounds: shot.VolumeBounds, OmittedVolumeBounds: omittedVolumes,
		GiantBoundaries: shot.GiantBoundaries,
		GlobeOutlines:   shot.World.GlobeOutlines,
		SkyGradient:     shot.SkyGradient,
	}
	scene.Focus = sharedFocus(scene.Batches[:len(scene.Batches)-shot.World.PlaneProxies], shot.Eye)
	if detail != "" {
		if err := frameSceneDetail(&scene, detail); err != nil {
			return stagewire.Snapshot{}, err
		}
	}
	if clipped := scene.FarClippedBatches(); clipped > 0 {
		notices = append(notices, farClipNotice(clipped, "Geometry groups"))
	}
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
		if err == nil {
			err = workingJobForm(wanted.Scene().Form())
		}
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

func openSnapshotSession(ctx context.Context, wanted job.Job, detail string) (*session.Session, error) {
	if detail == "" {
		return session.Open(ctx, wanted, session.Stamp{})
	}
	collection, name, ok := strings.Cut(detail, "/")
	if !ok || name == "" || (collection != "scene.set.things" && collection != "scene.set.stands") {
		return nil, fmt.Errorf("invalid authored scene detail %q", detail)
	}
	return session.OpenWorkingDetail(ctx, wanted, session.WorkingDetail{Collection: collection, Name: name})
}

// frameSceneDetail gives the selected geometry a local working camera.
func frameSceneDetail(scene *placement.DisplayScene, id string) error {
	low := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, batch := range scene.Batches {
		if !batch.Authored || batch.Author.Collection+"/"+batch.Author.Name != id || !batch.HasBounds {
			continue
		}
		minimum, maximum := batch.Bounds.Min, batch.Bounds.Max
		low = maths.V(min(low.X, minimum.X), min(low.Y, minimum.Y), min(low.Z, minimum.Z))
		high = maths.V(max(high.X, maximum.X), max(high.Y, maximum.Y), max(high.Z, maximum.Z))
	}
	if math.IsInf(low.X, 0) {
		return fmt.Errorf("scene detail %q has no inspectable geometry", id)
	}
	centre := low.Add(high).Scale(0.5)
	distance := max(2, high.Sub(low).Len()) / math.Tan(scene.Lens.Wide*0.5)
	distance *= max(1, float64(scene.Width)/float64(scene.Height))
	eye := centre.Add(maths.V(0.7, 0.4, 1).Unit().Scale(distance))
	scene.Eye = maths.Subspace{Origin: eye, Orientation: maths.LookAt(eye, centre)}
	scene.Focus = centre
	return nil
}

// workingJobForm distinguishes traced scenes from jobs that only develop
// already-rendered frames. Some traced scenes still lack an inspectable eye.
func workingJobForm(form string) error {
	switch form {
	case "set", "named", "smoke":
		return nil
	case "frames":
		return fmt.Errorf("working view does not show jobs made from already-rendered frames")
	default:
		return fmt.Errorf("working view does not stage jobs with scene form %q", form)
	}
}
