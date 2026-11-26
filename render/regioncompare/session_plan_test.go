//go:build draw

package regioncompare

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/render/regionset"
)

func TestPlanSessionAndCompare(t *testing.T) {
	source, region := sessionFixture(t)
	if err := source.PlanSession([]regionset.Region{region}, 2, t.TempDir(), true); err != nil {
		t.Fatal(err)
	}
	ref := source.References[region.Name]
	set := `job: job.yaml
threshold: 0.1
regions:
  - {name: box, observation: box, left: 1, top: 1, width: 2, height: 2}
`
	result, err := Run(strings.NewReader(set), 4, 4, source.Candidates, map[string]Identity{"box": ref.Identity}, source)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Regions) != 1 || result.Regions[0].ReferenceRays != 2 {
		t.Fatalf("result: %+v", result)
	}
}

func TestPlanSessionRefusesUnassertedClosureAndDuplicateRegions(t *testing.T) {
	source, region := sessionFixture(t)
	original := source.Candidates[region.Name]
	if err := source.PlanSession([]regionset.Region{region}, 2, t.TempDir(), false); err == nil {
		t.Fatal("closure not asserted")
	}
	if err := source.PlanSession([]regionset.Region{region, region}, 2, t.TempDir(), true); err == nil {
		t.Fatal("duplicate accepted")
	}
	if source.Candidates[region.Name] != original {
		t.Fatal("failed plan changed candidate identities")
	}
}
