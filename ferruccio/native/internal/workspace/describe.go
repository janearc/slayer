package workspace

import (
	"time"

	"github.com/janearc/slayer/ferruccio/native/internal/look"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// WriteLook writes down what the window shows now, appends it to the set's
// looks file and says in the status line where it went. It works only when
// asked: nothing here runs while the view is drawn. It returns the path of
// the looks file.
func (work *Workspace) WriteLook() (string, error) {
	path, err := look.Place(work.state.path, work.evidenceFolder(), work.exampleDraft)
	if err != nil {
		return "", err
	}
	if err := look.Append(path, look.Render(look.Describe(work.lookInput(time.Now())))); err != nil {
		return "", err
	}
	work.status = "Look written · " + path
	work.caption = work.status
	work.invalidate()
	return path, nil
}

// describeView is what the Describe button and its key do: write a look, and
// show the problem if it cannot be written.
func (work *Workspace) describeView() {
	_, err := work.WriteLook()
	work.problem(err)
}

// lookInput gathers what the window already knows, and why it knows nothing
// where it does not.
func (work *Workspace) lookInput(now time.Time) look.Input {
	text := work.state.document.Text()
	display := work.working.mode
	if display == "" {
		display = "rendered"
	}
	input := look.Input{When: now, Text: text, File: work.state.path, Display: display, Composition: work.composition, SelectedID: work.play.selected}
	input.Objects, _ = work.state.document.Objects()
	if read, err := set.Read(work.state.path, []byte(text)); err == nil {
		input.Things = read.Things()
	}
	input.Scene, input.SceneWhy = work.lookScene()
	input.Pointer, input.PointerWhy = work.lookPointer()
	return input
}

// lookScene is the working display's scene when it is the one for this
// draft, or nil and why not.
func (work *Workspace) lookScene() (*placement.DisplayScene, string) {
	live := &work.working
	switch {
	case live.failure != "":
		return nil, "the display refused this draft: " + live.failure
	case live.scene == nil:
		return nil, "the working view has not been built for this draft"
	case !work.workingCurrent():
		return nil, "the working view is not current for this draft"
	}
	return live.scene, ""
}

// lookPointer is where the pointer last was over the picture, as fractions
// of its width and height, or nil and why not. It is the last place and not
// the place now, because pressing a button moves the pointer off the picture.
func (work *Workspace) lookPointer() (*[2]float64, string) {
	at, over := work.viewer.imagePoint(work.viewer.hoverPoint)
	if !over {
		return nil, "the pointer has not been over the picture"
	}
	return &[2]float64{float64(at.X), float64(at.Y)}, ""
}
