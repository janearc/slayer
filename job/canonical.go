package job

import (
	"encoding/json"
	"slices"

	"github.com/janearc/slayer/maths"
)

// canonical is the job's canonical form: JSON with its keys sorted and
// its numbers written as encoding/json writes a float64, holding every
// value that changes the picture or the sound, with the job's own
// defaults filled in, and no outputs, limits or note.
//
// It is refused where it cannot be written as JSON (a number that is
// not finite, a map whose keys are not words, in a named scene's with):
// two such jobs would otherwise share the key of nothing.
func (wire document) canonical() ([]byte, error) {
	form := wire
	form.Outputs, form.Limits, form.Titles = nil, nil, nil
	form.Film = canonicalFilm(wire.Film)
	if form.Quality == "" {
		form.Quality = defaultQuality
	}
	form.Shortcuts = canonicalShortcuts(wire)
	if form.film() && form.Start == nil {
		form.Start = &quantity{Value: nothing, Unit: seconds}
	}
	form.Camera = canonicalCamera(wire.Camera)
	form.Print, form.Stock = canonicalDevelopment(wire.Print, wire.Stock)
	form.Sound = canonicalSound(wire.Sound)
	raw, err := json.Marshal(form)
	if err != nil {
		return nil, err
	}
	var tree map[string]any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	if wire.Titles != nil {
		tree["titles"] = canonicalTitles(*wire.Titles)
	}
	return json.Marshal(tree)
}

// canonicalFilm is the film without its preset's name: the preset has
// already given its size and rate, which are what change the picture.
// A mode of light is left out too: it is what a film that says no mode
// draws, and a job that says it keeps the key it had without it.
func canonicalFilm(film *filmWire) *filmWire {
	if film == nil {
		return nil
	}
	plain := *film
	plain.Preset = ""
	if plain.Mode == modeLight {
		plain.Mode = ""
	}
	return &plain
}

// canonicalCamera is the camera with its easings and its up filled in.
func canonicalCamera(lens *cameraWire) *cameraWire {
	if lens == nil {
		return nil
	}
	plain := *lens
	plain.Moves = slices.Clone(lens.Moves)
	for i := range plain.Moves {
		plain.Moves[i].Ease = eased(plain.Moves[i].Ease)
	}
	if plain.Moves != nil && plain.Up == nil {
		plain.Up = &point{X: maths.Up.X, Y: maths.Up.Y, Z: maths.Up.Z}
	}
	if lens.Zoom != nil {
		zoom := *lens.Zoom
		zoom.Ease = eased(zoom.Ease)
		plain.Zoom = &zoom
	}
	return &plain
}

// eased is an easing, linear where none is said.
func eased(ease string) string {
	if ease == "" {
		return defaultEase
	}
	return ease
}

// canonicalDevelopment is the print and the stock with their unsaid
// numbers written as the zero they mean.
func canonicalDevelopment(curve *printWire, film *stockWire) (*printWire,
	*stockWire) {
	if curve != nil {
		plain := *curve
		plain.Bloom, plain.Stops = filled(curve.Bloom), filled(curve.Stops)
		curve = &plain
	}
	if film != nil {
		plain := *film
		plain.Seed, plain.Rated = filled(film.Seed), filled(film.Rated)
		film = &plain
	}
	return curve, film
}

// filled is the number said, or a zero where none was.
func filled[Number int64 | float64](value *Number) *Number {
	number, _ := pointed(value)
	return &number
}

// canonicalSound is the sound with an audio file's start filled in.
func canonicalSound(track *soundWire) *soundWire {
	if track == nil || track.File == "" || track.From != nil {
		return track
	}
	plain := *track
	plain.From = &quantity{Value: nothing, Unit: seconds}
	return &plain
}

// canonicalTitles is the titles with their elements as a list, since
// their order is the order they are drawn in and a JSON object's keys
// are sorted; each element carries its name and its alignment.
func canonicalTitles(words titlesWire) map[string]any {
	list := make([]any, len(words.Elements))
	for i, each := range words.Elements {
		title := each.title
		if title.Kind == "text" && title.Align == "" {
			title.Align = defaultAlign
		}
		title.Count, title.Value = block(title.Count), block(title.Value)
		raw, _ := json.Marshal(title)
		var element map[string]any
		_ = json.Unmarshal(raw, &element)
		element["name"] = each.name
		list[i] = element
	}
	tree := map[string]any{"elements": list}
	if words.Theme != "" {
		tree["theme"] = words.Theme
	}
	if words.Rows != nothing {
		tree["rows"] = words.Rows
	}
	return tree
}

// block is a ramp written as a block, so a number alone and the same
// number as from and to are the same.
func block(value *ramp) *ramp {
	if value == nil {
		return nil
	}
	plain := *value
	plain.alone = false
	return &plain
}
