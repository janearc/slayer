// the Objective-C side of the movie bridge: an AVAssetWriter with an
// input for pictures and one for sound, held in an object Go only ever
// sees as a pointer. built with ARC. every entry point is under its own
// autorelease pool and turns an Objective-C exception into an error.

#import <AVFoundation/AVFoundation.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import <Foundation/Foundation.h>
#include <stdatomic.h>
#include <stdlib.h>
#include <string.h>
#include "bridge.h"

// the bytes of a pixel, in Go's RGBA order and the encoder's BGRA order
enum {
	pixel_bytes = 4,
	rgba_red = 0,
	rgba_green = 1,
	rgba_blue = 2,
	rgba_alpha = 3,
	bgra_blue = 0,
	bgra_green = 1,
	bgra_red = 2,
	bgra_alpha = 3,
	opaque = 0xff,
};

// where an avcC box keeps the profile and the level: ISO/IEC 14496-15,
// 5.3.3.1, after the one-byte configuration version
enum {
	avcc_profile = 1,
	avcc_level = 3,
	avcc_least = 4,
};

// the four letters of a codec, and the nul after them
enum { fourcc_letters = 4 };

// a movie being written. the pictures come from Go's thread; the sound
// is pulled by the writer, when it wants it, on a queue of its own, so
// neither input waits on the other.
@interface SlayerMovie : NSObject {
@public
	atomic_int stopped;   // set by slayer_movie_interrupt from any thread
	atomic_int abandoned; // set by slayer_movie_cancel: pull no more sound
	atomic_int deaf;      // set when the sound could not be handed over
}
@property(nonatomic, strong) AVAssetWriter *writer;
@property(nonatomic, strong) AVAssetWriterInput *video;
@property(nonatomic, strong) AVAssetWriterInputPixelBufferAdaptor *adaptor;
@property(nonatomic, strong) AVAssetWriterInput *audio;
@property(nonatomic, assign) CMAudioFormatDescriptionRef format;
@property(nonatomic, assign) slayer_movie_ask ask; // its pointers dropped
@property(nonatomic, strong) NSData *sound;       // all of it, as floats
@property(nonatomic, assign) long heard;          // pairs handed over
@property(nonatomic, strong) dispatch_queue_t listener;
@property(nonatomic, strong) dispatch_semaphore_t heard_all;
@property(nonatomic, assign) BOOL ended;          // the sound is finished
@property(nonatomic, copy) NSString *why_deaf;
@end

@implementation SlayerMovie
- (void)dealloc {
	if (_format != NULL) {
		CFRelease(_format);
	}
}
@end

static void start_listening(SlayerMovie *movie);

// say writes why something failed into err, cut to fit
static void say(char *err, int errlen, NSString *what) {
	if (err == NULL || errlen <= 0) {
		return;
	}
	strncpy(err, [what UTF8String], errlen - 1);
	err[errlen - 1] = 0;
}

// failed says what failed and the system's own words for why
static void failed(char *err, int errlen, NSString *what, NSError *why) {
	NSString *reason = why ? why.localizedDescription : @"no reason given";
	NSString *detail = why.userInfo[NSUnderlyingErrorKey]
		? [why.userInfo[NSUnderlyingErrorKey] description] : @"";
	say(err, errlen, [NSString stringWithFormat:@"%@: %@ %@", what,
	                  reason, detail]);
}

// thrown says what an Objective-C exception said
static void thrown(char *err, int errlen, NSString *what, NSException *e) {
	say(err, errlen, [NSString stringWithFormat:@"%@: %@: %@", what,
	                  e.name, e.reason]);
}

// pause sleeps the moment between looks, so no wait spins a core
static void pause_a_moment(SlayerMovie *movie) {
	[NSThread sleepForTimeInterval:movie.ask.pause];
}

// bt709 is the colour every picture and the file are tagged with
static NSDictionary *bt709(void) {
	return @{
		AVVideoColorPrimariesKey : AVVideoColorPrimaries_ITU_R_709_2,
		AVVideoTransferFunctionKey : AVVideoTransferFunction_ITU_R_709_2,
		AVVideoYCbCrMatrixKey : AVVideoYCbCrMatrix_ITU_R_709_2,
	};
}

