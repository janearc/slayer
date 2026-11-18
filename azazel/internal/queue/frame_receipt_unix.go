//go:build darwin || linux

package queue

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/janearc/slayer/azazel/internal/budget"
	"github.com/janearc/slayer/internal/observability"
	"golang.org/x/sys/unix"
)

// 64 MiB accommodates ordinary 20-minute, 24-fps receipts while bounding disk.
const frameReceiptLimit = 64 << 20
const frameTrailerReserve = 256
const frameDrainLimit = 256

type frameReceipt struct {
	parent, child     *os.File
	file              *os.File
	nonce             string
	sequence, attempt uint64
	stop, done        chan struct{}
	limit             uint64
	started           bool
	once              sync.Once
}

// newFrameReceipt creates the socket pair and the receipt file for one
// attempt under root. It refuses when the disk is short on space.
func newFrameReceipt(root string, job Job) (*frameReceipt, error) {
	available, err := receiptFree(root)
	if err != nil || !receiptSpaceSafe(available) {
		return nil, fmt.Errorf("receipt disk space unavailable")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil, err
	}
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	parent, child := os.NewFile(uintptr(pair[0]), "frame-receiver"), os.NewFile(uintptr(pair[1]), "frame-sender")
	fail := func(err error) (*frameReceipt, error) { parent.Close(); child.Close(); return nil, err }
	if err = unix.SetNonblock(pair[0], true); err != nil {
		return fail(err)
	}
	if err = unix.SetNonblock(pair[1], true); err != nil {
		return fail(err)
	}
	unix.CloseOnExec(pair[0])
	unix.CloseOnExec(pair[1])
	name := fmt.Sprintf("%020d.%04d.frames.jsonl", job.Sequence, job.Attempt)
	file, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return fail(err)
	}
	return &frameReceipt{parent: parent, child: child, file: file, nonce: hex.EncodeToString(random[:]), sequence: job.Sequence, attempt: job.Attempt, stop: make(chan struct{}), done: make(chan struct{}), limit: frameReceiptLimit}, nil
}

// environment adds the frame receipt variables for the child to env.
func (r *frameReceipt) environment(env []string) []string {
	env = setEnvironment(env, "AZAZEL_FRAME_FD", "4")
	env = setEnvironment(env, "AZAZEL_FRAME_NONCE", r.nonce)
	env = setEnvironment(env, "AZAZEL_FRAME_SEQUENCE", strconv.FormatUint(r.sequence, 10))
	return setEnvironment(env, "AZAZEL_FRAME_ATTEMPT", strconv.FormatUint(r.attempt, 10))
}

// start is called only after the child inherited its sender. The reader owns
// its descriptors; no stalled disk or consumer can hold up the renderer.
func (r *frameReceipt) start() { r.started = true; r.child.Close(); go r.read() }

// close stops the reader and releases its descriptors. A reader that never
// started releases them in the background.
func (r *frameReceipt) close() {
	r.once.Do(func() {
		if !r.started {
			go func() { r.child.Close(); r.parent.Close(); r.file.Close() }()
			return
		}
		close(r.stop)
		select {
		case <-r.done:
		case <-time.After(50 * time.Millisecond):
		}
	})
}

// read copies frame packets from the child into the receipt file until the
// reader is stopped.
func (r *frameReceipt) read() {
	defer close(r.done)
	defer r.parent.Close()
	defer r.file.Close()
	guard := &receiptDiskWriter{file: r.file, free: func() (uint64, error) { return receiptFree(filepath.Dir(r.file.Name())) }}
	state := r.receive(func(buffer []byte) (int, int, error) {
		n, _, flags, _, err := unix.Recvmsg(int(r.parent.Fd()), buffer, nil, unix.MSG_DONTWAIT)
		return n, flags, err
	}, guard)
	trailer, _ := json.Marshal(struct {
		Receipt  string `json:"receipt"`
		Complete bool   `json:"complete"`
		Invalid  uint64 `json:"invalid"`
		Overflow bool   `json:"overflow"`
		Cutoff   bool   `json:"cutoff"`
	}{"frame-observation-v1", state.complete(), state.invalid, state.overflow, state.cutoff})
	_, _ = guard.Write(append(trailer, '\n'))
}

// receiptSpaceSafe reports whether free bytes leave room for a receipt above
// the queue disk floor.
func receiptSpaceSafe(free uint64) bool { return free >= budget.QueueDiskFloorBytes+frameReceiptLimit }

