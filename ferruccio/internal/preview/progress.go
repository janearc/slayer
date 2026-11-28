package preview

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Progress is an observed phase and, when available, its completed units.
// A zero Total means the phase has no reliable denominator yet.
type Progress struct {
	Phase       string
	Done, Total int
}

// readAttemptProgress reads only this verified job attempt's recent queue log.
// Missing or incomplete logs simply leave the progress indeterminate.
func readAttemptProgress(queue string, sequence, attempt uint64) Progress {
	if !filepath.IsAbs(queue) || sequence == 0 || attempt == 0 {
		return Progress{}
	}
	path := filepath.Join(queue, fmt.Sprintf("%020d.%04d.log", sequence, attempt))
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return Progress{}
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return Progress{}
	}
	const tail = 64 << 10
	start := max(int64(0), info.Size()-tail)
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return Progress{}
	}
	raw, err := io.ReadAll(io.LimitReader(file, tail))
	if err != nil {
		return Progress{}
	}
	lines := strings.Split(string(raw), "\n")
	if start > 0 && len(lines) > 0 {
		lines = lines[1:]
	}
	// Do not present a line still being written as a complete observation.
	if len(raw) > 0 && raw[len(raw)-1] != '\n' && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	var result Progress
	for _, line := range lines {
		if progress, ok := parseProgressLine(line); ok {
			result = progress
		}
	}
	return result
}

func parseProgressLine(line string) (Progress, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return Progress{}, false
	}
	phase, counts := "", ""
	switch {
	case fields[0] == "progress" && strings.HasPrefix(fields[1], "rows="):
		phase, counts = "Rendering", strings.TrimPrefix(fields[1], "rows=")
	case fields[0] == "staging" && strings.HasPrefix(fields[1], "at="):
		phase = "Staging"
		name := strings.TrimPrefix(fields[1], "at=")
		if name != "" && len(name) <= 32 && strings.Trim(name, "abcdefghijklmnopqrstuvwxyz0123456789-") == "" {
			phase += " · " + name
		}
		if len(fields) > 2 {
			counts = fields[2]
		}
	default:
		return Progress{}, false
	}
	result := Progress{Phase: phase}
	parts := strings.Split(counts, "/")
	if len(parts) != 2 {
		return result, true
	}
	done, doneErr := strconv.Atoi(parts[0])
	total, totalErr := strconv.Atoi(parts[1])
	if doneErr == nil && totalErr == nil && total > 0 && done >= 0 && done <= total {
		result.Done, result.Total = done, total
	}
	return result, true
}
