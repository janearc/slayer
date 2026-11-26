package trace

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/janearc/slayer/render/frame"
)

// exactRows draws complete picture rows, retaining only rows which completed.
func exactRows(keeping Keeping, width, height int,
	pixel func(x, y int) (frame.Pixel, bool), valid func() bool,
) (*frame.Frame, error) {
	if keeping.told() {
		return nil, keeping.stop.Err()
	}
	made := &painting{keeping: keeping, picture: frame.New(width, height), left: make(chan int, height)}
	made.takeUp()
	close(made.left)
	var workers sync.WaitGroup
	var incomplete atomic.Bool
	for range max(fewestWorker, runtime.GOMAXPROCS(0)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for row := range made.left {
				pixels := make([]frame.Pixel, width)
				for x := range pixels {
					if keeping.told() {
						return
					}
					value, whole := pixel(x, row)
					if !whole {
						incomplete.Store(true)
						return
					}
					pixels[x] = value
				}
				made.put(row, pixels)
				if keeping.rows != nil && valid() {
					keeping.rows.Drawn(row, pixels)
				}
			}
		}()
	}
	workers.Wait()
	if incomplete.Load() || int(made.finished.Load()) != height {
		if keeping.told() {
			return nil, keeping.stop.Err()
		}
		return nil, fmt.Errorf("trace: exact draw did not finish every row")
	}
	return made.picture, nil
}
