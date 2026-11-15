package finish

import (
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/job"
)

//go:generate sh mocks/generate.sh

// titled is one title as a job gives it: package job's Title, named by
// the methods this package reads, so a title of any kind can be tried.
type titled interface {
	Name() string
	Kind() string
	Words() (words, format string, counted bool, count job.Ramp)
	Value() (job.Ramp, bool)
	Box() (left, top, right, bottom float64)
	At() (across, down float64, align string)
	Roles() (ink, ground, fill, empty string)
	From() (time.Duration, bool)
	Until() (time.Duration, bool)
}

// Titles is what a job draws over its frames: a hud, the theme its roles
// come from, and the rows of cells a picture is laid out in.
type Titles struct {
	drawn hud.Hud
	theme hud.Theme
	rows  int
}

// kind is how one kind of title is made into a hud element, and which of
// its roles it must name: the rest may be left out and are then clear.
type kind struct {
	make     func(each titled) (hud.Element, error)
	required func(each titled) string
}

// kinds are the titles a picture can hold, by name. A label pinned to a
// thing in the world is not among them: it needs the scene.
var kinds = map[string]kind{
	"text": {textOf, func(each titled) string {
		ink, _, _, _ := each.Roles()
		return ink
	}},
	"frame": {frameOf, func(each titled) string {
		ink, _, _, _ := each.Roles()
		return ink
	}},
	"bar": {barOf, func(each titled) string {
		_, _, fill, _ := each.Roles()
		return fill
	}},
}

// NewTitles is the titles a job says, and whether it says any. Its theme
// is the colourway file the job names, or the hud's default. A title of
// a kind a picture cannot hold, or a role the theme has not, is refused
// by the title's name, here and not on the frame that first shows it.
func NewTitles(made job.Job) (Titles, bool, error) {
	written, said := made.Titles()
	if !said {
		return Titles{}, false, nil
	}
	theme, err := themeOf(written)
	if err != nil {
		return Titles{}, false, err
	}
	rows, said := written.Rows()
	if !said {
		rows = defaultRows
	}
	var each []titled
	for _, title := range written.Elements() {
		each = append(each, title)
	}
	drawn, err := elementsOf(each, theme)
	if err != nil {
		return Titles{}, false, err
	}
	return Titles{drawn: drawn, theme: theme, rows: rows}, true, nil
}

// themeOf is the colourway the titles name, or the hud's own.
func themeOf(written job.Titles) (hud.Theme, error) {
	path, named := written.Theme()
	if !named {
		return hud.DefaultTheme(), nil
	}
	theme, err := hud.ReadTheme(path)
	if err != nil {
		return hud.Theme{}, fmt.Errorf("finish: the titles' theme "+
			"%s cannot be read: %w", path, err)
	}
	return theme, nil
}

// elementsOf is the hud of the titles, each over the ones before.
func elementsOf(each []titled, theme hud.Theme) (hud.Hud, error) {
	var drawn hud.Hud
	for _, title := range each {
		made, found := kinds[title.Kind()]
		if !found {
			return hud.Hud{}, fmt.Errorf("finish: title %q is a %q; "+
				"titles are text, frame or bar, and a label pinned to "+
				"the world needs the scene, which frames in do not "+
				"have", title.Name(), title.Kind())
		}
		if err := rolesHeld(title, made, theme); err != nil {
			return hud.Hud{}, err
		}
		element, err := made.make(title)
		if err != nil {
			return hud.Hud{}, fmt.Errorf("finish: title %q: %w",
				title.Name(), err)
		}
		drawn.Elements = append(drawn.Elements, element)
	}
	return drawn, nil
}

// rolesHeld refuses a title that names a role the theme has not, or no
// role where its kind needs one.
func rolesHeld(title titled, made kind, theme hud.Theme) error {
	ink, ground, fill, empty := title.Roles()
	if made.required(title) == "" {
		return fmt.Errorf("finish: title %q is a %s and names no ink "+
			"or fill role; name one from the theme", title.Name(),
			title.Kind())
	}
	for _, role := range []string{ink, ground, fill, empty} {
		if role == "" {
			continue
		}
		if _, err := theme.Colour(role); err != nil {
			return fmt.Errorf("finish: title %q: %w; name a role the "+
				"theme has", title.Name(), err)
		}
	}
	return nil
}

// spanOf is the film time a title is drawn in.
func spanOf(each titled) hud.Span {
	from, hasFrom := each.From()
	until, hasUntil := each.Until()
	switch {
	case hasFrom && hasUntil:
		return hud.Between(from, until)
	case hasFrom:
		return hud.From(from)
	case hasUntil:
		return hud.Until(until)
	}
	return hud.Span{}
}

// aligned is each way words stand on their place, by the word a job
// says it in.
var aligned = map[string]hud.Align{"left": hud.Left, "centre": hud.Centre,
	"right": hud.Right}

// gauged is a job's ramp as a gauge: its from until its start, then
// evenly to its to over its seconds.
func gauged(value job.Ramp) hud.Gauge {
	from, to, start, over := value.Values()
	return hud.Ramp{From: from, To: to, Start: seconds(start),
		Over: seconds(over)}
}

// seconds is a number of seconds as a duration.
func seconds(count float64) time.Duration {
	return time.Duration(count * float64(time.Second))
}

// textOf is a title of words, or of a count written out.
func textOf(each titled) (hud.Element, error) {
	across, down, align := each.At()
	stand, found := aligned[align]
	if !found {
		return nil, fmt.Errorf("it sits %q; it sits left, centre or "+
			"right", align)
	}
	ink, ground, _, _ := each.Roles()
	words, format, counted, count := each.Words()
	var wording hud.Wording = hud.Words(words)
	if counted {
		wording = hud.Count{Of: gauged(count), Format: format}
	}
	return hud.Text{Words: wording, At: hud.Point{Across: across,
		Down: down}, Align: stand, Ink: ink, Ground: ground,
		Within: spanOf(each)}, nil
}

// frameOf is a title that is a box's edge.
func frameOf(each titled) (hud.Element, error) {
	ink, _, _, _ := each.Roles()
	return hud.Frame{Box: boxOf(each), Ink: ink, Within: spanOf(each)}, nil
}

// barOf is a title that is a gauge shown as a bar.
func barOf(each titled) (hud.Element, error) {
	value, said := each.Value()
	if !said {
		return nil, errors.New("a bar has no value; give it one from " +
			"nought to one")
	}
	_, _, fill, empty := each.Roles()
	return hud.Bar{Box: boxOf(each), Value: gauged(value), Fill: fill,
		Empty: empty, Within: spanOf(each)}, nil
}

// boxOf is the part of the picture a title takes.
func boxOf(each titled) hud.Box {
	left, top, right, bottom := each.Box()
	return hud.Box{Left: left, Top: top, Right: right, Bottom: bottom}
}

// Draw draws the titles on a developed picture, at a time of film time:
// those whose span holds it, over the picture as it is.
func (made Titles) Draw(img *image.RGBA, at time.Duration) error {
	return made.drawn.Picture(img, made.rows, at, made.theme, hud.View{})
}
