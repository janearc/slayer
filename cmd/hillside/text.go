package main

import (
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/typography"
)

// letters stands a text's letters on the ground where the set says, its
// baseline so far above it, facing its bearing and turning as it says,
// each letter a thing in the text's colour.
func (stage *staging) letters(each set.Text) error {
	east, north, up := each.At()
	place, err := stage.at(east, north)
	if err != nil {
		return err
	}
	place.Y += up
	stage.places[each.Name()] = place
	stage.note(each.Name(), "text", place)
	made, err := spelled(each, filepath.Dir(stage.given.set))
	if err != nil {
		return err
	}
	red, green, blue, gloss := each.Colour()
	material := &thing.Material{Name: each.Name(),
		Colour: maths.RGB(red, green, blue), Gloss: gloss}
	moving := turning(each, place)
	for _, one := range typography.Things(made, material, moving) {
		stage.world.Add(one)
	}
	stage.tallied(made)
	if stage.made.kept > stage.given.faces {
		return fmt.Errorf("%w: past -faces of %d", errTooMany,
			stage.given.faces)
	}
	return nil
}

// spelled is a text's letters made in its font, the font found from
// dir, the set file's own directory, when it is a file.
func spelled(each set.Text, dir string) ([]typography.Letter, error) {
	face, err := typography.Named(each.Font(), dir)
	if err != nil {
		return nil, err
	}
	size, depth, bevel, smooth, tolerance := each.Measures()
	return typography.Make(face, typography.Text{Words: each.Words(),
		Size: size, Depth: depth, Bevel: bevel,
		Smooth: smooth * maths.Degree, Tolerance: tolerance})
}

// tallied counts a text's letters as things, each one's faces as drawn,
// and each mesh's faces once as kept, since a letter used twice is one
// mesh standing twice.
func (stage *staging) tallied(made []typography.Letter) {
	kept := map[*mesh.Mesh]bool{}
	for _, letter := range made {
		if letter.Mesh == nil {
			continue
		}
		stage.made.things++
		stage.made.faces += letter.Mesh.FaceCount()
		if !kept[letter.Mesh] {
			kept[letter.Mesh] = true
			stage.made.kept += letter.Mesh.FaceCount()
		}
	}
}

// turning is how a text stands and moves: at its place, its front, which
// faces plus z in the letters' own space, turned to its bearing, and
// turning about up once in its spin's seconds.
//
// the scene's plus z is south and plus x east, so the bearing b,
// clockwise from north, is the front turned pi less b about up.
func turning(each set.Text, place maths.Vec) thing.Motion {
	bearing, spin := each.Facing()
	facing := math.Pi - bearing*maths.Degree
	return thing.Spin{In: maths.Subspace{Origin: place,
		Orientation: maths.About(maths.Up, facing)}, Axis: maths.Up,
		Period: time.Duration(spin * float64(time.Second))}
}
