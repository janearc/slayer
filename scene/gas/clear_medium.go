package gas

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// ClearMedium snapshots this table's RGB extinction into an immutable clear
// boundary medium. The tracer's channels use the same 610, 550 and 465 nm
// samples as EyeOptics. Both absorption and scattering remove energy from the
// unscattered beam. metresPerUnit converts world distances to metres.
func (table VisibleTable) ClearMedium(name string,
	metresPerUnit float64) (*thing.ClearMedium, error) {
	red, err := table.At(610)
	if err != nil {
		return nil, err
	}
	green, err := table.At(550)
	if err != nil {
		return nil, err
	}
	blue, err := table.At(465)
	if err != nil {
		return nil, err
	}
	extinction := maths.RGB(red.Absorb+red.Scatter,
		green.Absorb+green.Scatter, blue.Absorb+blue.Scatter)
	return thing.NewClearMedium(name, extinction, metresPerUnit)
}
