// the C face of the movie bridge: what Go calls, in plain C types, so
// cgo never sees an Objective-C object. a movie is an opaque handle;
// every call that can fail writes why into err and returns non-zero.

#ifndef SLAYER_MOVIE_BRIDGE_H
#define SLAYER_MOVIE_BRIDGE_H

#include <stdint.h>

// what the writer is asked for, checked by Go before it gets here
typedef struct {
	const char *path;     // the temporary file, which must not exist
	int width;            // coded pixels across
	int height;           // coded pixels down, padding included
	int picture_height;   // the rows the caller's frames hold
	int rate;             // frames a second
	long bit_rate;        // video bits a second
	const char *profile;  // VideoToolbox's name, H264_High_4_1
	int timescale;        // ticks a second the video's times count in
	int keyframes;        // the most frames between keyframes
	int sound_rate;       // samples a second, or 0 for no audio track
	int channels;         // samples at each instant of the sound
	const float *sound;   // the whole sound, interleaved, copied by open
	long sound_count;     // pairs in it: the picture's length exactly
	long sound_chunk;     // pairs handed to the writer at a time
	long audio_bit_rate;  // AAC bits a second, both channels
	double pause;         // seconds between looks at a busy encoder
	double limit;         // seconds to wait for it before giving up
} slayer_movie_ask;

// slayer_movie_open starts an mp4 at ask->path. NULL on failure.
void *slayer_movie_open(const slayer_movie_ask *ask, char *err, int errlen);

// slayer_movie_frame encodes RGBA rows at stride bytes apart as frame
// index, shown at index/rate seconds. the sound is not handed over
// here: the writer pulls it, as it wants it, on a queue of its own.
int slayer_movie_frame(void *movie, long index, const uint8_t *rgba,
                       long stride, char *err, int errlen);

// slayer_movie_finish waits for all the sound to be handed over, ends
// the movie at frames/rate seconds, and waits for the file.
int slayer_movie_finish(void *movie, long frames, char *err, int errlen);

// slayer_movie_cancel stops writing; what was written is left for the
// caller to remove.
void slayer_movie_cancel(void *movie);

// slayer_movie_interrupt makes any wait the movie is in give up. it is
// safe from any thread until the movie is released.
void slayer_movie_interrupt(void *movie);

// slayer_movie_release lets the movie go.
void slayer_movie_release(void *movie);

// what slayer_movie_probe read from a file
typedef struct {
	int width, height;
	long frames;
	double rate;
	double seconds;
	char video[8];          // the codec's four letters
	int profile, level;     // from the avcC box; 0 when not H.264
	char primaries[64];
	char transfer[64];
	char matrix[64];
	long keyframe_gap;
	char audio[8];          // empty with no audio track
	double audio_rate;
	int audio_channels;
	double audio_seconds;
} slayer_movie_facts;

// slayer_movie_probe reads a file's facts, waiting at most limit
// seconds for it to load.
int slayer_movie_probe(const char *path, double limit,
                       slayer_movie_facts *facts, char *err, int errlen);

// slayer_movie_sound decodes a file's first audio track to float32
// samples at rate, interleaved, in the file's own channel count. the
// result is malloc'd, for the caller to free; NULL on failure.
float *slayer_movie_sound(const char *path, int rate, double limit,
                          long *samples, int *channels, char *err,
                          int errlen);

// what slayer_film_open read of a film it opened for its pictures
typedef struct {
	double seconds;  // the film's length
	double rate;     // its frames a second, nominal
	int width;       // its own pixels across
	int height;      // and down
} slayer_film_facts;

// slayer_film_open starts reading a film's pictures from from seconds,
// each scaled to width by height by the decoder. NULL on failure.
void *slayer_film_open(const char *path, double from, int width,
                       int height, double limit, slayer_film_facts *facts,
                       char *err, int errlen);

// slayer_film_next decodes the next picture into RGBA rows stride bytes
// apart and says its time in at. it returns slayer_film_frame for a
// picture, slayer_film_end when there are no more, and slayer_film_failed
// with why in err.
enum { slayer_film_frame = 0, slayer_film_failed = 1, slayer_film_end = 2 };
int slayer_film_next(void *film, uint8_t *rgba, long stride, double *at,
                     char *err, int errlen);

// slayer_film_release stops reading and lets the film go.
void slayer_film_release(void *film);

#endif
