package main

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/janearc/puzzlemouth/pzm"
)

//go:embed mission-census.json
var censusRaw []byte

//go:embed mission-census.schema.json
var censusSchema []byte

type missionCandidate struct {
	Kind       string      `json:"kind" yaml:"kind"`
	Name       string      `json:"name" yaml:"name"`
	Priority   int         `json:"priority" yaml:"priority"`
	Flight     string      `json:"flight" yaml:"flight"`
	References []reference `json:"references" yaml:"references"`
	Notes      string      `json:"notes" yaml:"notes"`
}

type missionSurvey struct {
	Scope    string             `json:"scope" yaml:"scope"`
	Complete bool               `json:"complete" yaml:"complete"`
	Reviewed string             `json:"reviewed-on" yaml:"reviewed-on"`
	Missions []missionCandidate `json:"missions" yaml:"missions"`
}

type missionStatus struct {
	missionCandidate
	Presence  string `json:"presence" yaml:"presence"`
	Parts     *int   `json:"parts" yaml:"parts"`
	Minimum   int    `json:"minimum-parts" yaml:"minimum-parts"`
	Shortfall int    `json:"shortfall" yaml:"shortfall"`
	Review    string `json:"maturity-review" yaml:"maturity-review"`
}

type missionReport struct {
	Scope    string          `json:"scope" yaml:"scope"`
	Complete bool            `json:"complete" yaml:"complete"`
	Reviewed string          `json:"reviewed-on" yaml:"reviewed-on"`
	Missions []missionStatus `json:"missions" yaml:"missions"`
}

// census reconciles the cited mission seed against this checkout's factory.
func census(call pzm.Call) error {
	menu, err := readMenu()
	if err != nil {
		return err
	}
	survey, err := readCensus(censusRaw)
	if err != nil {
		return err
	}
	return writeCensus(call.Out(), *menu, *survey, call.Args())
}

// readCensus rejects duplicate identities and keeps the declared priority order.
func readCensus(raw []byte) (*missionSurvey, error) {
	var survey missionSurvey
	if err := readDocument("mission-census.json", raw, censusSchema, &survey); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, candidate := range survey.Missions {
		if seen[candidate.Kind] {
			return nil, fmt.Errorf("duplicate mission identity %q", candidate.Kind)
		}
		seen[candidate.Kind] = true
	}
	slices.SortStableFunc(survey.Missions, func(a, b missionCandidate) int {
		return cmp.Compare(a.Priority, b.Priority)
	})
	return &survey, nil
}

// writeCensus emits nothing if selection or specimen construction fails.
func writeCensus(out io.Writer, menu menu, survey missionSurvey, names []string) error {
	available := map[string]entry{}
	for _, entry := range menu.Entries {
		available[entry.Kind] = entry
	}
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	report := missionReport{Scope: survey.Scope, Complete: survey.Complete,
		Reviewed: survey.Reviewed, Missions: []missionStatus{}}
	for _, candidate := range survey.Missions {
		if len(names) != 0 && !wanted[candidate.Kind] {
			continue
		}
		delete(wanted, candidate.Kind)
		status := missionStatus{missionCandidate: candidate, Presence: "missing",
			Minimum: 600, Shortfall: 600, Review: "not-built"}
		if existing, present := available[candidate.Kind]; present {
			measured, err := measureEntry(existing)
			if err != nil {
				return fmt.Errorf("census %s: %w", candidate.Kind, err)
			}
			status.Presence, status.Parts, status.Review = "present", &measured.Parts, "pending"
			status.Shortfall = max(0, status.Minimum-measured.Parts)
		}
		report.Missions = append(report.Missions, status)
	}
	if len(wanted) != 0 {
		var unknown []string
		for name := range wanted {
			unknown = append(unknown, name)
		}
		slices.Sort(unknown)
		return fmt.Errorf("unknown census missions: %v", unknown)
	}
	writer := json.NewEncoder(out)
	writer.SetIndent("", "  ")
	return writer.Encode(report)
}
