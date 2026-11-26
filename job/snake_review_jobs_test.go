package job_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/snakes"
)

// Each review job must resolve the actual checked species and both cameras
// before its two independent Azazel requests can be submitted.
func TestSnakeReviewJobsResolveProfilesAndBothViews(t *testing.T) {
	file, raw := snakes.Embedded()
	book, err := snakes.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(book.Species()) != 9 {
		t.Fatalf("review needs nine snake profiles, got %d", len(book.Species()))
	}
	for _, species := range book.Species() {
		t.Run(species, func(t *testing.T) {
			path := filepath.Join("..", "jobs", "bioforms", "snake-review", species+".yaml")
			wanted, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			scene, ok := wanted.ImportedSet()
			if !ok {
				t.Fatal("job did not import its set")
			}
			recipe, ok := scene.Recipe("specimen")
			if !ok || !recipe.IsAnimal() || recipe.AnimalSpecies() != species {
				t.Fatalf("job did not select %s", species)
			}
			for _, view := range []string{"full", "head"} {
				if !slices.Contains(scene.Cameras(), view) {
					t.Errorf("job lacks %s camera", view)
				}
				if _, err := wanted.With(job.CameraName(view)); err != nil {
					t.Errorf("%s camera override: %v", view, err)
				}
			}
		})
	}
}
