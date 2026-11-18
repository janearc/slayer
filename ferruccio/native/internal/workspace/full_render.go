package workspace

// StartFullRender freezes the authored still dimensions and rays in exact quality.
// It uses the same cancellation, observation and verified-result acceptance as preview.
func (work *Workspace) StartFullRender() error { return work.startRender(true) }
