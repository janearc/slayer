package publication

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// immutablePublisher owns an exclusive filesystem, not a grant authority.
type immutablePublisher struct {
	filesystem immutableFilesystem
}

// installationSpec supplies selected destination identity and verified bytes.
type installationSpec struct {
	parent string
	name   string
	body   []byte
}

// publish writes and closes complete bytes before absent-destination install.
func (owner immutablePublisher) publish(
	request installationSpec,
) (result publicationResult) {
	if owner.filesystem == nil || !filepath.IsAbs(request.parent) ||
		request.parent != filepath.Clean(request.parent) {
		return publicationResult{stage: publicationRefused,
			err: errors.New("publication needs a filesystem " +
				"and clean absolute parent")}
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
	cleanup := temporaryCleanup{remover: owner.filesystem, opened: opened}
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
	if err := owner.filesystem.Install(installSpec{
		from: opened.name,
		to:   filepath.Join(request.parent, request.name),
	}); err != nil {
		stage := publicationInstallUncertain
		if errors.Is(err, os.ErrExist) {
			stage = publicationExists
		}
		return publicationResult{stage: stage, err: err}
	}
	return publicationResult{stage: publicationVisible}
}
