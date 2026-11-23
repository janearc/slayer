package pages

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// The seconds barkeep's line is reckoned in.
const (
	// paceWindow is how far back the pace and the time left are taken.
	paceWindow = 600 // defined: progress-while.sh's ten minutes
	// liveLine is the age under which a line is someone's live render.
	liveLine = 120 // defined: barkeep's renderStale
	// secondsAMinute turns a count a second into a count a minute.
	secondsAMinute = 60 // defined: a minute
)

// unknownLeft is the time left while it cannot be said.
const unknownLeft = -1 // defined: barkeep reads -1 as not yet known

// epochField is the place of the line's time among its five numbers.
const epochField = 4 // defined: done total frames_per_minute eta epoch

// lineFields is how many numbers barkeep's line holds.
const lineFields = 5 // defined: the five numbers barkeep reads from its line

// pace is one render's counts over the last ten minutes, oldest first:
// when each was taken and how many pictures were done by then.
type pace struct {
	whens  []int64
	counts []int64
}

// add keeps one count and gives the pictures a minute and the seconds
// left, both over the window, as progress-while.sh reckons them.
func (p *pace) add(now, done, total int64) (perMinute, left int64) {
	p.whens, p.counts = append(p.whens, now), append(p.counts, done)
	for now-p.whens[0] > paceWindow {
		p.whens, p.counts = p.whens[1:], p.counts[1:]
	}
	span, made := now-p.whens[0], done-p.counts[0]
	if span > 0 && made > 0 {
		perMinute = made * secondsAMinute / span
	}
	switch {
	case done >= total:
		return perMinute, 0
	case span > 0 && made > 0:
		return perMinute, (total - done) * span / made
	}
	return perMinute, unknownLeft
}

// barkeep keeps barkeep's one line and the time of the last line it
// wrote there, so that it knows its own.
type barkeep struct {
	path string
	ours int64
}

// lineTime is the time a barkeep line says it was written, if it can be
// read.
func lineTime(path string) (int64, bool) {
	data, err := os.ReadFile(path)
	fields := strings.Fields(string(data))
	if err != nil || len(fields) < lineFields {
		return 0, false
	}
	when, err := strconv.ParseInt(fields[epochField], 10, 64)
	return when, err == nil
}

// free says whether the line is barkeep's to write: there is none, it
// cannot be read, it was its own, or it is older than a live render's.
func (keeper *barkeep) free(now int64) bool {
	when, read := lineTime(keeper.path)
	return !read || when == keeper.ours || now-when >= liveLine
}

// write puts one line in place whole, by a rename.
func (keeper *barkeep) write(done, total, perMinute, left, now int64) error {
	line := fmt.Sprintf("%d %d %d %d %d\n", done, total, perMinute, left, now)
	if err := writeBeside(keeper.path, []byte(line), written); err != nil {
		return err
	}
	keeper.ours = now
	return nil
}
