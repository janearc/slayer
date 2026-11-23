package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestCensusStartsWithRequestedMissionsAndIncludesUnlaunchedConcepts(t *testing.T) {
	var out, failed bytes.Buffer
	if tool(context.Background()).Run([]string{"census"}, &out, &failed) != 0 {
		t.Fatal(failed.String())
	}
	var report struct {
		Complete bool   `json:"complete"`
		Scope    string `json:"scope"`
		Reviewed string `json:"reviewed-on"`
		Missions []struct {
			Kind     string `json:"kind"`
			Flight   string `json:"flight"`
			Presence string `json:"presence"`
			Parts    *int   `json:"parts"`
			Minimum  int    `json:"minimum-parts"`
			Review   string `json:"maturity-review"`
		} `json:"missions"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Complete || report.Scope == "" || report.Reviewed != "2026-11-15" || len(report.Missions) < 20 {
		t.Fatal("initial census misrepresented its coverage", report.Scope)
	}
	for index, kind := range []string{"jimo", "new-horizons", "cassini", "huygens"} {
		row := report.Missions[index]
		if row.Kind != kind || row.Presence != "present" || row.Parts == nil ||
			*row.Parts <= 0 || row.Minimum != 600 || row.Review != "pending" {
			t.Fatalf("priority mission misrepresented: %+v", row)
		}
	}
	if report.Missions[0].Flight != "unlaunched" {
		t.Fatal("JIMO was excluded or labelled as flown")
	}
	var spitzer, webb, voyager bool
	for _, row := range report.Missions {
		if row.Kind == "spitzer" || row.Kind == "webb" {
			spitzer = spitzer || row.Kind == "spitzer"
			webb = webb || row.Kind == "webb"
			if row.Presence != "present" || row.Parts == nil || *row.Parts < 600 || row.Review != "pending" {
				t.Fatal("part-floor pass was mistaken for maturity acceptance", row)
			}
		}
		// Voyager 1 is the existing mission of the slate that is still under
		// the part floor, and must be listed as such, review pending.
		if row.Kind == "voyager-1" {
			voyager = true
			if row.Parts == nil || *row.Parts >= 600 || row.Review != "pending" {
				t.Fatal("existing under-floor mission disappeared", row)
			}
		}
	}
	if !spitzer || !webb || !voyager {
		t.Fatal("census did not reconcile existing missions")
	}
}

func Example_census() {
	var out, failed bytes.Buffer
	tool(context.Background()).Run([]string{"census", "jimo", "new-horizons"}, &out, &failed)
	var report missionReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		panic(err)
	}
	for _, row := range report.Missions {
		fmt.Println(row.Kind, row.Flight, row.Presence, row.Minimum)
	}
	// Output:
	// jimo unlaunched present 600
	// new-horizons launched present 600
}

func TestCensusRejectsDuplicateAndUnsupportedRecords(t *testing.T) {
	base, err := readCensus(censusRaw)
	if err != nil {
		t.Fatal(err)
	}
	first := base.Missions[0]
	for _, change := range []func(*missionSurvey){
		func(s *missionSurvey) { s.Missions = []missionCandidate{first, first} },
		func(s *missionSurvey) { s.Complete = true },
		func(s *missionSurvey) { s.Missions[0].Flight = "successful-ish" },
		func(s *missionSurvey) { s.Missions[0].References = nil },
	} {
		copy := *base
		copy.Missions = append([]missionCandidate(nil), base.Missions...)
		change(&copy)
		raw, err := json.Marshal(copy)
		if err != nil {
			t.Fatal(err)
		}
		if parsed, err := readCensus(raw); err == nil || parsed != nil {
			t.Fatal("invalid census accepted")
		}
	}
	base.Missions[0], base.Missions[4] = base.Missions[4], base.Missions[0]
	raw, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	sorted, err := readCensus(raw)
	if err != nil || sorted.Missions[0].Priority != 1 || sorted.Missions[2].Priority != 2 {
		t.Fatal("priority order was lost", err)
	}
}

func TestCensusFailureDoesNotPublishPartialData(t *testing.T) {
	var out bytes.Buffer
	broken := missionSurvey{Missions: []missionCandidate{{Kind: "absent"}}}
	if err := writeCensus(&out, menu{Entries: []entry{{Kind: "absent"}}}, broken, nil); err == nil || out.Len() != 0 {
		t.Fatal("failed construction produced a partial report", err)
	}
	file, err := os.CreateTemp(t.TempDir(), "closed-writer")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writeCensus(file, menu{}, broken, nil); err == nil {
		t.Fatal("failed output writer reported success")
	}
	oldMenu, oldCensus := menuRaw, censusRaw
	t.Cleanup(func() { menuRaw, censusRaw = oldMenu, oldCensus })
	for _, field := range []*[]byte{&menuRaw, &censusRaw} {
		*field = []byte("{")
		var failed bytes.Buffer
		if tool(context.Background()).Run([]string{"census"}, &out, &failed) == 0 || out.Len() != 0 || failed.Len() == 0 {
			t.Fatal("malformed input did not fail atomically")
		}
		menuRaw, censusRaw = oldMenu, oldCensus
	}
}

func TestCensusSelectionRefusesUnknownMissionsWithoutPartialOutput(t *testing.T) {
	var out, failed bytes.Buffer
	if tool(context.Background()).Run([]string{"census", "jimo", "made-up-mission"}, &out, &failed) == 0 || out.Len() != 0 {
		t.Fatal("unknown mission produced a partial census")
	}
}
