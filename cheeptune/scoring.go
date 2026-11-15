package cheeptune

import (
	_ "embed"
	"slices"
)

//go:embed SCORING.md
var scoring []byte

// Scoring is SCORING.md's bytes: how to write a score, for a model that
// has never seen the format. slayer edit puts a copy beside a job so the
// agent in its console can read it.
func Scoring() []byte { return slices.Clone(scoring) }
