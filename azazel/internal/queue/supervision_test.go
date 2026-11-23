package queue

import (
	"context"
	"errors"
	"go.uber.org/mock/gomock"
	"io"
	"strings"
	"testing"
	"time"
)

// One failed host sample does not kill a render that then reads well.
func TestOneFailedHostSampleDoesNotKillARender(t *testing.T) {
	s, j := executableFixture(t, "exec /bin/sleep 1")
	monitor := unreadHost(t)
	first := monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
	group := gomock.Cond(func(group int) bool { return group > 1 })
	slow := monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, group).Return(Vitals{}, errors.New("sample timed out")).After(first)
	monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, group).Return(Vitals{Ready: true}, nil).AnyTimes().After(slow)
	result := s.execute(t.Context(), j, io.Discard, monitor)
	if result.State != Succeeded || result.Uncertain {
		t.Fatal("one failed sample stopped the render", result)
	}
}

// Anti-test: reads that keep failing stop the render once the allowance is
// spent, and the stored reason says how many failed and over how long.
func TestReadsThatKeepFailingStopTheRenderAndSayHowMany(t *testing.T) {
	s, j := executableFixture(t, "exec /bin/sleep 5")
	s.readLimit = 300 * time.Millisecond
	monitor := unreadHost(t)
	first := monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
	monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).
		Return(Vitals{}, errors.New("sample timed out")).AnyTimes().After(first)
	result := s.execute(t.Context(), j, io.Discard, monitor)
	if result.State != Failed || result.Uncertain || !strings.Contains(result.Reason, "reads failed over") || !strings.Contains(result.Reason, "sample timed out") {
		t.Fatal(result)
	}
}

// A stop signal from a good sample acts at once, whatever the allowance.
func TestAStopFromAGoodSampleActsAtOnce(t *testing.T) {
	s, j := executableFixture(t, "exec /bin/sleep 5")
	s.readLimit = time.Hour
	monitor := unreadHost(t)
	first := monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
	monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).
		Return(Vitals{Stop: true, Reason: "disk floor"}, nil).After(first)
	start := time.Now()
	result := s.execute(t.Context(), j, io.Discard, monitor)
	if result.State != Failed || !result.Stop || time.Since(start) > 4*time.Second {
		t.Fatal(result)
	}
}

// The failure counter spends its allowance from the first failure in a row
// and forgets them when a tick reads well.
func TestReadFailuresCountInARowAndClear(t *testing.T) {
	var failures readFailures
	start := time.Now()
	if failures.spent(start, time.Minute) || failures.spent(start.Add(30*time.Second), time.Minute) {
		t.Fatal("spent early")
	}
	if !failures.spent(start.Add(time.Minute), time.Minute) {
		t.Fatal("not spent at the allowance")
	}
	if got := failures.explain(errors.New("late"), start.Add(time.Minute)).Error(); !strings.Contains(got, "3 host and ledger reads failed over 1m0s") {
		t.Fatal(got)
	}
	failures.clear()
	if failures.spent(start.Add(2*time.Minute), time.Minute) {
		t.Fatal("cleared failures still counted")
	}
}

// Anti-test: an error the allowance cannot help is acted on at once, and
// with no failures counted it is reported as it is.
func TestReadsThatCannotBeRetriedAreNotRetried(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if readMayRetry(cancelled, errors.New("x")) || readMayRetry(t.Context(), ErrOwner) || readMayRetry(t.Context(), ErrMissing) || readMayRetry(t.Context(), ErrUnsafe) {
		t.Fatal("an error that cannot recover was retried")
	}
	if !readMayRetry(t.Context(), errors.New("timeout")) {
		t.Fatal("a plain read error was not retried")
	}
	plain := errors.New("plain")
	if got := (readFailures{}).explain(plain, time.Now()); got != plain {
		t.Fatal(got)
	}
}
