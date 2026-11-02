// Package film describes film gauges, camera projections, and emitters. It
// defines the recording area and the rays connecting pixels to a scene.
package film

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/number"
)

// what a gauge counts in, each defined
const (
	// none is no grain at all, as a detector has. It is defined as zero.
	none = 0
	// square is the aspect of a frame with no height to speak of. It is
	// defined as one: as wide as it is tall.
	square = 1
)

var errGauge = errors.New("ninjafilm: a gauge has a name, a finite " +
	"width and height above nought in millimetres, and a grain of " +
	"nought or more")

// Gauge is a film's frame. It says how wide and how tall the exposed
// picture is, in millimetres. With a focal length it says how much a
// lens sees, the way a cinematographer says it. On Super 35 a lens of
// 32 is normal, 50 is a little long and 24 is wide.
//
// Its grain is how coarse the film's grain is, as a share of the width
// of the frame. Small gauges are grainy and big ones are clean. Nought
// is no grain at all, as a detector has.
type Gauge struct {
	name                  string
	width, height, grains float64
}

// NewGauge is a gauge of a name and a size in millimetres. Its grain is
// a share of its width. It says why not when it cannot be.
//
//ninja:vow practice slayer's lens makes its gauges here
func NewGauge(name string, width, height, grain float64) (made Gauge,
	err error) {
	if name == "" || !number.Positive(width) || !number.Positive(height) ||
		!(grain >= none && grain <= math.MaxFloat64) {
		return made, errGauge
	}
	return Gauge{name, width, height, grain}, nil
}

// Name is what the gauge is called.
//
//ninja:vow practice slayer's lens names its gauge here
func (gauge Gauge) Name() string { return gauge.name }

// Width is how wide the frame is, in millimetres.
func (gauge Gauge) Width() float64 { return gauge.width }

// Height is how tall the frame is, in millimetres.
//
//ninja:vow practice slayer's lens reads the height here
func (gauge Gauge) Height() float64 { return gauge.height }

// Grain is how coarse the grain is, as a share of the frame's width.
//
//ninja:vow practice slayer's film looks read the grain here
func (gauge Gauge) Grain() float64 { return gauge.grains }

// Grainless is the gauge with its grain taken off. Rolls make their
// own.
func (gauge Gauge) Grainless() Gauge {
	gauge.grains = none
	return gauge
}

// Aspect is how wide the frame is against how tall.
//
//ninja:vow practice slayer's lens reads the aspect here
func (gauge Gauge) Aspect() float64 {
	if !number.Positive(gauge.height) {
		return square
	}
	return gauge.width / gauge.height
}

// Table is the gauges of a table of csv text, one to a row. Tables that
// will not read are said when a gauge is asked for.
//
// The grain of each gauge is the same emulsion's. It is a share of the
// width of the frame that works out to a clump of 46 to 47 microns on
// most. The clump is a look and not a measurement. No published grain
// of any stock is behind it.
type Table struct {
	rows   []data.Row
	unread error
}

// From is the table of gauges that csv text makes.
func From(text string) Table {
	rows, err := data.Parse(text)
	return Table{rows, err}
}

// Named is the gauge of a name in the table. Names not in it are
// refused, and the refusal lists the names that are.
func (table Table) Named(name string) (made Gauge, err error) {
	var called []string
	for _, row := range table.rows {
		each, err := row.Text("name")
		if err != nil {
			return made, err
		}
		if each == name {
			return gaugeOf(row, name)
		}
		called = append(called, each)
	}
	slices.Sort(called)
	return made, errors.Join(table.unread, fmt.Errorf("ninjafilm: no "+
		"gauge is called %q; the table holds %q", name, called))
}

// Must is the gauge of a name in the table. It serves the gauges the
// code leans on. Tables without them are mistakes in the code. They
// stop the program.
//
//ninja:vow practice the lens leans on two gauges; tests do the rest
func (table Table) Must(name string) Gauge {
	made, err := table.Named(name)
	if err != nil {
		panic(err)
	}
	return made
}

// gaugeOf is the gauge of a name that a row of the table of gauges
// holds.
func gaugeOf(row data.Row, name string) (made Gauge, err error) {
	width, err := row.Float("width_mm")
	if err != nil {
		return made, err
	}
	height, err := row.Float("height_mm")
	if err != nil {
		return made, err
	}
	grain, err := row.Float("grain")
	if err != nil {
		return made, err
	}
	return NewGauge(name, width, height, grain)
}

// Shipped is the table of gauges that ships with ninjafilm.
//
//ninja:vow practice slayer reads the gauges here; tests so far
func Shipped() Table { return From(data.GaugesTable()) }
