package observability

import (
	"io"
	"os"
	"sync"
	"sync/atomic"
)

var summaryFailures atomic.Uint64
var summaryLogDrops atomic.Uint64
var summaryLogOnce sync.Once
var summaryLog chan summaryLogEntry

type summaryLogEntry struct {
	file *os.File
	text string
}

// NoteSummaryFailure records an optional reporter failure without any I/O.
func NoteSummaryFailure(writer io.Writer, text string) {
	summaryFailures.Add(1)
	file, ok := writer.(*os.File)
	if !ok || file == nil {
		summaryLogDrops.Add(1)
		return
	}
	if len(text) > 2048 {
		text = text[:2048]
	}
	summaryLogOnce.Do(func() {
		summaryLog = make(chan summaryLogEntry, 16)
		go func() {
			for entry := range summaryLog {
				_, _ = io.WriteString(entry.file, entry.text)
			}
		}()
	})
	select {
	case summaryLog <- summaryLogEntry{file, text}:
	default:
		summaryLogDrops.Add(1)
	}
}

// SummaryFailures is the number of optional reporter failures in this process.
func SummaryFailures() uint64 { return summaryFailures.Load() }

// SummaryLogDrops counts warnings omitted because the optional log was full.
func SummaryLogDrops() uint64 { return summaryLogDrops.Load() }
