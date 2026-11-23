//go:build darwin && cgo

package movie

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework CoreMedia
#cgo LDFLAGS: -framework CoreVideo -framework VideoToolbox
#cgo LDFLAGS: -framework Foundation
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"image"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// profilePrefix is how VideoToolbox names an H.264 High profile level:
// H264_High_4_1 for 4.1. Defined by VTCompressionProperties.h.
const profilePrefix = "H264_High_"

// bridge is a movie being written through AVFoundation. The writer
// calls frame, finish, cancel and release one at a time; interrupt may
// come from any goroutine, so it and release share a lock.
type bridge struct {
	mu     sync.Mutex
	handle unsafe.Pointer
}

// openPipe starts writing the movie the settings describe to temporary.
func openPipe(settings Settings, temporary string) (pipe, error) {
	path := C.CString(temporary)
	defer C.free(unsafe.Pointer(path))
	name := profilePrefix + strings.ReplaceAll(settings.level.name, ".", "_")
	profile := C.CString(name)
	defer C.free(unsafe.Pointer(profile))
	ask := C.slayer_movie_ask{
		path:           path,
		width:          C.int(settings.coded.X),
		height:         C.int(settings.coded.Y),
		picture_height: C.int(settings.picture.Y),
		rate:           C.int(settings.rate),
		bit_rate:       C.long(settings.bitRate),
		profile:        profile,
		timescale:      C.int(settings.rate * ticksPerFrame),
		keyframes:      C.int(keyframesWithin),
		sound_rate:     C.int(settings.sound.rate),
		channels:       C.int(channels),
		audio_bit_rate: C.long(audioBitRate),
		pause:          C.double(waitPause.Seconds()),
		limit:          C.double(waitLimit.Seconds()),
	}
	if settings.sound.present() {
		// the whole sound, cut or padded to the picture, copied by open;
		// pinned, since the ask that points at it is Go's memory too
		heard := settings.sound.span(0, settings.frames, settings.rate)
		var pinner runtime.Pinner
		pinner.Pin(&heard[0])
		defer pinner.Unpin()
		ask.sound = (*C.float)(unsafe.Pointer(&heard[0]))
		ask.sound_count = C.long(len(heard) / channels)
		ask.sound_chunk = C.long(soundChunk)
	}
	why := make([]C.char, whyLength)
	handle := C.slayer_movie_open(&ask, &why[0], C.int(len(why)))
	if handle == nil {
		return nil, errors.New("movie: " + C.GoString(&why[0]))
	}
	return &bridge{handle: handle}, nil
}

// frame hands over one picture, shown at index/rate seconds.
func (movie *bridge) frame(index int, picture *image.RGBA) error {
	rows := (*C.uint8_t)(unsafe.Pointer(&picture.Pix[0]))
	why := make([]C.char, whyLength)
	failed := C.slayer_movie_frame(movie.handle, C.long(index), rows,
		C.long(picture.Stride), &why[0], C.int(len(why)))
	if failed != 0 {
		return errors.New("movie: " + C.GoString(&why[0]))
	}
	return nil
}

// finish ends the movie after frames pictures and waits for the file.
func (movie *bridge) finish(frames int) error {
	why := make([]C.char, whyLength)
	failed := C.slayer_movie_finish(movie.handle, C.long(frames), &why[0],
		C.int(len(why)))
	if failed != 0 {
		return errors.New("movie: " + C.GoString(&why[0]))
	}
	return nil
}

// cancel stops writing.
func (movie *bridge) cancel() { C.slayer_movie_cancel(movie.handle) }

// interrupt makes any wait give up; after release it does nothing.
func (movie *bridge) interrupt() {
	movie.mu.Lock()
	defer movie.mu.Unlock()
	if movie.handle != nil {
		C.slayer_movie_interrupt(movie.handle)
	}
}

// release lets the movie go, once.
func (movie *bridge) release() {
	movie.mu.Lock()
	defer movie.mu.Unlock()
	if movie.handle != nil {
		C.slayer_movie_release(movie.handle)
		movie.handle = nil
	}
}

