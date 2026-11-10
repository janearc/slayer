package publication

import "path/filepath"

// preparedPublication supplies checked opaque bytes under a caller-held lock.
// Its fixed private entry names are candidates for the shared contract.
type preparedPublication struct {
	parent string
	intent []byte
	state  []byte
}

// completionStage identifies the last attempted phase, never grant authority.
type completionStage uint8

const (
	completionUnknown completionStage = iota
	completionIntentFailed
	completionStateFailed
	completionRemovalUncertain
	completionComplete
)

// completionResult retains the failed phase and all underlying error causes.
type completionResult struct {
	stage completionStage
	err   error
}

// complete publishes prepared intent and state before clearing the intent.
// Its caller validates records and holds the transaction lock throughout.
func (owner publisher) complete(request preparedPublication) completionResult {
	intent := owner.replace(replacement{parent: request.parent,
		name: "intent", body: request.intent})
	if intent.stage != publicationVisible {
		return completionResult{
			stage: completionIntentFailed, err: intent.err}
	}
	state := owner.replace(replacement{parent: request.parent,
		name: "state", body: request.state})
	if state.stage != publicationVisible {
		return completionResult{
			stage: completionStateFailed, err: state.err}
	}
	if err := owner.filesystem.Remove(pathSpec{
		path: filepath.Join(request.parent, "intent"),
	}); err != nil {
		return completionResult{
			stage: completionRemovalUncertain, err: err}
	}
	return completionResult{stage: completionComplete}
}
