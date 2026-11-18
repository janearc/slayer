package studio

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
)

// openDocument protects the current draft before offering a bounded file read.
func (workspace *Workspace) openDocument() {
	proceed := func() {
		picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				workspace.problem(err)
				return
			}
			if reader == nil {
				return
			}
			source, readError := io.ReadAll(io.LimitReader(reader, (4<<20)+1))
			closeError := reader.Close()
			if readError != nil {
				workspace.problem(readError)
				return
			}
			if closeError != nil {
				workspace.problem(closeError)
				return
			}
			if len(source) > 4<<20 {
				workspace.problem(fmt.Errorf("this prototype opens documents up to 4 MiB; the original file was left untouched"))
				return
			}
			if err := workspace.load(reader.URI().Path(), source); err != nil {
				workspace.problem(err)
			}
		}, workspace.window)
		picker.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
		picker.Show()
	}
	if workspace.document.Dirty() {
		dialog.ShowConfirm("Keep your draft?", "Opening another document will replace the unsaved draft. Continue?", func(yes bool) {
			if yes {
				proceed()
			}
		}, workspace.window)
	} else {
		proceed()
	}
}

// saveDocument chooses a path without truncating it before the atomic save.
func (workspace *Workspace) saveDocument() {
	path := widget.NewEntry()
	if workspace.sourcePath != "" {
		path.SetText(workspace.sourcePath)
	} else {
		path.SetText(workspace.document.Name())
	}
	browse := widget.NewButton("Choose folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				workspace.problem(err)
				return
			}
			if uri != nil {
				path.SetText(filepath.Join(uri.Path(), workspace.document.Name()))
			}
		}, workspace.window)
	})
	picker := dialog.NewCustomConfirm("Save draft", "Save", "Cancel", container.NewVBox(paragraph("Save the current text, including any unfinished edits."), path, browse), func(yes bool) {
		if !yes {
			return
		}
		write := func() {
			if err := workspace.document.Save(path.Text); err != nil {
				workspace.problem(err)
				return
			}
			workspace.sourcePath, _ = filepath.Abs(path.Text)
			workspace.syncDocument()
			workspace.status.SetText("Saved · " + filepath.Base(path.Text))
		}
		if _, err := os.Stat(path.Text); err == nil {
			dialog.ShowConfirm("Replace this file?", "The existing file will be replaced by this draft.", func(replace bool) {
				if replace {
					write()
				}
			}, workspace.window)
		} else if os.IsNotExist(err) {
			write()
		} else {
			workspace.problem(err)
		}
	}, workspace.window)
	picker.Resize(fyne.NewSize(660, 240))
	picker.Show()
}

// openImage displays a supplied still without claiming that Slayer rendered it.
func (workspace *Workspace) openImage() {
	picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			workspace.problem(err)
			return
		}
		if reader == nil {
			return
		}
		picture, _, decodeError := image.Decode(reader)
		closeError := reader.Close()
		if decodeError != nil {
			workspace.problem(decodeError)
			return
		}
		if closeError != nil {
			workspace.problem(closeError)
			return
		}
		workspace.picture.Resource = nil
		workspace.picture.File = ""
		workspace.picture.Image = picture
		workspace.picture.FillMode = canvas.ImageFillContain
		workspace.picture.Refresh()
		workspace.caption.Text = "Loaded image · " + filepath.Base(reader.URI().Path())
		workspace.caption.Refresh()
		workspace.creditDetail.Text = "Image supplied by you · rendering is not started by loading it"
		workspace.creditDetail.Refresh()
	}, workspace.window)
	picker.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg"}))
	picker.Show()
}

// load replaces the model only after its schema and source are constructed.
func (workspace *Workspace) load(name string, source []byte) error {
	document, err := draft.Open(filepath.Base(name), source)
	if err != nil {
		return err
	}
	workspace.document = document
	workspace.sourcePath, err = filepath.Abs(name)
	if err != nil {
		return err
	}
	workspace.showDocument()
	workspace.syncDocument()
	return nil
}

// LoadFile is the explicit initial-file path supplied through Puzzlemouth.
func (workspace *Workspace) LoadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	source, err := io.ReadAll(io.LimitReader(file, (4<<20)+1))
	closeError := file.Close()
	if err != nil {
		return err
	}
	if closeError != nil {
		return closeError
	}
	if len(source) > 4<<20 {
		return fmt.Errorf("document exceeds the prototype's 4 MiB limit")
	}
	return workspace.load(path, source)
}

// protectClose leaves unsaved work in place unless the user chooses to discard.
func (workspace *Workspace) protectClose() {
	workspace.window.SetCloseIntercept(func() {
		finishClose := func() { workspace.window.SetCloseIntercept(nil); workspace.window.Close() }
		close := func() {
			if workspace.cancelRender != nil {
				workspace.renderState.SetText("Stopping the renderer before closing…")
				workspace.cancelRender()
				done := workspace.renderDone
				go func() { <-done; fyne.Do(finishClose) }()
				return
			}
			finishClose()
		}
		if !workspace.document.Dirty() {
			close()
			return
		}
		dialog.ShowConfirm("Leave this draft?", "There are unsaved edits. Close without saving?", func(discard bool) {
			if discard {
				close()
			}
		}, workspace.window)
	})
}

// problem reports failed actions in the owning window without discarding work.
func (workspace *Workspace) problem(err error) {
	workspace.status.SetText("Needs attention")
	dialog.ShowError(fmt.Errorf("Atelier: %w", err), workspace.window)
}
