package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/ferruccio/internal/capture"
	"github.com/janearc/slayer/ferruccio/internal/studio"
)

// nativeProof checks the real window and captures only its own canvas.
func nativeProof(call pzm.Call) error {
	if len(call.Args()) != 0 || call.Value("out") == "" {
		return fmt.Errorf("proof needs --out and no positional arguments")
	}
	if call.On("render") && call.Value("file") == "" {
		return fmt.Errorf("render proof needs --file")
	}
	if call.DryRun() {
		_, err := fmt.Fprintln(call.Out(), "open native Atelier, capture its canvas, close only Atelier")
		return err
	}
	application := app.NewWithID("org.slayerscenes.atelier.proof")
	application.Settings().SetTheme(studio.Theme())
	window := application.NewWindow("Atelier · native proof")
	workspace, err := studio.New(studio.Options{Window: window, Study: "venus", Tools: toolsFor(call), Examples: call.Value("examples")})
	if err != nil {
		return err
	}
	if call.Value("file") != "" {
		if err := workspace.LoadFile(call.Value("file")); err != nil {
			return err
		}
	}
	window.SetContent(workspace.Content())
	window.Resize(fyne.NewSize(1600, 1000))
	result := make(chan error, 1)
	finish := func() {
		previewPath, previewError := workspace.PreviewResult()
		if call.On("render") && (previewError != nil || previewPath == "") {
			result <- fmt.Errorf("native render proof failed: output=%q, error=%v", previewPath, previewError)
			application.Quit()
			return
		}
		picture := window.Canvas().Capture()
		receipt := map[string]any{"build": build, "renderer": "Fyne native GLFW canvas", "native_window": true, "width": picture.Bounds().Dx(), "height": picture.Bounds().Dy(), "source_file": filepath.Base(call.Value("file")), "preview_file": previewPath}
		err := capture.Write(call.Value("out"), picture, receipt)
		if err == nil {
			err = json.NewEncoder(call.Out()).Encode(receipt)
		}
		result <- err
		application.Quit()
	}
	timer := time.AfterFunc(time.Second, func() {
		fyne.Do(func() {
			if !call.On("render") {
				finish()
				return
			}
			workspace.StartPreview()
			done := workspace.PreviewDone()
			if done == nil {
				result <- fmt.Errorf("preview did not start")
				application.Quit()
				return
			}
			go func() {
				<-done
				// Capture the painted frame after the image update reaches the draw loop.
				time.AfterFunc(300*time.Millisecond, func() { fyne.Do(finish) })
			}()
		})
	})
	defer timer.Stop()
	window.ShowAndRun()
	select {
	case err := <-result:
		return err
	default:
		return fmt.Errorf("window closed before the capture completed")
	}
}
