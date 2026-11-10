package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/set"
)

func TestShippedSequencePlansKeepTheirDocumentedTimingAndScenes(t *testing.T) {
	for _, test := range []struct {
		name        string
		frames, fps int
		start       time.Duration
	}{
		{"fire-night-sequence.yaml", 2880, 24, 0},
		{"fire-fuel-sequence.yaml", 180, 3, 0},
		{"fire-woodland-motion.yaml", 96, 24, 40 * time.Second},
		{"fire-woodland-spread.yaml", 120, 2, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join("../../doc", test.name)
			plan, err := readSequence(path)
			if err != nil {
				t.Fatal(err)
			}
			if plan.frames != test.frames || plan.fps != test.fps || plan.start != test.start {
				t.Fatalf("shipped timing changed: %+v", plan)
			}
			raw, err := os.ReadFile(plan.scene)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = set.Read(plan.scene, raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}
