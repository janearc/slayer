package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/improv/searching"
)

// progressReading counts finished PNG files, not encoder submissions. ETA
// extrapolates recent output intervals; it cannot predict later shot costs.
type progressReading struct {
	Updated       time.Time  `json:"updated"`
	Status        string     `json:"status"`
	Written       int        `json:"png_frames_written"`
	Total         int        `json:"frames_expected"`
	Percent       float64    `json:"percent"`
	Bytes         int64      `json:"png_bytes_written"`
	Elapsed       float64    `json:"seconds_since_first_png"`
	Idle          float64    `json:"seconds_since_last_png"`
	PerMinute     float64    `json:"recent_frames_per_minute"`
	SecondsFrame  float64    `json:"recent_seconds_per_frame"`
	Remaining     *float64   `json:"estimated_seconds_remaining,omitempty"`
	ETA           *time.Time `json:"estimated_finish,omitempty"`
	Frame         string     `json:"latest_frame,omitempty"`
	EstimateBasis string     `json:"estimate_basis"`
}

// completedPNG holds only the metadata needed by the progress watcher.
type completedPNG struct {
	name string
	at   time.Time
	size int64
}

// scanProgress ignores unfinished PNGs, foreign names and frames outside
// the requested excerpt. It reads only each file's signature and trailer.
func scanProgress(root string, from, count int, now time.Time) (progressReading, error) {
	reading := progressReading{Updated: now, Status: "observing", Total: count,
		EstimateBasis: "last 24 completed PNGs; later scenes may cost differently"}
	entries, err := os.ReadDir(filepath.Join(root, "frames"))
	if err != nil {
		return reading, err
	}
	var frames []completedPNG
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || len(name) != len("frame-000000.png") {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "frame-"), ".png"))
		if err != nil || name != fmt.Sprintf("frame-%06d.png", index) || index < from || index-from >= count {
			continue
		}
		file, err := os.Open(filepath.Join(root, "frames", name))
		if err != nil {
			return reading, err
		}
		info, statErr := file.Stat()
		if statErr != nil {
			file.Close()
			return reading, statErr
		}
		complete := info.Mode().IsRegular() && endedPNG(file, info.Size())
		file.Close()
		if complete {
			frames = append(frames, completedPNG{name, info.ModTime(), info.Size()})
		}
	}
	return measureProgress(reading, frames, now), nil
}

// endedPNG distinguishes a completed writer output from a file still being
// encoded. This is not a full PNG integrity or image-acceptance check.
func endedPNG(file io.ReaderAt, size int64) bool {
	if size < 45 {
		return false
	}
	var header [8]byte
	var end [12]byte
	_, firstErr := file.ReadAt(header[:], 0)
	_, lastErr := file.ReadAt(end[:], size-12)
	return firstErr == nil && lastErr == nil &&
		bytes.Equal(header[:], []byte{137, 80, 78, 71, 13, 10, 26, 10}) &&
		bytes.Equal(end[:], []byte{0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130})
}

// measureProgress uses wall time including the age of the unfinished frame.
// After a long gap it withdraws the ETA instead of promising a past finish.
func measureProgress(reading progressReading, frames []completedPNG, now time.Time) progressReading {
	sort.Slice(frames, func(i, j int) bool { return frames[i].at.Before(frames[j].at) })
	reading.Written = len(frames)
	if reading.Total > 0 {
		reading.Percent = 100 * float64(len(frames)) / float64(reading.Total)
	}
	for _, file := range frames {
		reading.Bytes += file.size
	}
	if len(frames) == 0 {
		return reading
	}
	last := frames[len(frames)-1]
	reading.Frame = filepath.Join("frames", last.name)
	reading.Elapsed = math.Max(0, now.Sub(frames[0].at).Seconds())
	reading.Idle = math.Max(0, now.Sub(last.at).Seconds())
	if len(frames) < 3 {
		return reading
	}
	first := max(0, len(frames)-24)
	intervals := float64(len(frames) - first - 1)
	span := last.at.Sub(frames[first].at).Seconds()
	if span <= 0 {
		return reading
	}
	if reading.Idle > math.Max(30, 3*span/intervals) {
		reading.Status = "no recent frame"
		return reading
	}
	reading.SecondsFrame = (span + reading.Idle) / intervals
	reading.PerMinute = 60 / reading.SecondsFrame
	remaining := float64(max(0, reading.Total-reading.Written)) * reading.SecondsFrame
	reading.Remaining = &remaining
	if remaining < float64(math.MaxInt64)/float64(time.Second) {
		eta := now.Add(time.Duration(remaining * float64(time.Second)))
		reading.ETA = &eta
	}
	return reading
}

// line keeps the count, recent performance and estimated finish in one
// timestamped Barkeep message. It never describes PNGs as a finished movie.
func (reading progressReading) line() string {
	filled := min(12, max(0, int(reading.Percent*12/100)))
	bar := "[" + strings.Repeat("#", filled) + strings.Repeat("-", 12-filled) + "]"
	pace, left := "pace pending", "ETA pending"
	if reading.PerMinute > 0 {
		pace = fmt.Sprintf("%.1f fr/min %.1fs/fr", reading.PerMinute, reading.SecondsFrame)
	}
	if reading.ETA != nil && reading.Remaining != nil {
		left = fmt.Sprintf("~%s left ETA %s", time.Duration(*reading.Remaining*float64(time.Second)).Round(time.Second), reading.ETA.Local().Format("15:04"))
	}
	return fmt.Sprintf("%s heard. %s %d/%d %.1f%% PNG %s elapsed %s %s (%s)",
		reading.Updated.Local().Format("15:04:05"), bar, reading.Written, reading.Total, reading.Percent,
		pace, time.Duration(reading.Elapsed*float64(time.Second)).Round(time.Second), left, reading.Status)
}

