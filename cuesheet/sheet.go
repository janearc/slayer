package cuesheet

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// Channel is one thing a sheet may play: its name, the least and most it
// may be, and the value it rests at when no cue names it.
type Channel struct {
	Name              string
	Least, Most, Rest float64
}

// the refusals a sheet is given
var (
	// ErrChannel is a channel the scene has not.
	ErrChannel = errors.New("cuesheet: no such channel")
	// ErrRange is a value outside its channel's range.
	ErrRange = errors.New("cuesheet: a value out of range")
	// ErrWhen is a moment that is not in the music: a bar past its end,
	// a beat past its bar's, a time before it starts.
	ErrWhen = errors.New("cuesheet: a moment not in the music")
	// ErrCue is a cue that is not written as a cue is.
	ErrCue = errors.New("cuesheet: a cue written wrong")
)

// the words a cue is written in, beside the channels it names
const (
	keyAt      = "at"
	keyAtSec   = "at_sec"
	keyFade    = "fade"
	keyUntil   = "until"
	keyFlash   = "flash"
	keyPulse   = "pulse"
	keyOn      = "on"
	keyHit     = "hit"
	keyDecay   = "decay"
	keyFollow  = "follow"
	keyUp      = "up"
	keyQuiet   = "quiet"
	keyStep    = "step"
	keyHot     = "hot"
	keySoft    = "soft"
	keyAttack  = "attack"
	keyRelease = "release"
	hitStep    = 'x'
)

// Sheet is a cue sheet read against a clock and a scene's channels: for
// each channel, what it does through the music.
type Sheet struct {
	clock    Clock
	channels []Channel
	tracks   map[string]*track
}

// document is a sheet as written.
type document struct {
	Cues []yaml.Node `yaml:"cues"`
}

// Read is a sheet read from raw against the clock and the channels a
// scene declares. A channel it has not, a value out of a channel's range,
// a moment not in the music and a cue written wrong are each refused by
// name, with the line.
func Read(raw []byte, channels []Channel, clock Clock) (*Sheet, error) {
	if clock == nil {
		return nil, fmt.Errorf("%w: a sheet needs a clock", ErrClock)
	}
	var written document
	if err := yaml.Unmarshal(raw, &written); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCue, err)
	}
	sheet := &Sheet{clock: clock, channels: append([]Channel(nil),
		channels...), tracks: map[string]*track{}}
	for _, one := range channels {
		sheet.tracks[one.Name] = &track{channel: one}
	}
	for index := range written.Cues {
		node := &written.Cues[index]
		if err := sheet.cue(node); err != nil {
			return nil, fmt.Errorf("line %d: %w", node.Line, err)
		}
	}
	return sheet, nil
}

// fields is a cue's keys and values, in the order written.
type fields map[string]*yaml.Node

// cue reads one cue into the tracks.
func (sheet *Sheet) cue(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%w: a cue is a map of words to values",
			ErrCue)
	}
	written := fields{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		written[node.Content[i].Value] = node.Content[i+1]
	}
	at, err := sheet.when(written)
	if err != nil {
		return err
	}
	if err := sheet.levels(written, at); err != nil {
		return err
	}
	if err := sheet.flashes(written, at); err != nil {
		return err
	}
	if err := sheet.follows(written, at); err != nil {
		return err
	}
	if err := sheet.stepped(written, at); err != nil {
		return err
	}
	return sheet.quiets(written, at)
}

// reserved are the words a cue is written in that are not channels.
var reserved = map[string]bool{keyAt: true, keyAtSec: true,
	keyFade: true, keyUntil: true, keyFlash: true, keyPulse: true,
	keyOn: true, keyHit: true, keyDecay: true, keyFollow: true,
	keyUp: true, keyQuiet: true, keyStep: true, keyAttack: true, keyHot: true,
	keySoft:    true,
	keyRelease: true}

// when is a cue's moment: at, as bar.beat, or at_sec, as seconds.
func (sheet *Sheet) when(written fields) (time.Duration, error) {
	if node, found := written[keyAtSec]; found {
		count, err := numberOf(keyAtSec, node)
		if err != nil {
			return 0, err
		}
		if count < none {
			return 0, fmt.Errorf("%w: at_sec %v is before the music",
				ErrWhen, count)
		}
		return seconds(count), nil
	}
	node, found := written[keyAt]
	if !found {
		return 0, fmt.Errorf("%w: a cue says when, by at or at_sec",
			ErrCue)
	}
	return sheet.barBeat(node.Value)
}

