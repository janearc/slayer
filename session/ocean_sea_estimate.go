package session

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// leastSamples is the fewest samples a spectrum estimate takes, which a
// sea with no edge says in place of any: its report then says none.
const leastSamples = 2

// unboundedSpectrumEstimate is the spectrum estimate of a sea with no edge.
// Its extent is the longest wave band's side, where the sea repeats, and it
// has no samples. A cox-munk glint's slopes are the sea's own bands', not a
// sheet's.
func unboundedSpectrumEstimate(surface set.WaveSurface) (OceanSpectrumEstimate,
	error) {
	model, err := stage.SeaOf(surface)
	if err != nil {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean: %w", err)
	}
	sea, err := arid.NewSea(model, surface.Seconds(), surface.Seed())
	if err != nil {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean: %w", err)
	}
	report, err := estimateSeaInputs(model, surface.Source(), sea.Period(),
		leastSamples, surface.Seed(), surface.Seconds())
	if err != nil {
		return OceanSpectrumEstimate{}, err
	}
	report.Samples = 0
	report.PeakWavelengthMetres = model.PeakWavelength()
	measured := sea.Measure()
	report.Measured = &measured
	report.DirectSunMode = surface.DirectSunMode()
	report.GlintModel = surface.Glint().Model()
	if report.GlintModel == "" {
		report.GlintModel = set.GlintCoxMunk
	}
	report.GlintRoughness = surface.Glint().Roughness()
	if report.GlintModel == set.GlintCoxMunk {
		budget, err := sea.GlintBudget(model)
		if err != nil {
			return OceanSpectrumEstimate{}, fmt.Errorf("ocean glint: %w", err)
		}
		report.GlintSlopes = &budget
	}
	return report, nil
}

// measuredLine is the receipt's account of a sea with no edge as held and
// as drawn, beside the peak wave the spectrum asks; or nothing for a
// sheet. Heights are significant heights, slopes mean squares both ways
// together.
func (report OceanSpectrumEstimate) measuredLine() string {
	measured := report.Measured
	if measured == nil {
		return ""
	}
	text := fmt.Sprintf("; asked: peak wave %.4g m; held in the bands: "+
		"height %.4g m, peak wave %.4g m, slope %.4g; met by a ray: "+
		"height %.4g m, peak wave %.4g m, slope %.4g, skewness %.3g; "+
		"shaded by: slope %.4g; breaking: %.3g of the sea (whitecaps of "+
		"every age by Monahan and O'Muircheartaigh 1980: %.3g); bands",
		report.PeakWavelengthMetres,
		measured.HeldHeightMetres, measured.HeldPeakMetres,
		measured.HeldSlope, measured.MetHeightMetres,
		measured.MetPeakMetres, measured.MetSlope, measured.MetSkewness,
		measured.ShadedSlope, measured.Breaking,
		oceantex.Whitecaps(report.WindMetresPerSecond))
	for _, band := range measured.Bands {
		text += fmt.Sprintf(" [%.4g m by %.4g m: height %.4g m, slope "+
			"%.4g of %.4g expected]", band.SideMetres, band.SpacingMetres,
			heldTimes*math.Sqrt(band.HeightVariance), band.SlopeVariance,
			band.SlopeExpected)
	}
	return text
}

// heldTimes is how many standard deviations a significant height is:
// four, the definition wave buoys report by.
const heldTimes = 4
