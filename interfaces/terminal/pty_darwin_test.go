package terminal

import (
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ptyPair is a pseudo-terminal made for the test and closed with it: a
// master end the test reads and writes as a terminal's screen and
// keyboard, and a slave end that is a terminal as far as ioctl, Measure
// and Open can tell. it is not the terminal the tests are run from, and
// is never in the test's os.Stdin, os.Stdout or os.Stderr.
//
// this is the darwin way to make one (/dev/ptmx, then grant, unlock and
// ask the slave's name); other systems need their own, so the tests that
// use it are darwin only.
func ptyPair(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal to be had: %v", err)
	}
	for _, req := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
		if err := unix.IoctlSetInt(int(master.Fd()), req, 0); err != nil {
			master.Close()
			t.Skipf("no pseudo-terminal to be had: ioctl %#x: %v", req, err)
		}
	}
	var name [128]byte
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0]))); e != 0 {
		master.Close()
		t.Skipf("no pseudo-terminal to be had: its name: %v", e)
	}
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	slave, err = os.OpenFile(string(name[:n]), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		t.Skipf("no pseudo-terminal to be had: %v", err)
	}
	t.Cleanup(func() {
		slave.Close()
		master.Close()
	})
	return master, slave
}

// reopen is the slave a second time, for reading only, which is a
// terminal that a write to cannot get through.
func reopen(t *testing.T, slave *os.File) *os.File {
	t.Helper()
	f, err := os.OpenFile(slave.Name(), os.O_RDONLY|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

// resize tells the slave how big it is, through the master, with the
// cells and the pixels given.
func resize(t *testing.T, master *os.File, cols, rows, xpix, ypix int) {
	t.Helper()
	ws := &unix.Winsize{Col: uint16(cols), Row: uint16(rows), Xpixel: uint16(xpix), Ypixel: uint16(ypix)}
	if err := unix.IoctlSetWinsize(int(master.Fd()), unix.TIOCSWINSZ, ws); err != nil {
		t.Fatal(err)
	}
}

// screenOf is what the master has been sent so far, waiting up to three
// seconds for at least want bytes and no longer. a master read would
// block with nothing to read, so it asks first.
func screenOf(t *testing.T, master *os.File, want int) string {
	t.Helper()
	var got []byte
	buf := make([]byte, 256)
	end := time.Now().Add(3 * time.Second)
	for len(got) < want && time.Now().Before(end) {
		fds := []unix.PollFd{{Fd: int32(master.Fd()), Events: unix.POLLIN}}
		if n, err := unix.Poll(fds, 100); err != nil || n == 0 {
			continue
		}
		n, err := unix.Read(int(master.Fd()), buf)
		if err != nil || n <= 0 {
			break
		}
		got = append(got, buf[:n]...)
	}
	return string(got)
}

// settings is the terminal's modes, to see whether they changed.
func settings(t *testing.T, f *os.File) unix.Termios {
	t.Helper()
	tio, err := unix.IoctlGetTermios(int(f.Fd()), unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	return *tio
}
