package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/janearc/slayer/improv/searching"
	"github.com/janearc/slayer/interfaces/movie"
)

// openMovie prepares only the sound interval used by this take.
func openMovie(ctx context.Context, edit *searching.Edit, into string, from, count int) (*movie.Writer, error) {
	sound, err := excerptSound(edit.Plan.Music, from, count, edit.Plan.FPS)
	if err != nil {
		return nil, err
	}
	debug.FreeOSMemory()
	settings, err := movie.NewSettings(movie.Request{Path: filepath.Join(into, "animation.mp4"),
		Width: edit.Plan.Width, Height: edit.Plan.Height, Rate: edit.Plan.FPS, Frames: count, Sound: sound, Pad: true})
	if err != nil {
		return nil, err
	}
	return movie.New(ctx, settings)
}

// encodeFrames opens the encoder only after native scene resources close.
// It reads the actual saved pictures in order, keeping one decoded PNG.
func encodeFrames(ctx context.Context, edit *searching.Edit, into string, from, count int) error {
	writer, err := openMovie(ctx, edit, into, from, count)
	if err != nil {
		return err
	}
	defer writer.Abort()
	for index := range count {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		path := filepath.Join(into, "frames", fmt.Sprintf("frame-%06d.png", from+index))
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		picture, decoded := png.Decode(file)
		if err := errors.Join(decoded, file.Close()); err != nil {
			return err
		}
		rgba := image.NewRGBA(picture.Bounds())
		draw.Draw(rgba, rgba.Bounds(), picture, picture.Bounds().Min, draw.Src)
		if err := writer.Frame(index, rgba); err != nil {
			return err
		}
	}
	return writer.Close()
}
