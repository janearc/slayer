package main

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
	"github.com/janearc/slayer/trace"
)

const (
	// still is the moment drawn: nothing in the house moves yet
	still = 0
	// square is a pixel's width over its height
	square = 1
)

// places is where the set's assets stand, given on the command line as
// -at "the box=-1,0,0", in the world's space. A set file does not place its
// assets yet, and where things are is the scene's, not this command's.
type places map[string]maths.Vec

// String is the places as -at would say them.
func (at places) String() string { return fmt.Sprint(map[string]maths.Vec(at)) }

// Set reads one -at: a name, then x, y and z in metres.
func (at places) Set(said string) error {
	name, where, found := strings.Cut(said, "=")
	parts := strings.Split(where, ",")
	if !found || len(parts) != 3 {
		return fmt.Errorf("-at wants name=x,y,z, not %q", said)
	}
	var metres [3]float64
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return fmt.Errorf("-at %q: %w", said, err)
		}
		metres[i] = value
	}
	at[name] = maths.V(metres[0], metres[1], metres[2])
	return nil
}

// where answers the set's question of where an asset stands.
func (at places) where(asset string) (maths.Vec, bool) {
	spot, found := at[asset]
	return spot, found
}

// shoot draws the cameras named from the set file into out, and says
// what it drew. One camera is drawn at the film's frame, its raw written
// beside the picture; several side by side, each at half the frame.
func shoot(setPath string, cameras []string, at places,
	out string) (string, error) {
	written, err := os.ReadFile(setPath)
	if err != nil {
		return "", err
	}
	read, err := set.Read(setPath, written)
	if err != nil {
		return "", err
	}
	said, has := read.World()
	if !has {
		return "", fmt.Errorf("%s says no world to draw", setPath)
	}
	film, has := read.Film()
	if !has {
		return "", fmt.Errorf("%s says no film to draw on", setPath)
	}
	world, err := stage.World(said)
	if err != nil {
		return "", fmt.Errorf("%s: %w", setPath, err)
	}
	var shots []set.Shot
	for _, name := range cameras {
		shot, err := read.Shot(name, at.where)
		if err != nil {
			return "", err
		}
		shots = append(shots, shot)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	wide, high, rays := film.Size()
	lens, printing := stage.Lens(film), stage.Print(film)
	if len(shots) == 1 {
		drawn, err := trace.Sampled(world, shots[0].Eye(), lens, still,
			wide, high, rays)
		if err != nil {
			return "", err
		}
		if err := raw.Write(strings.TrimSuffix(out, ".png")+".pfm",
			drawn); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s: %dx%d, %d rays a pixel\n", out, wide, high,
			rays*rays), file.PNG(out, printing.Develop(drawn), square)
	}
	wide, high = wide/2, high/2
	gap := wide / 32
	picture := image.NewRGBA(image.Rect(0, 0,
		len(shots)*wide+(len(shots)-1)*gap, high))
	draw.Draw(picture, picture.Bounds(), image.Black, image.Point{},
		draw.Src)
	var told strings.Builder
	for i, each := range shots {
		drawn, err := trace.Sampled(world, each.Eye(), lens, still, wide,
			high, rays)
		if err != nil {
			return "", err
		}
		left := i * (wide + gap)
		draw.Draw(picture, image.Rect(left, 0, left+wide, high),
			printing.Develop(drawn), image.Point{}, draw.Src)
		fmt.Fprintf(&told, "%s: %s, %q\n", out, each.Name(), each.Title())
	}
	return told.String(), file.PNG(out, picture, square)
}
