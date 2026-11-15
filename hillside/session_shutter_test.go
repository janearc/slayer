package hillside

// The numbers the shutter's tests pick, each by the session slice's
// author, 2026-11-12.
const (
	// shutterRate is two frames a second, so a frame's interval is half a
	// second and a shutter's glimpses are well apart.
	shutterRate = 2
	// shutterGlimpses is four glimpses a frame.
	shutterGlimpses = 4
	// shutterStretch is how many times slower than the film scene time
	// runs in a stretched film: ten seconds of scene in one of film.
	shutterStretch = 10
)
