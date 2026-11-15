// Command magnetar lays walk's labels and panel over the real
// magnetar's traced stills: it reads a walk script, takes one short by
// name, and writes it as a film, or one frame of it as a picture.
//
//	magnetar SCRIPT PICTURES SHORT OUT.mp4
//	magnetar SCRIPT PICTURES SHORT OUT.png
//	magnetar film PLAN FRAMES OUT.mp4
//
// A picture is the short's middle frame. film writes the short in the
// house form from magnetar-film.yaml, the scene's frames under FRAMES.
// The scripts and pictures are the scene's; this command only hands
// them to packages walk and host.
package main

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/interfaces/movie"
)

// the arguments the command takes, defined
const asked = 5

// halves is how a short's frames are halved to find its middle, defined
const halves = 2

// errUsage is a call with the wrong arguments.
var errUsage = errors.New("magnetar SCRIPT PICTURES SHORT OUT.mp4|OUT.png" +
	", or magnetar film PLAN FRAMES OUT.mp4")

// main runs the command on the program's arguments. A failure prints the
// error and exits with 1.
func main() {
	if err := run(os.Args, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "magnetar:", err)
		os.Exit(1)
	}
}

// run reads the script, finds the short, and writes it where asked,
// saying what it wrote.
func run(args []string, said io.Writer) error {
	if len(args) == asked && args[1] == "film" {
		return film(args[2], args[3], args[4], said)
	}
	if len(args) != asked {
		return errUsage
	}
	script, err := walk.ReadScript(args[1], args[2])
	if err != nil {
		return err
	}
	short, found := script.Short(args[3])
	if !found {
		return fmt.Errorf("the script has no short %q", args[3])
	}
	out := args[4]
	if filepath.Ext(out) == ".png" {
		return still(short, out, said)
	}
	written, err := short.Write(context.Background(), out, movie.Sound{})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(said, "%s: %d frames\n", out, written)
	return err
}

// still writes the short's middle frame as a picture.
func still(short walk.Short, out string, said io.Writer) error {
	made, err := short.Frame(short.Frames()/halves, walk.NewStills())
	if err != nil {
		return err
	}
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(file, made); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(said, out)
	return err
}
