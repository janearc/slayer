package regioncompare

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"

	"github.com/janearc/slayer/render/regionjudge"
	"github.com/janearc/slayer/render/regionset"
)

// WriteReport writes the comparison's linear-light scores and elapsed times.
// A finite reference is an estimate, not proof of convergence.
func WriteReport(out io.Writer, result Result) error {
	if out == nil || result.Sheet == nil || len(result.Regions) == 0 {
		return fmt.Errorf("regioncompare: completed comparison and writer are required")
	}
	type repeat struct {
		Seconds float64            `json:"seconds"`
		Score   regionjudge.Report `json:"repeat_minus_first"`
	}
	type row struct {
		Region           regionset.Region   `json:"region"`
		Candidate        string             `json:"candidate_identity_sha256"`
		Reference        string             `json:"reference_identity_sha256"`
		CandidateRays    int                `json:"candidate_rays_per_side"`
		ReferenceRays    int                `json:"reference_rays_per_side"`
		DrawSeconds      float64            `json:"candidate_seconds"`
		ReferenceSeconds float64            `json:"reference_prepare_and_read_seconds"`
		Score            regionjudge.Report `json:"linear_light"`
		Repeats          []repeat           `json:"repeats,omitempty"`
	}
	rows := make([]row, 0, len(result.Regions))
	for _, r := range result.Regions {
		var repeats []repeat
		for _, each := range r.Repeats {
			repeats = append(repeats, repeat{Seconds: each.Duration.Seconds(), Score: each.Report})
		}
		rows = append(rows, row{Region: r.Region, Candidate: hex.EncodeToString(r.Candidate[:]), Reference: hex.EncodeToString(r.Reference[:]),
			CandidateRays: r.CandidateRays, ReferenceRays: r.ReferenceRays, DrawSeconds: r.DrawDuration.Seconds(),
			ReferenceSeconds: r.ReferenceDuration.Seconds(), Score: r.Report, Repeats: repeats})
	}
	report := struct {
		Job       string  `json:"job"`
		Threshold float64 `json:"isolated_positive_threshold"`
		Caveat    string  `json:"caveat"`
		Regions   []row   `json:"regions"`
	}{result.Job, result.Threshold, "A finite reference is an estimate; signed differences help reveal lost light.", rows}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// WriteSheet encodes the labelled comparison sheet as PNG.
func WriteSheet(out io.Writer, result Result) error {
	if out == nil || result.Sheet == nil || len(result.Regions) == 0 {
		return fmt.Errorf("regioncompare: completed comparison and writer are required")
	}
	return png.Encode(out, result.Sheet)
}
