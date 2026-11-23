package session

import (
	"path/filepath"

	"github.com/janearc/slayer/manufacture"
)

// lettered is the row given the files of the fonts its script is set in,
// each found on the source shelf as the sources.yaml nearest the set pins
// it, and checked against its pin. A row whose recipe says no script, or
// a script set in go's own fonts, is returned as it is. It downloads
// nothing: a font that is not on the shelf is one error that names the
// source and says how to fetch it.
func (stage *staging) lettered(row *manufacture.Kind,
	script string) (*manufacture.Kind, error) {
	if script == "" {
		return row, nil
	}
	keys, err := manufacture.SignFaces(script)
	if err != nil || len(keys) == 0 {
		return row, err
	}
	files := map[string]string{}
	for _, key := range keys {
		asset, err := pinnedAssetAt(key, "", filepath.Dir(stage.given.set), "")
		if err != nil {
			return nil, err
		}
		files[key] = asset.path
	}
	made, err := manufacture.WithFaces(*row, files)
	if err != nil {
		return nil, err
	}
	return &made, nil
}
