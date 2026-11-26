package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestRealSceneSharedInstances(t *testing.T) {
	path := os.Getenv("FERRUCCIO_STAGE_SCENE")
	if path == "" {
		t.Skip("set FERRUCCIO_STAGE_SCENE to measure a real scene")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wanted, err := job.StillFrom(path, source)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	owner, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	opened := time.Since(started)
	got, err := owner.WorkingInstances(0)
	if err != nil {
		t.Fatal(err)
	}
	meshes := make(map[*placement.DisplayMesh]bool)
	count := 0
	for _, batch := range got.Batches {
		meshes[batch.Mesh] = true
		count += len(batch.Places)
	}
	t.Logf("opened in %s, instances in %s: %d placements, %d shared meshes, %d batches, %d seas omitted", opened, time.Since(started)-opened, count, len(meshes), len(got.Batches), got.OmittedSeas)
}

func workingFixture(t *testing.T) *Session {
	t.Helper()
	dir := t.TempDir()
	setText := `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 2]}
      orientation: {look_at: [1, 0, 0]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      first: {material: matte, middle: [10, 20, 3], size: [2, 4, 6], source: eye}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: set.yaml}
film: {width: 32, height: 24, rays: 1}
renderer: cpu
outputs: {picture: picture.png}
`
	for name, contents := range map[string]string{"set.yaml": setText, "job.yaml": jobText} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	wanted, err := job.Read(filepath.Join(dir, "job.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestWorkingViewStagesUnsavedSetBuffer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scene.yaml")
	saved := `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 2]}
      orientation: {look_at: [1, 0, 0]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      first: {material: matte, middle: [10, 20, 3], size: [2, 4, 6], source: eye}
`
	if err := os.WriteFile(path, []byte(saved), 0600); err != nil {
		t.Fatal(err)
	}
	buffer := strings.Replace(saved, "[2, 4, 6]", "[6, 4, 6]", 1)
	wanted, err := job.StillFrom(path, []byte(buffer))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	view, err := opened.WorkingView(0)
	if err != nil {
		t.Fatal(err)
	}
	low, high, has := view.World.Mesh.Bounds()
	if !has || low.X != 7 || high.X != 13 {
		t.Fatalf("buffered box bounds = %v %v %v", low, high, has)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != saved {
		t.Fatalf("saved set changed: %v", err)
	}
}

func TestWorkingViewUsesStagedFrameAndCamera(t *testing.T) {
	session := workingFixture(t)
	got, err := session.WorkingView(0)
	if err != nil {
		t.Fatal(err)
	}
	census, err := session.Census(0)
	if err != nil {
		t.Fatal(err)
	}
	if census.Eye == nil || got.Eye.Origin != census.Eye.Position || got.Eye.Forward() != census.Eye.Look {
		t.Fatalf("working eye %+v differs from census %+v", got.Eye, census.Eye)
	}
	if got.Width != 32 || got.Height != 24 || got.Lens.Wide != census.Eye.Wide*maths.Degree {
		t.Fatalf("working film %+v, census %+v", got, census.Eye)
	}
	shared, err := session.WorkingViewShared(0)
	if err != nil {
		t.Fatal(err)
	}
	if shared.Eye != got.Eye || shared.Lens != got.Lens || shared.Width != got.Width || shared.Height != got.Height {
		t.Fatal("shared view changed the staged camera or film")
	}
	low, high, has := got.World.Mesh.Bounds()
	if !has || low != maths.V(9, 0, -22) || high != maths.V(11, 6, -18) {
		t.Fatalf("staged box bounds %v %v %v", low, high, has)
	}
	author, has := got.World.Mesh.AuthorAt(maths.NewRay(maths.V(10, 3, 0), maths.V(0, 0, -1)))
	if !has || author.Collection != "scene.world.boxes" || author.Name != "first" {
		t.Fatalf("staged box identity = %+v %v", author, has)
	}
}

func TestWorkingViewRejectsBadIndexAndClosedSession(t *testing.T) {
	session := workingFixture(t)
	if _, err := session.WorkingView(1); err == nil {
		t.Fatal("bad index accepted")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WorkingView(0); err == nil {
		t.Fatal("closed session accepted")
	}
}