// add_video adds the H.264 input and the adaptor its pictures go through
static BOOL add_video(SlayerMovie *movie, const char *profile, char *err,
                      int errlen) {
	slayer_movie_ask ask = movie.ask;
	NSDictionary *compression = @{
		AVVideoAverageBitRateKey : @(ask.bit_rate),
		AVVideoMaxKeyFrameIntervalKey : @(ask.keyframes),
		AVVideoProfileLevelKey : @(profile),
		AVVideoH264EntropyModeKey : AVVideoH264EntropyModeCABAC,
		AVVideoExpectedSourceFrameRateKey : @(ask.rate),
	};
	NSDictionary *settings = @{
		AVVideoCodecKey : AVVideoCodecTypeH264,
		AVVideoWidthKey : @(ask.width),
		AVVideoHeightKey : @(ask.height),
		AVVideoColorPropertiesKey : bt709(),
		AVVideoCompressionPropertiesKey : compression,
	};
	AVAssetWriterInput *video =
		[AVAssetWriterInput assetWriterInputWithMediaType:AVMediaTypeVideo
		                                   outputSettings:settings];
	video.expectsMediaDataInRealTime = NO;
	video.mediaTimeScale = ask.timescale;
	NSDictionary *source = @{
		(id)kCVPixelBufferPixelFormatTypeKey : @(kCVPixelFormatType_32BGRA),
		(id)kCVPixelBufferWidthKey : @(ask.width),
		(id)kCVPixelBufferHeightKey : @(ask.height),
	};
	movie.adaptor = [AVAssetWriterInputPixelBufferAdaptor
		assetWriterInputPixelBufferAdaptorWithAssetWriterInput:video
		                           sourcePixelBufferAttributes:source];
	if (![movie.writer canAddInput:video]) {
		say(err, errlen, @"the writer will not take the video settings");
		return NO;
	}
	[movie.writer addInput:video];
	movie.video = video;
	return YES;
}

// lpcm is how the sound arrives: packed float32, interleaved
static AudioStreamBasicDescription lpcm(slayer_movie_ask ask) {
	AudioStreamBasicDescription format;
	memset(&format, 0, sizeof format);
	format.mSampleRate = ask.sound_rate;
	format.mFormatID = kAudioFormatLinearPCM;
	format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
	format.mChannelsPerFrame = ask.channels;
	format.mBitsPerChannel = sizeof(float) * CHAR_BIT;
	format.mBytesPerFrame = sizeof(float) * ask.channels;
	format.mFramesPerPacket = 1;
	format.mBytesPerPacket = format.mBytesPerFrame;
	return format;
}

// add_audio adds the AAC input, fed stereo LPCM
static BOOL add_audio(SlayerMovie *movie, char *err, int errlen) {
	slayer_movie_ask ask = movie.ask;
	AudioChannelLayout layout;
	memset(&layout, 0, sizeof layout);
	layout.mChannelLayoutTag = kAudioChannelLayoutTag_Stereo;
	NSDictionary *settings = @{
		AVFormatIDKey : @(kAudioFormatMPEG4AAC),
		AVSampleRateKey : @(ask.sound_rate),
		AVNumberOfChannelsKey : @(ask.channels),
		AVEncoderBitRateKey : @(ask.audio_bit_rate),
		AVChannelLayoutKey : [NSData dataWithBytes:&layout
		                                    length:sizeof layout],
	};
	AVAssetWriterInput *audio =
		[AVAssetWriterInput assetWriterInputWithMediaType:AVMediaTypeAudio
		                                   outputSettings:settings];
	audio.expectsMediaDataInRealTime = NO;
	AudioStreamBasicDescription arrives = lpcm(ask);
	CMAudioFormatDescriptionRef format = NULL;
	OSStatus status = CMAudioFormatDescriptionCreate(
		kCFAllocatorDefault, &arrives, sizeof layout, &layout, 0, NULL,
		NULL, &format);
	if (status != noErr) {
		say(err, errlen, [NSString stringWithFormat:
		    @"describing the sound: OSStatus %d", (int)status]);
		return NO;
	}
	movie.format = format;
	if (![movie.writer canAddInput:audio]) {
		say(err, errlen, @"the writer will not take the sound settings");
		return NO;
	}
	[movie.writer addInput:audio];
	movie.audio = audio;
	return YES;
}

// start makes the writer and its inputs and starts it at time zero
static BOOL start(SlayerMovie *movie, const slayer_movie_ask *ask,
                  char *err, int errlen) {
	NSURL *url = [NSURL fileURLWithPath:@(ask->path)];
	NSError *why = nil;
	movie.writer = [AVAssetWriter assetWriterWithURL:url
	                                        fileType:AVFileTypeMPEG4
	                                           error:&why];
	if (movie.writer == nil) {
		failed(err, errlen, @"opening the movie", why);
		return NO;
	}
	movie.writer.shouldOptimizeForNetworkUse = YES;
	if (!add_video(movie, ask->profile, err, errlen)) {
		return NO;
	}
	if (ask->sound_rate > 0 && !add_audio(movie, err, errlen)) {
		return NO;
	}
	if (![movie.writer startWriting]) {
		failed(err, errlen, @"starting the movie", movie.writer.error);
		return NO;
	}
	[movie.writer startSessionAtSourceTime:kCMTimeZero];
	if (movie.audio != nil) {
		start_listening(movie);
	}
	return YES;
}

void *slayer_movie_open(const slayer_movie_ask *ask, char *err, int errlen) {
	@autoreleasepool {
		SlayerMovie *movie = [SlayerMovie new];
		slayer_movie_ask kept = *ask;
		kept.path = NULL;
		kept.profile = NULL;
		kept.sound = NULL;
		movie.ask = kept;
		if (ask->sound_count > 0) {
			size_t bytes = ask->sound_count * ask->channels * sizeof(float);
			movie.sound = [NSData dataWithBytes:ask->sound length:bytes];
		}
		@try {
			if (!start(movie, ask, err, errlen)) {
				return NULL;
			}
		} @catch (NSException *e) {
			thrown(err, errlen, @"opening the movie", e);
			return NULL;
		}
		return (__bridge_retained void *)movie;
	}
}

