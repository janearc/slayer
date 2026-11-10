//ninja:vow dojo private lattice guards have no public recipe inputs
package manufacture

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed testdata/large-primary.yaml
var largePrimaryRaw []byte

type axialExpectation struct{ Column, Row int }

func (point axialExpectation) cell() axialPanel {
	return axialPanel{column: point.Column, row: point.Row}
}

type primaryInventoryCase struct{ Count, Hole, Want int }
type primarySectorCase struct {
	Count, Hole int
	Centres     []axialExpectation
}
type largePrimaryCases struct {
	Start       axialExpectation
	Rotation    []axialExpectation
	Inventories []primaryInventoryCase
	Sectors     []primarySectorCase
}

func readLargePrimary(test *testing.T) largePrimaryCases {
	test.Helper()
	var data largePrimaryCases
	if err := yamlfile.Read("large-primary.yaml", largePrimaryRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	if len(data.Rotation) != hexagonCorners ||
		len(data.Inventories) == spaceZero || len(data.Sectors) == spaceZero {
		test.Fatal("incomplete lattice expectations")
	}
	return data
}

func TestLargePrimaryRotationVisitsEachSector(test *testing.T) {
	data := readLargePrimary(test)
	cell := data.Start.cell()
	for _, expected := range data.Rotation {
		if cell != expected.cell() {
			test.Fatal("wrong lattice rotation", cell, expected)
		}
		cell = cell.rotate()
	}
	if cell != data.Start.cell() {
		test.Fatal("six rotations did not return to the first sector", cell)
	}
}

func TestLargePrimaryRefusesIncompleteOrbits(test *testing.T) {
	for _, row := range readLargePrimary(test).Inventories {
		points := annularPanels(row.Count, row.Hole)
		if len(points) != row.Want {
			test.Fatal("invalid sixfold inventory accepted", row, points)
		}
	}
}

func TestLargePrimarySectorRefusesInvalidInputs(test *testing.T) {
	for _, row := range readLargePrimary(test).Sectors {
		cells := annularSector(row.Count, row.Hole)
		if len(cells) != len(row.Centres) {
			test.Fatal("wrong sector inventory", row, cells)
		}
		for index, expected := range row.Centres {
			if cells[index] != expected.cell() {
				test.Fatal("wrong nearest sector centre", cells, expected)
			}
		}
	}
}
