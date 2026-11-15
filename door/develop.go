package door

// develop prints light already drawn, so a frame drawn once can be
// developed many ways without drawing again. It takes a job, for its
// print, stock and titles, and raws: .pfm files, or a folder of plates
// as a frames scene reads them (render/finish).
//
// Each raw is a frame of its own, developed through the job's stock or
// print, as draw develops, with the job's titles at its film time. With
// --sum the raws are added light to light into one long exposure, as
// cmd/develop did. --white balances the light first.

import (
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/finish"
	"github.com/janearc/slayer/render/frame"
)

// developHelp is develop's page.
const developHelp = `# develop

slayer develop JOB RAW... develops light already drawn, through the
job's print or stock, with the job's titles, as draw develops it. RAW
is one or more .pfm raws, or one folder of plates as a frames scene
reads them. JOB is a job, a plan or a set file, read as draw reads it;
only its print, stock, titles and rate are used.

Each raw is a frame of its own, developed and written by itself; this
is a change from cmd/develop, which added several raws into one. A
raw named as draw names frames, frame-000042.pfm, is frame 42: its
grain and its titles' time are frame 42's. Raws named otherwise are
numbered from --frame N upward, or from 0, and the line says so.

With --sum the raws are added light to light into one long exposure,
as cmd/develop did, and developed as one picture. --white balances the
light so that a black body at that many kelvin prints neutral; leave
it out to leave the light alone (0 no longer means that).

One picture goes to out/NAME.png, NAME the raw's; several go to a
folder, out/JOB/, each named for its raw. --out says the picture or the
folder; for one picture, a folder there already gets FOLDER/NAME.png.
A job with no print and no stock develops through the tone cmd/develop
used: a knee at 0.7 and a top at 8 whites.

--dry-run says each picture it would write and writes nothing.
`

// jobAndRaw is the fewest words develop takes: a job and one raw.
// Defined.
const jobAndRaw = 2

// The names of develop's flags.
const (
	sumFlag   = "sum"
	whiteFlag = "white"
	frameFlag = "frame"
)

// frameStem is a frame's file name as draw writes it, without its
// extension: a raw named so says its own frame number.
var frameStem = strings.TrimSuffix(frameName, pictureExtension)

// The tone a job with no print and no stock develops through: the
// orbital's own development, picked for it and kept by cmd/develop.
const (
	plainKnee = 0.7
	plainTop  = 8
)

// developing is the develop verb, with what it runs under.
type developing struct {
	env  Env
	shop workshop
}

// plate is one raw: its name, how its light is read, its frame number
// and whether that number was its name's or --frame's (said) or only
// counted from nought (unsaid).
type plate struct {
	name   string
	read   func() (*frame.Frame, error)
	number int
	said   bool
}

// sheet is one picture to write: its path and the plates it is made
// from, one, or several summed; it is numbered by its first.
type sheet struct {
	path   string
	plates []plate
}

// develop is the verb: the job read, the raws found, the prints worked
// out; then a dry run says them, or each is developed and written.
func (held developing) develop(call pzm.Call) error {
	if len(call.Args()) < jobAndRaw {
		return errors.New("develop takes a job and its raws: slayer " +
			"develop JOB RAW..., or JOB FOLDER")
	}
	given := resolved(call.Args()[nothing], held.env.directory)
	wanted, err := jobOf(given)
	if err != nil {
		return err
	}
	plates, err := platesOf(call.Args()[one:], held.env.directory)
	if err != nil {
		return err
	}
	kelvin, err := whiteOf(call)
	if err != nil {
		return err
	}
	plates, err = numbered(call, plates)
	if err != nil {
		return err
	}
	prints, err := held.prints(call, given, plates)
	if err != nil {
		return err
	}
	if call.DryRun() {
		return wouldDevelop(call.Out(), prints)
	}
	made, err := held.openDarkroom(wanted, kelvin)
	if err != nil {
		return err
	}
	for _, each := range prints {
		if err := made.print(call.Out(), each); err != nil {
			return err
		}
	}
	return nil
}

// platesOf is the raws named: one folder's plates, or each file a raw.
func platesOf(named []string, directory string) ([]plate, error) {
	first := resolved(named[nothing], directory)
	if held, err := os.Stat(first); err == nil && held.IsDir() {
		if len(named) > one {
			return nil, errors.New("develop takes one folder of plates, " +
				"or raws; not a folder and more")
		}
		return folderPlates(first)
	}
	var plates []plate
	for _, name := range named {
		path := resolved(name, directory)
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("the raw %s is not there: %w", name,
				err)
		}
		plates = append(plates, plate{name: path,
			read: func() (*frame.Frame, error) { return raw.Read(path) }})
	}
	return plates, nil
}

