//go:build darwin

package metal

import (
	"testing"
	"time"
)

// the sizing fixture: a picture of fifty rows, band buffers of eight
const sizedHeight, sizedMost = 50, 8

// sizedDrawing is the fixture's drawing under the default limit, as
// drawingOf makes one before its first band, without a device.
func sizedDrawing() *bandDrawing {
	kernel := &Kernel{limit: defaultLimit}
	drawing := kernel.drawingOf(nil, sizedHeight, sizedMost)
	return &drawing
}

// the aim is a sixtieth of the limit, or the kernel's own where it has
// one.
func TestTheAimIsAShareOfTheLimit(t *testing.T) {
	kernel := &Kernel{limit: 300 * time.Millisecond}
	if got := kernel.bandAim(); got != 5*time.Millisecond {
		t.Errorf("the aim under a limit of 300 ms is %v; want 5ms", got)
	}
	kernel.aimBands(time.Second)
	if got := kernel.bandAim(); got != time.Second {
		t.Errorf("the aim set to a second is %v", got)
	}
}

// with nothing known, the first band is one row.
func TestTheFirstBandIsOneRow(t *testing.T) {
	if got, want := sizedDrawing().after(span{}), (span{0, 1}); got != want {
		t.Errorf("the first band is %+v; want %+v", got, want)
	}
}

// a band inside the aim lets the next hold twice its rows, and no more.
func TestAQuickBandLetsTheNextGrow(t *testing.T) {
	drawing := sizedDrawing()
	drawing.timed(span{0, 2}, time.Millisecond)
	if got, want := drawing.after(span{0, 2}), (span{2, 4}); got != want {
		t.Errorf("after a quick band of two rows came %+v; want %+v",
			got, want)
	}
}

// however quick, no band holds more rows than its buffers.
func TestNoBandOutgrowsItsBuffers(t *testing.T) {
	drawing := sizedDrawing()
	drawing.timed(span{0, sizedMost}, time.Millisecond)
	want := span{sizedMost, sizedMost}
	if got := drawing.after(span{0, sizedMost}); got != want {
		t.Errorf("after a quick full band came %+v; want %+v", got, want)
	}
}

// a band of eight rows that took a second and seven seconds a row makes
// the next as tall as fits the aim of thirty seconds: four rows.
func TestASlowBandMakesTheNextFitTheAim(t *testing.T) {
	drawing := sizedDrawing()
	drawing.timed(span{0, 1}, time.Second)
	drawing.timed(span{1, 8}, time.Second+8*7*time.Second)
	if got, want := drawing.after(span{1, 8}), (span{9, 4}); got != want {
		t.Errorf("after eight rows at seven seconds came %+v; want %+v",
			got, want)
	}
}

// a row that cannot meet the aim is drawn whole, one row a band, and
// said once.
func TestARowOverTheAimIsDrawnWhole(t *testing.T) {
	drawing := sizedDrawing()
	drawing.timed(span{0, 1}, 10*time.Second)
	drawing.timed(span{1, 1}, 70*time.Second)
	if got, want := drawing.after(span{1, 1}), (span{2, 1}); got != want {
		t.Errorf("after a row of seventy seconds came %+v; want %+v",
			got, want)
	}
	if !drawing.said {
		t.Error("a row over the aim was not said")
	}
}

// the aim is never under four times the least a band took: bands whose
// least is twenty seconds aim at eighty, not thirty.
func TestTheAimIsNeverUnderFourFixedCosts(t *testing.T) {
	kernel := &Kernel{limit: defaultLimit}
	drawing := kernel.drawingOf(nil, 500, 64)
	drawing.timed(span{0, 1}, 20*time.Second)
	drawing.timed(span{1, 32}, 20*time.Second+32*time.Second)
	if got, want := drawing.after(span{1, 32}), (span{33, 60}); got != want {
		t.Errorf("after 32 rows of a second over twenty came %+v; want %+v",
			got, want)
	}
}

// fixed bands keep their height whatever their time.
func TestAFixedBandKeepsItsHeight(t *testing.T) {
	kernel := &Kernel{limit: defaultLimit, fixed: 3}
	drawing := kernel.drawingOf(nil, sizedHeight, sizedMost)
	drawing.timed(span{0, 3}, time.Hour)
	if got, want := drawing.after(span{0, 3}), (span{3, 3}); got != want {
		t.Errorf("after a slow fixed band came %+v; want %+v", got, want)
	}
}

// the last band is cut at the picture's foot, and after it is no band.
func TestTheLastBandStopsAtTheFoot(t *testing.T) {
	drawing := sizedDrawing()
	drawing.down = sizedMost
	last := drawing.after(span{0, sizedHeight - 3})
	if want := (span{sizedHeight - 3, 3}); last != want {
		t.Errorf("the band at the foot is %+v; want %+v", last, want)
	}
	if after := drawing.after(last); after.down != 0 {
		t.Errorf("after the foot came %+v; want no band", after)
	}
}

// a band stopped for the display is sent again half as tall, a row
// stays a row, and the bands after are no taller.
func TestAStopHalvesTheRows(t *testing.T) {
	drawing := sizedDrawing()
	drawing.down = sizedMost
	if got, want := drawing.halved(span{6, 4}), (span{6, 2}); got != want {
		t.Errorf("four rows stopped are sent again as %+v; want %+v",
			got, want)
	}
	if got, want := drawing.halved(span{6, 1}), (span{6, 1}); got != want {
		t.Errorf("a row stopped is sent again as %+v; want %+v", got, want)
	}
	if drawing.down != 1 {
		t.Errorf("after the stops bands are %d rows; want 1", drawing.down)
	}
}

// after a stop, a quick band does not grow the next past the rows the
// stopped band was sent again as, for the rest of the picture.
func TestAStoppedBandDoesNotGrowBack(t *testing.T) {
	drawing := sizedDrawing()
	drawing.down = sizedMost
	again := drawing.halved(span{6, 4})
	drawing.timed(again, time.Millisecond)
	if got, want := drawing.after(again), (span{8, 2}); got != want {
		t.Errorf("after a quick band of two rows sent again came %+v; "+
			"want %+v", got, want)
	}
	drawing.timed(span{8, 2}, time.Millisecond)
	if got := drawing.after(span{8, 2}); got.down > again.down {
		t.Errorf("a later band grew to %d rows past the stop's %d",
			got.down, again.down)
	}
}
