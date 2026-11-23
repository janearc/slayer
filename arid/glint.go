package arid

import (
	"fmt"

	"github.com/janearc/slayer/oceantex"
)

// GlintSlopes is the slope budget of a sea sampled as NewHeightField
// samples it, extent metres a side at samples a side: the spacing
// between samples sets the Nyquist wave number at which the sheet's own
// slope ends and the glint lobe's begins. See oceantex.SlopeBudget.
//
// The budget's resolved part is the spectrum's integral to that wave
// number. The sheet itself samples the cascades' heights, some finer
// than its cells, so the slope its cells carry may differ.
func GlintSlopes(sea oceantex.Sea, extent float64,
	samples int) (oceantex.SlopeBudget, error) {
	if samples < 2 {
		return oceantex.SlopeBudget{}, fmt.Errorf(
			"scene/ocean: a sheet of %d samples a side has no spacing",
			samples)
	}
	return sea.SlopeBudget(extent / float64(samples-1))
}
