// Package look writes down, in plain text, what Ferruccio shows: the camera,
// the selected thing, what is under the pointer and what is in view. A look
// holds only what the display knows; what it cannot tell, it says it does
// not know, and why. It is computed when asked, never while drawing.
package look

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// MostSeen is how many things in view a look lists. The count of the rest
// is kept.
const MostSeen = 12

// defaultWideDegrees is the field of view a lens with none has.
const defaultWideDegrees = 60

// Look is one description of what the window showed.
type Look struct {
	When                       time.Time
	DraftSHA256                string
	File, Display, Composition string
	Camera                     Camera
	Selected                   *Selected
	Pointer                    Pointer
	// Seen is the things in view, nearest first, at most MostSeen; More is
	// how many more there were. SeenWhy is why Seen is not known.
	Seen    []Seen
	More    int
	SeenWhy string
}

// Camera is where the view stands and how wide it sees, in the set's own
// coordinates: east, north and up, in metres. Why is set when it is not known.
type Camera struct {
	Why           string
	Eye, Focus    [3]float64
	WideDegrees   float64
	Width, Height int
}

// Selected is the thing the person selected: what the set says of it and
// where the display stands it now.
type Selected struct {
	Name, Kind, Recipe, Detail string
	Seed                       uint64
	// SetKnown is whether the set could be read for Recipe, Seed and Detail.
	SetKnown bool
	// Stands is where the display has it, which differs from the set's text
	// while it is being dragged; StandsKnown is whether there is one.
	Stands      [3]float64
	StandsKnown bool
	// Heading is clockwise degrees from north, as the set authors it;
	// HeadingKnown is whether the set says one.
	Heading      float64
	HeadingKnown bool
}

// Pointer is what was under the pointer the last time it was over the
// picture. Why is set when that is not known. Over is empty where the ray
// met nothing.
type Pointer struct {
	Why      string
	At       [2]float64
	Over     string
	Point    [3]float64
	Distance float64
	Part     string
}

// Seen is one thing in view. Share is the part of the picture, from zero to
// one, that the outline of its bounding box covers. It is a bound, not a
// count of pixels, and a thing behind another has its share too.
type Seen struct {
	Name, Kind, Recipe string
	Distance, Share    float64
}

// Input is everything Describe is given. Nothing here is guessed: a field
// left empty is reported as not known.
type Input struct {
	When                       time.Time
	Text                       string
	File, Display, Composition string
	// Scene is the working display's scene, or nil with SceneWhy saying why
	// there is none.
	Scene    *placement.DisplayScene
	SceneWhy string
	// Objects are the draft's placements and Things the set's; either may
	// be nil when it could not be read.
	Objects []draft.Object
	Things  []set.Thing
	// SelectedID is the draft identity of the selected placement, or empty.
	SelectedID string
	// Pointer is where the pointer last was over the picture, as fractions
	// of its width and height from the top left; nil with PointerWhy saying
	// why it is not known.
	Pointer    *[2]float64
	PointerWhy string
}

// Describe is the look of an input. It reads nothing but the input.
func Describe(input Input) Look {
	sum := sha256.Sum256([]byte(input.Text))
	described := Look{When: input.When, DraftSHA256: hex.EncodeToString(sum[:]), File: input.File, Display: input.Display, Composition: input.Composition}
	described.Camera = cameraOf(input)
	described.Selected = selectedOf(input)
	described.Pointer = pointerOf(input)
	described.Seen, described.More, described.SeenWhy = seenOf(input)
	return described
}

// cameraOf is the scene's camera, or why there is none.
func cameraOf(input Input) Camera {
	scene := input.Scene
	if scene == nil {
		return Camera{Why: "not known: " + input.SceneWhy}
	}
	wide := scene.Lens.Wide / maths.Degree
	if !(wide > 0 && wide < 180) {
		wide = defaultWideDegrees
	}
	return Camera{Eye: placement.SetPoint(scene.Eye.Origin), Focus: placement.SetPoint(scene.Focus), WideDegrees: wide, Width: scene.Width, Height: scene.Height}
}

// selectedOf is the selected placement, or nil when none is selected or the
// draft no longer has it.
func selectedOf(input Input) *Selected {
	for _, object := range input.Objects {
		if object.ID() != input.SelectedID || input.SelectedID == "" {
			continue
		}
		chosen := &Selected{Name: object.Name(), Kind: object.Kind()}
		chosen.Heading, chosen.HeadingKnown = object.Heading()
		chosen.Stands, chosen.StandsKnown = object.Position(), true
		if input.Scene != nil {
			for _, shown := range input.Scene.Objects {
				if shown.ID == input.SelectedID {
					chosen.Stands = placement.SetPoint(shown.At.Origin)
				}
			}
		}
		for _, each := range input.Things {
			if each.Name() == object.Name() {
				chosen.Recipe, chosen.Seed, chosen.Detail, chosen.SetKnown = each.Recipe(), each.Seed(), each.Complexity(), true
			}
		}
		return chosen
	}
	return nil
}

// pointerOf is what the pointer's last place on the picture meets.
func pointerOf(input Input) Pointer {
	if input.Pointer == nil {
		return Pointer{Why: "not known: " + input.PointerWhy}
	}
	if input.Scene == nil {
		return Pointer{At: *input.Pointer, Why: "not known: " + input.SceneWhy}
	}
	found := Pointer{At: *input.Pointer}
	if hit, ok := input.Scene.PickAt(input.Pointer[0], input.Pointer[1]); ok {
		found.Over, found.Point, found.Distance, found.Part = hit.Name, placement.SetPoint(hit.Point), hit.Distance, hit.Part
	}
	return found
}

// seenOf is the things in view with the kind and recipe the set gives each,
// cut to MostSeen, with how many were left out, or why none is known.
func seenOf(input Input) ([]Seen, int, string) {
	if input.Scene == nil {
		return nil, 0, "not known: " + input.SceneWhy
	}
	kinds := map[string]set.Thing{}
	for _, each := range input.Things {
		kinds[each.Name()] = each
	}
	var seen []Seen
	for _, each := range input.Scene.InView() {
		entry := Seen{Name: each.Name, Distance: each.Distance, Share: each.Share}
		if thing, found := kinds[each.Name]; found {
			entry.Kind, entry.Recipe = thing.Kind(), thing.Recipe()
		}
		seen = append(seen, entry)
	}
	more := max(0, len(seen)-MostSeen)
	return seen[:len(seen)-more], more, ""
}