// wait_for waits until the input takes more, and gives up when
// interrupted, failed, deaf or out of time
static BOOL wait_for(SlayerMovie *movie, AVAssetWriterInput *input,
                     NSString *what, char *err, int errlen) {
	NSDate *limit = [NSDate dateWithTimeIntervalSinceNow:movie.ask.limit];
	while (!input.readyForMoreMediaData) {
		if (atomic_load(&movie->stopped)) {
			say(err, errlen, [NSString stringWithFormat:
			    @"stopped while waiting to hand over %@", what]);
			return NO;
		}
		if (atomic_load(&movie->deaf)) {
			say(err, errlen, movie.why_deaf);
			return NO;
		}
		if (movie.writer.status != AVAssetWriterStatusWriting) {
			failed(err, errlen, @"the writer stopped writing",
			       movie.writer.error);
			return NO;
		}
		if (limit.timeIntervalSinceNow < 0) {
			say(err, errlen, [NSString stringWithFormat:
			    @"the encoder took no %@ for %g seconds", what,
			    movie.ask.limit]);
			return NO;
		}
		pause_a_moment(movie);
	}
	return YES;
}

// sound_buffer wraps count pairs of the sound from instant first
static CMSampleBufferRef sound_buffer(SlayerMovie *movie, long first,
                                      long count, OSStatus *status) {
	size_t pair = movie.ask.channels * sizeof(float);
	size_t bytes = count * pair;
	const uint8_t *from = (const uint8_t *)movie.sound.bytes + first * pair;
	CMBlockBufferRef block = NULL;
	*status = CMBlockBufferCreateWithMemoryBlock(
		kCFAllocatorDefault, NULL, bytes, kCFAllocatorDefault, NULL, 0,
		bytes, kCMBlockBufferAssureMemoryNowFlag, &block);
	if (*status == noErr) {
		*status = CMBlockBufferReplaceDataBytes(from, block, 0, bytes);
	}
	CMSampleBufferRef buffer = NULL;
	if (*status == noErr) {
		CMTime when = CMTimeMake(first, movie.ask.sound_rate);
		*status = CMAudioSampleBufferCreateReadyWithPacketDescriptions(
			kCFAllocatorDefault, block, movie.format, count, when, NULL,
			&buffer);
	}
	if (block != NULL) {
		CFRelease(block);
	}
	return buffer;
}

// end_sound marks the sound finished, once, and says so to finish
static void end_sound(SlayerMovie *movie, NSString *why) {
	if (movie.ended) {
		return;
	}
	movie.ended = YES;
	if (why != nil) {
		movie.why_deaf = why;
		atomic_store(&movie->deaf, 1);
	} else if (!atomic_load(&movie->abandoned)) {
		[movie.audio markAsFinished];
	}
	dispatch_semaphore_signal(movie.heard_all);
}

// hear hands the writer sound for as long as it will take it, a chunk
// at a time, and finishes the sound when it is all handed over
static void hear(SlayerMovie *movie) {
	long total = movie.ask.sound_count;
	while (!movie.ended && movie.audio.readyForMoreMediaData) {
		if (atomic_load(&movie->abandoned) || movie.heard >= total) {
			end_sound(movie, nil);
			return;
		}
		long count = MIN(movie.ask.sound_chunk, total - movie.heard);
		OSStatus status = noErr;
		CMSampleBufferRef buffer = sound_buffer(movie, movie.heard, count,
		                                        &status);
		if (buffer == NULL) {
			end_sound(movie, [NSString stringWithFormat:
			    @"wrapping sound: OSStatus %d", (int)status]);
			return;
		}
		BOOL took = [movie.audio appendSampleBuffer:buffer];
		CFRelease(buffer);
		if (!took) {
			end_sound(movie, [NSString stringWithFormat:
			    @"encoding sound: %@", movie.writer.error]);
			return;
		}
		movie.heard += count;
	}
}

// start_listening lets the writer pull the sound on a queue of its own
static void start_listening(SlayerMovie *movie) {
	movie.listener = dispatch_queue_create("slayer.movie.sound",
	                                       DISPATCH_QUEUE_SERIAL);
	movie.heard_all = dispatch_semaphore_create(0);
	__weak SlayerMovie *weak = movie;
	[movie.audio requestMediaDataWhenReadyOnQueue:movie.listener
	                                   usingBlock:^{
		SlayerMovie *held = weak;
		if (held != nil) {
			@try {
				hear(held);
			} @catch (NSException *e) {
				end_sound(held, [NSString stringWithFormat:
				    @"handing over sound: %@: %@", e.name, e.reason]);
			}
		}
	}];
}