// folderPlates is a folder's plates, in name order, as finish reads
// them.
func folderPlates(folder string) ([]plate, error) {
	opened, err := finish.OpenPlates(folder)
	if err != nil {
		return nil, err
	}
	var plates []plate
	for i, name := range opened.Names() {
		place := i
		plates = append(plates, plate{name: name,
			read: func() (*frame.Frame, error) {
				return opened.Frame(place)
			}})
	}
	return plates, nil
}

// whiteOf is the white balance asked for, in kelvin, or nought for
// none; a value that is not a temperature is refused by name.
func whiteOf(call pzm.Call) (float64, error) {
	if !call.Given(whiteFlag) {
		return nothing, nil
	}
	kelvin, err := strconv.ParseFloat(call.Value(whiteFlag), 64)
	if err != nil || !(kelvin > nothing) || math.IsInf(kelvin, one) {
		return nothing, fmt.Errorf("--white takes a temperature in "+
			"kelvin above nought, not %q", call.Value(whiteFlag))
	}
	return kelvin, nil
}

// numbered is the plates with their frame numbers: a name in draw's
// frame naming says its own; the rest count up from --frame, or from
// nought, unsaid.
func numbered(call pzm.Call, plates []plate) ([]plate, error) {
	next, err := wholeFlag(call, frameFlag, nothing)
	if err != nil || next < nothing {
		return nil, fmt.Errorf("--frame takes a frame number, nought or "+
			"more, not %q", call.Value(frameFlag))
	}
	said := call.Given(frameFlag)
	for i := range plates {
		if number, named := frameNumber(plates[i].name); named {
			plates[i].number, plates[i].said = number, true
			continue
		}
		plates[i].number, plates[i].said = next, said
		next++
	}
	return plates, nil
}

// frameNumber is the frame a file's name says, in draw's frame naming,
// and whether it says one.
func frameNumber(path string) (int, bool) {
	var number int
	stem := baseName(path)
	if _, err := fmt.Sscanf(stem, frameStem, &number); err != nil {
		return nothing, false
	}
	return number, fmt.Sprintf(frameStem, number) == stem
}

// prints is each picture to write: one for one raw or for raws summed,
// at --out, in --out when it is a folder, or at out/NAME.png; one a raw
// in a folder otherwise, --out or out/JOB/, named for its raw. Two raws
// that would write one picture are refused by name.
func (held developing) prints(call pzm.Call, given string,
	plates []plate) ([]sheet, error) {
	if call.On(sumFlag) || len(plates) == one {
		name := baseName(plates[nothing].name) + pictureExtension
		path := filepath.Join(held.env.directory, outFolder, name)
		if call.Given("out") {
			path = resolved(call.Value("out"), held.env.directory)
		}
		if found, err := os.Stat(path); err == nil && found.IsDir() {
			path = filepath.Join(path, name)
		}
		return []sheet{{path: path, plates: plates}}, nil
	}
	folder := filepath.Join(held.env.directory, outFolder,
		baseName(given))
	if call.Given("out") {
		folder = resolved(call.Value("out"), held.env.directory)
	}
	prints := make([]sheet, nothing, len(plates))
	from := map[string]string{}
	for _, each := range plates {
		path := filepath.Join(folder, baseName(each.name)+pictureExtension)
		if first, taken := from[path]; taken {
			return nil, fmt.Errorf("%s and %s would both be written to "+
				"%s; give raws with different names, or develop them "+
				"one at a time with --out", first, each.name, path)
		}
		from[path] = each.name
		prints = append(prints, sheet{plates: []plate{each}, path: path})
	}
	return prints, nil
}

// baseName is a path's base name without its extension.
func baseName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// wouldDevelop says each picture develop would write, and writes none.
func wouldDevelop(out io.Writer, prints []sheet) error {
	var lines []string
	for _, each := range prints {
		line := "would write " + each.path + " from " +
			filepath.Base(each.plates[nothing].name) + " as " +
			each.frameWords()
		if len(each.plates) > one {
			line += fmt.Sprintf(" and %d more, summed",
				len(each.plates)-one)
		}
		lines = append(lines, line)
	}
	_, err := fmt.Fprintln(out, strings.Join(lines, "\n"))
	return err
}

