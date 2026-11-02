package main

// The orbital as a stream of terminal cells, for -cells.

import (
	"errors"
	"flag"
	"io"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/cells"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
)

// startsAt is when frame zero of the scene happens. It is defined as half a
// second in, as the pictures are.
const startsAt = 500 * time.Millisecond

// square is the pixel shape of a picture file. It is defined as zero, which
// film.Pinhole reads as square.
const square = 0

// the flags that name a file, which cells do not write
const (
	outFlag = "out"
	rawFlag = "raw"
)

// errFiles is what -cells says to a flag that names a file.
var errFiles = errors.New("-cells writes no file: -out and -raw are for " +
	"pictures")

// given is the names of the flags that were set on the command line.
type given map[string]bool

// note records one flag as set.
func (found given) note(one *flag.Flag) { found[one.Name] = true }

// givenFlags is the flags set on the command line.
func givenFlags() given {
	found := given{}
	flag.Visit(found.note)
	return found
}

// errKeysAlone is what -keys says when there is no -cells to wait between.
var errKeysAlone = errors.New("-keys is for -cells: it waits for a key " +
	"between frames of terminal cells")

// cellsFlags is what the flags say about drawing to cells. With keys, in
// and out are the terminal the keys come from and the frames go to.
type cellsFlags struct {
	size          string
	set           given
	first, frames int
	rate, rays    int
	printed       printing
	keys          bool
	in, out       *os.File
}

// cellsMode is the orbital drawn as cells, if the flags asked for it.
// With in it is drawn a key at a time, and exit is how a signal ends it.
type cellsMode struct {
	on     bool
	scene  *cells.Scene
	stream *terminal.Stream
	in     *os.File
	exit   func(int)
}

// shellExit is what a shell adds to a signal's number to say a command
// died of it. It is defined as 128.
const shellExit = 128

// signalRoom is how many signals the watch holds. It is defined as one:
// the first ends the session.
const signalRoom = 1

// hangup is a watch for the signals that stop a process, kept while a
// terminal is taken over.
type hangup struct {
	caught  chan os.Signal
	done    chan struct{}
	over    chan struct{}
	session *terminal.Session
	exit    func(int)
}

// leaveOnSignal answers SIGTERM, SIGINT and SIGHUP by giving the terminal
// back and exiting 128 plus the signal. Call stop when the session ends.
func leaveOnSignal(session *terminal.Session, exit func(int)) *hangup {
	watch := &hangup{caught: make(chan os.Signal, signalRoom),
		done: make(chan struct{}), over: make(chan struct{}),
		session: session, exit: exit}
	signal.Notify(watch.caught, syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGHUP)
	go watch.wait()
	return watch
}

// wait is the watch: it ends the session on a signal, or ends with stop.
func (watch *hangup) wait() {
	defer close(watch.over)
	select {
	case got := <-watch.caught:
		watch.session.End()
		watch.exit(shellExit + int(got.(syscall.Signal)))
	case <-watch.done:
	}
}

// stop ends the watch, and waits until it has ended.
func (watch *hangup) stop() {
	signal.Stop(watch.caught)
	close(watch.done)
	<-watch.over
}

// beams tells the lantern's beams the time of the frame about to be drawn.
type beams struct{}

// Set is the time the beams are drawn at.
func (beams) Set(at time.Duration) { beamClock = at }

// askCells reads the flags for -cells. Building a scene takes seconds.
// So every refusal that can be made is made here, before it is. With no
// size given the mode is off, and refused if it was to wait for keys.
func askCells(flags cellsFlags) (cellsMode, error) {
	if flags.size == "" && flags.keys {
		return cellsMode{}, errKeysAlone
	}
	if flags.size == "" {
		return cellsMode{}, nil
	}
	if flags.set[outFlag] || flags.set[rawFlag] {
		return cellsMode{}, errFiles
	}
	size, err := terminal.ParseSize(flags.size)
	if err != nil {
		return cellsMode{}, err
	}
	var in *os.File
	if flags.keys {
		if err := terminal.Check(flags.in, flags.out); err != nil {
			return cellsMode{}, err
		}
		in = flags.in
	}
	pace, err := cells.NewPace(startsAt, flags.rate)
	if err != nil {
		return cellsMode{}, err
	}
	lens := lensFor(flags.printed, terminal.SextantPixel)
	scene, err := cells.New(lens, developer(flags.printed), flags.rays,
		pace, beams{})
	if err != nil {
		return cellsMode{}, err
	}
	stream, err := terminal.NewStream(size, flags.first, flags.frames,
		scene)
	if err != nil {
		return cellsMode{}, err
	}
	return cellsMode{on: true, scene: scene, stream: stream, in: in,
		exit: os.Exit}, nil
}

// chat is where the orbital says what it is doing. Frames own standard
// output in cells mode, so it is standard error there.
func (mode cellsMode) chat() io.Writer {
	if mode.on {
		return os.Stderr
	}
	return os.Stdout
}

// run places the world and the eye in the scene and writes the stream to
// out until it ends or out will not take more. With keys it waits for one
// between frames. It takes the terminal then, and gives it back.
func (mode cellsMode) run(out *os.File, world *thing.World,
	eye maths.Subspace) error {
	mode.scene.Place(world, eye)
	if mode.in == nil {
		return mode.stream.Run(out)
	}
	session, err := terminal.Open(mode.in, out)
	if err != nil {
		return err
	}
	defer session.End()
	defer leaveOnSignal(session, mode.exit).stop()
	return mode.stream.RunBy(out, session.Next)
}

// lensFor is the pinhole the printing asks for, its pixels as wide over
// high as pixel says.
func lensFor(printed printing, pixel float64) film.Pinhole {
	//ninja:vow stranger film has no maker; ninjafilm's camera replaces it
	return film.Pinhole{Wide: printed.wide, Pixel: pixel}
}

// developer is how light becomes a picture: at the printing's exposure,
// with the knee and top the orbital has always used.
func developer(printed printing) colour.Tone {
	//ninja:vow table the orbital's own knee and top, kept as they were
	return colour.Tone{Exposure: math.Exp2(printed.stops), //ninja:vow stranger ninjafilm's darkroom replaces it
		Knee: 0.7, Top: 8}
}