// copy_rows writes RGBA rows into a BGRA buffer, opaque, and black rows
// below the picture to the coded height
static void copy_rows(SlayerMovie *movie, const uint8_t *rgba, long stride,
                      uint8_t *base, size_t row) {
	slayer_movie_ask ask = movie.ask;
	for (int y = 0; y < ask.height; y++) {
		uint8_t *out = base + y * row;
		if (y >= ask.picture_height) {
			memset(out, 0, (size_t)ask.width * pixel_bytes);
			for (int x = 0; x < ask.width; x++) {
				out[x * pixel_bytes + bgra_alpha] = opaque;
			}
			continue;
		}
		const uint8_t *in = rgba + y * stride;
		for (int x = 0; x < ask.width; x++) {
			const uint8_t *from = in + x * pixel_bytes;
			uint8_t *to = out + x * pixel_bytes;
			to[bgra_blue] = from[rgba_blue];
			to[bgra_green] = from[rgba_green];
			to[bgra_red] = from[rgba_red];
			to[rgba_alpha] = opaque;
		}
	}
}

// picture is a pixel buffer from the adaptor's pool holding the frame,
// tagged BT.709 so the encoder codes the bytes as they are
static CVPixelBufferRef picture(SlayerMovie *movie, const uint8_t *rgba,
                                long stride, char *err, int errlen) {
	CVPixelBufferRef pixels = NULL;
	CVReturn status = CVPixelBufferPoolCreatePixelBuffer(
		NULL, movie.adaptor.pixelBufferPool, &pixels);
	if (status != kCVReturnSuccess || pixels == NULL) {
		say(err, errlen, [NSString stringWithFormat:
		    @"no picture buffer from the pool: CVReturn %d", status]);
		return NULL;
	}
	CVPixelBufferLockBaseAddress(pixels, 0);
	copy_rows(movie, rgba, stride, CVPixelBufferGetBaseAddress(pixels),
	          CVPixelBufferGetBytesPerRow(pixels));
	CVPixelBufferUnlockBaseAddress(pixels, 0);
	CVBufferSetAttachment(pixels, kCVImageBufferColorPrimariesKey,
	                      kCVImageBufferColorPrimaries_ITU_R_709_2,
	                      kCVAttachmentMode_ShouldPropagate);
	CVBufferSetAttachment(pixels, kCVImageBufferTransferFunctionKey,
	                      kCVImageBufferTransferFunction_ITU_R_709_2,
	                      kCVAttachmentMode_ShouldPropagate);
	CVBufferSetAttachment(pixels, kCVImageBufferYCbCrMatrixKey,
	                      kCVImageBufferYCbCrMatrix_ITU_R_709_2,
	                      kCVAttachmentMode_ShouldPropagate);
	return pixels;
}

// frame waits for the encoder and hands it the picture
static BOOL frame(SlayerMovie *movie, long index, const uint8_t *rgba,
                  long stride, char *err, int errlen) {
	if (!wait_for(movie, movie.video, @"pictures", err, errlen)) {
		return NO;
	}
	CVPixelBufferRef pixels = picture(movie, rgba, stride, err, errlen);
	if (pixels == NULL) {
		return NO;
	}
	CMTime when = CMTimeMake(index, movie.ask.rate);
	BOOL took = [movie.adaptor appendPixelBuffer:pixels
	                        withPresentationTime:when];
	CVPixelBufferRelease(pixels);
	if (!took) {
		failed(err, errlen, @"encoding a picture", movie.writer.error);
		return NO;
	}
	return YES;
}

int slayer_movie_frame(void *handle, long index, const uint8_t *rgba,
                       long stride, char *err, int errlen) {
	@autoreleasepool {
		SlayerMovie *movie = (__bridge SlayerMovie *)handle;
		@try {
			return frame(movie, index, rgba, stride, err, errlen) ? 0 : 1;
		} @catch (NSException *e) {
			thrown(err, errlen, @"encoding a frame", e);
			return 1;
		}
	}
}



// wait_done waits for a signal, interrupted or timed out
static BOOL wait_upto(SlayerMovie *movie, dispatch_semaphore_t done,
                     NSString *what, double seconds, char *err, int errlen);

static BOOL wait_done(SlayerMovie *movie, dispatch_semaphore_t done,
                      NSString *what, char *err, int errlen) {
	return wait_upto(movie, done, what, movie.ask.limit, err, errlen);
}

// wait_upto waits for a signal for at most some seconds, or until stopped
static BOOL wait_upto(SlayerMovie *movie, dispatch_semaphore_t done,
                     NSString *what, double seconds, char *err, int errlen) {
	NSDate *limit = [NSDate dateWithTimeIntervalSinceNow:seconds];
	int64_t moment = (int64_t)(movie.ask.pause * NSEC_PER_SEC);
	while (dispatch_semaphore_wait(
	           done, dispatch_time(DISPATCH_TIME_NOW, moment)) != 0) {
		if (atomic_load(&movie->stopped)) {
			say(err, errlen, [NSString stringWithFormat:
			    @"stopped while waiting for %@", what]);
			return NO;
		}
		if (limit.timeIntervalSinceNow < 0) {
			say(err, errlen, [NSString stringWithFormat:
			    @"waited %g seconds for %@", seconds, what]);
			return NO;
		}
	}
	return YES;
}

