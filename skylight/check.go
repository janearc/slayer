package skylight

import (
	"errors"
	"fmt"
	"math"
)

// The refusals Solve gives, each by name.
var (
	// ErrPhotons is a call with no photons to walk.
	ErrPhotons = errors.New("skylight: photons must be one or more")
	// ErrWorkers is a call with no workers to walk them.
	ErrWorkers = errors.New("skylight: workers must be one or more")
	// ErrRadius is a planet with no finite, positive radius.
	ErrRadius = errors.New("skylight: the radius must be finite and " +
		"above nought")
	// ErrShells is a column with no shells.
	ErrShells = errors.New("skylight: a column needs a shell")
	// ErrGap is a shell that begins above where the one below ends, or
	// a first shell that begins above the ground.
	ErrGap = errors.New("skylight: a gap between shells")
	// ErrOverlap is a shell that begins below where the one below ends.
	ErrOverlap = errors.New("skylight: shells overlap")
	// ErrThickness is a shell whose top is not finite and above its
	// bottom.
	ErrThickness = errors.New("skylight: a shell's top must be finite " +
		"and above its bottom")
	// ErrCoefficient is a scattering, absorption or extinction that is
	// negative, endless or no number.
	ErrCoefficient = errors.New("skylight: a coefficient must be finite " +
		"and not negative")
	// ErrAlbedo is a particle or ground albedo outside nought to one.
	ErrAlbedo = errors.New("skylight: an albedo must be from nought to one")
	// ErrAsymmetry is an asymmetry not strictly between minus one and
	// one, where Henyey-Greenstein's phase is no longer a spread of ways.
	ErrAsymmetry = errors.New("skylight: an asymmetry must be strictly " +
		"between minus one and one")
	// ErrBins is a grid with fewer than one bin on an axis.
	ErrBins = errors.New("skylight: each axis needs a bin")
	// ErrHeight is a grid with no kept heights, or one outside the
	// column, from the ground to the top.
	ErrHeight = errors.New("skylight: a kept height must be in the column")
	// ErrOrder is kept heights that do not rise.
	ErrOrder = errors.New("skylight: kept heights must rise")
)

// check is nil when Solve can walk the call, and otherwise names the
// first thing wrong with it.
func check(column Column, grid Grid, photons, workers int) error {
	if photons < one {
		return fmt.Errorf("%w: %d", ErrPhotons, photons)
	}
	if workers < one {
		return fmt.Errorf("%w: %d", ErrWorkers, workers)
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
		return fmt.Errorf("%w: %g m", ErrRadius, column.Radius)
	}
	if len(column.Shells) == none {
		return ErrShells
	}
	if !(column.GroundAlbedo >= none && column.GroundAlbedo <= one) {
		return fmt.Errorf("%w: the ground's is %g", ErrAlbedo,
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
			ErrGap, index, shell.Bottom, below)
	case shell.Bottom < below:
		return fmt.Errorf("%w: shell %d begins at %g m, below ends at %g m",
			ErrOverlap, index, shell.Bottom, below)
	case !finite(shell.Top) || !(shell.Top > shell.Bottom):
		return fmt.Errorf("%w: shell %d from %g m to %g m", ErrThickness,
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
			return fmt.Errorf("%w: shell %d has %g", ErrCoefficient, index,
				coefficient)
		}
	}
	if !(shell.ParticleAlbedo >= none && shell.ParticleAlbedo <= one) {
		return fmt.Errorf("%w: shell %d's particles' is %g", ErrAlbedo,
			index, shell.ParticleAlbedo)
	}
	if !(math.Abs(shell.ParticleAsymmetry) < one) {
		return fmt.Errorf("%w: shell %d has %g", ErrAsymmetry, index,
			shell.ParticleAsymmetry)
	}
	return nil
}

// checkGrid is nil when the grid has a bin on each axis and its kept
// heights rise from the ground to no higher than the top.
func checkGrid(grid Grid, top float64) error {
	if min(grid.Zeniths, grid.Views, grid.Azimuths) < one {
		return fmt.Errorf("%w: %d by %d by %d", ErrBins, grid.Zeniths,
			grid.Views, grid.Azimuths)
	}
	if len(grid.Heights) == none {
		return fmt.Errorf("%w: none is kept", ErrHeight)
	}
	below := math.Inf(-one)
	for _, height := range grid.Heights {
		if !(height >= none && height <= top) {
			return fmt.Errorf("%w: %g m, the top is %g m", ErrHeight,
				height, top)
		}
		if !(height > below) {
			return fmt.Errorf("%w: %g m after %g m", ErrOrder, height, below)
		}
		below = height
	}
	return nil
}

// finite is whether a number is neither endless nor no number.
func finite(number float64) bool {
	return !math.IsInf(number, none) && !math.IsNaN(number)
}
