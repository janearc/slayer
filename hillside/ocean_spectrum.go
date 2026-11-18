package hillside

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/set"
)

// OceanSpectrumEstimate is the spectrum model's estimate for one authored
// ocean. It describes the continuous spectrum, not observations or the exact
// finite wave sheet sampled for a scene.
type OceanSpectrumEstimate struct {
	Model                   string  `json:"model"`
	Source                  string  `json:"source"`
	WindMetresPerSecond     float64 `json:"wind_metres_per_second"`
	FetchMetres             float64 `json:"fetch_metres"`
	DepthMetres             float64 `json:"depth_metres"`
	TowardRadians           float64 `json:"toward_radians"`
	ExtentMetres            float64 `json:"extent_metres"`
	Samples                 int     `json:"samples"`
	Seed                    uint64  `json:"seed"`
	Seconds                 float64 `json:"seconds"`
	SignificantHeightMetres float64 `json:"significant_height_metres"`
	PeakPeriodSeconds       float64 `json:"peak_period_seconds,omitempty"`
	HasPeakPeriod           bool    `json:"has_peak_period"`
	DirectSunMode           string  `json:"direct_sun_mode,omitempty"`
	// GlintModel is the authored glint model, empty when none is said;
	// GlintRoughness is an artistic glint's roughness, its Satin.
	GlintModel     string  `json:"glint_model,omitempty"`
	GlintRoughness float64 `json:"glint_roughness,omitempty"`
	// GlintSlopes is a cox-munk glint's mean-square slope budget at the
	// sheet's spacing: Cox and Munk's total, the spectrum's resolved
	// part, and the unresolved rest the Beckmann lobe carries.
	GlintSlopes *oceantex.SlopeBudget `json:"glint_slopes,omitempty"`
}

const oceanSpectrumModel = "oceantex.Sea"

// oceanSpectrumEstimate reports oceantex's spectrum estimate for the
// authored sea inputs. Calm conditions have zero significant height and no
// spectral peak, so no period is reported. Invalid or numerically degenerate
// estimates fail staging rather than printing nonfinite diagnostics.
func oceanSpectrumEstimate(surface set.WaveSurface) (OceanSpectrumEstimate,
	error) {
	wind, fetch, depth, toward := surface.Sea()
	sea := oceantex.Sea{Wind: wind, Fetch: fetch, Depth: depth,
		Toward: toward}
	report, err := estimateSeaInputs(sea, surface.Source(),
		surface.Extent(), surface.Samples(), surface.Seed(),
		surface.Seconds())
	if err != nil {
		return OceanSpectrumEstimate{}, err
	}
	report.DirectSunMode = surface.DirectSunMode()
	report.GlintModel = surface.Glint().Model()
	report.GlintRoughness = surface.Glint().Roughness()
	if report.GlintModel == set.GlintCoxMunk {
		budget, err := ocean.GlintSlopes(sea, surface.Extent(),
			surface.Samples())
		if err != nil {
			return OceanSpectrumEstimate{}, fmt.Errorf("ocean glint: %w", err)
		}
		report.GlintSlopes = &budget
	}
	return report, nil
}