// finish waits for the sound to be handed over, ends the session at
// the picture's end, and waits for the file to be written
static BOOL finish(SlayerMovie *movie, long frames, char *err, int errlen) {
	[movie.video markAsFinished];
	if (movie.audio != nil) {
		if (!wait_done(movie, movie.heard_all, @"the sound", err, errlen)) {
			return NO;
		}
		if (atomic_load(&movie->deaf)) {
			say(err, errlen, movie.why_deaf);
			return NO;
		}
	}
	[movie.writer endSessionAtSourceTime:CMTimeMake(frames, movie.ask.rate)];
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	[movie.writer finishWritingWithCompletionHandler:^{
		dispatch_semaphore_signal(done);
	}];
	// the finish rewrites the whole file to put its index first, which
	// for a long film takes far longer than one frame: a minute a
	// thousand frames, and never less than the frame's own limit
	double finishing = movie.ask.limit * (1.0 + (double)frames / 1000.0);
	if (!wait_upto(movie, done, @"the movie to finish", finishing, err,
	              errlen)) {
		return NO;
	}
	if (movie.writer.status != AVAssetWriterStatusCompleted) {
		failed(err, errlen, @"finishing the movie", movie.writer.error);
		return NO;
	}
	return YES;
}

int slayer_movie_finish(void *handle, long frames, char *err, int errlen) {
	@autoreleasepool {
		SlayerMovie *movie = (__bridge SlayerMovie *)handle;
		@try {
			return finish(movie, frames, err, errlen) ? 0 : 1;
		} @catch (NSException *e) {
			thrown(err, errlen, @"finishing the movie", e);
			return 1;
		}
	}
}

void slayer_movie_cancel(void *handle) {
	@autoreleasepool {
		SlayerMovie *movie = (__bridge SlayerMovie *)handle;
		atomic_store(&movie->abandoned, 1);
		if (movie.listener != nil) {
			// a pull of sound under way ends before the writer stops
			dispatch_sync(movie.listener, ^{
			});
		}
		@try {
			if (movie.writer.status == AVAssetWriterStatusWriting) {
				[movie.writer cancelWriting];
			}
		} @catch (NSException *e) {
			// the file is removed by the caller whatever the writer said
		}
	}
}

void slayer_movie_interrupt(void *handle) {
	SlayerMovie *movie = (__bridge SlayerMovie *)handle;
	atomic_store(&movie->stopped, 1);
}

void slayer_movie_release(void *handle) {
	@autoreleasepool {
		SlayerMovie *movie = (__bridge_transfer SlayerMovie *)handle;
		movie = nil;
	}
}

// fourcc writes a codec's four letters, most significant first
static void fourcc(FourCharCode code, char *out) {
	for (int i = 0; i < fourcc_letters; i++) {
		out[i] = (char)(code >> (CHAR_BIT * (fourcc_letters - 1 - i)));
	}
	out[fourcc_letters] = 0;
}

// copy_tag copies a format's text extension, or leaves out empty
static void copy_tag(CMFormatDescriptionRef format, CFStringRef key,
                     char *out, int outlen) {
	CFTypeRef value = CMFormatDescriptionGetExtension(format, key);
	if (value != NULL && CFGetTypeID(value) == CFStringGetTypeID()) {
		CFStringGetCString(value, out, outlen, kCFStringEncodingUTF8);
	}
}

// is_key is whether a sample in a buffer is a keyframe: one with no
// attachment saying it is not
static BOOL is_key(CMSampleBufferRef sample, CMItemCount place) {
	CFArrayRef all = CMSampleBufferGetSampleAttachmentsArray(sample, false);
	if (all == NULL || place >= CFArrayGetCount(all)) {
		return YES;
	}
	CFDictionaryRef one = CFArrayGetValueAtIndex(all, place);
	CFBooleanRef not_sync =
		CFDictionaryGetValue(one, kCMSampleAttachmentKey_NotSync);
	return not_sync == NULL || !CFBooleanGetValue(not_sync);
}

// count_frames reads every coded picture as it is in the file, counting
// them and the longest run from a keyframe to the next
static BOOL count_frames(AVAsset *asset, AVAssetTrack *video,
                         slayer_movie_facts *facts, char *err, int errlen) {
	NSError *why = nil;
	AVAssetReader *reader = [AVAssetReader assetReaderWithAsset:asset
	                                                      error:&why];
	if (reader == nil) {
		failed(err, errlen, @"reading the pictures", why);
		return NO;
	}
	AVAssetReaderTrackOutput *output = [AVAssetReaderTrackOutput
		assetReaderTrackOutputWithTrack:video
		                 outputSettings:nil];
	output.alwaysCopiesSampleData = NO;
	[reader addOutput:output];
	if (![reader startReading]) {
		failed(err, errlen, @"reading the pictures", reader.error);
		return NO;
	}
	long since = 0;
	CMSampleBufferRef sample;
	while ((sample = [output copyNextSampleBuffer]) != NULL) {
		CMItemCount count = CMSampleBufferGetNumSamples(sample);
		for (CMItemCount i = 0; i < count; i++) {
			since = is_key(sample, i) ? 1 : since + 1;
			facts->frames++;
			facts->keyframe_gap = MAX(facts->keyframe_gap, since);
		}
		CFRelease(sample);
	}
	if (reader.status != AVAssetReaderStatusCompleted) {
		failed(err, errlen, @"reading the pictures", reader.error);
		return NO;
	}
	return YES;
}

