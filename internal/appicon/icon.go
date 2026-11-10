// Package appicon holds the supplied Miami artwork used by Slayer's viewers.
package appicon

import "embed"

//go:embed miami-icon-*-1024.png
var artwork embed.FS

// PNG returns an owned copy of the light or dark application icon.
func PNG(dark bool) ([]byte, error) {
	name := "miami-icon-light-1024.png"
	if dark {
		name = "miami-icon-dark-1024.png"
	}
	return artwork.ReadFile(name)
}
