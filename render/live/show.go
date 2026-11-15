package live

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/render/finish"
	"github.com/janearc/slayer/render/frame"
)

// developed is the frame shown as last developed: which frame, at which
// exposure and development, and its picture.
type developed struct {
	shown   *held
	thirds  int
	version int
	picture *image.RGBA
}

// helpLines are the keys, shown in the foot by `?`.
var helpLines = []string{
	"left right: a frame back or on    [ ]: a second back or on",
	"space: play or pause    c: the next camera    e E: exposure " +
		"down or up a third of a stop",
	"f: the frame at full size    s: write the moment, camera and " +
		"exposure into the job    q: quit",
	"a preview is not the final render to the bit; slayer draw " +
		"draws that.    any key returns",
}

// developer makes the job's development again: its print or stock, or
// the scene's tone, and its titles. One that cannot be made is said, and
// the last kept.
func (watching *Watcher) developer() {
	developing, err := finish.NewDeveloper(watching.wanted,
		watching.facts.tone)
	if err != nil {
		watching.foot.note(err.Error())
		return
	}
	titles, has, err := finish.NewTitles(watching.wanted)
	if err != nil {
		watching.foot.note(err.Error())
		return
	}
	watching.developing, watching.titles = developing, nil
	if has {
		watching.titles = &titles
	}
	watching.version++
}

// picture is the frame shown, developed at the exposure the keys chose,
// with its hud and titles, and whether it was developed afresh: it is
// kept until the frame, the exposure or the development changes.
func (watching *Watcher) picture() (*image.RGBA, bool, error) {
	want := developed{shown: watching.shown, thirds: watching.view.thirds,
		version: watching.version}
	kept := watching.developed
	if kept.picture != nil && kept.shown == want.shown &&
		kept.thirds == want.thirds && kept.version == want.version {
		return kept.picture, false, nil
	}
	light := watching.shown.drawn.Light()
	if watching.view.thirds != none {
		light = scaled(light, gainOf(watching.view.thirds))
	}
	picture, err := watching.developing.Develop(watching.shown.index,
		[]*frame.Frame{light})
	if err == nil {
		err = watching.shown.drawn.Overlay(picture)
	}
	if err == nil && watching.titles != nil {
		err = watching.titles.Draw(picture, watching.filmTime())
	}
	if err != nil {
		return nil, false, err
	}
	want.picture = picture
	watching.developed = want
	return picture, true, nil
}

// filmTime is the shown frame's time in the film, from its start, which
// the titles are timed by.
func (watching *Watcher) filmTime() time.Duration {
	return time.Duration(watching.shown.index) * time.Second /
		time.Duration(watching.rate())
}

// show puts the frame shown on the screen, fitted to the pane, and the
// foot and status row under it, and keeps them in the status folder.
func (watching *Watcher) show() {
	screen := watching.given.Screen.Screen()
	var picture *image.RGBA
	var cells *terminal.Cells
	fresh := false
	if watching.shown != nil && watching.view.mode != helping {
		var err error
		picture, fresh, err = watching.picture()
		if err != nil {
			watching.foot.note(err.Error())
		}
	}
	if picture != nil {
		place := placed(picture.Bounds().Dx(), picture.Bounds().Dy(),
			paneOf(screen))
		cells = terminal.Chop(canvas(picture, place))
		watching.said(watching.given.Screen.Present(cells))
	}
	lines := watching.foot.lines()
	if watching.view.mode == helping {
		lines = helpLines
	}
	watching.said(watching.given.Screen.Foot(append(lines,
		watching.status(picture))))
	watching.publish(picture, cells, screen.Pixel, fresh)
}

// said keeps a screen that could not be written to as a note; the next
// show tries again.
func (watching *Watcher) said(err error) {
	if err != nil {
		watching.foot.note(err.Error())
	}
}

// status is the status row: what is being done, the quality and rung
// shown, the moment, camera and exposure, and how to get help.
func (watching *Watcher) status(picture *image.RGBA) string {
	var parts []string
	if watching.words != "" {
		parts = append(parts, watching.words)
	}
	quality, _ := watching.wanted.Quality()
	parts = append(parts, quality+", "+watching.rungWords(picture))
	parts = append(parts, watching.momentWords())
	if camera := watching.cameraWords(); camera != "" {
		parts = append(parts, "camera "+camera)
	}
	if watching.view.thirds != none {
		parts = append(parts, fmt.Sprintf("exposure %+d/3 stop",
			watching.view.thirds))
	}
	if watching.view.playing {
		parts = append(parts, "playing")
	}
	return strings.Join(append(parts, "? help  q quit"), "  ")
}

// rungWords says the rung shown, and a full frame's size.
func (watching *Watcher) rungWords(picture *image.RGBA) string {
	if watching.shown == nil {
		return "nothing drawn yet"
	}
	if watching.shown.rung.size == (Size{}) && picture != nil {
		return fmt.Sprintf("full size %dx%d", picture.Bounds().Dx(),
			picture.Bounds().Dy())
	}
	return watching.shown.rung.words
}

// momentWords says the frame and its scene time, or a still's moment.
func (watching *Watcher) momentWords() string {
	times := watching.facts.times
	index := min(watching.view.index, len(times)-one)
	if index < none {
		return "not staged yet"
	}
	if watching.still() {
		return "at " + times[index].String()
	}
	return fmt.Sprintf("frame %d of %d at %s", index+one, len(times),
		times[index])
}

// cameraWords is the camera shown by name: the one the keys chose, or
// the job's; nothing for the scene's own.
func (watching *Watcher) cameraWords() string {
	if watching.view.camera != "" {
		return watching.view.camera
	}
	lens, said := watching.wanted.Camera()
	if !said {
		return ""
	}
	name, _ := lens.Name()
	return name
}
