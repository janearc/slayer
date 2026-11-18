//go:build darwin

package metal

/*
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"os"
	"time"
)

// bandReceipt is the environment variable that, set to anything, has
// every band of every grid say its rows and its time on the device on
// standard error. It is how a picture's slowest command is found and
// measured against the limit, without changing what is drawn.
const bandReceipt = "SLAYER_BAND_TIMES"

// receiptsWanted is whether bands say their times, read once a process.
var receiptsWanted = os.Getenv(bandReceipt) != ""

// receipt says one band's rows, its time on the device and whether
// someone was using the machine, when asked for.
func receipt(first, down int, spent time.Duration) {
	if !receiptsWanted {
		return
	}
	fmt.Fprintf(os.Stderr, "metal band rows %d to %d: %v on the device; "+
		"%s\n", first, first+down-1, spent.Round(time.Microsecond),
		presence())
}

// stopReceipt says, when asked for, that the system stopped a band for
// the display, whether someone was using the machine, and that its rows
// are sent again again rows tall.
func stopReceipt(first, again int, why error) {
	if !receiptsWanted {
		return
	}
	fmt.Fprintf(os.Stderr,
		"metal band from row %d stopped for the display (%v); %s; "+
			"sent again %d rows tall\n", first, why, presence(), again)
}

// presence is the time since the machine's last input and whether its
// main display sleeps, read and never touched: the interactivity the
// system protects when it stops a band.
func presence() string {
	idle := "not read"
	if nanoseconds := int64(C.slayer_idle_nanoseconds()); nanoseconds >= 0 {
		idle = time.Duration(nanoseconds).Round(time.Millisecond).String()
	}
	return fmt.Sprintf("input idle %s, display asleep %t", idle,
		C.slayer_display_asleep() != 0)
}
