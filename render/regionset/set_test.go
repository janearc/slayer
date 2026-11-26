package regionset

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

const valid = `job: street.yaml
threshold: 0.05
regions:
  - name: water
    observation: bright reflections
    left: 0
    top: 0
    width: 8
    height: 6
`

func TestReadsARegionAtTheFrameEdge(t *testing.T) {
	got, err := Read(strings.NewReader(valid), 8, 6)
	want := Region{Name: "water", Observation: "bright reflections", Width: 8, Height: 6}
	if err != nil || got.Job != "street.yaml" || got.Threshold != 0.05 ||
		len(got.Regions) != 1 || got.Regions[0] != want {
		t.Fatalf("read %+v, %v", got, err)
	}
}

func TestRefusesMissingAndInvalidFields(t *testing.T) {
	cases := map[string]string{
		"job missing":         strings.Replace(valid, "job: street.yaml\n", "", 1),
		"blank job":           strings.Replace(valid, "street.yaml", "'  '", 1),
		"threshold missing":   strings.Replace(valid, "threshold: 0.05\n", "", 1),
		"threshold zero":      strings.Replace(valid, "0.05", "0", 1),
		"threshold negative":  strings.Replace(valid, "0.05", "-1", 1),
		"threshold nan":       strings.Replace(valid, "0.05", ".nan", 1),
		"threshold infinity":  strings.Replace(valid, "0.05", ".inf", 1),
		"empty regions":       "job: street.yaml\nthreshold: 1\nregions: []\n",
		"name missing":        strings.Replace(valid, "name: water", "name: ''", 1),
		"name whitespace":     strings.Replace(valid, "name: water", "name: ' water '", 1),
		"observation missing": strings.Replace(valid, "    observation: bright reflections\n", "", 1),
		"left missing":        strings.Replace(valid, "    left: 0\n", "", 1),
		"top missing":         strings.Replace(valid, "    top: 0\n", "", 1),
		"width missing":       strings.Replace(valid, "    width: 8\n", "", 1),
		"height missing":      strings.Replace(valid, "    height: 6\n", "", 1),
		"negative left":       strings.Replace(valid, "left: 0", "left: -1", 1),
		"negative top":        strings.Replace(valid, "top: 0", "top: -1", 1),
		"zero width":          strings.Replace(valid, "width: 8", "width: 0", 1),
		"negative height":     strings.Replace(valid, "height: 6", "height: -1", 1),
		"outside right":       strings.Replace(valid, "left: 0", "left: 1", 1),
		"outside bottom":      strings.Replace(valid, "top: 0", "top: 1", 1),
		"unknown key":         valid + "    typo: 1\n",
		"duplicate key":       valid + "    left: 1\n",
		"second document":     valid + "---\njob: other.yaml\n",
		"duplicate name":      valid + "  - name: water\n    observation: other\n    left: 0\n    top: 0\n    width: 1\n    height: 1\n",
		"coordinate overflow": strings.Replace(valid, "left: 0", fmt.Sprintf("left: %d", math.MaxInt), 1),
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Read(strings.NewReader(text), 8, 6)
			if err == nil || got.Job != "" || got.Regions != nil {
				t.Fatalf("invalid set returned %+v, %v", got, err)
			}
		})
	}
}

func TestRefusesInvalidFrame(t *testing.T) {
	for _, dimensions := range [][2]int{{0, 6}, {8, 0}, {-1, 6}, {8, -1}} {
		if _, err := Read(strings.NewReader(valid), dimensions[0], dimensions[1]); err == nil {
			t.Fatalf("accepted frame %v", dimensions)
		}
	}
}
