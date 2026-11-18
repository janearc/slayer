package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"testing/synctest"
)

// TestPipelineBoundedOverlap blocks the writer deliberately. A serial
// implementation cannot draw frame one; an unbounded one draws frame two.
func TestPipelineBoundedOverlap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		done := make(chan error, 1)
		var drawn, written []int
		go func() {
			_, err := pipeline(t.Context(), 4, func(_ context.Context, index int) (func() error, error) {
				drawn = append(drawn, index)
				return func() error {
					if index == 0 {
						<-release
					}
					written = append(written, index)
					return nil
				}, nil
			}, false)
			done <- err
		}()
		synctest.Wait()
		if !reflect.DeepEqual(drawn, []int{0, 1}) || len(written) != 0 {
			t.Errorf("blocked writer: drew %v, wrote %v; want only two live frames", drawn, written)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(written, []int{0, 1, 2, 3}) {
			t.Fatalf("lost, repeated or reordered output: %v", written)
		}
	})
}

// TestPipelineOutputFailure stops the in-flight producer and preserves the
// actual disk/encoder error instead of reporting only context cancellation.
func TestPipelineOutputFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		broken := errors.New("disk refused the frame")
		second := make(chan struct{})
		draws, outputs := 0, 0
		_, err := pipeline(t.Context(), 5, func(ctx context.Context, index int) (func() error, error) {
			draws++
			if index == 1 {
				close(second)
				<-ctx.Done()
				return nil, context.Cause(ctx)
			}
			return func() error {
				outputs++
				<-second
				return broken
			}, nil
		}, false)
		if !errors.Is(err, broken) || draws != 2 || outputs != 1 {
			t.Fatalf("failure %v, draws %d, outputs %d", err, draws, outputs)
		}
	})
}

// TestPipelineDrawFailureJoinsOutput ensures cleanup cannot close the movie
// or scene while the finishing worker still owns an earlier frame.
func TestPipelineDrawFailureJoinsOutput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		broken := errors.New("trace failed")
		writing, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		go func() {
			_, err := pipeline(t.Context(), 3, func(_ context.Context, index int) (func() error, error) {
				if index == 1 {
					<-writing
					return nil, broken
				}
				return func() error {
					close(writing)
					<-release
					return nil
				}, nil
			}, false)
			done <- err
		}()
		synctest.Wait()
		select {
		case err := <-done:
			t.Errorf("returned before output was joined: %v", err)
		default:
		}
		close(release)
		synctest.Wait()
		select {
		case err := <-done:
			if !errors.Is(err, broken) {
				t.Fatal(err)
			}
		default:
			t.Error("did not return after output finished")
		}
	})
}

// TestPipelineCancellationDropsWaitingFrame exercises a full handoff when
// cancellation arrives; the waiting frame must never reach the movie.
func TestPipelineCancellationDropsWaitingFrame(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(nil)
		stopped := errors.New("owner stopped the render")
		release := make(chan struct{})
		done := make(chan error, 1)
		outputs := 0
		go func() {
			_, err := pipeline(ctx, 4, func(_ context.Context, index int) (func() error, error) {
				return func() error {
					outputs++
					<-release
					return nil
				}, nil
			}, false)
			done <- err
		}()
		synctest.Wait()
		cancel(stopped)
		close(release)
		if err := <-done; !errors.Is(err, stopped) || outputs != 1 {
			t.Fatalf("cancelled output count %d, error %v", outputs, err)
		}
	})
}

// TestPipelineSerialControl keeps the comparison path strictly alternating.
func TestPipelineSerialControl(t *testing.T) {
	var events []int
	stats, err := pipeline(t.Context(), 3, func(_ context.Context, index int) (func() error, error) {
		events = append(events, index*2)
		return func() error {
			events = append(events, index*2+1)
			return nil
		}, nil
	}, true)
	if err != nil || stats.Mode != "serial" || !reflect.DeepEqual(events, []int{0, 1, 2, 3, 4, 5}) {
		t.Fatalf("serial control %v, %v, %+v", events, err, stats)
	}
}
