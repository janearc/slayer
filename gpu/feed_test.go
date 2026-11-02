//go:build darwin

package gpu

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// feeding is one frame of the swarm, encoded and baked once, so that
// what is measured is handing it to the gpu and nothing before
type feeding struct {
	width, height int
	inputs        [][]byte
}

// swarmToFeed is the swarm command's own scene at its defaults: a star
// of radius one in sixty orbits of panels, in haze, seen from 26 radii,
// at 1280 by 720 and one ray a pixel
func swarmToFeed(b *testing.B) feeding {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(0.002, 0.002, 0.004), Seed: 7}
	_, err := lantern.Lantern{Radius: 1, Orbits: 60, Panels: 120,
		Kelvin: 4800, Green: 38, Wide: 0.4, Turn: time.Minute,
		Skin: lantern.BlackGlass(0), Lining: lantern.Matte(0),
	}.Place(world)
	if err != nil {
		b.Fatal(err)
	}
	wisps := medium.Wispy{Of: medium.Puff{Radius: 14, Thick: 0.05},
		Seed: 3, Scale: 2.5, Amount: 0.45}
	world.AddVolume(thing.Volume{Medium: wisps, Material: &thing.Material{
		Name: "haze", Colour: maths.RGB(0.9, 0.82, 0.72)}})
	from := maths.V(0, 4, 26).Unit().Scale(26)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	lens := film.Pinhole{Wide: 45 * maths.Degree}
	prepared := feeding{width: 1280, height: 720}
	encoded, err := encode(world, eye, lens, 0, prepared.width,
		prepared.height)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := encoded.bakeAll(); err != nil {
		b.Fatal(err)
	}
	prepared.inputs = encoded.inputs()
	return prepared
}

// BenchmarkFeeders draws that frame again and again from one to six
// cpu threads at once, each with a kernel of its own and so a command
// queue of its own, to see whether more feeders keep the gpu busier.
// Then one to four feeders again, each frame sent as one command instead
// of bands of rows, to see what waiting between the bands costs.
func BenchmarkFeeders(b *testing.B) {
	prepared := swarmToFeed(b)
	for feeders := 1; feeders <= 6; feeders++ {
		b.Run(fmt.Sprintf("feeders=%d", feeders), func(b *testing.B) {
			feed(b, prepared, feeders, rows)
		})
	}
	for feeders := 1; feeders <= 4; feeders++ {
		name := fmt.Sprintf("feeders=%d,unbanded", feeders)
		b.Run(name, func(b *testing.B) {
			feed(b, prepared, feeders, 0)
		})
	}
}

// feed is one case: feeders goroutines sharing out b.N frames, with
// GOMAXPROCS set to match, reporting frames a second and the cpu time
// the whole process spent on each frame
func feed(b *testing.B, prepared feeding, feeders, band int) {
	kernels := opened(b, feeders)
	previous := runtime.GOMAXPROCS(feeders)
	defer runtime.GOMAXPROCS(previous)
	before := cpuTime(b)
	b.ResetTimer()
	var claimed atomic.Int64
	var group sync.WaitGroup
	for _, kernel := range kernels {
		group.Add(1)
		go func() {
			defer group.Done()
			drawWhileLeft(b, kernel, prepared, band, &claimed)
		}()
	}
	group.Wait()
	b.StopTimer()
	spent := cpuTime(b) - before
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "frames/s")
	b.ReportMetric(float64(spent.Milliseconds())/float64(b.N),
		"cpu-ms/frame")
}

// drawWhileLeft is one feeder: it claims frames from the shared count
// and draws each, until b.N have been claimed or a draw fails
func drawWhileLeft(b *testing.B, kernel *metal.Kernel, prepared feeding,
	band int, claimed *atomic.Int64) {
	for claimed.Add(1) <= int64(b.N) {
		_, err := kernel.Grid(prepared.width, prepared.height, band,
			per, prepared.inputs)
		if err != nil {
			b.Error(err)
			return
		}
	}
}

// opened is count kernels, each compiled on its own, closed when the
// case ends
func opened(b *testing.B, count int) []*metal.Kernel {
	kernels := make([]*metal.Kernel, 0, count)
	text := source()
	for range count {
		kernel, err := metal.Open(text, "trace")
		if err != nil {
			b.Skip(err)
		}
		b.Cleanup(kernel.Close)
		kernels = append(kernels, kernel)
	}
	return kernels
}

// cpuTime is the user and system time this process has spent so far
func cpuTime(b *testing.B) time.Duration {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		b.Fatal(err)
	}
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}
