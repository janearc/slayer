package door

// Package door's numbers and words. Each number is defined; the door
// picks no number of its own.

// The counts the door is made of.
const (
	nothing = 0 // defined: no frames, no words, no overrides
	one     = 1 // defined: a single file, frame or line
	pair    = 2 // defined: two of a thing, as a width and a height
)

// rowsPerFrame is what the meter is told a frame holds: the door counts
// whole frames, so a frame is one row to it. Defined.
const rowsPerFrame = 1

// square is a picture's pixel as one pixel of its png, as hillside's
// save writes a still. Defined.
const square = 1

// The modes the door writes with, as Slayer's other commands write.
const (
	folderMode = 0o755 // defined: a folder anyone may read
	fileMode   = 0o644 // defined: a file anyone may read
)

// frameName is a film's frame file, numbered from nought: hillside's
// sequence verb names its frames so, and the door keeps the name.
const frameName = "frame-%06d.png"

// framePattern is a glob that matches every name frameName gives, and
// only those.
const framePattern = "frame-[0-9][0-9][0-9][0-9][0-9][0-9].png"

// mapPattern is a glob that matches every name a scene of maps gives a
// map: a word, a hyphen and its clock, eight digits or more, as the
// weather names its maps. Defined by scene/named's clockName.
const mapPattern = "[a-z]*-[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]*.png"

// manifestName is the frames folder's record of what was written.
const manifestName = "manifest.json"

// partSuffix marks a file being written beside the one it replaces.
const partSuffix = ".part"

// The environment the door reads, and the places under it.
const (
	rendersVariable = "SLAYER_RENDERS" // the renders home
	teamVariable    = "SLAYER_TEAM"    // the team whose notes file it is
	notesName       = "latest-notes.txt"
	stillsFolder    = "frames"
	// cacheFolder is under the renders home, where the renders page
	// makes no cards; working frames go in it by team.
	cacheFolder = "cache"
	// workFolder is under the system's temporary folder, where working
	// frames go when there is no renders home or no team.
	workFolder   = "slayer-frames"
	tempVariable = "TMPDIR"
	filmsFolder  = "animations"
	outFolder    = "out"
	dateLayout   = "2006-01-02"
	// datePattern is a glob that matches a date written in dateLayout.
	datePattern = "[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]"
)

// The extensions of what the door writes by default.
const (
	pictureExtension = ".png"
	movieExtension   = ".mp4"
	jobExtension     = ".yaml"
)

// workDigits is how many hex digits of the job file's path name its
// working folder, beside the job's name. Picked by the door slice's
// author, 2026-11-12: twelve make two jobs of one name apart.
const workDigits = 12

// stereo is a sound's channels, left then right. Defined.
const stereo = 2
