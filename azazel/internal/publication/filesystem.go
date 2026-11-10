// Package publication holds private local file publication primitives.
package publication

// filesystem is the publisher's trusted namespace boundary.
// Successful creation transfers a valid handle and same-directory name.
// Failed creation transfers neither. A rename error may have an uncertain
// effect. Callers must reconcile state before using it as authority.
type filesystem interface {
	CreateTemp(tempSpec) openedTemp
	Rename(renameSpec) error
	Remove(pathSpec) error
}

// temporary accepts one complete payload and one close attempt.
type temporary interface {
	Write(writeSpec) writeResult
	Close() error
}

// tempSpec locates private files beside the requested destination.
type tempSpec struct {
	parent  string
	pattern string
}

// openedTemp transfers ownership only when err is nil.
type openedTemp struct {
	file temporary
	name string
	err  error
}

// renameSpec replaces one destination with the completed temporary file.
type renameSpec struct {
	from string
	to   string
}

// pathSpec identifies the single temporary name eligible for removal.
type pathSpec struct {
	path string
}

// writeSpec supplies the already validated file bytes.
type writeSpec struct {
	body []byte
}

// writeResult reports both byte count and a possible underlying cause.
type writeResult struct {
	written int
	err     error
}
