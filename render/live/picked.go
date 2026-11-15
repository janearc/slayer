package live

import "time"

// The counts the package is made of, each defined.
const (
	none = 0 // defined: no rows, no frames, no stops
	one  = 1 // defined: a single frame, rung or line
	pair = 2 // defined: two of a thing
)

// stopBase is what one stop multiplies light by. Defined: a stop doubles
// the light.
const stopBase = 2.0

// thirdsInStop is how many steps of `e` and `E` make a stop. Defined by
// the spec: a key steps exposure by a third of a stop.
const thirdsInStop = 3

// The numbers the package picked. Each is picked by the author of this
// slice, 2026-11-12, open to cutting.
const (
	// Ceiling is the longest a rung of the ladder may be predicted to
	// take and still be started. Past it the picture stays at the rung
	// before: a preview that takes longer than this is not a preview.
	Ceiling = 15 * time.Second

	// PollEvery is how often the job file's time and size are looked at.
	// Often enough that a save shows within a blink, rarely enough to
	// cost nothing.
	PollEvery = 250 * time.Millisecond

	// FootRows is how many rows of notes and errors sit above the status
	// row. Three show a refusal and the two notes before it.
	FootRows = 3

	// statusRows is the status row itself. Defined: one row.
	statusRows = 1
)

// divisors are the ladder's sizes: the fitted picture divided by each, in
// order, at one ray. Picked by the author of this slice, 2026-11-12: an
// eighth is drawn in a blink on any scene, and each rung is four times
// the pixels of the one before.
var divisors = []int{8, 4, 2, 1}

// finerRays are the rays along each side of a pixel the ladder's last
// rungs draw the whole size at, after one. Picked by the author of this
// slice, 2026-11-12: two and four smooth edges at a pane's size; past
// four a pane's cells cannot show the difference.
var finerRays = []int{2, 4}

// The modes the files are written with, as the door writes them.
const (
	folderMode = 0o755 // defined: a folder anyone may read
	fileMode   = 0o644 // defined: a file anyone may read
)

// The files of the status folder.
const (
	framePicture  = "frame.png"
	screenPicture = "screen.png"
	statusFile    = "status.json"
)

// square is a picture's pixel as one pixel of its png. Defined.
const square = 1

// half is half of a share, to round to the nearest. Defined.
const half = 0.5
