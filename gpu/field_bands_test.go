//go:build draw && darwin

package gpu

import (
	"fmt"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

func gravityBandWorld(t testing.TB) (*thing.World, maths.Subspace, film.Sensor) {
	t.Helper()
	world := thing.NewWorld()
	body, err := template.Quasar()
	if err != nil {
		t.Fatal(err)
	}
	body = body.WithoutJets()
	if err := body.Place(world, maths.Vec{}, 1/body.Radius(), 0); err != nil {
		t.Fatal(err)
	}
	from := maths.V(0, 8, 45)
	return world, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 2.6, 0))}, film.Pinhole{Wide: maths.Degree}
}

func TestGravityDispatchBandsPreserveEverySample(t *testing.T) {
	world, eye, lens := gravityBandWorld(t)
	var baseline *frame.Frame
	for _, rows := range []int{2, 8, 16, 32} {
		ops := fieldDefaults()
		ops.rows = rows
		picture, _, err := fieldSampledUsing(world, eye, lens, 0, 32, 17, 1, ops, 32)
		if err != nil {
			t.Fatal(err)
		}
		if baseline != nil {
			for y := range 17 {
				for x := range 32 {
					if picture.At(x, y) != baseline.At(x, y) {
						t.Fatal("dispatch band changed a sample", rows, x, y)
					}
				}
			}
		}
		baseline = picture
	}
}

func BenchmarkGravityDispatchBands(b *testing.B) {
	world, eye, lens := gravityBandWorld(b)
	for _, rows := range []int{2, 8, 16, 32} {
		b.Run(fmt.Sprint(rows), func(b *testing.B) {
			ops := fieldDefaults()
			ops.rows = rows
			for b.Loop() {
				_, spent, err := fieldSampledUsing(world, eye, lens, 0, 480, 319, 1, ops, 32)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(spent.trace.Seconds()*1000, "gpu-ms")
			}
		})
	}
}
