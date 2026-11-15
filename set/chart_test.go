package set_test

import (
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/planet-charts.yaml
var planetChartsRaw []byte

// planetCharts is testdata/planet-charts.yaml: the lines a planet is given
// and the chart it is expected to say.
type planetCharts struct {
	Charts []struct {
		Name, Line string
		Chart      float64
	}
}

// onePlaceInPlanet is how many places of the set take the added line. It
// is defined as one: the planet's block says made_up once.
const onePlaceInPlanet = 1

// charted is Melancholia's planet with a line added to its block; nil
// where the set is refused.
func charted(t *testing.T, line string) (*set.Planet, error) {
	t.Helper()
	source, err := os.ReadFile("../sets/melancholia.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := strings.Replace(string(source), "    made_up: true\n",
		"    made_up: true\n"+line, onePlaceInPlanet)
	if !strings.Contains(written, "    made_up: true\n") {
		t.Fatal("the set's planet does not say made_up as expected")
	}
	made, err := set.Read("melancholia.yaml", []byte(written))
	if err != nil {
		return nil, err
	}
	planet, _ := made.Planet()
	return &planet, nil
}

// a planet that does not say its chart has two points to a pixel; one
// that says has what it says, and zero is no chart; a chart under zero
// or over sixteen is refused.
func TestAPlanetSaysItsChart(t *testing.T) {
	var data planetCharts
	if err := yamlfile.Read("planet-charts.yaml", planetChartsRaw,
		&data); err != nil {
		t.Fatal(err)
	}
	for _, row := range data.Charts {
		planet, err := charted(t, row.Line)
		if err != nil {
			t.Fatal(err)
		}
		if got := planet.Chart(); got != row.Chart {
			t.Errorf("%s: with %q the chart is %v, not %v", row.Name,
				row.Line, got, row.Chart)
		}
	}
	for _, line := range []string{"    chart: -1\n", "    chart: 17\n"} {
		if _, err := charted(t, line); err == nil {
			t.Errorf("a planet with %q was read", strings.TrimSpace(line))
		}
	}
}
