package emptyspace

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed policy.yaml
var policyRaw []byte

// DefaultOptions reads picked proof/storage budgets through Puzzlemouth.
func DefaultOptions() (Options, error) {
	book, err := numbers.Read("policy.yaml", policyRaw)
	if err != nil {
		return Options{}, err
	}
	var values [4]int64
	for index, name := range []string{"brick-side", "bytes", "rays", "query-bricks"} {
		value, err := book.Get(name)
		if err != nil {
			return Options{}, err
		}
		if !finite(value.Value) || value.Value < 1 || value.Value >= float64(math.MaxInt) ||
			value.Value != math.Trunc(value.Value) {
			return Options{}, fmt.Errorf("emptyspace: invalid policy %s", name)
		}
		values[index] = int64(value.Value)
	}
	return Options{BrickSide: int(values[0]), Bytes: values[1],
		Rays: int(values[2]), QueryBricks: int(values[3])}, nil
}
