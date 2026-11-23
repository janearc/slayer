//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
)

// sentRows is how many rows the sent checks draw, a band of one row each,
// so that there is more than one band for the call to come before.
const sentRows = 3

// the two things a draw tells the checks, in the order it tells them.
const (
	toldSent = "sent"
	toldBand = "band"
)

// sentInputs is the one input of the spin kernel, over at once.
func sentInputs() [][]byte {
	return [][]byte{binary.LittleEndian.AppendUint32(nil, spinQuick)}
}

// sentFaults is what is wrong with what a draw told, in order, and the
// rows its bands held: that every input is on the device is told once,
// before any band, and the bands hold every row.
func sentFaults(told []string, rows int) []string {
	var faults []string
	sent := 0
	for _, each := range told {
		switch {
		case each == toldSent:
			sent++
		case sent == 0:
			faults = append(faults, "a band was read before the call")
		}
	}
	if sent != 1 {
		faults = append(faults, fmt.Sprintf("the call came %d times, "+
			"want once", sent))
	}
	if rows != sentRows {
		faults = append(faults, fmt.Sprintf("%d rows were drawn, want %d",
			rows, sentRows))
	}
	return faults
}

// TestGridBandsSentSaysSentOnceBeforeTheFirstBand draws bands of one row
// and notes the order of what it is told: the call that every input is on
// the device comes once, and before any band is read.
func TestGridBandsSentSaysSentOnceBeforeTheFirstBand(t *testing.T) {
	kernel, err := Open(spinProbe, "spin")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	var told []string
	var rows int
	err = kernel.GridBandsSent(1, sentRows, 1, 1, sentInputs(),
		func() { told = append(told, toldSent) },
		func(_, down int, _ []float32) error {
			told = append(told, toldBand)
			rows += down
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range sentFaults(told, rows) {
		t.Error(fault)
	}
}

// Anti-test: a second call before the first band is reported, and so is
// a band read before any call.
func TestASecondSendOrABandBeforeTheSendIsReported(t *testing.T) {
	twice := []string{toldSent, toldSent, toldBand, toldBand, toldBand}
	if faults := sentFaults(twice, sentRows); len(faults) != 1 {
		t.Errorf("a second send gave %v, want one fault", faults)
	}
	early := []string{toldBand, toldSent, toldBand, toldBand}
	if faults := sentFaults(early, sentRows); len(faults) != 1 {
		t.Errorf("a band before the send gave %v, want one fault", faults)
	}
}

// drawnFaults is what is wrong with a draw that was told of no call: it
// drew every row and gave no error.
func drawnFaults(rows int, err error) []string {
	var faults []string
	if err != nil {
		faults = append(faults, fmt.Sprintf("the draw gave %v", err))
	}
	if rows != sentRows {
		faults = append(faults, fmt.Sprintf("%d rows were drawn, want %d",
			rows, sentRows))
	}
	return faults
}

// With no call to make the rows are drawn as GridBands draws them.
func TestGridBandsSentOfNoCallDrawsAsGridBandsDoes(t *testing.T) {
	kernel, err := Open(spinProbe, "spin")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	var rows int
	err = kernel.GridBandsSent(1, sentRows, 1, 1, sentInputs(), nil,
		func(_, down int, _ []float32) error {
			rows += down
			return nil
		})
	for _, fault := range drawnFaults(rows, err) {
		t.Error(fault)
	}
}

// Anti-test: a draw that stopped short of its rows, with an error, is
// reported both ways.
func TestADrawThatStoppedShortIsReported(t *testing.T) {
	stopped := errors.New("stopped")
	if faults := drawnFaults(sentRows-1, stopped); len(faults) != 2 {
		t.Errorf("a draw stopped short gave %v, want two faults", faults)
	}
}
