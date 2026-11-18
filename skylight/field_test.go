package skylight

import (
	"math"
	"testing"
)

// The hand-made field the reading is checked on.
const (
	readBins     = 2     // picked: two bins on each axis, two centres
	readHigh     = 1000  // picked: the second kept height, in metres
	readNear     = 900   // picked: nearer the second height
	readLow      = 400   // picked: nearer the first
	readShare    = 1e-12 // picked: a few roundings of the values read
	firstQuarter = 0.25  // defined: the first of two bins' centres
)

// handField is a field of two kept heights and two bins on each axis,
// whose radiance in each bin is its index and whose count is twice it.
func handField() *Field {
	field := &Field{shape: shape{readBins, readBins, readBins},
		heights: []float64{none, readHigh}}
	size := len(field.heights) * readBins * readBins * readBins
	for index := range size {
		field.radiance = append(field.radiance, float64(index))
		field.counts = append(field.counts, int64(two*index))
	}
	return field
}

// a reading is a place and look, and the radiance it should read.
type reading struct {
	name                          string
	height, zenith, view, azimuth float64
	want                          float64
}

// readings are places on the hand field: bin centres, halfway between
// them, past the last centre, at a folded azimuth, and at the nearer of
// two heights. A bin's index is ((height 2 + zenith) 2 + view) 2 +
// azimuth.
func readings() []reading {
	zenith := spanOf(none, math.Pi, none, readBins)
	centre := (zenith.low + zenith.high) / two
	view, azimuth := -half, firstQuarter*math.Pi
	return []reading{
		{"a centre", none, centre, view, azimuth, none},
		{"the zeniths' middle", none, math.Pi / two, view, azimuth, two},
		{"past the last view", none, centre, one, azimuth, two},
		{"an azimuth folded", none, centre, view, -azimuth, none},
		{"an azimuth a turn on", none, centre, view, azimuth + fullTurn,
			none},
		{"the views' middle", none, centre, none, azimuth, one},
		{"the nearer height", readNear, centre, view, azimuth,
			two * two * two},
	}
}

// the field reads its bins at their centres, between centres in
// proportion, the end bins' values past them, and the nearest kept
// height; and counts the crossings behind the bin a look falls in.
func TestTheFieldReadsBetweenCentres(t *testing.T) {
	field := handField()
	for _, each := range readings() {
		got := field.Radiance(each.height, each.zenith, each.view,
			each.azimuth)
		if !(math.Abs(got-each.want) <= readShare) {
			t.Errorf("%s: %g, want %g", each.name, got, each.want)
		}
	}
	last := readings()[len(readings())-one]
	if got := field.Count(last.height, last.zenith, last.view,
		last.azimuth); got != two*two*two*two {
		t.Errorf("the nearer height counts %d", got)
	}
}

// the anti-test: a look off the bins' centres does not read its own
// bin's value, and a height nearer the ground does not read the higher
// kept height, so the check would catch a field read by nearest bin.
func TestTheFieldIsNotReadByNearestBin(t *testing.T) {
	field := handField()
	off := spanOf(none, math.Pi, none, readBins).high * (one - firstQuarter)
	inside := field.Radiance(none, off, -half, firstQuarter*math.Pi)
	if math.Abs(inside-none) <= readShare {
		t.Errorf("a look off the centre reads its bin: %g", inside)
	}
	low := field.Radiance(readLow, firstQuarter*math.Pi, -half,
		firstQuarter*math.Pi)
	if math.Abs(low-two*two*two) <= readShare {
		t.Errorf("a height nearer the ground reads the higher one")
	}
}