// probe_video reads the picture's size, rate, codec, profile, level and
// colour tags, and counts its frames
static BOOL probe_video(AVAsset *asset, AVAssetTrack *video,
                        slayer_movie_facts *facts, char *err, int errlen) {
	facts->width = (int)video.naturalSize.width;
	facts->height = (int)video.naturalSize.height;
	facts->rate = video.nominalFrameRate;
	CMFormatDescriptionRef format =
		(__bridge CMFormatDescriptionRef)video.formatDescriptions.firstObject;
	fourcc(CMFormatDescriptionGetMediaSubType(format), facts->video);
	copy_tag(format, kCMFormatDescriptionExtension_ColorPrimaries,
	         facts->primaries, sizeof facts->primaries);
	copy_tag(format, kCMFormatDescriptionExtension_TransferFunction,
	         facts->transfer, sizeof facts->transfer);
	copy_tag(format, kCMFormatDescriptionExtension_YCbCrMatrix,
	         facts->matrix, sizeof facts->matrix);
	CFStringRef atoms_key =
		kCMFormatDescriptionExtension_SampleDescriptionExtensionAtoms;
	NSDictionary *atoms = (__bridge NSDictionary *)
		CMFormatDescriptionGetExtension(format, atoms_key);
	NSData *avcc = atoms[@"avcC"];
	if ([avcc isKindOfClass:[NSData class]] && avcc.length >= avcc_least) {
		const uint8_t *bytes = avcc.bytes;
		facts->profile = bytes[avcc_profile];
		facts->level = bytes[avcc_level];
	}
	return count_frames(asset, video, facts, err, errlen);
}

// probe_audio reads the sound's codec, rate, channels and length
static void probe_audio(AVAssetTrack *audio, slayer_movie_facts *facts) {
	id first = audio.formatDescriptions.firstObject;
	CMAudioFormatDescriptionRef format =
		(__bridge CMAudioFormatDescriptionRef)first;
	const AudioStreamBasicDescription *basic =
		CMAudioFormatDescriptionGetStreamBasicDescription(format);
	if (basic != NULL) {
		fourcc(basic->mFormatID, facts->audio);
		facts->audio_rate = basic->mSampleRate;
		facts->audio_channels = (int)basic->mChannelsPerFrame;
	}
	facts->audio_seconds = CMTimeGetSeconds(audio.timeRange.duration);
}

// asset_at is the file at a path, read for exact times
static AVURLAsset *asset_at(const char *path) {
	NSURL *url = [NSURL fileURLWithPath:@(path)];
	NSDictionary *exact = @{AVURLAssetPreferPreciseDurationAndTimingKey : @YES};
	return [AVURLAsset URLAssetWithURL:url options:exact];
}

// first_track is an asset's first track of a kind, nil when it has
// none; the asset is loaded, waiting at most limit seconds
static AVAssetTrack *first_track(AVAsset *asset, AVMediaType kind,
                                 double limit, char *err, int errlen,
                                 BOOL *failure) {
	__block AVAssetTrack *found = nil;
	__block NSError *why = nil;
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	[asset loadTracksWithMediaType:kind
	             completionHandler:^(NSArray<AVAssetTrack *> *tracks,
	                                 NSError *error) {
		found = tracks.firstObject;
		why = error;
		dispatch_semaphore_signal(done);
	}];
	int64_t wait = (int64_t)(limit * NSEC_PER_SEC);
	if (dispatch_semaphore_wait(
	        done, dispatch_time(DISPATCH_TIME_NOW, wait)) != 0) {
		say(err, errlen, [NSString stringWithFormat:
		    @"the file did not load in %g seconds", limit]);
		*failure = YES;
		return nil;
	}
	if (why != nil) {
		failed(err, errlen, @"loading the file", why);
		*failure = YES;
	}
	return found;
}

int slayer_movie_probe(const char *path, double limit,
                       slayer_movie_facts *facts, char *err, int errlen) {
	@autoreleasepool {
		memset(facts, 0, sizeof *facts);
		@try {
			AVURLAsset *asset = asset_at(path);
			BOOL failure = NO;
			AVAssetTrack *video = first_track(asset, AVMediaTypeVideo, limit,
			                                  err, errlen, &failure);
			if (failure) {
				return 1;
			}
			if (video == nil) {
				say(err, errlen, [NSString stringWithFormat:
				    @"no video track in %s", path]);
				return 1;
			}
			facts->seconds = CMTimeGetSeconds(asset.duration);
			if (!probe_video(asset, video, facts, err, errlen)) {
				return 1;
			}
			AVAssetTrack *audio = first_track(asset, AVMediaTypeAudio, limit,
			                                  err, errlen, &failure);
			if (failure) {
				return 1;
			}
			if (audio != nil) {
				probe_audio(audio, facts);
			}
			return 0;
		} @catch (NSException *e) {
			thrown(err, errlen, @"probing the movie", e);
			return 1;
		}
	}
}

