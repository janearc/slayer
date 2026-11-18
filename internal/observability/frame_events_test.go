//go:build darwin || linux

package observability

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func eventPair(t *testing.T) (int, int, func()) {
	t.Helper()
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = unix.SetNonblock(pair[0], true); err != nil {
		t.Fatal(err)
	}
	if err = unix.SetNonblock(pair[1], true); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	closeReceive := func() { once.Do(func() { unix.Close(pair[1]) }) }
	t.Cleanup(closeReceive)
	return pair[0], pair[1], closeReceive
}

func TestFrameEventDeliveryKeptAndClosure(t *testing.T) {
	send, recv, _ := eventPair(t)
	sender, err := NewFrameSender(send, "0123456789abcdef0123456789abcdef", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sender.Close)
	if !sender.Emit(FrameEvent{Frame: 7, Kept: true, SceneTime: time.Second}) {
		t.Fatal("event not sent")
	}
	var buffer [MaxFramePacket]byte
	n, _, err := unix.Recvfrom(recv, buffer[:], unix.MSG_DONTWAIT)
	if err != nil {
		t.Fatal(err)
	}
	var event FramePacket
	if err = json.Unmarshal(buffer[:n], &event); err != nil || event.Version != 1 || event.Kind != "frame" || event.Sequence != 3 || event.Attempt != 2 || event.Frame != 7 || !event.Kept {
		t.Fatal(event, err)
	}
	sender.Close()
	n, _, err = unix.Recvfrom(recv, buffer[:], unix.MSG_DONTWAIT)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(buffer[:n], &event); err != nil || event.Kind != "end" || event.Attempted != 1 || event.Dropped != 0 {
		t.Fatal(event, err)
	}
	if sender.Emit(FrameEvent{}) {
		t.Fatal("accepted event after close")
	}
}

func TestFrameEventBackpressureAndClosedPeerNeverBlock(t *testing.T) {
	send, _, closeReceive := eventPair(t)
	sender, err := NewFrameSender(send, "0123456789abcdef0123456789abcdef", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sender.Close)
	started := time.Now()
	for i := 0; i < 10000; i++ {
		sender.Emit(FrameEvent{Frame: i})
	}
	if sender.Dropped() == 0 || time.Since(started) > time.Second {
		t.Fatal("not bounded", sender.Dropped(), time.Since(started))
	}
	closeReceive()
	if sender.Emit(FrameEvent{Frame: 10001}) {
		t.Fatal("closed peer accepted")
	}
	sender.Close()
}

func TestFrameSenderRejectsInvalidDescriptorAndIdentity(t *testing.T) {
	if _, err := NewFrameSender(-1, "bad", 0, 0); err == nil {
		t.Fatal("accepted invalid")
	}
}

func TestConcurrentEmitAndCloseNeverClaimsIncompleteTerminal(t *testing.T) {
	for iteration := 0; iteration < 25; iteration++ {
		pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
		if err != nil {
			t.Fatal(err)
		}
		if err = unix.SetNonblock(pair[1], true); err != nil {
			t.Fatal(err)
		}
		sender, err := NewFrameSender(pair[0], "0123456789abcdef0123456789abcdef", 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var group sync.WaitGroup
		for i := 0; i < 16; i++ {
			group.Add(1)
			go func(frame int) { defer group.Done(); <-start; sender.Emit(FrameEvent{Frame: frame}) }(i)
		}
		close(start)
		sender.Close()
		group.Wait()
		var count uint64
		var terminal *FramePacket
		for {
			var buffer [MaxFramePacket]byte
			n, _, err := unix.Recvfrom(pair[1], buffer[:], unix.MSG_DONTWAIT)
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK || err == unix.ECONNRESET {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			var packet FramePacket
			if err = json.Unmarshal(buffer[:n], &packet); err != nil {
				t.Fatal(err)
			}
			if terminal != nil {
				t.Fatal("packet after terminal")
			}
			if packet.Kind == "frame" {
				count++
			} else if packet.Kind == "end" {
				copy := packet
				terminal = &copy
			}
		}
		if terminal != nil && (terminal.Ordinal != count || terminal.Attempted != count+terminal.Dropped) {
			t.Fatal("ambiguous terminal", count, *terminal)
		}
		unix.Close(pair[1])
	}
}
