package cuesheet

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// track is what one channel does through the music: the levels it is
// set to, the flashes and follows that ride on them, and when each of
// those ends.
type track struct {
	channel Channel
	levels  []level
	flashes []flash
	follows []follow
	steps   []step
	quiets  []time.Duration
}

// level is a channel set at a moment, arriving at until: at once when
// until is the moment, else linearly from what it held.
type level struct {
	from, until time.Duration
	to          float64
}

// flash is a channel struck from a moment on: at each of strikes it
// jumps to peak and falls back by decay beats to a third of the way.
type flash struct {
	from    time.Duration
	peak    float64
	decay   float64
	strikes []time.Duration
}

// step is a channel moved on by so much at each of strikes, from a moment
// on: a count, not a flash, so it stays where the strikes leave it.
type step struct {
	from    time.Duration
	by      float64
	strikes []time.Duration
}

// follow is a channel raised to up while a note of a voice sounds, from a
// moment on.
type follow struct {
	from            time.Duration
	up              float64
	notes           []Note
	attack, release float64 // beats; nought is at once
}

// flashes reads the cue's flash: its peaks, what strikes it, and its
// decay.
func (sheet *Sheet) flashes(written fields, at time.Duration) error {
	node, found := written[keyFlash]
	if !found {
		return nil
	}
	peaks, err := sheet.mapOf(keyFlash, node)
	if err != nil {
		return err
	}
	decay, err := sheet.decayOf(written)
	if err != nil {
		return err
	}
	strikes, err := sheet.strikes(written, at)
	if err != nil {
		return err
	}
	for found, peak := range peaks {
		found.flashes = append(found.flashes, flash{from: at, peak: peak,
			decay: decay, strikes: strikes})
	}
	return nil
}

// stepped reads the cue's step: how far each strike moves each channel,
// and what strikes it.
func (sheet *Sheet) stepped(written fields, at time.Duration) error {
	node, found := written[keyStep]
	if !found {
		return nil
	}
	moves, err := sheet.amounts(node)
	if err != nil {
		return err
	}
	strikes, err := sheet.strikes(written, at)
	if err != nil {
		return err
	}
	for found, by := range moves {
		found.steps = append(found.steps, step{from: at, by: by,
			strikes: strikes})
	}
	return nil
}

// amounts is a step's map of channel to how far a strike moves it: any
// number, since a step is a move and not a value.
func (sheet *Sheet) amounts(node *yaml.Node) (map[*track]float64, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%w: step is a map of channel to amount",
			ErrCue)
	}
	held := map[*track]float64{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		found, err := sheet.track(node.Content[i].Value)
		if err != nil {
			return nil, err
		}
		by, err := numberOf(keyStep, node.Content[i+1])
		if err != nil {
			return nil, err
		}
		held[found] = by
	}
	return held, nil
}

// decayOf is a flash's decay in beats, which must be above nought.
func (sheet *Sheet) decayOf(written fields) (float64, error) {
	node, found := written[keyDecay]
	if !found {
		return 0, fmt.Errorf("%w: a flash says its decay, in beats",
			ErrCue)
	}
	decay, err := numberOf(keyDecay, node)
	if err != nil {
		return 0, err
	}
	if !(decay > none) {
		return 0, fmt.Errorf("%w: decay %v; a flash falls over more "+
			"than nought beats", ErrCue, decay)
	}
	return decay, nil
}

// strikes are the moments a flash is struck: each x of its pulse, every
// bar from the cue's, or each note of its voice from the cue's moment.
func (sheet *Sheet) strikes(written fields,
	at time.Duration) ([]time.Duration, error) {
	if node, found := written[keyPulse]; found {
		return sheet.pulsed(node.Value, at)
	}
	node, found := written[keyOn]
	if !found {
		return nil, fmt.Errorf("%w: a flash says what strikes it, by "+
			"pulse or on", ErrCue)
	}
	notes, err := sheet.notesOf(node.Value, written)
	if err != nil {
		return nil, err
	}
	var struck []time.Duration
	for _, one := range notes {
		if one.On >= at {
			struck = append(struck, one.On)
		}
	}
	return struck, nil
}

