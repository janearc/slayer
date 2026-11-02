package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// keep writes the picture's raw at path, its light as a pfm.
func keep(drawn *frame.Frame, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return raw.Write(path, drawn)
}

// proving proves the raw -prove names against the hillside these flags make:
// the pixels -pixel names, then -many more the raw's bytes pick, made again on
// the cpu, each held to the raw bit for bit.
//
// the sheet goes in -proof, or beside the raw, with the raw developed and every
// proved pixel marked on it. it gives back zero when the proof holds and one
// when it fails or cannot be made.
func proving(world *thing.World, given asked, across float64,
	arguments []string, said, wrong io.Writer) int {
	eye, lens := eyeFor(given, across), lensFor(given)
	replay := func(x, y int) (frame.Pixel, []trace.Step, error) {
		if given.glass != nil {
			return replayed(world, given, eye, x, y)
		}
		return trace.Replay(world, eye, lens, given.at, given.width,
			given.height, given.samples, x, y)
	}
	asked := "hillside " + strings.Join(arguments, " ")
	proof, err := fermat.Prove(given.prove, asked, given.pixels,
		given.many, replay)
	if err == nil && (proof.Width != given.width ||
		proof.Height != given.height) {
		err = fmt.Errorf("%s is %d by %d, and these flags draw %d by %d",
			given.prove, proof.Width, proof.Height, given.width,
			given.height)
	}
	dir := sheetFor(given)
	if err == nil {
		err = sheet(proof, dir)
	}
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return 1
	}
	fmt.Fprintln(said, proof)
	fmt.Fprintf(said, "%s: the sheet, and the raw marked\n", dir)
	if !proof.Holds() {
		return 1
	}
	return 0
}

// named is the pixels -pixel names, in the order named, each X,Y.
type named [][2]int

// String is the pixels as they are named, between semicolons.
func (pixels *named) String() string {
	var said []string
	for _, at := range *pixels {
		said = append(said, fmt.Sprintf("%d,%d", at[0], at[1]))
	}
	return strings.Join(said, ";")
}

// Set takes one more pixel, X,Y, each a whole number.
func (pixels *named) Set(text string) error {
	across, down, found := strings.Cut(text, ",")
	x, wrongX := strconv.Atoi(strings.TrimSpace(across))
	y, wrongY := strconv.Atoi(strings.TrimSpace(down))
	if !found || wrongX != nil || wrongY != nil {
		return fmt.Errorf("%q is not a pixel X,Y", text)
	}
	*pixels = append(*pixels, [2]int{x, y})
	return nil
}

// inside is whether every pixel named is in a picture so wide and tall.
func (pixels named) inside(width, height int) bool {
	for _, at := range pixels {
		if at[0] < 0 || at[1] < 0 || at[0] >= width || at[1] >= height {
			return false
		}
	}
	return true
}

// sheetFor is where a proof's sheet goes: -proof, or beside the raw,
// named for it.
func sheetFor(given asked) string {
	if given.proof != "" {
		return given.proof
	}
	return strings.TrimSuffix(given.prove, filepath.Ext(given.prove)) +
		"-proof"
}

// sheet writes the proof's pages in dir, and the raw developed as the
// print is, with its picked pixels marked, as marked.png.
func sheet(proof fermat.Proof, dir string) error {
	if err := proof.Write(dir); err != nil {
		return err
	}
	kept, err := raw.Read(proof.Raw)
	if err != nil {
		return err
	}
	picture := developing.Develop(kept)
	proof.Mark(picture)
	const square = 1
	return file.PNG(filepath.Join(dir, "marked.png"), picture, square)
}
