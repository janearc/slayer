package regioncompare

import (
	"bytes"
	"encoding/json"
	"image/png"
	"strings"
	"testing"
)

func TestWritesComparisonReportAndSheet(t *testing.T) {
	result, err := Run(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}},
		map[string]Identity{"water": {2}}, &sourceStub{})
	if err != nil {
		t.Fatal(err)
	}
	var report bytes.Buffer
	if err := WriteReport(&report, result); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Job     string `json:"job"`
		Regions []struct {
			CandidateRays int    `json:"candidate_rays_per_side"`
			ReferenceRays int    `json:"reference_rays_per_side"`
			Candidate     string `json:"candidate_identity_sha256"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(report.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Job != "shot.yaml" || len(decoded.Regions) != 1 || decoded.Regions[0].CandidateRays != 4 ||
		decoded.Regions[0].ReferenceRays != 16 || len(decoded.Regions[0].Candidate) != 64 {
		t.Fatalf("report: %+v", decoded)
	}
	var sheet bytes.Buffer
	if err := WriteSheet(&sheet, result); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(&sheet); err != nil {
		t.Fatal(err)
	}
}

func TestRefusesIncompleteReportAndSheet(t *testing.T) {
	var out bytes.Buffer
	if WriteReport(&out, Result{}) == nil || WriteSheet(&out, Result{}) == nil || out.Len() != 0 {
		t.Fatal("incomplete comparison was written")
	}
}
