//go:build draw

package main

import (
	_ "embed"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

func TestCatalogueDeclaresEveryKindAndOnlyActualDetails(test *testing.T) {
	menu, err := readMenu()
	if err != nil {
		test.Fatal(err)
	}
	plan := defaultPlan(test)
	for _, entry := range menu.Entries {
		row, err := manufacture.KindNamed(entry.Kind)
		if err != nil {
			test.Fatal(err)
		}
		var first *maths.Box
		for _, detail := range entry.Details {
			object, err := manufacture.MakeWithComplexity(row,
				manufacture.Sow(plan.Seed), detail)
			if err != nil {
				test.Fatal(entry.Kind, detail, err)
			}
			bounds := object.Bounds()
			if first != nil && *first != bounds {
				test.Fatal("detail moved the physical envelope", entry.Kind)
			}
			first = &bounds
		}
	}
}
