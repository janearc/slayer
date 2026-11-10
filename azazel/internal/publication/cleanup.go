package publication

import "errors"

// temporaryRemover names only the owned entry eligible for cleanup.
type temporaryRemover interface {
	Remove(pathSpec) error
}

// temporaryCleanup owns only a successfully created private temporary.
// The caller serializes its lifetime and records close attempts before Close.
type temporaryCleanup struct {
	remover        temporaryRemover
	opened         openedTemp
	closeAttempted bool
}

// finish joins owned cleanup refusals into the caller's nonnil final result.
// Register once after creation. The pointer receiver sees later close state.
func (owner *temporaryCleanup) finish(result *publicationResult) {
	if result.stage == publicationVisible {
		return
	}
	if !owner.closeAttempted {
		owner.closeAttempted = true
		result.err = errors.Join(result.err, owner.opened.file.Close())
	}
	result.err = errors.Join(result.err,
		owner.remover.Remove(pathSpec{path: owner.opened.name}))
}