// darkroom is what a print is developed with: the job's development, its
// titles if it says any, its rate, and the white balance.
type darkroom struct {
	developer developer
	titles    titler
	rate      int
	kelvin    float64
}

// openDarkroom opens the job's development and titles.
func (held developing) openDarkroom(wanted job.Job, kelvin float64) (darkroom,
	error) {
	plain := colour.Tone{Exposure: unitExposure, Knee: plainKnee,
		Top: plainTop}
	developed, err := held.shop.Developer(wanted, plain)
	if err != nil {
		return darkroom{}, err
	}
	titles, has, err := held.shop.Titler(wanted)
	if err != nil {
		return darkroom{}, err
	}
	made := darkroom{developer: developed, kelvin: kelvin}
	if has {
		made.titles = titles
	}
	made.rate, _ = wanted.Clock().Rate()
	return made, nil
}

// frameWords is the sheet's frame number in words, and, when nothing
// said it, what to type to say it.
func (each sheet) frameWords() string {
	first := each.plates[nothing]
	words := fmt.Sprintf("frame %d", first.number)
	if !first.said {
		words += " (its name holds no frame number; say --frame N to " +
			"number it)"
	}
	return words
}

// print develops one picture, the frame its first plate's number,
// titles it and writes it.
func (room darkroom) print(out io.Writer, each sheet) error {
	number := each.plates[nothing].number
	light, err := summed(each.plates)
	if err != nil {
		return err
	}
	if room.kelvin != nothing {
		if err := balance(light, room.kelvin); err != nil {
			return err
		}
	}
	picture, err := room.developer.Develop(number, []*frame.Frame{light})
	if err != nil {
		return err
	}
	if err := room.title(picture, number); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(each.path), folderMode); err != nil {
		return err
	}
	if err := file.PNG(each.path, picture, square); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "wrote "+each.path+" as "+each.frameWords())
	return err
}

// title draws the job's titles on a frame at its film time: nought for
// the first, or a job with no rate.
func (room darkroom) title(picture *image.RGBA, number int) error {
	if room.titles == nil {
		return nil
	}
	var at time.Duration
	if room.rate > nothing {
		at = filmTime(number, room.rate)
	}
	return room.titles.Draw(picture, at)
}

// summed is the plates' light, added light to light: one plate is its
// own light.
func summed(plates []plate) (*frame.Frame, error) {
	frames := make([]*frame.Frame, nothing, len(plates))
	for _, each := range plates {
		light, err := each.read()
		if err != nil {
			return nil, err
		}
		frames = append(frames, light)
	}
	return raw.Sum(frames)
}

// balance scales a frame's red and blue so that a black body at the
// temperature comes out neutral, its green kept: what a camera's white
// balance does. A temperature so low its light has no red or no blue is
// refused, and the light left as it was.
func balance(light *frame.Frame, kelvin float64) error {
	spectrum := bands.Split(swatch.Blackbody(kelvin))
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	if !(red > nothing && blue > nothing) {
		return fmt.Errorf("--white %v: a black body that cold has no "+
			"white to balance to; give a warmer temperature", kelvin)
	}
	redGain, blueGain := float32(green/red), float32(green/blue)
	for i := range light.Pix {
		light.Pix[i].R *= redGain
		light.Pix[i].B *= blueGain
	}
	return nil
}

// declareDevelop adds develop to the tool: it writes its pictures.
func declareDevelop(made *pzm.Tool, env Env, shop workshop) {
	writes := verb.WritesTo("the developed pictures, at --out or " +
		"under out/")
	prints := verb.NewFacts(writes, verb.Changes, verb.ThisMachine,
		verb.NoOne, verb.Light)
	held := developing{env: env, shop: shop}
	made.Verb("develop", verb.NoShort, "turn raws already drawn into "+
		"pictures through a job's print or stock", developHelp, "Example_develop",
		prints, held.develop,
		pzm.Value("out", verb.NoShort, "the picture, or the folder of "+
			"pictures"),
		pzm.Switch(sumFlag, verb.NoShort, "add the raws into one long "+
			"exposure"),
		pzm.Value(whiteFlag, verb.NoShort, "white balance, in kelvin"),
		pzm.Value(frameFlag, verb.NoShort, "the first unnumbered raw's "+
			"frame number"))
}