// progress watches an existing take without changing or rebuilding its
// producer. Configuration and optional Barkeep routing go through Pzm.
func progress(ctx context.Context, call pzm.Call) error {
	if len(call.Args()) != 1 {
		return fmt.Errorf("one render directory is required")
	}
	root := call.Args()[0]
	edit, err := searching.Read(filepath.Join(root, "shoot.yaml"))
	if err != nil {
		return err
	}
	from, err := integer(call, "from", 0)
	if err != nil {
		return err
	}
	count, err := integer(call, "frames", edit.Frames()-from)
	if err != nil || from < 0 || count < 1 || from >= edit.Frames() || count > edit.Frames()-from {
		return fmt.Errorf("progress range outside song")
	}
	pid, err := integer(call, "pid", 0)
	if err != nil || pid < 0 {
		return fmt.Errorf("pid must be nonnegative")
	}
	bar, err := progressBar(call)
	if err != nil {
		return err
	}
	if call.DryRun() {
		return nil
	}
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		reading, err := scanProgress(root, from, count, time.Now())
		if err != nil {
			return err
		}
		completed, err := progressComplete(root, from, count)
		if err != nil {
			return err
		}
		framesComplete, err := progressFramesComplete(root, from, count)
		if err != nil {
			return err
		}
		if completed && reading.Written == count {
			reading.Status = "completed movie"
			reading.ETA, reading.Remaining = nil, nil
		} else if framesComplete && reading.Written == count {
			reading.Status = "completed frame set"
			reading.ETA, reading.Remaining = nil, nil
		} else if pid > 0 && syscall.Kill(pid, 0) == syscall.ESRCH {
			reading.Status = "stopped without completion"
			reading.ETA, reading.Remaining = nil, nil
		} else if reading.Written == count {
			reading.Status = "PNGs written; movie completion pending"
			reading.ETA, reading.Remaining = nil, nil
		}
		if err := saveProgress(root, reading); err != nil {
			return err
		}
		if path := call.Value("progress-file"); path != "" {
			if err := publishProgress(path, []byte(reading.column())); err != nil {
				return fmt.Errorf("Barkeep status file: %w", err)
			}
		}
		fmt.Fprintln(call.Out(), reading.line())
		if bar != nil {
			if err := bar.update(ctx, reading); err != nil {
				return fmt.Errorf("Barkeep update: %w", err)
			}
		}
		if !call.On("follow") || strings.HasPrefix(reading.Status, "completed") || strings.HasPrefix(reading.Status, "stopped") {
			return nil
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-tick.C:
		}
	}
}

// progressFramesComplete admits a finished still collection only with its
// explicit receipt and exact ordered range. scanProgress independently checks
// that all corresponding PNG writers have finished.
func progressFramesComplete(root string, from, count int) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var receipt struct {
		Status string `json:"status"`
		Frames []struct {
			File string `json:"file"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return false, err
	}
	if receipt.Status != "complete" {
		return false, nil
	}
	if len(receipt.Frames) != count {
		return false, fmt.Errorf("frame-set receipt names a different frame count")
	}
	for i, frame := range receipt.Frames {
		if frame.File != fmt.Sprintf("frame-%06d.png", from+i) {
			return false, fmt.Errorf("frame-set receipt names a different frame range")
		}
	}
	return true, nil
}

// progressComplete requires the producer's completion receipt and movie,
// with the same frame range. A full PNG directory alone is not completion.
func progressComplete(root string, from, count int) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "render.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var receipt struct {
		From  int `json:"source_start_frame"`
		Count int `json:"frames"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return false, err
	}
	if receipt.From != from || receipt.Count != count {
		return false, fmt.Errorf("completion receipt names a different frame range")
	}
	info, err := os.Stat(filepath.Join(root, "animation.mp4"))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0, err
}

// saveProgress atomically publishes only the watcher's own status file.
// The completed take's receipt and its producer provenance remain untouched.
func saveProgress(root string, reading progressReading) error {
	raw, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		return err
	}
	return publishProgress(filepath.Join(root, "progress.json"), append(raw, '\n'))
}

// column follows the side column's five-integer contract. Unknown ETA is
// -1; zero is reserved for a movie or frame set with its completion receipt. The reader
// expires the last snapshot after 120 seconds without a heartbeat.
func (reading progressReading) column() string {
	seconds := int64(-1)
	if reading.Status == "completed movie" || reading.Status == "completed frame set" {
		seconds = 0
	} else if reading.Status == "observing" && reading.Remaining != nil &&
		*reading.Remaining > 0 && *reading.Remaining < float64(math.MaxInt64) {
		seconds = int64(math.Ceil(*reading.Remaining))
	}
	return fmt.Sprintf("%d %d %.0f %d %d\n", reading.Written, reading.Total,
		math.Round(reading.PerMinute), seconds, reading.Updated.Unix())
}

// publishProgress renames a complete snapshot within its destination
// directory so readers never observe a truncated or mixed update.
func publishProgress(path string, raw []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".progress-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(raw)
	closed := file.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	return os.Rename(file.Name(), path)
}
