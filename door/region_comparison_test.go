package door

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
)

func comparisonTool(env Env) *pzm.Tool {
	made := pzm.New("slayer", "draw and compare", pzm.Build(""),
		pzm.Built(""), pzm.Who(""), pzm.Sessions(""))
	declareRegionComparison(made, env)
	return made
}

func TestComparisonRequiresExplicitSnapshotDetails(t *testing.T) {
	got := running(comparisonTool(Env{}), "compare-regions", "set.yaml")
	if got.code == 0 || !strings.Contains(got.said+got.wrongSaid, "--snapshot") {
		t.Fatalf("missing snapshot: %d %s%s", got.code, got.said, got.wrongSaid)
	}
}

func TestComparisonRefusesMissingOrExtraPaths(t *testing.T) {
	for _, args := range [][]string{{"compare-regions"}, {"compare-regions", "one.yaml", "two.yaml"}} {
		got := running(comparisonTool(Env{}), args...)
		if got.code == 0 || !strings.Contains(got.said+got.wrongSaid, "one set") {
			t.Fatalf("arguments %v: %d %s%s", args, got.code, got.said, got.wrongSaid)
		}
	}
}

func TestComparisonHelpDescribesTheClosureAssertion(t *testing.T) {
	got := running(comparisonTool(Env{}), "compare-regions", "--help")
	if got.code != 0 || !strings.Contains(got.said+got.wrongSaid, "--complete-inputs") {
		t.Fatalf("help: %d %s%s", got.code, got.said, got.wrongSaid)
	}
}

func TestComparisonRejectsAnUnknownFlag(t *testing.T) {
	got := running(comparisonTool(Env{}), "compare-regions", "set.yaml", "--invented")
	if got.code == 0 {
		t.Fatalf("accepted an unknown flag: %s", got.said)
	}
}
