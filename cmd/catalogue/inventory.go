package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

type inventoryRow struct {
	Kind       string      `json:"kind"`
	Group      string      `json:"group"`
	Animal     *animalSpec `json:"animal,omitempty"`
	Plant      *plantSpec  `json:"plant,omitempty"`
	Alga       *algaSpec   `json:"alga,omitempty"`
	Source     string      `json:"profile-identity-sha256,omitempty"`
	Detail     string      `json:"detail"`
	Seed       uint64      `json:"seed"`
	Parts      int         `json:"parts"`
	Minimum    int         `json:"minimum-parts,omitempty"`
	Shortfall  int         `json:"shortfall,omitempty"`
	Review     string      `json:"maturity-review"`
	Bounds     maths.Box   `json:"bounds-metres"`
	Notes      string      `json:"limitations"`
	References []reference `json:"references"`
}

// inventory measures selected specimens without drawing or declaring maturity.
func inventory(call pzm.Call) error {
	menu, err := menuForCall(call)
	if err != nil {
		return err
	}
	entries, err := menu.selectEntries(call.Args())
	if err != nil {
		return err
	}
	return writeInventory(call.Out(), entries)
}

// writeInventory constructs the complete report before emitting any bytes.
func writeInventory(out io.Writer, entries []entry) error {
	rows, err := measureEntries(entries)
	if err != nil {
		return err
	}
	writer := json.NewEncoder(out)
	writer.SetIndent("", "  ")
	return writer.Encode(rows)
}

// measureEntries refuses a partial report if any specimen cannot be built.
func measureEntries(entries []entry) ([]inventoryRow, error) {
	rows := make([]inventoryRow, 0, len(entries))
	for _, entry := range entries {
		row, err := measureEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("catalogue inventory %s: %w", entry.Kind, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// measureEntry keeps the model's limitations beside its generated population.
func measureEntry(entry entry) (inventoryRow, error) {
	kind, err := entry.kind()
	if err != nil {
		return inventoryRow{}, err
	}
	return measureKind(entry, kind)
}

// measureKind also supports checked custom recipes outside the named menu.
func measureKind(entry entry, kind manufacture.Kind) (inventoryRow, error) {
	made, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(1), "quick")
	if err != nil {
		return inventoryRow{}, err
	}
	minimum := entry.partMinimum()
	return inventoryRow{Kind: entry.Kind, Group: entry.Group,
		Animal: entry.Animal, Plant: entry.Plant, Alga: entry.Alga, Source: entry.source,
		Detail: "quick", Seed: 1, Parts: made.PartCount(), Minimum: minimum,
		Shortfall: max(0, minimum-made.PartCount()), Review: "pending",
		Bounds: made.Bounds(), Notes: entry.Notes, References: entry.References}, nil
}

// partMinimum applies the floor to existing instruments and spacecraft.
// The alias shares Spitzer's inventory; ancillary equipment has no such floor.
func (entry entry) partMinimum() int {
	switch entry.Group {
	case "Personal telescopes", "Ground observatories", "Observatories",
		"Visitors", "Crewed spacecraft", "Launchers", "Maunakea families",
		"Haleakala families", "Historical and unbuilt concepts":
		return 600
	}
	return 0
}
