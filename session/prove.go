package session

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// errReplayMedia is the refusal of a replay of a picture whose camera
// starts in an explicit stack of media.
var errReplayMedia = errors.New("initial media: pixel replay does not " +
	"support an explicit start stack")

// replayRefusal is why a picture drawn as given asks cannot be replayed
// pixel by pixel, or nil where it can.
func replayRefusal(given asked) error {
	if len(given.initialMedia) > 0 {
		return errReplayMedia
	}
	return nil
}

// replaying is the replay of one pixel of the picture world is drawn as
// given asks, seen from the eye given asks for: through the lens a rig
// gives, exposed over its instants, or through the lens and at the time
// the cpu draws it, on the exact tracer.
func replaying(world *thing.World, given asked, across float64) fermat.Replay {
	eye, lens := eyeFor(given, across), lensFor(given)
	return func(x, y int) (frame.Pixel, []trace.Step, error) {
		if given.glass != nil {
			return replayed(world, given, eye, x, y)
		}
		return trace.Replay(world, eye, lens, given.at, given.width,
			given.height, given.samples, x, y)
	}
}

// sizeRefusal is why a proof's raw is not a picture of the size given
// asks, or nil where it is.
func sizeRefusal(proof fermat.Proof, given asked) error {
	if proof.Width == given.width && proof.Height == given.height {
		return nil
	}
	return fmt.Errorf("%s is %d by %d, and these flags draw %d by %d",
		proof.Raw, proof.Width, proof.Height, given.width, given.height)
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
	err := replayRefusal(given)
	var proof fermat.Proof
	if err == nil {
		asked := "hillside " + strings.Join(arguments, " ")
		proof, err = fermat.Prove(given.prove, asked, given.pixels,
			given.many, replaying(world, given, across))
	}
	if err == nil {
		err = sizeRefusal(proof, given)
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

// Proving is what a raw is proved with: where the raw is, the pixels
// named to be proved first, how many more the raw's own bytes pick, and
// the words the proof records as what was asked.
type Proving struct {
	// Raw is the path of the raw, a pfm, to prove.
	Raw string
	// Pixels are the pixels named, each X,Y from the top left.
	Pixels [][2]int
	// Many is how many more pixels the raw's bytes pick; fermat.Many is
	// the usual. Nought is none, and refused where no pixel is named.
	Many int
	// Asked is what the proof records as what was asked of it.
	Asked string
}

// stager is a source that says a frame's world, eye, lens and time as it
// draws them, without drawing: the steps its light takes first.
type stager interface {
	staged(at time.Duration, given asked) (drawnLight, error)
}

// errProofOfNothing is the refusal of a proof that names no pixel and
// picks none.
var errProofOfNothing = errors.New("hillside: a proof of nothing: name " +
	"a pixel, or have the raw's bytes pick one or more")

// errProveOnGPU is the refusal of a proof of a picture drawn on the gpu.
var errProveOnGPU = errors.New("hillside: this session draws on the gpu, " +
	"whose light is not the exact tracer's, so there is nothing to hold " +
	"it to bit for bit; draw on the cpu to prove it")

// errProveShutter is the refusal of a proof of a picture that is the
// mean of the glimpses of a shutter.
var errProveShutter = errors.New("hillside: this film's frame is the mean " +
	"of the glimpses of its shutter, which one pixel replay does not " +
	"make; prove a frame of a film with no shutter")

// errProveSource is the refusal of a proof of frames no world is staged
// for: a film through a set's rig, whose frame is its glimpses exposed
// through the lens, each drawn at its own moment.
var errProveSource = errors.New("hillside: this session's frames are not " +
	"drawn from a world that can be replayed")

// Prove replays pixels of the raw proving names, as frame index draws
// them on the cpu: the same world, eye, lens, film time, size and rays
// Frame draws with. The pixels named are replayed first, then as many
// more as the raw's bytes pick, each held to the raw bit for bit.
//
// It refuses a session that draws on the gpu, a camera that starts in an
// explicit stack of media, a frame that is the mean of a shutter's
// glimpses, a film through a set's rig, and a raw of another size than
// the frame's, each by name. A proof that fails is a proof, not an
// error; Holds says which.
func (session *Session) Prove(index int, proving Proving) (fermat.Proof,
	error) {
	if err := session.provable(index, proving); err != nil {
		return fermat.Proof{}, err
	}
	given, err := session.asking(Ask{})
	if err != nil {
		return fermat.Proof{}, err
	}
	staged, err := session.replayable(index, given, proving.Raw)
	if err != nil {
		return fermat.Proof{}, err
	}
	proof, err := fermat.Prove(proving.Raw, proving.Asked, proving.Pixels,
		proving.Many, replaying(staged.world, staged.given,
			staged.made.across))
	if err == nil {
		err = sizeRefusal(proof, staged.given)
	}
	if err != nil {
		return fermat.Proof{}, err
	}
	return proof, nil
}

// provable is why frame index cannot be proved as proving asks, before
// anything is staged: a closed session, no such frame, a proof of
// nothing, or a session that draws on the gpu.
func (session *Session) provable(index int, proving Proving) error {
	if session.given.detail != nil {
		return fmt.Errorf("scene detail is available only as a working view")
	}
	if session.closed {
		return errClosed
	}
	if index < nothing || index >= len(session.times) {
		return fmt.Errorf("hillside: no frame %d; the job has %d, "+
			"counted from zero", index, len(session.times))
	}
	if proving.Many < nothing || proving.Many == nothing &&
		len(proving.Pixels) == nothing {
		return errProofOfNothing
	}
	if session.given.onGPU {
		return errProveOnGPU
	}
	return nil
}

// replayable is frame index staged as Frame draws it, and what the
// frame is asked there, refused where it cannot be replayed: its source
// stages no world, its camera starts in an explicit stack of media, its
// shortcuts are not the exact tracer's, or it is a shutter's mean. The
// raw is the one a proof is made of, which a shortcut's refusal names.
//
// A rig's lens takes its own instants from the frame's time; any other
// frame is drawn at its one glimpse's time.
func (session *Session) replayable(index int, given asked,
	raw string) (drawnLight, error) {
	source, stages := session.source.(stager)
	if !stages {
		return drawnLight{}, errProveSource
	}
	at := session.times[index]
	if given.lensed == nil && len(session.offsets) > singleRay {
		return drawnLight{}, errProveShutter
	}
	if given.lensed == nil && len(session.offsets) == singleRay {
		at += session.offsets[nothing]
	}
	staged, err := source.staged(at, given)
	if err != nil {
		return drawnLight{}, err
	}
	staged.given.prove = raw
	for _, refusal := range []error{replayRefusal(staged.given),
		fastRefusal(staged.given)} {
		if refusal != nil {
			return drawnLight{}, fmt.Errorf("hillside: %w", refusal)
		}
	}
	return staged, nil
}

// Sheet writes the proof's sheet in dir, or beside the proof's raw,
// named for it, where dir is empty: the proof's pages, and the raw
// developed as a print is with the proved pixels marked on it. It says
// where the sheet went.
func (session *Session) Sheet(proof fermat.Proof, dir string) (string,
	error) {
	dir = sheetFor(asked{prove: proof.Raw, proof: dir})
	return dir, sheet(proof, dir)
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
