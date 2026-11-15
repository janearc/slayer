package job

// schemaName is the schema's name: what a job's $schema says, and what
// resolves the schema's own references. It names no file to fetch.
const schemaName = "https://github.com/janearc/slayer/schemas/job-v1.json"

// nothing is defined as no frames, no time and a key the job left unsaid.
const nothing = 0

// one is defined as a single frame: the fewest a film has.
const one = 1

// halves is defined as the two halves an angle is cut into, either side
// of the middle of the picture.
const halves = 2

// The scene's forms, as the scene key writes them.
const (
	formSet    = "set"
	formSmoke  = "smoke"
	formNamed  = "named"
	formFrames = "frames"
)

// The job's built-in defaults, filled into its canonical form.
const (
	// defaultQuality is the quality of a job that says none.
	defaultQuality = "exact"
	// defaultEase is the easing of a move or a zoom that says none.
	defaultEase = "linear"
	// defaultAlign is where a text sits on its place, as a set's hud
	// has it.
	defaultAlign = "left"
)

// seconds and degrees are the units a quantity is written in.
const (
	seconds = "s"
	degrees = "deg"
)

// movieExtension is the extension that makes a command line's output a
// film's movie rather than its frames.
const movieExtension = ".mp4"

// builtIn is what starts a tune or a voicing that names a built-in
// rather than a file: it is left as written.
const builtIn = "@"
