//go:build darwin

package metal

import (
	"os"
	"strconv"
	"time"
)

// A grid is drawn in bands, one command each. On Apple Silicon the window
// server draws the display on the same gpu, and the system stops a
// background command that keeps the display from presenting. Such a
// command was stopped cleanly: the device is fine.
//
// So a band the system stops for the display is sent again, half as tall,
// a row at least, after the display has had a turn. Bands are sized by
// their time (see band_heights.go), so the kernel's limit is met only by
// a device that has hung. A pixel's light never depends on its band: the
// kernels address samples by their whole rows.

// stopTurn is the wait after the system stopped a band for the display,
// before the same rows are sent again: several frames of the display, so
// it has caught up.
const stopTurn = 100 * time.Millisecond

// mostStops is how many times one band's rows may be stopped for the
// display, each time sent again in half the height, a row at least,
// before the drawing fails: six halvings take a band of 64 rows to one,
// and that row may be sent twice more.
const mostStops = 8

// environmentBands is SLAYER_BANDS, a height every band is drawn at
// where a caller fixes none, for experiments; nought leaves the bands as
// they are. It changes how bands are sent, never what they draw.
var environmentBands, _ = strconv.Atoi(os.Getenv("SLAYER_BANDS"))
