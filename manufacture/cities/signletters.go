package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
)

// The two looks a sign's letters take. Tube neon is thin and pillowed and
// glows all through. A channel letter is a deep dark body with a face
// that glows.
const (
	lookTube    = "tube"
	lookChannel = "channel"
)

// glyphForm is one letter made solid by package typography, as a form
// that package cities can mesh: the letter's faces as triangles.
type glyphForm struct{ solid *mesh.Mesh }

// Mesh is the glyph's faces as triangles, in the order and the winding
// the typography package made them.
func (form glyphForm) Mesh(Detail) (*Mesh, error) {
	count := form.solid.FaceCount()
	triangles := make([]Triangle, count)
	for index := range count {
		one, two, three := form.solid.Corners(index)
		triangles[index] = Triangle{one, two, three}
	}
	return &Mesh{Triangles: triangles}, nil
}

// inkLayer is one solid layer of a letter: how deep it is, how far its
// faces are stepped in, how far behind the front of the letter's depth it
// starts, what it is made of, and whether it is the layer that glows.
type inkLayer struct {
	depth, bevel, from float64
	material           *thing.Material
	lit                bool
}

// layers are the solid layers a letter of an em is made of, in the look
// the job gives it and the colour given: for a screen's pillowed letters
// one layer that glows, for flat letters, painted on a board or dark on a
// lit face, one layer of the job's ink, and for a channel letter its dark
// body and the face that glows on it.
func (job signJob) layers(em float64, neon option) []inkLayer {
	drawing := job.drawing
	switch job.look {
	case lookTube:
		depth := em * drawing.fixed("sign-tube-depth")
		return []inkLayer{{depth: depth,
			bevel:    depth * drawing.fixed("sign-tube-bevel"),
			material: job.glow(neon), lit: true}}
	case lookLamped, lookBox, lookValance:
		return []inkLayer{{depth: em * drawing.fixed("sign-flat-depth"),
			material: job.lettering}}
	}
	body := em * drawing.fixed("sign-channel-depth")
	return []inkLayer{{depth: body, material: job.steel},
		{depth: em * drawing.fixed("sign-channel-face"), from: body,
			material: job.glow(neon), lit: true}}
}

// width is how wide a line of words runs, in ems, as the font sets it.
func (job signJob) width(words string) (float64, error) {
	face := job.faceFor()
	line, err := typography.Layout(face, words)
	if err != nil {
		return 0, fmt.Errorf("cities: sign words %q: %w", words, err)
	}
	return line.Width / face.Units(), nil
}

// letters are the letters of a text, each a mesh. A bevel typography cannot
// take on a letter, as it cannot at any width on the letter K, whose arms
// meet the stem at a point sharp enough that the corners of the step in
// cross, is taken off that letter alone, and the letter set without one.
// The face is the one the sign is set in, whole and letter by letter.
func (job signJob) letters(text typography.Text) ([]typography.Letter,
	error) {
	face := job.faceFor()
	made, err := typography.Make(face, text)
	if err == nil || text.Bevel == 0 {
		return made, err
	}
	bevel := text.Bevel
	text.Bevel = 0
	flat, err := typography.Make(face, text)
	if err != nil {
		return nil, err
	}
	text.Bevel = bevel
	for index, letter := range flat {
		text.Words = string(letter.Letter)
		if one, err := typography.Make(face, text); err == nil &&
			letter.Mesh != nil {
			flat[index].Mesh = one[0].Mesh
		}
	}
	return flat, nil
}

// stand is how far the letters of an em stand out of what they are on:
// the depth of all their layers.
func (job signJob) stand(em float64) float64 {
	layers := job.layers(em, job.neon)
	last := layers[len(layers)-1]
	return last.from + last.depth
}

// ink is a line of words set in letters of an em, with the middle of its
// baseline at a point of the sign's frame, standing on the plane through
// that point. A letter stands out of the plane toward +z. Of the letters
// that have a shape, the one numbered dark, counting from nought, has its
// glowing layer out; a number below nought is none.
func (job signJob) ink(words string, em float64, middle maths.Vec,
	neon option, dark int) ([]placedElement, error) {
	var parts []placedElement
	for _, layer := range job.layers(em, neon) {
		made, err := job.solids(words, em, middle, layer, dark)
		if err != nil {
			return nil, err
		}
		parts = append(parts, made...)
	}
	return parts, nil
}

// solids is one layer of a line of letters, each placed where the font
// sets it, in the layer's material, or dark if the layer glows and the
// letter is the one that is out.
func (job signJob) solids(words string, em float64, middle maths.Vec,
	layer inkLayer, dark int) ([]placedElement, error) {
	letters, err := job.letters(typography.Text{Words: words, Size: em,
		Depth: layer.depth, Bevel: layer.bevel,
		Tolerance: job.tolerance()})
	if err != nil {
		return nil, fmt.Errorf("cities: sign words %q: %w", words, err)
	}
	var parts []placedElement
	for _, letter := range letters {
		if letter.Mesh == nil {
			continue
		}
		material := layer.material
		if layer.lit && len(parts) == dark {
			material = job.unlit("letter")
		}
		at := maths.V(middle.X+letter.At.X,
			middle.Y+letter.At.Y+job.rise*em,
			middle.Z+layer.from+layer.depth/float64(half))
		parts = append(parts, placedElement{Element: solid{Name: "sign letter",
			Form: glyphForm{letter.Mesh}, Material: material},
			Place: standingAt(at)})
	}
	return parts, nil
}