// barBeat is the moment of "bar.beat", or "bar" for its first beat, both
// counted from one; a bar past the music's end or a beat past its bar's
// is refused.
func (sheet *Sheet) barBeat(written string) (time.Duration, error) {
	barText, beatText, hasBeat := strings.Cut(written, ".")
	bar, err := strconv.Atoi(barText)
	beat := fromFirst
	if err == nil && hasBeat {
		beat, err = strconv.Atoi(beatText)
	}
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not bar.beat", ErrCue, written)
	}
	meter, bars := sheet.clock.Meter(), sheet.clock.Bars()
	if bar < fromFirst || bar > bars || beat < fromFirst || beat > meter {
		return 0, fmt.Errorf("%w: %q; the music has bars 1 to %d of %d "+
			"beats", ErrWhen, written, bars, meter)
	}
	beats := float64((bar-fromFirst)*meter + beat - fromFirst)
	return sheet.clock.Moment(beats), nil
}

// track is the channel this cue names, or a refusal listing those the
// scene has.
func (sheet *Sheet) track(name string) (*track, error) {
	found, ok := sheet.tracks[name]
	if !ok {
		names := make([]string, 0, len(sheet.channels))
		for _, one := range sheet.channels {
			names = append(names, one.Name)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("%w: %q; the channels are %s", ErrChannel,
			name, strings.Join(names, ", "))
	}
	return found, nil
}

// value is a number for a channel, or a refusal saying its range.
func (sheet *Sheet) value(name string, node *yaml.Node) (*track,
	float64, error) {
	found, err := sheet.track(name)
	if err != nil {
		return nil, 0, err
	}
	count, err := numberOf(name, node)
	if err != nil {
		return nil, 0, err
	}
	one := found.channel
	if count < one.Least || count > one.Most {
		return nil, 0, fmt.Errorf("%w: %s is %v, outside %v to %v",
			ErrRange, name, count, one.Least, one.Most)
	}
	return found, count, nil
}

// numberOf is a node's number, or a refusal naming the key.
func numberOf(key string, node *yaml.Node) (float64, error) {
	count, err := strconv.ParseFloat(node.Value, 64)
	if node.Kind != yaml.ScalarNode || err != nil {
		return 0, fmt.Errorf("%w: %s is a number, not %q", ErrCue, key,
			node.Value)
	}
	return count, nil
}

// levels reads the cue's channel levels, with their glide.
func (sheet *Sheet) levels(written fields, at time.Duration) error {
	end, err := sheet.glideEnd(written, at)
	if err != nil {
		return err
	}
	for name, node := range written {
		if reserved[name] {
			continue
		}
		found, count, err := sheet.value(name, node)
		if err != nil {
			return err
		}
		found.levels = append(found.levels, level{from: at, until: end,
			to: count})
	}
	return nil
}

// glideEnd is when a cue's levels arrive: at fade seconds after the cue,
// at until, or at once.
func (sheet *Sheet) glideEnd(written fields,
	at time.Duration) (time.Duration, error) {
	if node, found := written[keyUntil]; found {
		end, err := sheet.barBeat(node.Value)
		if err != nil {
			return 0, err
		}
		if end < at {
			return 0, fmt.Errorf("%w: until %s is before the cue",
				ErrWhen, node.Value)
		}
		return end, nil
	}
	if node, found := written[keyFade]; found {
		count, err := numberOf(keyFade, node)
		if err != nil {
			return 0, err
		}
		return at + seconds(count), nil
	}
	return at, nil
}

// mapOf is a cue's map of channel to number under a key.
func (sheet *Sheet) mapOf(key string,
	node *yaml.Node) (map[*track]float64, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%w: %s is a map of channel to value",
			ErrCue, key)
	}
	held := map[*track]float64{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		found, count, err := sheet.value(node.Content[i].Value,
			node.Content[i+1])
		if err != nil {
			return nil, err
		}
		held[found] = count
	}
	return held, nil
}

// notesOf is the notes of the cue's voice, only its hit's when it names
// one.
func (sheet *Sheet) notesOf(voice string, written fields) ([]Note,
	error) {
	notes, err := sheet.clock.Notes(voice)
	if err != nil {
		return nil, err
	}
	hot, err := sheet.loudness(written, keyHot, 0)
	if err != nil {
		return nil, err
	}
	soft, err := sheet.loudness(written, keySoft, math.Inf(1))
	if err != nil {
		return nil, err
	}
	hit, hitSaid := written[keyHit]
	var struck []Note
	for _, one := range notes {
		if (hitSaid && one.Hit != hit.Value) || one.Loud < hot ||
			one.Loud >= soft {
			continue
		}
		struck = append(struck, one)
	}
	return struck, nil
}

// loudness is how loud a cue asks its voice's notes to be, under a key:
// hot, at least so loud (moonlake's word), or soft, quieter than so; or
// the fallback where it says none.
func (sheet *Sheet) loudness(written fields, key string,
	fallback float64) (float64, error) {
	node, found := written[key]
	if !found {
		return fallback, nil
	}
	return numberOf(key, node)
}
