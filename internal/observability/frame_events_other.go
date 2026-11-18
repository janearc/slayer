//go:build !darwin && !linux

package observability

import "errors"

// FrameSender is unavailable on platforms without the datagram transport.
type FrameSender struct{}

// NewFrameSender reports that frame events are unavailable here.
func NewFrameSender(int, string, uint64, uint64) (*FrameSender, error) {
	return nil, errors.New("frame events unavailable")
}

// Emit reports that no frame event is sent on this platform.
func (*FrameSender) Emit(FrameEvent) bool { return false }

// Dropped reports zero dropped frames, as none are sent here.
func (*FrameSender) Dropped() uint64 { return 0 }

// Close does nothing on this platform.
func (*FrameSender) Close() {}
