// Package workspace connects the GoGPU interface to real drafts and previews.
package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"io"
	"os"
	"path/filepath"
)

// documentState retains the original path separately from the display name.
type documentState struct {
	document *draft.Document
	path     string
}

// load replaces state only after the whole candidate has been read.
func (state *documentState) load(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	file, err := os.Open(absolute)
	if err != nil {
		return err
	}
	defer file.Close()
	source, err := io.ReadAll(io.LimitReader(file, 4*1024*1024+1))
	if err != nil {
		return err
	}
	if len(source) > 4*1024*1024 {
		return fmt.Errorf("YAML exceeds the 4 MiB editor limit")
	}
	document, err := draft.Open(filepath.Base(absolute), source)
	if err != nil {
		return err
	}
	state.document, state.path = document, absolute
	return nil
}

// accepts prevents an old render from replacing a newer document's view.
func (state *documentState) accepts(plan *preview.Plan) bool {
	if plan == nil || state.document == nil || state.path != plan.Source {
		return false
	}
	sum := sha256.Sum256([]byte(state.document.Text()))
	return hex.EncodeToString(sum[:]) == plan.Digest
}
