package studio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
)

// makeRenderControls connects the visible action to the actual preview adapter.
func (workspace *Workspace) makeRenderControls() {
	workspace.renderState = widget.NewLabel("Open a YAML file, then render its preview.")
	workspace.renderState.Wrapping = fyne.TextWrapWord
	workspace.renderLog = widget.NewMultiLineEntry()
	workspace.renderLog.TextStyle = fyne.TextStyle{Monospace: true}
	workspace.renderLog.Disable()
	workspace.progress = widget.NewProgressBarInfinite()
	workspace.progress.Hide()
	workspace.renderButton = widget.NewButtonWithIcon("Render preview", theme.MediaPlayIcon(), workspace.StartPreview)
	workspace.renderButton.Importance = widget.HighImportance
	workspace.stopButton = widget.NewButtonWithIcon("Stop", theme.CancelIcon(), func() {
		if workspace.cancelRender != nil {
			workspace.cancelRender()
		}
	})
	workspace.stopButton.Disable()
	workspace.stopButton.Hide()
}

// StartPreview freezes the draft and draws it on a worker while Fyne stays responsive.
func (workspace *Workspace) StartPreview() {
	if workspace.cancelRender != nil {
		return
	}
	plan, err := preview.Prepare(preview.Request{Path: workspace.sourcePath, Text: workspace.document.Text()})
	if err != nil {
		workspace.problem(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	workspace.cancelRender = cancel
	workspace.renderDone = make(chan struct{})
	workspace.renderButton.Disable()
	workspace.stopButton.Enable()
	workspace.stopButton.Show()
	workspace.renderLog.SetText("")
	workspace.lastPreview = ""
	workspace.previewError = nil
	workspace.renderState.SetText("Waiting or rendering · 640 × 360 · one ray per side")
	workspace.progress.Show()
	workspace.progress.Start()
	output := &renderOutput{workspace: workspace}
	go func() {
		err := preview.Run(ctx, workspace.tools, plan, output)
		closeError := plan.Close()
		cancel()
		if err == nil {
			err = closeError
		}
		fyne.Do(func() { workspace.finishPreview(plan, err); close(workspace.renderDone) })
	}()
}

// finishPreview accepts only an image belonging to the still-current document.
func (workspace *Workspace) finishPreview(plan *preview.Plan, err error) {
	workspace.cancelRender = nil
	workspace.progress.Stop()
	workspace.progress.Hide()
	workspace.stopButton.Disable()
	workspace.stopButton.Hide()
	workspace.syncDocument()
	workspace.previewError = err
	if err != nil {
		workspace.renderState.SetText("Preview failed · " + err.Error())
		return
	}
	sum := sha256.Sum256([]byte(workspace.document.Text()))
	if hex.EncodeToString(sum[:]) != plan.Digest || workspace.sourcePath != plan.Source {
		workspace.renderState.SetText("Preview finished for an older draft. Render again to see your changes.")
		return
	}
	workspace.showDocument()
	if err := workspace.DisplayImage(plan.Output); err != nil {
		workspace.previewError = err
		workspace.problem(err)
		return
	}
	workspace.lastPreview = plan.Output
	workspace.caption.Text = "Slayer preview · 640 × 360 · one ray per side"
	workspace.caption.Refresh()
	workspace.creditDetail.Text = "Rendered from " + workspace.document.Name()
	workspace.creditDetail.Refresh()
	workspace.renderState.SetText("Preview ready · " + workspace.document.Name())
}

// PreviewDone is the current request's completion, including its image update.
func (workspace *Workspace) PreviewDone() <-chan struct{} { return workspace.renderDone }

// PreviewResult is the real output path or the renderer's failure.
func (workspace *Workspace) PreviewResult() (string, error) {
	return workspace.lastPreview, workspace.previewError
}

// DisplayImage decodes the actual output rather than displaying a reference.
func (workspace *Workspace) DisplayImage(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	picture, _, err := image.Decode(file)
	closeError := file.Close()
	if err != nil {
		return err
	}
	if closeError != nil {
		return closeError
	}
	workspace.picture.Resource = nil
	workspace.picture.File = ""
	workspace.picture.Image = picture
	workspace.picture.FillMode = canvas.ImageFillContain
	workspace.picture.Refresh()
	return nil
}

// showRenderLog exposes the live output of Game and Slayer without invented progress.
func (workspace *Workspace) showRenderLog() {
	view := dialog.NewCustom("Renderer output", "Close", workspace.renderLog, workspace.window)
	view.Resize(fyne.NewSize(900, 550))
	view.Show()
}

type renderOutput struct {
	workspace *Workspace
	mutex     sync.Mutex
	text      string
}

// Write serializes stdout and stderr and bounds the visible transcript.
func (output *renderOutput) Write(raw []byte) (int, error) {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	output.text += string(raw)
	if len(output.text) > 65536 {
		output.text = output.text[len(output.text)-65536:]
	}
	transcript := output.text
	fyne.Do(func() { output.workspace.renderLog.SetText(transcript) })
	return len(raw), nil
}

// openExamples offers real, unchanged YAML from Slayer.
func (workspace *Workspace) openExamples() {
	if workspace.examples == "" {
		workspace.problem(fmt.Errorf("no examples directory configured"))
		return
	}
	menu := fyne.NewMenu("Examples",
		fyne.NewMenuItem("First picture · still job", func() { workspace.openExample("first-picture.yaml") }),
		fyne.NewMenuItem("Fizz · scene set", func() { workspace.openExample("fizz.yaml") }),
		fyne.NewMenuItem("Webb at L2 · scene set", func() { workspace.openExample("webb-at-l2.yaml") }))
	widget.ShowPopUpMenuAtPosition(menu, workspace.window.Canvas(), fyne.NewPos(400, 90))
}

// openExample applies the same unsaved-draft protection as opening a chosen file.
func (workspace *Workspace) openExample(name string) {
	open := func() {
		if err := workspace.LoadExample(name); err != nil {
			workspace.problem(err)
		}
	}
	if workspace.document.Dirty() {
		dialog.ShowConfirm("Replace this draft?", "There are unsaved edits. Open the example?", func(yes bool) {
			if yes {
				open()
			}
		}, workspace.window)
	} else {
		open()
	}
}

// LoadExample opens an editable copy so updating the app cannot erase saved work.
func (workspace *Workspace) LoadExample(name string) error {
	source, err := os.ReadFile(filepath.Join(workspace.examples, name))
	if err != nil {
		return err
	}
	folder, err := os.MkdirTemp("", "atelier-example-")
	if err != nil {
		return err
	}
	path := filepath.Join(folder, filepath.Base(name))
	if err := os.WriteFile(path, source, 0o600); err != nil {
		return err
	}
	return workspace.LoadFile(path)
}

// showDocument removes mission imagery when a real document becomes current.
func (workspace *Workspace) showDocument() {
	title, err := workspace.document.Value("scene", "name")
	if err != nil {
		title = workspace.document.Name()
	}
	workspace.worldTitle.Text = title
	workspace.worldTitle.Refresh()
	workspace.worldSubtitle.Text = "Slayer " + workspace.document.Kind()
	workspace.worldSubtitle.Refresh()
	workspace.picture.Resource = nil
	workspace.picture.Image = nil
	workspace.picture.File = ""
	workspace.picture.Refresh()
	workspace.caption.Text = "Render this document to see its image."
	workspace.caption.Refresh()
	workspace.creditDetail.Text = "Preview · 640 × 360 · original file stays unchanged"
	workspace.creditDetail.Refresh()
	workspace.renderState.SetText("Ready to render " + workspace.document.Name())
}
