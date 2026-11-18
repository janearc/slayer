//go:build darwin || linux

package observability

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// FrameSender owns one nonblocking datagram descriptor.
type FrameSender struct {
	fd                         int
	nonce                      string
	sequence, attempt, ordinal uint64
	mu                         sync.Mutex
	state                      atomic.Uint64
	attempted, dropped         atomic.Uint64
}

// NewFrameSender validates and owns an already-connected datagram socket.
func NewFrameSender(fd int, nonce string, sequence, attempt uint64) (*FrameSender, error) {
	if fd < 0 || !validFrameNonce(nonce) || sequence == 0 || attempt == 0 {
		return nil, fmt.Errorf("invalid frame sender")
	}
	typeOf, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_TYPE)
	if err != nil || typeOf != unix.SOCK_DGRAM {
		return nil, fmt.Errorf("frame descriptor is not a datagram socket")
	}
	unix.CloseOnExec(fd)
	return &FrameSender{fd: fd, nonce: nonce, sequence: sequence, attempt: attempt}, nil
}

// Emit tries once and drops on contention, backpressure or transport failure.
func (s *FrameSender) Emit(event FrameEvent) bool {
	if s == nil || !s.begin() {
		return false
	}
	defer s.state.Add(^uint64(0))
	s.attempted.Add(1)
	if !s.mu.TryLock() {
		s.dropped.Add(1)
		return false
	}
	defer s.mu.Unlock()
	if s.state.Load()&frameClosed != 0 {
		s.dropped.Add(1)
		return false
	}
	packet := FramePacket{Version: 1, Kind: "frame", Nonce: s.nonce, Sequence: s.sequence, Attempt: s.attempt, Ordinal: s.ordinal + 1, Attempted: s.attempted.Load(), Dropped: s.dropped.Load(), FrameEvent: event}
	if !s.send(packet) {
		s.dropped.Add(1)
		return false
	}
	s.ordinal++
	return true
}

const frameClosed = uint64(1) << 63

// begin reserves a send slot, or reports false once the sender is closed.
func (s *FrameSender) begin() bool {
	for {
		state := s.state.Load()
		if state&frameClosed != 0 || state == frameClosed-1 {
			return false
		}
		if s.state.CompareAndSwap(state, state+1) {
			return true
		}
	}
}

// send writes one packet to the socket without blocking, and reports
// whether it was sent.
func (s *FrameSender) send(packet FramePacket) bool {
	raw, err := json.Marshal(packet)
	if err != nil || len(raw) > MaxFramePacket {
		return false
	}
	n, err := unix.SendmsgN(s.fd, raw, nil, nil, unix.MSG_DONTWAIT|unix.MSG_NOSIGNAL)
	return err == nil && n == len(raw)
}

// Dropped is the number of attempted observations that were not delivered.
func (s *FrameSender) Dropped() uint64 {
	if s == nil {
		return 0
	}
	return s.dropped.Load()
}

// Close sends a best-effort terminal count and closes the sender once.
func (s *FrameSender) Close() {
	if s == nil {
		return
	}
	var active uint64
	for {
		state := s.state.Load()
		if state&frameClosed != 0 {
			return
		}
		if s.state.CompareAndSwap(state, state|frameClosed) {
			active = state
			break
		}
	}
	if active != 0 {
		// An in-flight event makes terminal accounting ambiguous. Do not send a
		// terminal; close after the nonblocking sender leaves its critical section.
		go func() { s.mu.Lock(); _ = unix.Close(s.fd); s.mu.Unlock() }()
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.send(FramePacket{Version: 1, Kind: "end", Nonce: s.nonce, Sequence: s.sequence, Attempt: s.attempt, Ordinal: s.ordinal, Attempted: s.attempted.Load(), Dropped: s.dropped.Load()})
	_ = unix.Close(s.fd)
}
