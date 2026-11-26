package job_test

import (
	"math"
	"path/filepath"
	"slices"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/sauropods"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Resolve the actual jobs and profiles, and check full-view camera frusta
// against each built part's bounds without drawing or submitting anything.
func TestSauropodReviewJobsResolveProfilesAndFrameBothViews(t *testing.T) {
	file, raw := sauropods.Embedded()
	book, err := sauropods.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(book.Species()) != 4 {
		t.Fatalf("need four sauropods, got %v", book.Species())
	}
	for _, species := range book.Species() {
		t.Run(species, func(t *testing.T) {
			profile, err := book.Profile(species)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("..", "jobs", "bioforms", "sauropod-review", species+".yaml")
			wanted, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			width, widthSet := wanted.Film().Width()
			height, heightSet := wanted.Film().Height()
			if !widthSet || !heightSet || width != 640 || height != 360 {
				t.Fatalf("wrong film: %d x %d", width, height)
			}
			scene, ok := wanted.ImportedSet()
			if !ok {
				t.Fatal("job did not import its set")
			}
			recipe, ok := scene.Recipe("specimen")
			if !ok || !recipe.IsAnimal() || recipe.AnimalSpecies() != species {
				t.Fatalf("job did not select %s", species)
			}
			kind, err := recipe.Construction()
			if err != nil || kind == nil {
				t.Fatalf("checked construction: %v", err)
			}
			made, err := manufacture.MakeWithComplexity(*kind, manufacture.Sow(7), "quick")
			if err != nil {
				t.Fatal(err)
			}
			for _, view := range []string{"full", "head"} {
				if !slices.Contains(scene.Cameras(), view) {
					t.Fatalf("missing %s camera", view)
				}
				if _, err := wanted.With(job.CameraName(view)); err != nil {
					t.Fatalf("%s camera override: %v", view, err)
				}
			}
			full, err := scene.Shot("full", nil)
			if err != nil {
				t.Fatal(err)
			}
			film, ok := scene.Film()
			if !ok || film.Wide() != 39 {
				t.Fatalf("unexpected set film: %+v", film)
			}
			across := math.Tan(film.Wide()*math.Pi/360) * .95
			vertical := across * float64(height) / float64(width)
			parts, err := made.Things(maths.Local())
			if err != nil || len(parts) == 0 {
				t.Fatalf("made parts: %v", err)
			}
			for _, part := range parts {
				box := part.Shape.Bounds()
				for _, x := range []float64{box.Min.X, box.Max.X} {
					for _, y := range []float64{box.Min.Y, box.Max.Y} {
						for _, z := range []float64{box.Min.Z, box.Max.Z} {
							world := part.At(0).Out(maths.V(x, y, z))
							point := full.Eye().In(world)
							if point.Z >= 0 || math.Abs(point.X) > -point.Z*across || math.Abs(point.Y) > -point.Z*vertical {
								t.Fatalf("full camera clips part bound %v (camera %v)", world, point)
							}
						}
					}
				}
			}
			head, err := scene.Shot("head", nil)
			if err != nil {
				t.Fatal(err)
			}
			// Inspect the actual head geometry, including the snout and eyes,
			// not just whether the eye station is the camera's focus.
			headStart := profile.Stations[profile.Head.EyeStation-1].X
			minX, maxX, headPoints := math.Inf(1), math.Inf(-1), 0
			clipped, behind := 0, 0
			checkHead := func(world maths.Vec) {
				point := head.Eye().In(world)
				if point.Z >= 0 {
					behind++
					return
				}
				x, y := point.X/(-point.Z*across), point.Y/(-point.Z*vertical)
				if math.Abs(x) > 1 || math.Abs(y) > 1 {
					clipped++
				}
				minX, maxX, headPoints = math.Min(minX, x), math.Max(maxX, x), headPoints+1
			}
			for _, part := range parts {
				if skin, ok := part.Shape.(*mesh.Mesh); ok {
					for _, point := range skin.Points() {
						world := part.At(0).Out(point)
						if world.X >= headStart {
							checkHead(world)
						}
					}
				} else {
					box := part.Shape.Bounds()
					for _, x := range []float64{box.Min.X, box.Max.X} {
						for _, y := range []float64{box.Min.Y, box.Max.Y} {
							for _, z := range []float64{box.Min.Z, box.Max.Z} {
								checkHead(part.At(0).Out(maths.V(x, y, z)))
							}
						}
					}
				}
			}
			if clipped > 0 || behind > 0 {
				t.Errorf("head camera clips %d points and leaves %d behind", clipped, behind)
			}
			if headPoints == 0 || maxX-minX < .3 {
				t.Errorf("head does not occupy enough of the frame: %d points, span %g", headPoints, maxX-minX)
			}
			// Near and far feet of each pair must project separately enough
			// to reveal four legs instead of overlapping into two columns.
			projectX := func(p maths.Vec) float64 { q := full.Eye().In(p); return q.X / -q.Z }
			for _, limb := range []sauropods.Limb{profile.Fore, profile.Hind} {
				left, right := projectX(maths.V(limb.FootX, 0, -limb.FootZ)), projectX(maths.V(limb.FootX, 0, limb.FootZ))
				a, b := projectX(maths.V(limb.FootX-limb.FootLength/2, 0, 0)), projectX(maths.V(limb.FootX+limb.FootLength/2, 0, 0))
				if math.Abs(left-right) < .55*math.Abs(a-b) {
					t.Errorf("full view overlaps near and far feet")
				}
			}
			world, ok := scene.World()
			if !ok {
				t.Fatal("review needs a ground receiver")
			}
			ground := false
			for _, box := range world.Boxes() {
				if box.Name() == "ground" && math.Abs(box.Middle().Y+box.Size().Y/2) < 1e-9 {
					ground = true
				}
			}
			if !ground {
				t.Error("no physical ground at the soles")
			}
		})
	}
}
