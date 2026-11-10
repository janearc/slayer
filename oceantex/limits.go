package oceantex

import (
	_ "embed"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed limits.yaml
var limitsRaw []byte

type resourceLimits struct {
	WaveCells   int `yaml:"wave-cells"`
	ImagePixels int `yaml:"image-pixels"`
	ShoreCells  int `yaml:"shore-cells"`
}

// readLimits bounds reference-model allocations before multiplying dimensions.
var readLimits = sync.OnceValues(func() (resourceLimits, error) {
	var limits resourceLimits
	if err := yamlfile.Read("limits.yaml", limitsRaw, &limits); err != nil {
		return limits, err
	}
	if limits.WaveCells < 4 || limits.ImagePixels < 1 || limits.ShoreCells < 9 {
		return limits, fmt.Errorf("oceantex: invalid resource limits")
	}
	return limits, nil
})

// finite reports whether a value is an ordinary number: not NaN, not infinite.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
