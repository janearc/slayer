package publication

// immutableFilesystem owns temporary creation and exclusive namespace install.
// Success consumes the source name. An existing-entry error leaves the
// incumbent untouched; other errors may have an uncertain namespace effect.
type immutableFilesystem interface {
	CreateTemp(tempSpec) openedTemp
	Install(installSpec) error
	Remove(pathSpec) error
}

// installSpec names a completed temporary and its absent destination.
type installSpec struct {
	from string
	to   string
}
