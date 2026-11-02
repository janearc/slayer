package hud

import (
	"errors"
	"fmt"
	"image/color"
	"os"

	"github.com/janearc/libtheme-css/colourway"
	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// ErrHud is a hud that cannot be drawn: a role its theme has not, a
// camera it cannot project through, a value out of its range. the error
// says which.
var ErrHud = errors.New("hud")

// Theme is the colours a hud draws in, by role: a colourway's own roles,
// and the general ones colourway resolves from them, ink, dim, border,
// ground, ok, warn and the rest. a hud names roles and never colours,
// so the same hud drawn in another theme is drawn in its colours.
type Theme struct {
	roles colourway.Resolved
}

// NewTheme is a theme of a sheet of roles, its general roles resolved.
func NewTheme(roles *css.Sheet) Theme {
	return Theme{roles: colourway.Resolve(roles)}
}

// ReadTheme is the theme of a colourway file: a css sheet of roles.
func ReadTheme(path string) (Theme, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, fmt.Errorf("%w: %w", ErrHud, err)
	}
	return NewTheme(css.Read(string(raw)).Roles), nil
}

// Colour is a role's colour as a screen shows it: the theme's swatch
// made srgb, clipped to what the screen's lamps can make. a role the
// theme has not, even once resolved, is an error naming it.
func (made Theme) Colour(role string) (color.RGBA, error) {
	if made.roles.Roles == nil {
		return color.RGBA{}, fmt.Errorf("%w: no theme for role %q",
			ErrHud, role)
	}
	swatch, ok := made.roles.Of(role)
	if !ok {
		return color.RGBA{}, fmt.Errorf("%w: the theme has no role %q",
			ErrHud, role)
	}
	lit, _ := srgb.FromSwatch(swatch)
	r, g, b := lit.Bytes()
	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
