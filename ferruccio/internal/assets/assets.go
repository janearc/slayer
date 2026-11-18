// Package assets holds the prototype's pinned schema and credited imagery.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed *.schema.json *.yaml *.jpg *.ttf
var files embed.FS

// Read returns a fresh copy of a bundled asset.
func Read(name string) ([]byte, error) { return fs.ReadFile(files, name) }