// pulsed are the moments of a pulse's hits, sixteen steps to a bar, from
// the cue's moment to the music's end.
func (sheet *Sheet) pulsed(pattern string,
	at time.Duration) ([]time.Duration, error) {
	if len(pattern) == 0 || strings.Trim(pattern, "x.") != "" {
		return nil, fmt.Errorf("%w: pulse %q is steps of x and \".\"",
			ErrCue, pattern)
	}
	meter := float64(sheet.clock.Meter())
	step := meter / sixteenth
	first := sheet.clock.Beats(at)
	end := float64(sheet.clock.Bars()) * meter
	var struck []time.Duration
	for index := 0; first+float64(index)*step < end; index++ {
		if pattern[index%len(pattern)] == hitStep {
			struck = append(struck,
				sheet.clock.Moment(first+float64(index)*step))
		}
	}
	return struck, nil
}

// follows reads the cue's follow: the voice and the values its notes
// raise.
func (sheet *Sheet) follows(written fields, at time.Duration) error {
	node, found := written[keyFollow]
	if !found {
		return nil
	}
	upNode, found := written[keyUp]
	if !found {
		return fmt.Errorf("%w: a follow says what goes up, by up", ErrCue)
	}
	ups, err := sheet.mapOf(keyUp, upNode)
	if err != nil {
		return err
	}
	notes, err := sheet.notesOf(node.Value, written)
	if err != nil {
		return err
	}
	attack, err := sheet.beatsOf(written, keyAttack)
	if err != nil {
		return err
	}
	release, err := sheet.beatsOf(written, keyRelease)
	if err != nil {
		return err
	}
	for found, up := range ups {
		found.follows = append(found.follows, follow{from: at, up: up,
			notes: notes, attack: attack, release: release})
	}
	return nil
}

// beatsOf is a count of beats a cue says under a key, nought when it
// says none; fewer than nought is refused.
func (sheet *Sheet) beatsOf(written fields, key string) (float64, error) {
	node, found := written[key]
	if !found {
		return 0, nil
	}
	count, err := numberOf(key, node)
	if err != nil {
		return 0, err
	}
	if count < none {
		return 0, fmt.Errorf("%w: %s %v; it is beats, nought or more",
			ErrCue, key, count)
	}
	return count, nil
}

// quiets reads the cue's quiet: the channels whose flash and follow end.
func (sheet *Sheet) quiets(written fields, at time.Duration) error {
	node, found := written[keyQuiet]
	if !found {
		return nil
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("%w: quiet is a list of channels", ErrCue)
	}
	for _, item := range node.Content {
		found, err := sheet.track(item.Value)
		if err != nil {
			return err
		}
		found.quiets = append(found.quiets, at)
	}
	return nil
}

// Channels are the channels the sheet plays, as it was given them.
func (sheet *Sheet) Channels() []Channel {
	return append([]Channel(nil), sheet.channels...)
}

// At is every channel's value at a moment.
func (sheet *Sheet) At(at time.Duration) map[string]float64 {
	values := make(map[string]float64, len(sheet.tracks))
	for name, one := range sheet.tracks {
		values[name] = one.at(at, sheet.clock)
	}
	return values
}

// at is the channel's value at a moment: its level, raised while its
// follow's voice sounds, and its latest flash over that, kept within its
// range.
func (one *track) at(at time.Duration, clock Clock) float64 {
	quiet := one.quietSince(at)
	value := one.level(at) + one.stepped(at)
	if up, share := one.following(at, quiet, clock); share > none {
		value += (up - value) * share
	}
	if peak, since, decay, struck := one.flashing(at, quiet,
		clock); struck {
		value += (peak - value) * math.Exp(-since/decay)
	}
	return math.Max(one.channel.Least, math.Min(one.channel.Most, value))
}

