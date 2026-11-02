package typography

import (
	"path/filepath"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// gos is go's own fonts by the names a scene may give them: the same on
// every machine, with nothing to fetch, as the tests use them.
var gos = map[string][]byte{
	"go":        goregular.TTF,
	"go-bold":   gobold.TTF,
	"go-italic": goitalic.TTF,
	"go-medium": gomedium.TTF,
	"go-mono":   gomono.TTF,
}

// Named is a font a scene names: one of go's own by its name, go,
// go-bold, go-italic, go-medium or go-mono, or else a font file, its
// path taken from dir when it is not absolute.
func Named(name, dir string) (*Font, error) {
	if raw, ok := gos[name]; ok {
		return Parse(raw)
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(dir, name)
	}
	return Load(name)
}
