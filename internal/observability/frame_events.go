package observability

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

// MaxFramePacket bounds one versioned local datagram.
const MaxFramePacket = 4096

// FrameEvent describes one completed frame. Kept frames have no draw timing.
type FrameEvent struct {
	Frame          int           `json:"frame"`
	ObservedAt     time.Time     `json:"observed_at"`
	SceneTime      time.Duration `json:"scene_time_ns"`
	SceneTimeKnown bool          `json:"scene_time_known"`
	Kept           bool          `json:"kept"`
	Still          bool          `json:"still"`
	DrawKnown      bool          `json:"draw_known"`
	WriteKnown     bool          `json:"write_known"`
	ReadKnown      bool          `json:"read_known"`
	EncodeKnown    bool          `json:"encode_known"`
	Draw           time.Duration `json:"draw_ns"`
	Develop        time.Duration `json:"develop_ns"`
	Titles         time.Duration `json:"titles_ns"`
	Write          time.Duration `json:"write_ns"`
	Read           time.Duration `json:"read_ns"`
	Encode         time.Duration `json:"encode_ns"`
}

// FramePacket binds a frame or terminal summary to one queue attempt.
type FramePacket struct {
	Version   int    `json:"version"`
	Kind      string `json:"kind"`
	Nonce     string `json:"nonce"`
	Sequence  uint64 `json:"sequence"`
	Attempt   uint64 `json:"attempt"`
	Ordinal   uint64 `json:"ordinal"`
	Attempted uint64 `json:"attempted"`
	Dropped   uint64 `json:"dropped"`
	FrameEvent
}

var frameSender atomic.Pointer[FrameSender]

// FrameEventsEnabled reports whether this process has an active sender.
func FrameEventsEnabled() bool { return frameSender.Load() != nil }

// EmitFrame never waits for a consumer; an unavailable sender is a no-op.
func EmitFrame(event FrameEvent) {
	if sender := frameSender.Load(); sender != nil {
		sender.Emit(event)
	}
}

// StartFrameEventsFromEnv starts only the supervisor's explicit FD handoff.
// Failure can be logged by the caller but must never fail a render.
func StartFrameEventsFromEnv() (func(), error) {
	raw := os.Getenv("AZAZEL_FRAME_FD")
	if raw == "" {
		return func() {}, nil
	}
	fd, err := strconv.Atoi(raw)
	if err != nil || fd != 4 {
		return nil, fmt.Errorf("invalid frame event descriptor")
	}
	sequence, err := strconv.ParseUint(os.Getenv("AZAZEL_FRAME_SEQUENCE"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid frame sequence")
	}
	attempt, err := strconv.ParseUint(os.Getenv("AZAZEL_FRAME_ATTEMPT"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid frame attempt")
	}
	sender, err := NewFrameSender(fd, os.Getenv("AZAZEL_FRAME_NONCE"), sequence, attempt)
	if err != nil {
		return nil, err
	}
	return InstallFrameSender(sender)
}

// InstallFrameSender enables one sender for this process until stopped.
func InstallFrameSender(sender *FrameSender) (func(), error) {
	if sender == nil || !frameSender.CompareAndSwap(nil, sender) {
		if sender != nil {
			sender.Close()
		}
		return nil, fmt.Errorf("frame sender already started or unavailable")
	}
	return func() {
		if frameSender.CompareAndSwap(sender, nil) {
			sender.Close()
		}
	}, nil
}

// validFrameNonce reports whether nonce is 16 bytes of lowercase hex.
func validFrameNonce(nonce string) bool {
	decoded, err := hex.DecodeString(nonce)
	return err == nil && len(decoded) == 16 && hex.EncodeToString(decoded) == nonce
}
