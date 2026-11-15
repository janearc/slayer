package hud

import (
	_ "embed"

	"github.com/janearc/libtheme-css/css"
)

// defaultSheet is the default colourway, as its file holds it.
//
//go:embed default.css
var defaultSheet string

// DefaultTheme is the hud's own colourway, for a picture whose maker
// names none: roles ink, ground, dim, border, ok, warn and err, light
// ink on a dark ground. Where its colours came from is in default.css.
func DefaultTheme() Theme {
	return NewTheme(css.Read(defaultSheet).Roles)
}
