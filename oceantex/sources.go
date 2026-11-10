package oceantex

import (
	_ "embed"
	"fmt"
	"github.com/janearc/puzzlemouth/yamlfile"
	"math"
	"slices"
	"sort"
)

//go:embed sources.yaml
var sourcesYAML string

// Source is one number, or one list of numbers, and where it came from:
// an entry of sources.yaml.
type Source struct {
	Name    string
	Values  []float64 // one value, or a table
	Unit    string
	From    string // the publication
	URL     string // the page it was read on
	Fetched string // the day it was read, as 2006-01-02
	Read    string // "fetch", "search" or "none"
	Kind    string // "sourced", "definition" or "glue"
	Note    string
}

// sources is sources.yaml, read once at start-up. A file that does not
// parse stops the program: the numbers are not optional.
var sources = func() map[string]Source {
	parsed, err := parseSources(sourcesYAML)
	if err != nil {
		panic("oceantex: sources.yaml: " + err.Error())
	}
	return parsed
}()

// Sources is every number the package uses and where it came from, in
// name order.
func Sources() []Source {
	out := make([]Source, 0, len(sources))
	for _, s := range sources {
		s.Values = slices.Clone(s.Values)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// num is the one value of a named source.
func num(name string) float64 {
	s, ok := sources[name]
	if !ok || len(s.Values) != 1 {
		panic("oceantex: sources.yaml has no single value named " + name)
	}
	return s.Values[0]
}

// nums is the table of a named source.
func nums(name string) []float64 {
	s, ok := sources[name]
	if !ok || len(s.Values) == 0 {
		panic("oceantex: sources.yaml has no values named " + name)
	}
	return s.Values
}

// sourceEntry is decoded by Pzm; values stay separate from attribution.
type sourceEntry struct {
	Value                                      *float64
	Values                                     []float64
	Unit, From, URL, Fetched, Read, Kind, Note string
}

// parseSources reads the complete ledger through Pzm's strict reader.
func parseSources(text string) (map[string]Source, error) {
	var entries map[string]sourceEntry
	if err := yamlfile.Read("sources.yaml", []byte(text), &entries); err != nil {
		return nil, err
	}
	out := make(map[string]Source, len(entries))
	for name, entry := range entries {
		if name == "" || (entry.Value != nil) == (len(entry.Values) > 0) {
			return nil, fmt.Errorf("source %q needs one value or a table", name)
		}
		values := slices.Clone(entry.Values)
		if entry.Value != nil {
			values = []float64{*entry.Value}
		}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("source %q has a non-finite value", name)
			}
		}
		if entry.From == "" || entry.URL == "" || entry.Fetched == "" || entry.Read == "" || entry.Kind == "" {
			return nil, fmt.Errorf("source %q is missing attribution", name)
		}
		out[name] = Source{Name: name, Values: values, Unit: entry.Unit, From: entry.From,
			URL: entry.URL, Fetched: entry.Fetched, Read: entry.Read, Kind: entry.Kind, Note: entry.Note}
	}
	return out, nil
}
