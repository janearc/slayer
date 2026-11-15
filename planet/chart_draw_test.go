//go:build draw

package planet

import (
	"math"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
)

// apartOf is how far a chart is from its face over points the eye sees:
// the mean and the largest difference of any of the light's three,
// against the mean light.
func apartOf(t *testing.T, face *Face, chart *Chart, radius float64) (mean,
	largest float64) {
	t.Helper()
	var sum, light float64
	var asked int
	for k := range pickedCount(t, "chart-points") {
		point, seen := seenPoint(t, face, radius, k)
		if !seen {
			continue
		}
		want, got := face.Within(point, nothing), chart.Within(point, nothing)
		apart := lightApart(got, want)
		sum, light = sum+apart, light+(want.R+want.G+want.B)/chartLight
		asked++
		largest = math.Max(largest, apart)
	}
	return sum / light, largest / (light / float64(asked))
}

// a chart is its face at the chart's own points, to the float32 it keeps,
// and nearer the face between them the finer it is: halving the cell
// brings the mean difference down by more than a third.
func TestAChartIsItsFace(t *testing.T) {
	face, radius := chartFace(t)
	finer := picked(t, "chart-fine-cell")
	fine, _ := NewChart(face, radius, finer*maths.Degree)
	down, along := picked(t, "chart-grid-down"), picked(t, "chart-grid-along")
	point := face.ways.toward(down*finer, along*finer).Scale(radius)
	want, got := face.Within(point, nothing), fine.Within(point, nothing)
	if !(math.Abs(got.G-want.G) <= picked(t, "chart-grid-close")*want.G) ||
		!positive(want.G) {
		t.Fatalf("at a point of the chart it reads %v, the face %v", got, want)
	}
	coarse, _ := NewChart(face, radius, picked(t, "chart-coarse-cell")*maths.Degree)
	rough, _ := apartOf(t, face, coarse, radius)
	near, _ := apartOf(t, face, fine, radius)
	t.Logf("mean difference over mean light: %.4g at 1 degree, %.4g at half",
		rough, near)
	ceiling := rough * picked(t, "chart-finer-top") / picked(t, "chart-finer-bottom")
	if !(near < ceiling) || !positive(near) {
		t.Fatalf("mean difference %g at half a degree, %g at one", near, rough)
	}
}

// rayReader is a ray that reads the chart over many points, then the point
// every ray reads, for the first's light.
type rayReader struct {
	t      *testing.T
	face   *Face
	chart  *Chart
	radius float64
	point  maths.Vec
	want   maths.Light
	draws  int64
	group  *sync.WaitGroup
}

// many rays at once paint each tile once and read the same light.
func TestAChartIsSafeForManyRays(t *testing.T) {
	face, radius := chartFace(t)
	chart, _ := NewChart(face, radius, picked(t, "chart-rays-cell")*maths.Degree)
	point := face.ways.toward(picked(t, "chart-rays-down"),
		picked(t, "chart-rays-along")).Scale(radius)
	want := chart.Within(point, nothing)
	var group sync.WaitGroup
	for range pickedInt(t, "chart-rays") {
		group.Add(single)
		reader := &rayReader{t: t, face: face, chart: chart, radius: radius,
			point: point, want: want, group: &group,
			draws: pickedCount(t, "chart-ray-draws")}
		go reader.run()
	}
	group.Wait()
}

// run reads the chart at the reader's draws, then at its point, and
// reports a light that is not the first's.
func (reader *rayReader) run() {
	defer reader.group.Done()
	for k := range reader.draws {
		other, _ := seenPoint(reader.t, reader.face, reader.radius, k)
		reader.chart.Within(other, nothing)
	}
	if got := reader.chart.Within(reader.point, nothing); got != reader.want {
		reader.t.Errorf("a ray read %v, the first %v", got, reader.want)
	}
}
