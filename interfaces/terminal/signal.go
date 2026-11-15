package terminal

// Giving the terminal back when the process is told to stop.

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalExit is what a shell adds to a signal's number to say a command
// died of it. It is defined as 128.
const SignalExit = 128

// signalRoom is how many signals the watch holds. It is defined as one:
// the first ends the session.
const signalRoom = 1

// Watch is a watch for the signals that stop a process, kept while a
// terminal is taken over.
type Watch struct {
	caught   chan os.Signal
	done     chan struct{}
	over     chan struct{}
	giveBack func()
	exit     func(int)
}

// LeaveOnSignal answers SIGTERM, SIGINT and SIGHUP by calling giveBack,
// which gives the terminal back, then exit with SignalExit plus the
// signal's number. Call Stop when the terminal is given back by other means.
func LeaveOnSignal(giveBack func(), exit func(int)) *Watch {
	watch := &Watch{caught: make(chan os.Signal, signalRoom),
		done: make(chan struct{}), over: make(chan struct{}),
		giveBack: giveBack, exit: exit}
	signal.Notify(watch.caught, syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGHUP)
	go watch.wait()
	return watch
}

// wait is the watch: it gives back on a signal, or ends with Stop.
func (watch *Watch) wait() {
	defer close(watch.over)
	select {
	case got := <-watch.caught:
		watch.giveBack()
		watch.exit(SignalExit + int(got.(syscall.Signal)))
	case <-watch.done:
	}
}

// Stop ends the watch, and waits until it has ended. It must not be called
// from giveBack or exit, which the watch is waiting on.
func (watch *Watch) Stop() {
	signal.Stop(watch.caught)
	close(watch.done)
	<-watch.over
}
