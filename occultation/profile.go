// Package occultation reads ring optical-depth profiles published by the PDS
// Ring-Moon Systems Node. Stellar occultations measure normal optical depth as
// a function of radius; callers can query a radial interval.
package occultation

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// opaque is the depth read where the star was gone, which the tables
// mark 99. it is deeper than the photometer could see, so at least the
// deepest it measured anywhere: 5.6 in voyager 2's delta sco profile at
// 1 km. read as 6, picked.
const opaque = 6

// A Profile is a ring's normal optical depth outward from an inner
// radius, in bins of one width in metres, each the mean over its bin.
// it keeps running sums of the depth and of the light kept straight
// through, so any stretch reads in the same time.
type Profile struct {
	inner, step float64
	depths      []float64
	summed      []float64
	kept        []float64
}

// Read is a profile from a ring profile table: a row a bin, its middle's radius
// in km, the star's signal and its uncertainty, the median normal optical depth
// and its lower and upper limits, by commas.
//
// a depth of 99 is opaque. a depth less than zero is noise in an empty gap,
// read as zero. a row with no depth, its signal zero and its uncertainty 99, is
// refused.
func Read(table io.Reader) (Profile, error) {
	scanner := bufio.NewScanner(table)
	var radii, depths []float64
	for row := 1; scanner.Scan(); row++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		radius, depth, err := parse(line)
		if err != nil {
			return Profile{}, fmt.Errorf("row %d: %w", row, err)
		}
		radii, depths = append(radii, radius), append(depths, depth)
	}
	if err := scanner.Err(); err != nil {
		return Profile{}, err
	}
	return binned(radii, depths)
}

// parse is a row's radius in metres and its depth.
func parse(line string) (radius, depth float64, err error) {
	fields := strings.Split(line, ",")
	if len(fields) != 6 {
		return 0, 0, fmt.Errorf("%d fields, want 6", len(fields))
	}
	var read [6]float64
	for k, field := range fields {
		read[k], err = strconv.ParseFloat(strings.TrimSpace(field), 64)
		if err != nil {
			return 0, 0, err
		}
	}
	switch {
	case read[3] >= 99:
		depth = opaque
	case read[1] == 0 && read[2] >= 99:
		return 0, 0, fmt.Errorf("no depth at %g km", read[0])
	default:
		depth = math.Max(0, read[3])
	}
	return read[0] * 1e3, depth, nil
}

// binned is a profile from its bins' middles, evenly spaced and rising,
// and their depths.
func binned(radii, depths []float64) (Profile, error) {
	if len(radii) < 2 {
		return Profile{}, fmt.Errorf("%d bins, want two or more", len(radii))
	}
	step := radii[1] - radii[0]
	for k := 1; k < len(radii); k++ {
		gap := radii[k] - radii[k-1]
		if !(step > 0) || math.Abs(gap-step) > 1e-6*step {
			return Profile{}, fmt.Errorf("bins %g m apart at %g m, not %g",
				gap, radii[k], step)
		}
	}
	return made(radii[0]-step/2, step, depths), nil
}

// made is a profile and its running sums.
func made(inner, step float64, depths []float64) Profile {
	summed := make([]float64, len(depths)+1)
	kept := make([]float64, len(depths)+1)
	for k, depth := range depths {
		summed[k+1] = summed[k] + depth*step
		kept[k+1] = kept[k] + math.Exp(-depth)*step
	}
	return Profile{inner: inner, step: step, depths: depths, summed: summed,
		kept: kept}
}

// Joined is profiles end to end, each starting where the one before it
// ends, in bins of one width.
func Joined(parts ...Profile) (Profile, error) {
	if len(parts) == 0 {
		return Profile{}, fmt.Errorf("no profiles to join")
	}
	depths := append([]float64(nil), parts[0].depths...)
	for k := 1; k < len(parts); k++ {
		was, next := parts[k-1], parts[k]
		if math.Abs(next.step-was.step) > 1e-9*was.step ||
			math.Abs(next.inner-was.Outer()) > 1e-6*was.step {
			return Profile{}, fmt.Errorf("profile %d starts at %g m in %g m "+
				"bins, after one ending at %g m in %g m bins", k, next.inner,
				next.step, was.Outer(), was.step)
		}
		depths = append(depths, next.depths...)
	}
	return made(parts[0].inner, parts[0].step, depths), nil
}

// Inner is the radius in metres the profile starts at.
func (profile Profile) Inner() float64 { return profile.inner }

// Outer is the radius in metres the profile ends at.
func (profile Profile) Outer() float64 {
	return profile.inner + profile.step*float64(len(profile.depths))
}

// Mean is the mean depth from one radius to another, in metres.
func (profile Profile) Mean(from, to float64) float64 {
	from, to = profile.within(from), profile.within(to)
	if !(to > from) {
		return 0
	}
	return (profile.under(profile.summed, to) -
		profile.under(profile.summed, from)) / (to - from)
}

// Seen is the depth across so many metres of radius about a radius, as light
// sees it: the depth that keeps, straight through, the mean of what each place
// across it keeps. a gap beside an opaque band lets its light through, where
// their mean depth would not.
//
// across zero metres it is the bin's own depth; outside the profile, zero.
func (profile Profile) Seen(radius, across float64) float64 {
	if radius < profile.inner || radius >= profile.Outer() {
		return 0
	}
	from := profile.within(radius - across/2)
	to := profile.within(radius + across/2)
	if !(to > from) {
		return profile.depths[profile.bin(radius)]
	}
	mean := (profile.under(profile.kept, to) -
		profile.under(profile.kept, from)) / (to - from)
	return -math.Log(mean)
}

// under is a running sum read at a radius inside the profile, straight
// across the bin it falls in.
func (profile Profile) under(sums []float64, radius float64) float64 {
	k := profile.bin(radius)
	part := radius - profile.inner - float64(k)*profile.step
	return sums[k] + (sums[k+1]-sums[k])*part/profile.step
}

// bin is the bin a radius inside the profile falls in.
func (profile Profile) bin(radius float64) int {
	k := int(math.Floor((radius - profile.inner) / profile.step))
	return max(0, min(k, len(profile.depths)-1))
}

// within is a radius brought inside the profile.
func (profile Profile) within(radius float64) float64 {
	return math.Max(profile.inner, math.Min(profile.Outer(), radius))
}