// decoded reads every sample of an audio output into one block of
// bytes, and the channel count of the first buffer
static NSData *decoded(AVAssetReaderTrackOutput *output, int *channels) {
	NSMutableData *all = [NSMutableData data];
	CMSampleBufferRef sample;
	while ((sample = [output copyNextSampleBuffer]) != NULL) {
		if (*channels == 0) {
			const AudioStreamBasicDescription *basic =
				CMAudioFormatDescriptionGetStreamBasicDescription(
					CMSampleBufferGetFormatDescription(sample));
			*channels = basic ? (int)basic->mChannelsPerFrame : 0;
		}
		CMBlockBufferRef block = CMSampleBufferGetDataBuffer(sample);
		size_t length = block ? CMBlockBufferGetDataLength(block) : 0;
		size_t at = all.length;
		[all setLength:at + length];
		if (length > 0) {
			CMBlockBufferCopyDataBytes(block, 0, length,
			                           (uint8_t *)all.mutableBytes + at);
		}
		CFRelease(sample);
	}
	return all;
}

// sound_output is a reader output giving float32 interleaved samples at
// rate, in the track's own channel count
static AVAssetReaderTrackOutput *sound_output(AVAssetTrack *track, int rate) {
	NSDictionary *settings = @{
		AVFormatIDKey : @(kAudioFormatLinearPCM),
		AVSampleRateKey : @(rate),
		AVLinearPCMBitDepthKey : @(sizeof(float) * CHAR_BIT),
		AVLinearPCMIsFloatKey : @YES,
		AVLinearPCMIsNonInterleaved : @NO,
		AVLinearPCMIsBigEndianKey : @NO,
	};
	return [AVAssetReaderTrackOutput assetReaderTrackOutputWithTrack:track
	                                                  outputSettings:settings];
}

// read_sound decodes the first audio track, or says why not
static NSData *read_sound(const char *path, int rate, double limit,
                          int *channels, char *err, int errlen) {
	AVURLAsset *asset = asset_at(path);
	BOOL failure = NO;
	AVAssetTrack *track = first_track(asset, AVMediaTypeAudio, limit, err,
	                                  errlen, &failure);
	if (failure) {
		return nil;
	}
	if (track == nil) {
		say(err, errlen, [NSString stringWithFormat:
		    @"no sound in %s: AVFoundation found no audio track", path]);
		return nil;
	}
	NSError *why = nil;
	AVAssetReader *reader = [AVAssetReader assetReaderWithAsset:asset
	                                                      error:&why];
	if (reader == nil) {
		failed(err, errlen, @"reading the sound", why);
		return nil;
	}
	AVAssetReaderTrackOutput *output = sound_output(track, rate);
	[reader addOutput:output];
	if (![reader startReading]) {
		failed(err, errlen, @"reading the sound", reader.error);
		return nil;
	}
	NSData *all = decoded(output, channels);
	if (reader.status != AVAssetReaderStatusCompleted) {
		failed(err, errlen, @"reading the sound", reader.error);
		return nil;
	}
	return all;
}

float *slayer_movie_sound(const char *path, int rate, double limit,
                          long *samples, int *channels, char *err,
                          int errlen) {
	@autoreleasepool {
		*samples = 0;
		*channels = 0;
		@try {
			NSData *all = read_sound(path, rate, limit, channels, err, errlen);
			if (all == nil) {
				return NULL;
			}
			float *out = malloc(all.length > 0 ? all.length : sizeof(float));
			if (out == NULL) {
				say(err, errlen, @"no memory for the decoded sound");
				return NULL;
			}
			memcpy(out, all.bytes, all.length);
			*samples = (long)(all.length / sizeof(float));
			return out;
		} @catch (NSException *e) {
			thrown(err, errlen, @"reading the sound", e);
			return NULL;
		}
	}
}

// a film being read for its pictures: the reader, and its one output,
// decoding to BGRA at the size asked
@interface SlayerFilm : NSObject
@property(nonatomic, strong) AVAssetReader *reader;
@property(nonatomic, strong) AVAssetReaderTrackOutput *output;
@property(nonatomic, assign) int width;
@property(nonatomic, assign) int height;
@end

@implementation SlayerFilm
@end

// picture_output is a reader output giving BGRA pictures scaled by the
// decoder to width by height
static AVAssetReaderTrackOutput *picture_output(AVAssetTrack *track,
                                                int width, int height) {
	NSDictionary *settings = @{
		(id)kCVPixelBufferPixelFormatTypeKey : @(kCVPixelFormatType_32BGRA),
		(id)kCVPixelBufferWidthKey : @(width),
		(id)kCVPixelBufferHeightKey : @(height),
	};
	AVAssetReaderTrackOutput *output =
		[AVAssetReaderTrackOutput assetReaderTrackOutputWithTrack:track
		                                           outputSettings:settings];
	output.alwaysCopiesSampleData = NO;
	return output;
}

// film_facts is what Go is told of a film: its length, rate and size
static void film_facts(AVAsset *asset, AVAssetTrack *video,
                       slayer_film_facts *facts) {
	facts->seconds = CMTimeGetSeconds(asset.duration);
	facts->rate = video.nominalFrameRate;
	facts->width = (int)video.naturalSize.width;
	facts->height = (int)video.naturalSize.height;
}

