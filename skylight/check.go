package skylight

import (
	"errors"
	"fmt"
	"math"
)

// refusal is a refusal of this package, held as a constant so that no one
// can assign it.
type refusal string

// Error is the refusal's words.
func (r refusal) Error() string { return string(r) }

// ErrShells is a column with no shells.
const ErrShells refusal = "skylight: a column needs a shell"

// The other refusals Solve gives, each by name.
var (
	// errPhotons is a call with no photons to walk.
	errPhotons = errors.New("skylight: photons must be one or more")
	// errWorkers is a call with no workers to walk them.
	errWorkers = errors.New("skylight: workers must be one or more")
	// errRadius is a planet with no finite, positive radius.
	errRadius = errors.New("skylight: the radius must be finite and " +
		"above nought")
	// errGap is a shell that begins above where the one below ends, or
	// a first shell that begins above the ground.
	errGap = errors.New("skylight: a gap between shells")
	// errOverlap is a shell that begins below where the one below ends.
	errOverlap = errors.New("skylight: shells overlap")
	// errThickness is a shell whose top is not finite and above its
	// bottom.
	errThickness = errors.New("skylight: a shell's top must be finite " +
		"and above its bottom")
	// errCoefficient is a scattering, absorption or extinction that is
	// negative, endless or no number.
	errCoefficient = errors.New("skylight: a coefficient must be finite " +
		"and not negative")
	// errAlbedo is a particle or ground albedo outside nought to one.
	errAlbedo = errors.New("skylight: an albedo must be from nought to one")
	// errAsymmetry is an asymmetry not strictly between minus one and
	// one, where Henyey-Greenstein's phase is no longer a spread of ways.
	errAsymmetry = errors.New("skylight: an asymmetry must be strictly " +
		"between minus one and one")
	// errBins is a grid with fewer than one bin on an axis.
	errBins = errors.New("skylight: each axis needs a bin")
	// errHeight is a grid with no kept heights, or one outside the
	// column, from the ground to the top.
	errHeight = errors.New("skylight: a kept height must be in the column")
	// errOrder is kept heights that do not rise.
	errOrder = errors.New("skylight: kept heights must rise")
	// errSuns is a span of suns not from nought to pi, or whose least
	// is past its most.
	errSuns = errors.New("skylight: a span of suns must run up, from " +
		"nought to pi")
)

// checkSuns is nil for a span of the sun's zenith a walk can be for:
// from nought to pi, its least at or below its most.
func checkSuns(suns Suns) error {
	if !(suns.From >= none && suns.From <= suns.To && suns.To <= math.Pi) {
		return fmt.Errorf("%w: %g to %g", errSuns, suns.From, suns.To)
	}
	return nil
}

// check is nil when Solve can walk the call, and otherwise names the
// first thing wrong with it.
func check(column Column, grid Grid, photons, workers int) error {
	if photons < one {
		return fmt.Errorf("%w: %d", errPhotons, photons)
	}
	if workers < one {
		return fmt.Errorf("%w: %d", errWorkers, workers)
	}
	if err := checkColumn(column); err != nil {
		return err
	}
	last := column.Shells[len(column.Shells)-one]
	return checkGrid(grid, last.Top)
}

// checkColumn is nil when the planet has a radius, the ground an albedo
// and the shells stack from the ground up, each sound.
func checkColumn(column Column) error {
	if !finite(column.Radius) || !(column.Radius > none) {
		return fmt.Errorf("%w: %g m", errRadius, column.Radius)
	}
	if len(column.Shells) == none {
		return ErrShells
	}
	if !(column.GroundAlbedo >= none && column.GroundAlbedo <= one) {
		return fmt.Errorf("%w: the ground's is %g", errAlbedo,
			column.GroundAlbedo)
	}
	below := 0.0
	for index, shell := range column.Shells {
		if err := checkPlace(index, shell, below); err != nil {
			return err
		}
		if err := checkShell(index, shell); err != nil {
			return err
		}
		below = shell.Top
	}
	return nil
}

// checkPlace is nil when the shell at index begins where the one below
// ends, or at the ground, and has a finite top above its bottom.
func checkPlace(index int, shell Shell, below float64) error {
	switch {
	case shell.Bottom > below:
		return fmt.Errorf("%w: shell %d begins at %g m, below ends at %g m",
			errGap, index, shell.Bottom, below)
	case shell.Bottom < below:
		return fmt.Errorf("%w: shell %d begins at %g m, below ends at %g m",
			errOverlap, index, shell.Bottom, below)
	case !finite(shell.Top) || !(shell.Top > shell.Bottom):
		return fmt.Errorf("%w: shell %d from %g m to %g m", errThickness,
			index, shell.Bottom, shell.Top)
	}
	return nil
}

// checkShell is nil when the shell at index has finite coefficients
// that are not negative, an albedo from nought to one and an asymmetry
// strictly between minus one and one.
func checkShell(index int, shell Shell) error {
	for _, coefficient := range []float64{shell.GasScattering,
		shell.GasAbsorption, shell.ParticleExtinction} {
		if !finite(coefficient) || coefficient < none {
			return fmt.Errorf("%w: shell %d has %g", errCoefficient, index,
				coefficient)
		}
	}
	if !(shell.ParticleAlbedo >= none && shell.ParticleAlbedo <= one) {
		return fmt.Errorf("%w: shell %d's particles' is %g", errAlbedo,
			index, shell.ParticleAlbedo)
	}
	if !(math.Abs(shell.ParticleAsymmetry) < one) {
		return fmt.Errorf("%w: shell %d has %g", errAsymmetry, index,
			shell.ParticleAsymmetry)
	}
	return nil
}

// checkGrid is nil when the grid has a bin on each axis and its kept
// heights rise from the ground to no higher than the top.
func checkGrid(grid Grid, top float64) error {
	if min(grid.Zeniths, grid.Views, grid.Azimuths) < one {
		return fmt.Errorf("%w: %d by %d by %d", errBins, grid.Zeniths,
			grid.Views, grid.Azimuths)
	}
	if len(grid.Heights) == none {
		return fmt.Errorf("%w: none is kept", errHeight)
	}
	below := math.Inf(-one)
	for _, height := range grid.Heights {
		if !(height >= none && height <= top) {
			return fmt.Errorf("%w: %g m, the top is %g m", errHeight,
				height, top)
		}
		if !(height > below) {
			return fmt.Errorf("%w: %g m after %g m", errOrder, height, below)
		}
		below = height
	}
	return nil
}

// finite is whether a number is neither endless nor no number.
func finite(number float64) bool {
	return !math.IsInf(number, none) && !math.IsNaN(number)
}
