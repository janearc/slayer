package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestInventoryIncludesSelectedCraftAndTheirLimitations(t *testing.T) {
	var out, failed bytes.Buffer
	status := tool(context.Background()).Run([]string{
		"inventory", "apollo-csm", "webb", "owl"}, &out, &failed)
	if status != 0 {
		t.Fatal(failed.String())
	}
	var rows []inventoryRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("lost selected spacecraft: %s", out.String())
	}
	for _, row := range rows {
		if row.Parts <= 0 || row.Seed != 1 || row.Detail != "quick" || row.Notes == "" {
			t.Fatalf("missing measured inventory context: %+v", row)
		}
		if row.Minimum != 600 || row.Review != "pending" {
			t.Fatalf("incorrect maturity claim: %+v", row)
		}
		if row.Kind == "owl" && row.Parts < 3264 {
			t.Fatalf("OWL panels were lost inside mesh batches: %d", row.Parts)
		}
	}
}

func Example_inventory() {
	var out, failed bytes.Buffer
	tool(context.Background()).Run([]string{"inventory", "spitzer"}, &out, &failed)
	var rows []inventoryRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		panic(err)
	}
	fmt.Println(rows[0].Kind, rows[0].Minimum, rows[0].Review)
	// Output:
	// spitzer 600 pending
}

func TestInventoryKeepsAncillaryEquipmentOutsideTheFloor(t *testing.T) {
	row, err := measureEntry(entry{Kind: "human-scale-2m", Group: "Scale references"})
	if err != nil || row.Minimum != 0 || row.Shortfall != 0 || row.Parts == 0 {
		t.Fatal("scale reference was treated as a spacecraft", row, err)
	}
	if _, err := measureEntry(entry{Kind: "absent"}); err == nil {
		t.Fatal("absent factory kind was accepted")
	}
	if rows, err := measureEntries([]entry{{Kind: "spitzer"}, {Kind: "absent"}}); err == nil || rows != nil {
		t.Fatal("a failed later entry left a partial inventory", rows, err)
	}
	var partial bytes.Buffer
	if err := writeInventory(&partial, []entry{{Kind: "spitzer"}, {Kind: "absent"}}); err == nil || partial.Len() != 0 {
		t.Fatal("a failed inventory wrote partial output", err)
	}
	if _, err := measureKind(entry{}, manufacture.Kind{}); err == nil {
		t.Fatal("an invalid custom recipe was counted")
	}
	before := menuRaw
	t.Cleanup(func() { menuRaw = before })
	menuRaw = []byte("entries: [")
	var out, failed bytes.Buffer
	if tool(context.Background()).Run([]string{"inventory"}, &out, &failed) == 0 || out.Len() != 0 {
		t.Fatal("malformed menu produced an inventory")
	}
}

// anti-test of TestInventoryIncludesSelectedCraftAndTheirLimitations
func TestInventoryRefusesUnknownKinds(t *testing.T) {
	var out, failed bytes.Buffer
	if status := tool(context.Background()).Run([]string{
		"inventory", "not-a-spacecraft"}, &out, &failed); status == 0 {
		t.Fatal("unknown model admitted to inventory")
	}
	if out.Len() != 0 {
		t.Fatal("refused selection wrote a partial inventory")
	}
}
