package live

import (
	"encoding/json"
	"errors"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/terminal"
)

// statusSchema names the form of status.json, so a reader can refuse
// one it does not know.
const statusSchema = "slayer-watch-status-v1"

// The states the view is in, as status.json says them.
const (
	stateStaging = "staging" // the job is being opened
	stateDrawing = "drawing" // a rung is being drawn
	stateShown   = "shown"   // the ladder is done; the picture stands
	stateStopped = "stopped" // watch has quit; nothing more is written
)

// modeWords are the modes as status.json says them, in the order of mode.
var modeWords = [...]string{"preview", "help", "full"}

// report is the status folder's status.json: what a graphical front end
// or another program reads of the view. frame.png and screen.png are
// written before it, so the pictures it counts are on disk when it is
// read.
type report struct {
	// Schema is statusSchema.
	Schema string `json:"schema"`
	// Job is the job file watched, absolute.
	Job string `json:"job"`
	// State is staging, drawing, shown or stopped.
	State string `json:"state"`
	// Mode is preview, help or full.
	Mode string `json:"mode"`
	// Quality is preview or exact, as drawn.
	Quality string `json:"quality"`
	// Time is the shown frame's scene time, in seconds.
	Time float64 `json:"time"`
	// Frame is the shown frame, from nought, of Frames; Rate is the
	// film's frames a second, one for a still.
	Frame  int `json:"frame"`
	Frames int `json:"frames"`
	Rate   int `json:"rate"`
	// Playing is whether a film is playing.
	Playing bool `json:"playing"`
	// Camera is the camera shown by name; empty for the scene's own.
	Camera string `json:"camera"`
	// Thirds is the exposure the keys chose, in thirds of a stop.
	Thirds int `json:"exposure_thirds"`
	// Rung is the rung shown, in words.
	Rung string `json:"rung"`
	// Pictures counts the frame.png and screen.png pairs written; it
	// moves when they change. Width, Height and Rays are frame.png's
	// picture as drawn; nought before any.
	Pictures int `json:"pictures"`
	Width    int `json:"width"`
	Height   int `json:"height"`
	Rays     int `json:"rays"`
	// Error is the trouble with the job or its scene now, or empty: the
	// last good picture stays while it stands.
	Error string `json:"error"`
	// Notes are the foot's lines, oldest first.
	Notes []string `json:"notes"`
	// When is when this was written, RFC 3339.
	When string `json:"when"`
}

// published is what the status folder holds of the pictures: how many
// were written, and the last one's size and rays.
type published struct {
	pictures, width, height, rays int
}

// publish writes the status folder, where there is one: the picture and
// the cells when they are fresh, then the status every time. A folder
// that cannot be written is said once at the foot.
func (watching *Watcher) publish(picture *image.RGBA,
	cells *terminal.Cells, pixel float64, fresh bool) {
	folder := watching.given.Folder
	if folder == "" {
		return
	}
	err := os.MkdirAll(folder, folderMode)
	if err == nil && fresh && picture != nil && cells != nil {
		err = errors.Join(
			file.PNG(filepath.Join(folder, framePicture), picture,
				square),
			file.PNG(filepath.Join(folder, screenPicture),
				terminal.Picture(cells), pixel))
		if err == nil {
			watching.kept = published{
				pictures: watching.kept.pictures + one,
				width:    picture.Bounds().Dx(),
				height:   picture.Bounds().Dy(),
				rays:     watching.shown.rung.size.rays}
		}
	}
	if err == nil {
		err = writeWhole(filepath.Join(folder, statusFile),
			watching.report())
	}
	if err != nil && !watching.unpublished {
		watching.unpublished = true
		watching.foot.note("the status folder is not written: " +
			err.Error())
	}
}

// report is the view's status now.
func (watching *Watcher) report() report {
	quality, _ := watching.wanted.Quality()
	made := report{Schema: statusSchema, Job: watching.given.Path,
		State: watching.state(), Mode: modeWords[watching.view.mode],
		Quality: quality, Frames: len(watching.facts.times),
		Rate: watching.rate(), Playing: watching.view.playing,
		Camera: watching.cameraWords(), Thirds: watching.view.thirds,
		Pictures: watching.kept.pictures, Width: watching.kept.width,
		Height: watching.kept.height, Rays: watching.kept.rays,
		Error: watching.foot.trouble, Notes: watching.foot.notes,
		When: watching.given.Now().Format(time.RFC3339Nano)}
	if made.Notes == nil {
		made.Notes = []string{}
	}
	if times := watching.facts.times; len(times) > none {
		index := min(watching.view.index, len(times)-one)
		made.Frame, made.Time = index, times[index].Seconds()
	}
	if watching.shown != nil {
		made.Rung = watching.shown.rung.words
	}
	return made
}

// state is what the view is doing, in status.json's words.
func (watching *Watcher) state() string {
	switch {
	case watching.stopped:
		return stateStopped
	case watching.words == stagingWords:
		return stateStaging
	case watching.flight != nil:
		return stateDrawing
	}
	return stateShown
}

// writeWhole writes a status as JSON beside path and renames it into
// place, so a reader sees the whole of this one or of the one before.
func writeWhole(path string, said report) error {
	raw, err := json.MarshalIndent(said, "", "  ")
	if err != nil {
		return err
	}
	return replaced(path, append(raw, '\n'), fileMode)
}

// replaced writes raw beside path and renames it into place, with the
// mode given; what was at path is left as it was if any step fails.
func replaced(path string, raw []byte, mode os.FileMode) error {
	beside, err := os.CreateTemp(filepath.Dir(path),
		"."+filepath.Base(path)+".")
	if err != nil {
		return err
	}
	_, err = beside.Write(raw)
	err = errors.Join(err, beside.Chmod(mode), beside.Close())
	if err == nil {
		err = os.Rename(beside.Name(), path)
	}
	if err != nil {
		besideName := beside.Name()
		err = errors.Join(err, os.Remove(besideName))
	}
	return err
}
