package publication

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
)

// publisher owns a trusted filesystem adapter, not an admission authority.
type publisher struct {
	filesystem filesystem
}

// replacement names a destination and holds its complete new contents.
type replacement struct {
	parent string
	name   string
	body   []byte
}

// publicationStage distinguishes confirmed visibility, refusal and uncertainty.
type publicationStage uint8

const (
	publicationUnknown publicationStage = iota
	publicationRefused
	publicationCreateFailed
	publicationWriteFailed
	publicationCloseFailed
	publicationRenameUncertain
	publicationVisible
	publicationExists
	publicationInstallUncertain
)

// publicationResult retains the primary stage and joined failure causes.
type publicationResult struct {
	stage publicationStage
	err   error
}

// replace publishes a complete snapshot through a same-directory temporary.
// Deferred cleanup keeps ownership in this operation until rename succeeds.
func (owner publisher) replace(request replacement) (result publicationResult) {
	if owner.filesystem == nil || !filepath.IsAbs(request.parent) ||
		request.parent != filepath.Clean(request.parent) {
		return publicationResult{
			stage: publicationRefused,
			err: errors.New("publication needs a filesystem " +
				"and clean absolute parent"),
		}
	}
	if request.name == "" || request.name == "." || request.name == ".." ||
		strings.ContainsAny(request.name, "/\x00") {
		return publicationResult{stage: publicationRefused,
			err: errors.New("publication needs one plain basename")}
	}
	opened := owner.filesystem.CreateTemp(tempSpec{
		parent: request.parent, pattern: "." + request.name + "-*"})
	if opened.err != nil {
		return publicationResult{
			stage: publicationCreateFailed, err: opened.err,
		}
	}
	cleanup := temporaryCleanup{
		remover: owner.filesystem, opened: opened,
	}
	defer cleanup.finish(&result)
	written := opened.file.Write(writeSpec{body: request.body})
	if written.err == nil && written.written != len(request.body) {
		written.err = io.ErrShortWrite
	}
	if written.err != nil {
		return publicationResult{
			stage: publicationWriteFailed, err: written.err,
		}
	}
	cleanup.closeAttempted = true
	if err := opened.file.Close(); err != nil {
		return publicationResult{
			stage: publicationCloseFailed, err: err,
		}
	}
	if err := owner.filesystem.Rename(renameSpec{
		from: opened.name,
		to:   filepath.Join(request.parent, request.name),
	}); err != nil {
		return publicationResult{
			stage: publicationRenameUncertain, err: err,
		}
	}
	return publicationResult{stage: publicationVisible}
}
