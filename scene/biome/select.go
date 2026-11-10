package biome

import (
	"fmt"
	"math"
)

// Budget bounds examined cells and generated candidates, including rejected
// roots. It is a work/count budget, not an exact allocation or whole-process
// RSS cap.
type Budget struct{ Cells, Candidates int }

// Selection owns selected roots and reports the work used to find them.
// Generated includes candidates rejected by the query's half-open boundaries.
// This lets a caller share one allowance across several region queries.
type Selection struct {
	Roots            []Candidate
	Cells, Generated int
}

// Select clips only root selection. It does not change population identities.
// Callers expand a visible query by their maximum canopy/geometry radius.
// A budget failure returns no partial population and silently drops nothing.
func (region *Region) Select(query Rect, budget Budget) ([]Candidate, error) {
	selection, err := region.SelectWithUsage(query, budget)
	return selection.Roots, err
}

// SelectWithUsage returns owned root values and bounded work counts.
// On error no partial population or successful work record is returned.
func (region *Region) SelectWithUsage(query Rect, budget Budget) (Selection, error) {
	if region == nil || region.cell <= 0 || !query.valid() ||
		budget.Cells < 0 || budget.Candidates < 0 {
		return Selection{}, fmt.Errorf("biome: invalid query, region or work budget")
	}
	inside := query.intersection(region.definition.Bounds)
	if !inside.valid() || region.mean == 0 {
		return Selection{}, nil
	}
	span, err := region.cellSpan(inside, budget.Cells)
	if err != nil {
		return Selection{}, err
	}
	return region.selectCells(inside, span, budget.Candidates)
}

// cellSpan keeps checked lattice limits together, including a rounding halo.
type cellSpan struct{ firstColumn, lastColumn, firstRow, lastRow int64 }

// cellSpan checks products before visiting any candidate or allocating results.
func (region *Region) cellSpan(query Rect, limit int) (cellSpan, error) {
	firstColumn, lastColumn, err := latticeRange(query.MinEast, query.MaxEast, region.cell)
	if err != nil {
		return cellSpan{}, err
	}
	firstRow, lastRow, err := latticeRange(query.MinSouth, query.MaxSouth, region.cell)
	if err != nil {
		return cellSpan{}, err
	}
	columns, rows := lastColumn-firstColumn+1, lastRow-firstRow+1
	if columns > int64(limit)/rows {
		return cellSpan{}, fmt.Errorf("biome: query exceeds its %d-cell work budget", limit)
	}
	return cellSpan{firstColumn, lastColumn, firstRow, lastRow}, nil
}

// selectCells refuses excessive candidate work even if most roots miss the
// query.
func (region *Region) selectCells(query Rect, span cellSpan, limit int) (Selection, error) {
	selection := Selection{}
	for row := span.firstRow; row <= span.lastRow; row++ {
		for column := span.firstColumn; column <= span.lastColumn; column++ {
			count := region.count(column, row)
			if uint64(count) > uint64(limit-selection.Generated) {
				return Selection{}, fmt.Errorf("biome: query exceeds its %d-candidate work budget", limit)
			}
			selection.Cells++
			selection.Generated += int(count)
			for ordinal := uint32(0); ordinal < count; ordinal++ {
				candidate := region.candidate(column, row, ordinal)
				if query.Contains(candidate.East, candidate.South) {
					selection.Roots = append(selection.Roots, candidate)
				}
			}
		}
	}
	return selection, nil
}

// latticeRange leaves one cell of halo against division rounding at boundaries.
// The 52-bit bound leaves exact integer neighbours in binary64 and int64.
func latticeRange(low, high, cell float64) (int64, int64, error) {
	first, last := math.Floor(low/cell)-1, math.Ceil(high/cell)
	const exactNeighbour = 1 << 52
	if !finite(first) || !finite(last) || first <= -exactNeighbour || last >= exactNeighbour {
		return 0, 0, fmt.Errorf("biome: footprint exceeds the representable lattice range")
	}
	return int64(first), int64(last), nil
}

// latticePoint clamps a rounded draw inside its own half-open cell.
func latticePoint(index int64, cell, draw float64) float64 {
	low, high := float64(index)*cell, float64(index+1)*cell
	point := math.FMA(draw, high-low, low)
	if point >= high {
		return math.Nextafter(high, low)
	}
	return max(low, point)
}
