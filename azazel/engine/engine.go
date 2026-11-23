// Package engine draws one set file through the session package, the one
// slayer render uses. It is the render queue's adapter to the renderer, and
// the only place in azazel that imports it: the queue's own packages
// under azazel/internal do not.
package engine

import (
	"io"

	"github.com/janearc/slayer/session"
)

// Render draws the set file at path with the render settings, which are
// hillside's flags that may be given with a set: -w, -h and -n for the
// size and the rays, -out for where the picture goes, and the rest.
//
// It says what it did to said and what went wrong to wrong, and gives back
// hillside's exit number: nought for a picture written, one for a set it
// could not stage or draw, two for a setting it could not take. The stamp
// says which build is running.
func Render(path string, settings []string, said, wrong io.Writer,
	stamp session.Stamp) int {
	arguments := append([]string{"-set", path}, settings...)
	return session.Run(arguments, said, wrong, stamp)
}
