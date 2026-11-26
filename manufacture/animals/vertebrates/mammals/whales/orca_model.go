package whales

import (
	"fmt"
	"math"
	"sync"

	"github.com/janearc/slayer/bioforms/skeleton"
)

// orcaModel owns one immutable set of authored trunk stations and the
// expensive geometry derived from it. A snapshot never changes another
// profile's field, skeleton, skin, binding, or paint.
type orcaModel struct {
	stations []orcaStation

	trunkOnce sync.Once
	trunk     *orcaTrunkTable

	skeletonOnce sync.Once
	skeleton     *skeleton.Skeleton
	skeletonErr  error

	restOnce sync.Once
	rest     *orcaRestSkin

	bindingOnce sync.Once
	binding     [][]skeleton.Influence
	bindingErr  error
}

// newOrcaModel validates the authored stations and the interpolated trunk
// against the fixed meshing lattice. It owns its copy of the source data.
func newOrcaModel(rows [][4]float64) (*orcaModel, error) {
	if len(rows) < 3 || len(rows) > 128 {
		return nil, fmt.Errorf("orca: trunk needs 3 to 128 stations")
	}
	model := &orcaModel{stations: make([]orcaStation, len(rows))}
	// Leave a cell between the trunk and each lattice edge so the contour
	// closes instead of being silently cut off by the meshing domain.
	lowX := orcaLatticeLowX + orcaLatticeStep
	highX := orcaLatticeLowX + float64(orcaLatticeNX-2)*orcaLatticeStep
	lowY := orcaLatticeLowY + orcaLatticeStep
	highY := orcaLatticeLowY + float64(orcaLatticeNY-2)*orcaLatticeStep
	halfWidth := math.Min(-orcaLatticeLowZ, orcaLatticeLowZ+float64(orcaLatticeNZ-1)*orcaLatticeStep) - orcaLatticeStep
	for i, row := range rows {
		for _, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("orca: station %d is not finite", i)
			}
		}
		if row[1] <= 0 || row[2] <= 0 || row[0] <= lowX || row[0] >= highX ||
			(i > 0 && row[0] <= rows[i-1][0]) {
			return nil, fmt.Errorf("orca: invalid trunk station %d", i)
		}
		model.stations[i] = orcaStation{x: row[0], high: row[1], wide: row[2], centre: row[3]}
	}
	if rows[0][0] > orcaColumnEnd || rows[len(rows)-1][0] < orcaMouthFront {
		return nil, fmt.Errorf("orca: trunk does not span the skeleton and mouth")
	}
	for i, station := range model.orcaTrunkProfile().entries {
		if !(station.high > 0 && station.wide > 0 && station.wide < halfWidth &&
			station.centre-station.high > lowY && station.centre+station.high < highY) {
			return nil, fmt.Errorf("orca: interpolated trunk station %d exceeds its lattice", i)
		}
	}
	eye := model.orcaTrunkProfile().at(orcaEyeX)
	if math.Abs(orcaEyeY-eye.centre) >= eye.high {
		return nil, fmt.Errorf("orca: eye lies outside the authored trunk")
	}
	return model, nil
}
