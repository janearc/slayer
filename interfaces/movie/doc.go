// Package movie writes Slayer's frames, and a soundtrack, into an mp4
// that QuickTime Player plays. It runs no other program: on macOS it
// talks to AVFoundation through cgo. Anywhere else every call that would
// touch a file returns ErrNeedsMacOS, and the checks still run.
//
// # What it writes
//
// H.264 High profile video and, when a Sound is given, stereo AAC audio,
// in one mp4. The rules below were measured on this machine: a file that
// broke any of them decoded cleanly everywhere and still failed to play
// in QuickTime.
//
//   - The coded height is a multiple of 16. Any other height is refused
//     by name, unless the Request sets Pad, and then black rows are added
//     at the foot to reach the next multiple. Nothing is padded unasked.
//   - The width is even.
//   - The colour is tagged BT.709: primaries, transfer and matrix. The
//     frames are sRGB-encoded bytes and are coded as they are.
//   - A keyframe comes at least every 48 frames.
//   - The level is the lowest that the size, the rate and the bit rate
//     fit, from ITU-T H.264 table A-1 (levels.csv). A request that would
//     need more than 5.2 is refused by name: a 4K file at 6.0 showed as
//     blocks in QuickTime.
//   - The index is at the front, so playback starts before the file has
//     all arrived.
//   - It is written to a hidden sibling file and renamed into place on
//     Close. Abort, a refused frame or a cancelled context removes that
//     file and leaves nothing.
//
// # How it is used
//
// NewSettings checks a Request before anything starts. New opens the
// writer. Frame takes the pictures in order from index 0, each an
// *image.RGBA of exactly the requested size; alpha is not coded.
//
// Close finishes and publishes the file; Abort throws it away. One of
// the two is always called. A refused frame fails the writer and
// removes what it wrote; Close then returns the refusal.
//
// A cancelled context stops any wait the encoder is in, and the next
// call aborts. Every wait sleeps between looks, and gives up after a
// minute with the reason.
//
// The rate is a whole number of frames a second, so frame i is shown
// at exactly i/rate seconds. 24 and 30 are offered; 29.97 is not.
//
// # Sound
//
// A Sound is stereo float32 samples in -1..1, interleaved left then
// right, at 44100 or 48000 a second. No Sound means no audio track.
//
// Sound longer than the picture is cut at the picture's end. Shorter
// sound is followed by silence to the picture's end. Either way the
// audio track is as long as the picture, to within one AAC packet.
//
// The writer pulls the sound as it wants it, on a queue of its own,
// while the pictures come in on the caller's; it lays the two side by
// side in the file. Handed over in step with the pictures from one
// thread, a 1080x1920 movie stalled at frame 57: neither input took more.
//
// ReadSound reads any file AVFoundation reads (wav, m4a, mp3, aiff, the
// mp4s written here) into a Sound at 44100 a second. Mono is put on
// both channels; more than two channels are refused. Decoded samples
// past full scale, which AAC can make, are clipped to it.
//
// # Reading a file back
//
// Probe reads a finished file and reports its Facts: size, frame count,
// rate, duration, codec, profile, level, colour tags, the longest run
// between keyframes, whether the index is at the front, and the audio
// track's codec, rate, channels and duration.
//
// The tests use Probe to check every rule above without another tool.
//
// # Reading a film's pictures
//
// OpenFilm reads a film's first video track from a moment, in order,
// each picture an *image.RGBA scaled by the decoder to the size asked
// (shape is the caller's to keep). The first picture Next gives is the
// one showing at the moment asked; FrameAt is that one picture. Past the
// last picture, or opened at or after the film's end, is ErrPastEnd.
//
// Any codec AVFoundation decodes is read, HEVC included: a 4096 by 3008
// HEVC film reads at about 5 ms a picture scaled to 1080 across.
//
// Read back, a film written here is within a few steps in 255 of what
// was written: H.264's 4:2:0 colour halves the colour's resolution, and
// the encoder rounds. The tests hold it to film-error.
//
// # Kept in step with
//
// The compositor's encoder (improv/moonlake/omnipositor/encoder) has
// the same contract: settings checked first, frames in order by index,
// Close or Abort always, a temporary file then publication, and
// statistics.
//
// It pads the height top and bottom; this package pads at the foot, as
// asked, and only when told to.
package movie
