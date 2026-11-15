// Package nvim holds slayer.lua, the nvim side of `slayer edit`: the
// job schema for hover, autoread, and a key that draws the job. The
// file is the thing shipped; this package lets the slayer tool write a
// copy of it where nvim can load it.
package nvim

import (
	_ "embed"
	"slices"
)

//go:embed slayer.lua
var script []byte

// Script is slayer.lua's bytes.
func Script() []byte { return slices.Clone(script) }
