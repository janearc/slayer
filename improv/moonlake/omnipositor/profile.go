package omnipositor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type ProfileSpan struct {
	Frame    int           `json:"frame"`
	Stage    string        `json:"stage"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration_ns"`
}

type Profile struct {
	mu    sync.Mutex
	spans []ProfileSpan
	limit int
}

// NewProfile makes an empty profile that holds at most limit spans. limit
// must be from 1 to 1<<20.
func NewProfile(limit int) (*Profile, error) {
	if limit <= 0 || limit > 1<<20 {
		return nil, errors.New("profile: invalid span limit")
	}
	return &Profile{limit: limit}, nil
}

// Record adds span to the profile. It refuses a nil profile, a span with a
// negative frame, an empty, over-long, invalid UTF-8 or NUL-holding stage
// name, a zero start or negative duration, and any span past the limit.
func (profile *Profile) Record(span ProfileSpan) error {
	if profile == nil || span.Frame < 0 || len(span.Stage) == 0 || len(span.Stage) > 128 || !utf8.ValidString(span.Stage) || strings.ContainsRune(span.Stage, 0) || span.Start.IsZero() || span.Duration < 0 {
		return errors.New("profile: invalid span")
	}
	profile.mu.Lock()
	defer profile.mu.Unlock()
	if len(profile.spans) >= profile.limit {
		return errors.New("profile: span limit reached")
	}
	profile.spans = append(profile.spans, span)
	return nil
}

// Spans returns a copy of the recorded spans in the order they were added.
func (profile *Profile) Spans() []ProfileSpan {
	if profile == nil {
		return nil
	}
	profile.mu.Lock()
	defer profile.mu.Unlock()
	return append([]ProfileSpan(nil), profile.spans...)
}

type traceEvent struct {
	Name      string         `json:"name"`
	Category  string         `json:"cat"`
	Phase     string         `json:"ph"`
	Timestamp float64        `json:"ts"`
	Duration  float64        `json:"dur"`
	Process   int            `json:"pid"`
	Thread    int            `json:"tid"`
	Args      map[string]int `json:"args"`
}

// WriteTrace writes the spans to writer as trace-event JSON, in start order,
// with times in microseconds and the frame number in each event's args.
func (profile *Profile) WriteTrace(writer io.Writer) error {
	if profile == nil || writer == nil {
		return errors.New("profile: nil trace destination")
	}
	spans := profile.Spans()
	sort.SliceStable(spans, func(first, second int) bool { return spans[first].Start.Before(spans[second].Start) })
	events := make([]traceEvent, 0, len(spans))
	for _, span := range spans {
		micros := float64(span.Start.UnixMicro())
		duration := float64(span.Duration) / float64(time.Microsecond)
		if math.IsNaN(micros) || math.IsInf(micros, 0) || math.IsInf(duration, 0) {
			return errors.New("profile: invalid trace time")
		}
		events = append(events, traceEvent{Name: span.Stage, Category: "cpu-wall", Phase: "X", Timestamp: micros, Duration: duration, Process: 1, Thread: 1, Args: map[string]int{"frame": span.Frame}})
	}
	if err := json.NewEncoder(writer).Encode(struct {
		Events []traceEvent `json:"traceEvents"`
	}{events}); err != nil {
		return fmt.Errorf("profile: write trace: %w", err)
	}
	return nil
}