// level is the channel's level at a moment: its rest before any cue, the
// latest cue's value once arrived, and linearly between while it glides.
func (one *track) level(at time.Duration) float64 {
	held := one.channel.Rest
	for _, set := range one.ordered() {
		if set.from > at {
			break
		}
		if at >= set.until || set.until == set.from {
			held = set.to
			continue
		}
		share := float64(at-set.from) / float64(set.until-set.from)
		held += (set.to - held) * share
	}
	return held
}

// ordered are the channel's levels in the order they start, cues at one
// moment in the order written.
func (one *track) ordered() []level {
	held := append([]level(nil), one.levels...)
	sort.SliceStable(held, func(i, k int) bool {
		return held[i].from < held[k].from
	})
	return held
}

// stepped is how far the channel's steps have moved it by a moment: each
// strike of each step at or before it, and before the first quiet after
// the step began. A quiet stops the steps; where they left the channel
// stays.
func (one *track) stepped(at time.Duration) float64 {
	moved := 0.0
	for _, each := range one.steps {
		stop := one.quietAfter(each.from)
		for _, moment := range each.strikes {
			if moment <= at && moment < stop {
				moved += each.by
			}
		}
	}
	return moved
}

// quietAfter is the first quiet after a moment, or past every moment.
func (one *track) quietAfter(from time.Duration) time.Duration {
	first := time.Duration(math.MaxInt64)
	for _, moment := range one.quiets {
		if moment > from && moment < first {
			first = moment
		}
	}
	return first
}

// quietSince is the latest quiet at or before a moment, or before the
// music when there is none.
func (one *track) quietSince(at time.Duration) time.Duration {
	latest := time.Duration(math.MinInt64)
	for _, moment := range one.quiets {
		if moment <= at && moment > latest {
			latest = moment
		}
	}
	return latest
}

// following is the latest follow's up value, if one began after the
// last quiet, and how far toward it the channel stands: all the way while
// a note of its voice sounds, rising over its attack from the note's
// start and falling over its release from the note's end.
func (one *track) following(at, quiet time.Duration,
	clock Clock) (float64, float64) {
	var latest *follow
	for i := range one.follows {
		each := &one.follows[i]
		if each.from <= at && each.from > quiet &&
			(latest == nil || each.from >= latest.from) {
			latest = each
		}
	}
	if latest == nil {
		return 0, 0
	}
	share := 0.0
	now := clock.Beats(at)
	for _, note := range latest.notes {
		if note.On > at {
			break
		}
		share = math.Max(share, latest.swell(note, now, clock))
	}
	return latest.up, share
}

// swell is how far toward its up value a note holds the channel at a
// count of beats: rising over the attack while the note sounds, and
// falling over the release after it ends from where it had reached.
func (each *follow) swell(note Note, now float64, clock Clock) float64 {
	on, off := clock.Beats(note.On), clock.Beats(note.Off)
	rise := func(beats float64) float64 {
		if each.attack <= none {
			return 1
		}
		return math.Min(1, (beats-on)/each.attack)
	}
	if now < off {
		return rise(now)
	}
	if each.release <= none {
		return 0
	}
	return math.Max(0, rise(off)*(1-(now-off)/each.release))
}

// flashing is the latest flash's peak, beats since its latest strike at
// or before the moment, and its decay, if one began after the last quiet
// and has been struck.
func (one *track) flashing(at, quiet time.Duration,
	clock Clock) (float64, float64, float64, bool) {
	var latest *flash
	for i := range one.flashes {
		each := &one.flashes[i]
		if each.from <= at && each.from > quiet &&
			(latest == nil || each.from >= latest.from) {
			latest = each
		}
	}
	if latest == nil {
		return 0, 0, 0, false
	}
	struck := untold
	for _, moment := range latest.strikes {
		if moment <= at {
			struck = clock.Beats(moment)
		}
	}
	if struck == untold {
		return 0, 0, 0, false
	}
	return latest.peak, clock.Beats(at) - struck, latest.decay, true
}