// estimateSeaInputs returns the spectrum estimate for one sea state and
// its sampling.
func estimateSeaInputs(sea oceantex.Sea, source string, extent float64,
	samples int, seed uint64, seconds float64) (OceanSpectrumEstimate, error) {
	wind, fetch, depth, toward := sea.Wind, sea.Fetch, sea.Depth, sea.Toward
	if !finiteDiagnostic(wind) || !finiteDiagnostic(fetch) ||
		!finiteDiagnostic(depth) || !finiteDiagnostic(toward) ||
		wind < 0 || fetch < 0 || depth < 0 ||
		toward < 0 || toward > 2*math.Pi {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean spectrum: invalid authored sea inputs")
	}
	if !finiteDiagnostic(extent) || extent <= 0 || samples < 2 ||
		samples > ocean.MaxHeightFieldSamples ||
		!finiteDiagnostic(seconds) || seconds < 0 {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean spectrum: invalid surface sampling inputs")
	}
	height := sea.Height()
	if !finiteDiagnostic(height) || height < 0 {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean spectrum: invalid significant height estimate %g m", height)
	}
	report := OceanSpectrumEstimate{Model: oceanSpectrumModel,
		Source: source, WindMetresPerSecond: wind,
		FetchMetres: fetch, DepthMetres: depth, TowardRadians: toward,
		ExtentMetres: extent, Samples: samples,
		Seed: seed, Seconds: seconds,
		SignificantHeightMetres: height}
	if wind == 0 || fetch == 0 {
		if height != 0 {
			return OceanSpectrumEstimate{}, fmt.Errorf("ocean spectrum: calm input produced nonzero significant height %g m", height)
		}
		return report, nil
	}
	peak := sea.Peak()
	if !finiteDiagnostic(peak) || peak <= 0 {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean spectrum: invalid peak angular frequency %g rad/s", peak)
	}
	period := 2 * math.Pi / peak
	if !finiteDiagnostic(period) || period <= 0 || height <= 0 {
		return OceanSpectrumEstimate{}, fmt.Errorf("ocean spectrum: degenerate spectrum estimate (height %g m, peak period %g s)", height, period)
	}
	report.PeakPeriodSeconds = period
	report.HasPeakPeriod = true
	return report, nil
}

// finiteDiagnostic reports whether a diagnostic value is a finite number.
func finiteDiagnostic(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// line is the one human-readable CLI form of the structured estimate.
func (report OceanSpectrumEstimate) line() string {
	text := fmt.Sprintf("ocean spectrum model estimate (%s; not observation; finite sampled wave field may differ): source=%q wind=%.6g m/s fetch=%.6g m depth=%.6g m toward=%.6g rad; significant height=%.6g m",
		report.Model,
		report.Source, report.WindMetresPerSecond, report.FetchMetres,
		report.DepthMetres, report.TowardRadians,
		report.SignificantHeightMetres)
	if !report.HasPeakPeriod {
		text += "; peak period=none (calm/no spectral peak)"
	} else {
		text = fmt.Sprintf("%s; peak period=%.6g s", text,
			report.PeakPeriodSeconds)
	}
	if report.DirectSunMode != "" {
		text += "; direct-sun=" + report.DirectSunMode +
			" (configured CPU approximation; actual effect depends on the rays)"
	}
	return text + report.glintLine()
}

// glintLine is the receipt's account of the authored glint, or nothing
// when the set says none.
func (report OceanSpectrumEstimate) glintLine() string {
	slopes := report.GlintSlopes
	switch {
	case report.GlintModel == set.GlintArtistic:
		return fmt.Sprintf("; glint=ARTISTIC roughness=%.6g (GGX alpha "+
			"%.6g, chosen by eye; not a measured sea)",
			report.GlintRoughness,
			report.GlintRoughness*report.GlintRoughness)
	case slopes == nil:
		return ""
	}
	text := fmt.Sprintf("; glint=cox-munk, Beckmann lobe: mean-square "+
		"slope Cox-Munk 1954 total=%.6g (wind read as at 12.5 m), "+
		"spectrum resolved to Nyquist %.6g rad/m=%.6g, unresolved=%.6g",
		slopes.Total, slopes.Nyquist, slopes.Resolved, slopes.Unresolved)
	if slopes.Exceeds {
		text += " (the spectrum resolves more than Cox-Munk; unresolved " +
			"is the floor)"
	} else if slopes.Floored {
		text += " (floored)"
	}
	return text
}

// Summary is the one-line human-readable form of this estimate for command
// output. It does not print or alter the structured report.
func (report OceanSpectrumEstimate) Summary() string { return report.line() }

// OceanSpectrum returns the immutable spectrum estimate captured when this
// session staged its set, and whether the set authored an ocean.
func (session *Session) OceanSpectrum() (OceanSpectrumEstimate, bool) {
	if session.given.oceanSpectrum == nil {
		return OceanSpectrumEstimate{}, false
	}
	return *session.given.oceanSpectrum, true
}