// open_film is a reader over a film's first video track from from
// seconds, or nil with why
static SlayerFilm *open_film(const char *path, double from, int width,
                             int height, double limit,
                             slayer_film_facts *facts, char *err,
                             int errlen) {
	AVURLAsset *asset = asset_at(path);
	BOOL failure = NO;
	AVAssetTrack *video = first_track(asset, AVMediaTypeVideo, limit, err,
	                                  errlen, &failure);
	if (failure) {
		return nil;
	}
	if (video == nil) {
		say(err, errlen, [NSString stringWithFormat:
		    @"no video track in %s", path]);
		return nil;
	}
	film_facts(asset, video, facts);
	NSError *why = nil;
	SlayerFilm *film = [[SlayerFilm alloc] init];
	film.reader = [AVAssetReader assetReaderWithAsset:asset error:&why];
	if (film.reader == nil) {
		failed(err, errlen, @"reading the pictures", why);
		return nil;
	}
	film.reader.timeRange = CMTimeRangeMake(
		CMTimeMakeWithSeconds(from, video.naturalTimeScale),
		kCMTimePositiveInfinity);
	film.output = picture_output(video, width, height);
	film.width = width;
	film.height = height;
	[film.reader addOutput:film.output];
	if (![film.reader startReading]) {
		failed(err, errlen, @"reading the pictures", film.reader.error);
		return nil;
	}
	return film;
}

void *slayer_film_open(const char *path, double from, int width,
                       int height, double limit, slayer_film_facts *facts,
                       char *err, int errlen) {
	@autoreleasepool {
		memset(facts, 0, sizeof *facts);
		@try {
			SlayerFilm *film = open_film(path, from, width, height, limit,
			                             facts, err, errlen);
			return film == nil ? NULL : (__bridge_retained void *)film;
		} @catch (NSException *e) {
			thrown(err, errlen, @"opening the film", e);
			return NULL;
		}
	}
}

// copy_picture copies a decoded BGRA picture into RGBA rows, or says
// why it cannot: a picture of another size than was asked is refused
static BOOL copy_picture(SlayerFilm *film, CVPixelBufferRef pixels,
                         uint8_t *rgba, long stride, char *err, int errlen) {
	size_t wide = CVPixelBufferGetWidth(pixels);
	size_t high = CVPixelBufferGetHeight(pixels);
	if ((int)wide != film.width || (int)high != film.height) {
		say(err, errlen, [NSString stringWithFormat:
		    @"the decoder gave %zux%zu, not the %dx%d asked", wide, high,
		    film.width, film.height]);
		return NO;
	}
	CVPixelBufferLockBaseAddress(pixels, kCVPixelBufferLock_ReadOnly);
	const uint8_t *base = CVPixelBufferGetBaseAddress(pixels);
	size_t across = CVPixelBufferGetBytesPerRow(pixels);
	for (size_t y = 0; y < high; y++) {
		const uint8_t *from = base + y * across;
		uint8_t *to = rgba + y * stride;
		for (size_t x = 0; x < wide; x++) {
			to[rgba_red] = from[bgra_red];
			to[rgba_green] = from[bgra_green];
			to[rgba_blue] = from[bgra_blue];
			to[rgba_alpha] = opaque;
			from += pixel_bytes;
			to += pixel_bytes;
		}
	}
	CVPixelBufferUnlockBaseAddress(pixels, kCVPixelBufferLock_ReadOnly);
	return YES;
}

// next_picture decodes the next picture that has one: a sample with no
// image, which a reader may give, is passed over
static int next_picture(SlayerFilm *film, uint8_t *rgba, long stride,
                        double *at, char *err, int errlen) {
	CMSampleBufferRef sample;
	while ((sample = [film.output copyNextSampleBuffer]) != NULL) {
		CVImageBufferRef pixels = CMSampleBufferGetImageBuffer(sample);
		if (pixels == NULL) {
			CFRelease(sample);
			continue;
		}
		*at = CMTimeGetSeconds(CMSampleBufferGetPresentationTimeStamp(sample));
		BOOL copied = copy_picture(film, pixels, rgba, stride, err, errlen);
		CFRelease(sample);
		return copied ? slayer_film_frame : slayer_film_failed;
	}
	if (film.reader.status == AVAssetReaderStatusFailed) {
		failed(err, errlen, @"reading the pictures", film.reader.error);
		return slayer_film_failed;
	}
	return slayer_film_end;
}

int slayer_film_next(void *handle, uint8_t *rgba, long stride, double *at,
                     char *err, int errlen) {
	@autoreleasepool {
		@try {
			return next_picture((__bridge SlayerFilm *)handle, rgba, stride,
			                    at, err, errlen);
		} @catch (NSException *e) {
			thrown(err, errlen, @"reading the pictures", e);
			return slayer_film_failed;
		}
	}
}

void slayer_film_release(void *handle) {
	@autoreleasepool {
		SlayerFilm *film = (__bridge_transfer SlayerFilm *)handle;
		[film.reader cancelReading];
	}
}