// receiptFree returns the bytes available to unprivileged writers in
// directory.
func receiptFree(directory string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(directory, &stat); err != nil {
		return 0, err
	}
	if stat.Bsize == 0 || stat.Bavail > stat.Bfree || stat.Bfree > stat.Blocks || uint64(stat.Bavail) > math.MaxUint64/uint64(stat.Bsize) {
		return 0, fmt.Errorf("invalid receipt filesystem statistics")
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

type receiptDiskWriter struct {
	file io.Writer
	free func() (uint64, error)
}

// Write writes raw when the disk still has room for a receipt.
func (w *receiptDiskWriter) Write(raw []byte) (int, error) {
	free, err := w.free()
	if err != nil || !receiptSpaceSafe(free) {
		return 0, fmt.Errorf("receipt disk space unavailable")
	}
	return w.file.Write(raw)
}

// receive checks for stop on every packet and drains at most frameDrainLimit
// after stop, even if a peer sends continuously. A cutoff is never complete.
func (r *frameReceipt) receive(read func([]byte) (int, int, error), out io.Writer) receiptState {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var state receiptState
	stopping := false
	drained := 0
	for {
		if !stopping {
			select {
			case <-r.stop:
				stopping = true
			default:
			}
		}
		if stopping && drained >= frameDrainLimit {
			state.cutoff = true
			break
		}
		var buffer [observability.MaxFramePacket + 1]byte
		n, flags, err := read(buffer[:])
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			if stopping {
				break
			}
			select {
			case <-r.stop:
				stopping = true
			case <-ticker.C:
			}
			continue
		}
		if err == unix.ECONNRESET {
			break
		}
		if err != nil {
			state.invalid++
			break
		}
		if stopping {
			drained++
		}
		if err = r.accept(&state, buffer[:n], flags, out); err != nil {
			state.invalid++
			break
		}
	}
	return state
}

type receiptState struct {
	size, ordinal, invalid     uint64
	terminal, overflow, cutoff bool
	end                        observability.FramePacket
}

// complete reports whether the receipt holds a terminal packet and every frame
// it announced, with no overflow, cutoff or invalid packet.
func (s receiptState) complete() bool {
	return s.terminal && !s.overflow && !s.cutoff && s.invalid == 0 && s.end.Dropped == 0 && s.end.Attempted == s.ordinal && s.end.Ordinal == s.ordinal
}

// accept checks one packet, writes it to out when the receipt has room, and
// counts it.
func (r *frameReceipt) accept(state *receiptState, raw []byte, flags int, out io.Writer) error {
	packet, err := r.packet(raw, flags, state.ordinal, state.terminal)
	if err != nil {
		state.invalid++
		return nil
	}
	if packet.Kind == "frame" {
		state.ordinal++
	} else {
		state.terminal = true
		state.end = packet
	}
	limit := r.limit
	if limit == 0 {
		limit = frameReceiptLimit
	}
	if limit <= frameTrailerReserve || state.size+uint64(len(raw))+1 > limit-frameTrailerReserve {
		state.overflow = true
		return nil
	}
	line := append(append([]byte(nil), raw...), '\n')
	written, err := out.Write(line)
	if err == nil && written != len(line) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return err
	}
	state.size += uint64(written)
	return nil
}

// packet decodes one frame packet. It refuses empty, oversize or truncated
// input, unknown fields, and any packet after the terminal one.
func (r *frameReceipt) packet(raw []byte, flags int, ordinal uint64, terminal bool) (observability.FramePacket, error) {
	var p observability.FramePacket
	if len(raw) == 0 || len(raw) > observability.MaxFramePacket || flags&unix.MSG_TRUNC != 0 || terminal {
		return p, fmt.Errorf("invalid frame packet")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return p, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return p, fmt.Errorf("trailing frame packet")
	}
	if p.Version != 1 || p.Nonce != r.nonce || p.Sequence != r.sequence || p.Attempt != r.attempt {
		return p, fmt.Errorf("frame attempt mismatch")
	}
	if p.Kind == "frame" {
		if p.Ordinal != ordinal+1 || p.Frame < 0 || p.SceneTime < 0 || p.Draw < 0 || p.Develop < 0 || p.Titles < 0 || p.Write < 0 || p.Read < 0 || p.Encode < 0 {
			return p, fmt.Errorf("invalid frame observation")
		}
	} else if p.Kind != "end" || p.Ordinal != ordinal || p.Attempted < p.Dropped || p.Attempted-p.Dropped != ordinal {
		return p, fmt.Errorf("invalid frame terminal")
	}
	return p, nil
}
