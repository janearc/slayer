package lines

import (
	"math"
	"testing"
)

// The numbers this file counts in.
const (
	// airNanometre is how near, in nanometres, a line in the table must
	// lie to NIST's observed wavelength in air for it. The table gives
	// two places, so it can be half a hundredth off the rounded value;
	// NIST's own are good to a few thousandths. Picked by the test
	// author, 2026-11-17: a hundredth.
	airNanometre = 0.01
	// strongest is the strength of an element's strongest line. The
	// package says strengths are normalized to it. Defined: one.
	strongest = 1.0
)

// reading is one line as the table must have it: the wavelength NIST
// observed, and the strength the table gives it.
type reading struct{ nanometres, strength float64 }

// The wavelengths below are "Observed Wavelength Air (nm)" from the NIST
// Atomic Spectra Database (Kramida, Ralchenko, Reader and NIST ASD Team),
// lines of H I, Ne I and Ar II from 380 to 780 nm, read
// 2026-11-17 from
//
//	https://physics.nist.gov/cgi-bin/ASD/lines1.pl?spectra=H+I
//	https://physics.nist.gov/cgi-bin/ASD/lines1.pl?spectra=Ne+I
//	https://physics.nist.gov/cgi-bin/ASD/lines1.pl?spectra=Ar+II
//
// (low_w=380, upp_w=780, unit=1, show_obs_wl=1). The package's strengths
// are its own approximations, "useful for illustration, not quantitative
// spectroscopy"; NIST's intensities are in arbitrary units of each
// source and are not the package's ratios, so the strengths here are held
// as the package gives them: a change to one is a change to the glow.
var spectra = map[string][]reading{
	// hydrogen's balmer lines, alpha to delta, at a nebula's usual ratios
	"hydrogen": {{656.279, 1}, {486.135, 0.35}, {434.0472, 0.16},
		{410.1734, 0.09}},
	// neon I, 15 lines
	"neon": {{585.24878, 1}, {640.2248, 0.9}, {703.24128, 0.5},
		{614.30627, 0.45}, {588.18950, 0.4}, {594.48340, 0.4},
		{609.61630, 0.35}, {616.35937, 0.3}, {621.72812, 0.3},
		{626.64952, 0.3}, {633.44276, 0.35}, {638.29914, 0.3},
		{650.65277, 0.35}, {659.89528, 0.2}, {692.94672, 0.3}},
	// argon II, 10 lines
	"argon": {{487.98634, 1}, {514.53080, 0.8}, {476.48644, 0.4},
		{496.50794, 0.35}, {434.80635, 0.3}, {465.79009, 0.25},
		{457.93493, 0.2}, {472.68681, 0.2}, {501.71626, 0.15},
		{454.50516, 0.15}},
}

// Every line the package knows for a gas is one NIST observed, at its
// wavelength, with the strength the package gives it; no line is missing
// and none is extra.
func TestEveryKnownLineIsNISTsAtItsStrength(t *testing.T) {
	if len(known) != len(spectra) {
		t.Errorf("the package knows %d gases, NIST's lines are held "+
			"for %d", len(known), len(spectra))
	}
	for name, want := range spectra {
		got := known[name]
		if len(got) != len(want) {
			t.Errorf("%s has %d lines, not %d", name, len(got), len(want))
		}
		for _, line := range want {
			found := false
			for nanometres, strength := range got {
				if math.Abs(nanometres-line.nanometres) > airNanometre {
					continue
				}
				found = true
				if strength != line.strength {
					t.Errorf("%s at %v nm has strength %v, not %v",
						name, nanometres, strength, line.strength)
				}
			}
			if !found {
				t.Errorf("%s has no line within %v nm of NIST's %v nm",
					name, airNanometre, line.nanometres)
			}
		}
	}
}

// No two lines of a gas are one line: each of NIST's matches exactly one
// of the table's, so a table with two lines at one wavelength, or one
// that moved onto another's, is told from the table NIST gives.
func TestNoTwoKnownLinesAreWithinOneNISTLine(t *testing.T) {
	for name, want := range spectra {
		for _, line := range want {
			matches := 0
			for nanometres := range known[name] {
				if math.Abs(nanometres-line.nanometres) <= airNanometre {
					matches++
				}
			}
			if matches != 1 {
				t.Errorf("%s: NIST's %v nm is matched by %d lines",
					name, line.nanometres, matches)
			}
		}
	}
}

// Strengths are normalized to each gas's strongest: that is one, and none
// is above one or at nought or less.
func TestEachGasHasItsStrongestLineAtOneAndNoneOutsideNoughtToOne(
	t *testing.T) {
	for name, held := range known {
		top := 0.0
		for nanometres, strength := range held {
			if !(strength > 0 && strength <= strongest) {
				t.Errorf("%s at %v nm has strength %v", name, nanometres,
					strength)
			}
			top = math.Max(top, strength)
		}
		if top != strongest {
			t.Errorf("%s's strongest line is %v, not %v", name, top,
				strongest)
		}
	}
}