// probeFile reads a movie's facts through AVFoundation.
func probeFile(path string) (Facts, error) {
	where := C.CString(path)
	defer C.free(unsafe.Pointer(where))
	var read C.slayer_movie_facts
	why := make([]C.char, whyLength)
	if C.slayer_movie_probe(where, C.double(waitLimit.Seconds()), &read,
		&why[0], C.int(len(why))) != 0 {
		return Facts{}, errors.New("movie: " + C.GoString(&why[0]))
	}
	return factsOf(&read), nil
}

// factsOf is the facts the bridge read, in Go's types.
func factsOf(read *C.slayer_movie_facts) Facts {
	return Facts{
		width:         int(read.width),
		height:        int(read.height),
		frames:        int(read.frames),
		rate:          float64(read.rate),
		duration:      seconds(float64(read.seconds)),
		video:         codecName(C.GoString(&read.video[0])),
		profile:       int(read.profile),
		level:         int(read.level),
		primaries:     C.GoString(&read.primaries[0]),
		transfer:      C.GoString(&read.transfer[0]),
		matrix:        C.GoString(&read.matrix[0]),
		keyframeGap:   int(read.keyframe_gap),
		audio:         codecName(C.GoString(&read.audio[0])),
		audioRate:     float64(read.audio_rate),
		audioChannels: int(read.audio_channels),
		audioDuration: seconds(float64(read.audio_seconds)),
	}
}

// decodeSound decodes a file's first audio track at rate, and says how
// many channels the samples are interleaved over.
func decodeSound(path string, rate int) ([]float32, int, error) {
	where := C.CString(path)
	defer C.free(unsafe.Pointer(where))
	var count C.long
	var held C.int
	why := make([]C.char, whyLength)
	out := C.slayer_movie_sound(where, C.int(rate),
		C.double(waitLimit.Seconds()), &count, &held, &why[0],
		C.int(len(why)))
	if out == nil {
		return nil, 0, errors.New("movie: " + C.GoString(&why[0]))
	}
	defer C.free(unsafe.Pointer(out))
	samples := make([]float32, int(count))
	copy(samples, unsafe.Slice((*float32)(unsafe.Pointer(out)), int(count)))
	return samples, int(held), nil
}

// filmBridge is a film being read for its pictures through
// AVFoundation, one picture at a time, from one goroutine.
type filmBridge struct {
	handle unsafe.Pointer
}

// openFilm starts reading a film's pictures from a moment, each scaled
// by the decoder to a size, and says what the film is.
func openFilm(path string, from float64, size image.Point) (pictures,
	filmFacts, error) {
	where := C.CString(path)
	defer C.free(unsafe.Pointer(where))
	var read C.slayer_film_facts
	why := make([]C.char, whyLength)
	handle := C.slayer_film_open(where, C.double(from), C.int(size.X),
		C.int(size.Y), C.double(waitLimit.Seconds()), &read, &why[0],
		C.int(len(why)))
	if handle == nil {
		return nil, filmFacts{}, errors.New("movie: " + C.GoString(&why[0]))
	}
	facts := filmFacts{seconds: float64(read.seconds),
		rate: float64(read.rate),
		size: image.Pt(int(read.width), int(read.height))}
	return &filmBridge{handle: handle}, facts, nil
}

// next decodes the next picture into a frame of the asked size, and says
// its time in seconds, or that there are no more.
func (film *filmBridge) next(into *image.RGBA) (float64, bool, error) {
	rows := (*C.uint8_t)(unsafe.Pointer(&into.Pix[0]))
	var at C.double
	why := make([]C.char, whyLength)
	switch C.slayer_film_next(film.handle, rows, C.long(into.Stride), &at,
		&why[0], C.int(len(why))) {
	case C.slayer_film_frame:
		return float64(at), true, nil
	case C.slayer_film_end:
		return 0, false, nil
	}
	return 0, false, errors.New("movie: " + C.GoString(&why[0]))
}

// release stops reading, once.
func (film *filmBridge) release() {
	if film.handle != nil {
		C.slayer_film_release(film.handle)
		film.handle = nil
	}
}
