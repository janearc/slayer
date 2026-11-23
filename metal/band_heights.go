//go:build darwin

package metal

import (
	"fmt"
	"os"
	"time"
	"unsafe"
)

// A band is sized by its work, not by a count of rows. A command is
// taken to cost a fixed time and a time for each row it holds, both read
// from the bands already drawn: the fixed time is the least any band of
// the picture took, and a row's time is what the last band took beyond
// it, shared among its rows.
//
// The next band holds as many rows as fit the
// aim on those terms, never more than mostGrowth times the last band,
// never more than its buffers hold, and never less than one whole row.
//
// A row that cannot meet the aim is drawn whole all the same, and said
// once: a band that cutting cannot make shorter is not cut. The first
// band, with nothing known, is one row.

// aimShare is the share of the kernel's limit a band is sized to take:
// a sixtieth, so a band sized from the cheap rows before it can land on
// rows sixty times heavier and still finish inside the limit. Picked.
const aimShare = 60

// fixedTimes is how many times a command's fixed cost the aim is at the
// least: picked, so at most a quarter of a band's time is the cost every
// command pays before its rows.
const fixedTimes = 4

// mostGrowth is how many times the rows of the last band the next band
// may hold: picked, so a quick band lets the next grow, but one band of
// cheap rows cannot size a band of heavy ones far past them.
const mostGrowth = 2

// environmentAim is SLAYER_BAND_AIM, a duration such as "5s" that every
// band is sized to take where a caller sets none, for experiments; an
// empty or unreadable one leaves the share of the limit. It changes how
// bands are sent, never what they draw.
var environmentAim, _ = time.ParseDuration(os.Getenv("SLAYER_BAND_AIM"))

// aimBands has every grid band sized to take aim on the device in place
// of a share of the kernel's limit, or of SLAYER_BAND_AIM where it is
// nought; nought for both restores the share. Never below fixedTimes a
// command's fixed cost. A picture does not change with its bands.
func (k *Kernel) aimBands(aim time.Duration) {
	k.aim = max(0, aim)
	if k.aim == 0 {
		k.aim = max(0, environmentAim)
	}
}

// bandAim is the device time a band of the kernel is sized to take: its
// own aim where one is set, else a share of its limit.
func (k *Kernel) bandAim() time.Duration {
	if k.aim > 0 {
		return k.aim
	}
	return k.limit / aimShare
}

// span is a band of a picture: down rows from first. No rows is no band.
type span struct{ first, down int }

// drawingOf is the drawing of a picture height rows tall through a
// stream whose band buffers hold most rows: its first band one row, or
// every band the kernel's fixed height.
func (k *Kernel) drawingOf(stream unsafe.Pointer, height,
	most int) bandDrawing {
	drawing := bandDrawing{kernel: k, stream: stream, height: height,
		rows: most, down: 1}
	if k.fixed > 0 {
		drawing.down = min(k.fixed, most)
		drawing.fixed = true
	}
	return drawing
}

// timed sizes the bands after one that took spent on the device, unless
// every band is fixed. A band that took no more than the least is inside
// the aim, which is fixedTimes the least at the least, so the next grows.
func (drawing *bandDrawing) timed(band span, spent time.Duration) {
	if drawing.fixed {
		return
	}
	if drawing.least == 0 || spent < drawing.least {
		drawing.least = spent
	}
	aim := max(drawing.kernel.bandAim(), fixedTimes*drawing.least)
	grown := min(band.down*mostGrowth, drawing.rows)
	perRow := (spent - drawing.least) / time.Duration(band.down)
	fit := grown
	if perRow > 0 {
		fit = int((aim - drawing.least) / perRow)
	}
	drawing.down = max(1, min(fit, grown))
	if fit < 1 {
		drawing.overAim(band, spent, aim)
	}
}

// overAim says, once a drawing, that a row cannot meet the aim and is
// drawn whole all the same.
func (drawing *bandDrawing) overAim(band span, spent, aim time.Duration) {
	if drawing.said {
		return
	}
	drawing.said = true
	fmt.Fprintf(os.Stderr, "metal: rows %d to %d took %v on the device; "+
		"a row is over the aim of %v, so the rest are drawn a row a band\n",
		band.first, band.first+band.down-1, spent.Round(time.Millisecond),
		aim.Round(time.Millisecond))
}

// after is the band that follows one in the picture, as many rows as
// the size says, or no band at the picture's foot.
func (drawing *bandDrawing) after(band span) span {
	from := band.first + band.down
	if from >= drawing.height {
		return span{}
	}
	return span{from, min(drawing.down, drawing.height-from)}
}

// halved is a band the system stopped for the display, to be sent
// again: half its rows, one at least. No band for the rest of the
// picture is taller than that: the stop lowers the most rows a band
// holds, which timed grows no band past, since a band that regrew would
// likely be stopped again.
func (drawing *bandDrawing) halved(band span) span {
	band.down = max(1, band.down/2)
	drawing.rows = min(drawing.rows, band.down)
	drawing.down = min(drawing.down, band.down)
	return band
}
