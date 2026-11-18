package airs

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// sunHeader is how the spectrum's file begins: a line naming its first
// two columns, the wavelength in nanometres and the irradiance in watts
// a square metre a nanometre. The file goes on with the uncertainty and
// the bandwidth, which are not used.
//
// The file carries no version line, no date and no distance. That it is
// the TSIS-1 hybrid solar reference spectrum, at one astronomical unit,
// on a 0.025 nm grid at 0.1 nm bandwidth, is from where it was fetched
// (the laboratory for atmospheric and space physics), not from the file.
const sunHeader = "wavelength (nm),irradiance (W/m^2/nm)"

// SunSource is the key the sun's spectrum is pinned by in sources.yaml:
// the TSIS-1 hybrid solar reference spectrum on its 0.025 nm grid. slayer
// keeps no measured data in its tree; a run reads the file from the
// shelf, held to its pin, and gives it to ReadSun.
const SunSource = "tsis1-hsrs-p1nm"

// Sun is the sun's spectrum above any air at one astronomical unit: the
// irradiance, in watts a square metre a nanometre, at each wavelength in
// nanometres, ascending.
type Sun struct {
	wavelengths []float64
	irradiance  []float64
}

// ReadSun reads the spectrum from a comma-separated file in the form of
// the TSIS-1 hybrid spectrum's: the header sunHeader, then a line for
// each wavelength with the irradiance after it. It refuses a file that
// begins otherwise, a line that is not two numbers or more, an
// irradiance below nought, wavelengths that do not rise, and fewer than
// two lines; and it gives the error the reader gave.
func ReadSun(reader io.Reader) (Sun, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() || !strings.HasPrefix(scanner.Text(), sunHeader) {
		return Sun{}, fmt.Errorf("airs: the spectrum does not begin with %q",
			sunHeader)
	}
	var sun Sun
	for line := 2; scanner.Scan(); line++ {
		wavelength, irradiance, err := sample(scanner.Text())
		if err != nil {
			return Sun{}, fmt.Errorf("airs: the spectrum's line %d: %w", line, err)
		}
		if last := len(sun.wavelengths); last > 0 &&
			!(wavelength > sun.wavelengths[last-1]) {
			return Sun{}, fmt.Errorf("airs: the spectrum's line %d is at %g nm, "+
				"not after the line before", line, wavelength)
		}
		sun.wavelengths = append(sun.wavelengths, wavelength)
		sun.irradiance = append(sun.irradiance, irradiance)
	}
	if err := scanner.Err(); err != nil {
		return Sun{}, err
	}
	if len(sun.wavelengths) < 2 {
		return Sun{}, fmt.Errorf("airs: the spectrum has %d lines of data, "+
			"and a band needs two", len(sun.wavelengths))
	}
	return sun, nil
}

// sample is the wavelength and irradiance of a line of the file: its
// first two fields.
func sample(line string) (float64, float64, error) {
	first, rest, found := strings.Cut(line, ",")
	if !found {
		return 0, 0, fmt.Errorf("%q has no irradiance after its wavelength", line)
	}
	second, _, _ := strings.Cut(rest, ",")
	wavelength, err := strconv.ParseFloat(first, 64)
	if err != nil {
		return 0, 0, err
	}
	irradiance, err := strconv.ParseFloat(second, 64)
	if err != nil {
		return 0, 0, err
	}
	if !(wavelength > 0 && irradiance >= 0) {
		return 0, 0, fmt.Errorf("%q is not a wavelength above nought and an "+
			"irradiance of nought or more", line)
	}
	return wavelength, irradiance, nil
}

// Band is the sun's irradiance at one astronomical unit averaged over a
// band, from low to high in nanometres, ends included, in watts a square
// metre a nanometre. It is the mean of the file's samples in the band,
// which lie evenly, so over the band's width it is the integral. It
// refuses a band that runs backwards, runs past the file's ends, or has
// no sample in it.
func (sun Sun) Band(low, high float64) (float64, error) {
	last := len(sun.wavelengths) - 1
	if !(low <= high && low >= sun.wavelengths[0] && high <= sun.wavelengths[last]) {
		return 0, fmt.Errorf("airs: the band %g to %g nm is backwards or "+
			"outside the spectrum's %g to %g nm", low, high,
			sun.wavelengths[0], sun.wavelengths[last])
	}
	first, _ := slices.BinarySearch(sun.wavelengths, low)
	end, found := slices.BinarySearch(sun.wavelengths, high)
	if found {
		end++
	}
	if first >= end {
		return 0, fmt.Errorf("airs: the band %g to %g nm has no sample in it",
			low, high)
	}
	sum := 0.0
	for _, each := range sun.irradiance[first:end] {
		sum += each
	}
	return sum / float64(end-first), nil
}

// AtVenus is the sun's irradiance at Venus's mean distance, averaged over
// a band: Band's, over the square of the distance in astronomical units.
// The distance is the data's, to be pinned.
func (sun Sun) AtVenus(low, high float64) (float64, error) {
	if venusError != nil {
		return 0, venusError
	}
	near, err := sun.Band(low, high)
	if err != nil {
		return 0, err
	}
	distance := venusRead.Planet.Distance.Value
	return near / (distance * distance), nil
}
