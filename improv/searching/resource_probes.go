package searching

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/janearc/slayer/render/workload"
)

// ProbeFrame is an actual requested frame, not a reconstructed FPS estimate.
type ProbeFrame struct {
	Index int           `json:"frame"`
	At    time.Duration `json:"at_ns"`
}

// SceneProbe selects cold, early adjacent and late frames from one musical cut.
// Advancing says whether the adjacent pair advances scene time; actual geometry
// motion must still be measured, not assumed from a timestamp or scene name.
type SceneProbe struct {
	Shot      int          `json:"shot"`
	Scene     string       `json:"scene"`
	Kind      string       `json:"kind,omitempty"`
	Caption   string       `json:"caption"`
	Entry     ProbeFrame   `json:"entry"`
	Adjacent  []ProbeFrame `json:"adjacent"`
	Late      ProbeFrame   `json:"late"`
	Advancing bool         `json:"advancing_clock"`
}

// ResourceProbePlan selects each represented shot from the job's actual Times.
// It opens no models or GPU resources. Calling code owns pressure-aware probe
// execution, cold/warm measurements, content-keyed persistence and admission.
// Samples can expose changing complexity; they cannot prove a peak bound for
// arbitrary procedural animation, so runtime measurements remain necessary.
func (edit *Edit) ResourceProbePlan(times []time.Duration) ([]SceneProbe, error) {
	if edit == nil || len(times) == 0 || len(edit.ShotSpans) != len(edit.Plan.Shots) {
		return nil, fmt.Errorf("heard resource probes need a checked cut and requested frames")
	}
	for i, span := range edit.ShotSpans {
		if math.IsNaN(span[0]) || math.IsNaN(span[1]) || math.IsInf(span[0], 0) || math.IsInf(span[1], 0) || span[1] <= span[0] || i > 0 && span[0] < edit.ShotSpans[i-1][1] {
			return nil, fmt.Errorf("heard resource probes have invalid shot %d", i)
		}
	}
	for i, at := range times {
		if i > 0 && at < times[i-1] || edit.shot(at.Seconds()) < 0 {
			return nil, fmt.Errorf("heard resource probes cannot place requested frame %d", i)
		}
	}
	frame := func(index int) ProbeFrame { return ProbeFrame{index, times[index]} }
	var probes []SceneProbe
	for index, span := range edit.ShotSpans {
		first := sort.Search(len(times), func(i int) bool { return times[i].Seconds() >= span[0] })
		end := sort.Search(len(times), func(i int) bool { return times[i].Seconds() >= span[1] })
		if first == end {
			continue
		}
		shot := edit.Plan.Shots[index]
		probe := SceneProbe{Shot: index, Scene: shot.Scene, Caption: shot.Caption, Entry: frame(first), Late: frame(end - 1)}
		if shot.Stage != nil {
			probe.Kind = shot.Stage.Kind
		}
		// Ground instruments settle at authored progress .62. An early window
		// avoids accidentally benchmarking only that hold, while remaining a
		// candidate rather than a claim that every scene animates at .25.
		target := span[0] + .25*(span[1]-span[0])
		middle := sort.Search(end, func(i int) bool { return times[i].Seconds() >= target })
		middle = max(first, min(middle, end-1))
		if end-first > 1 {
			middle = min(middle, end-2)
			probe.Adjacent = []ProbeFrame{frame(middle), frame(middle + 1)}
			probe.Advancing = times[middle+1] > times[middle]
		} else {
			probe.Adjacent = []ProbeFrame{frame(first)}
		}
		probes = append(probes, probe)
	}
	return probes, nil
}

// Frames is the unique requested frame sequence for this probe. Roles can
// coincide in a short shot; execution need not render that frame twice.
func (probe SceneProbe) Frames() []ProbeFrame {
	all := append([]ProbeFrame{probe.Entry, probe.Late}, probe.Adjacent...)
	sort.Slice(all, func(i, j int) bool { return all[i].Index < all[j].Index })
	unique := all[:0]
	for _, frame := range all {
		if len(unique) == 0 || frame.Index != unique[len(unique)-1].Index {
			unique = append(unique, frame)
		}
	}
	return unique
}

// WorkloadProbePlan adapts the checked shot clocks for persistent profiles,
// preserving every role when a short shot deduplicates to a single frame.
// It does not claim source dependency completeness or execute any probe.
func (edit *Edit) WorkloadProbePlan(times []time.Duration) ([]workload.Probe, error) {
	planned, err := edit.ResourceProbePlan(times)
	if err != nil {
		return nil, err
	}
	result := make([]workload.Probe, 0, len(planned))
	for _, scene := range planned {
		probe := workload.Probe{Shot: scene.Shot, Scene: scene.Scene, Kind: scene.Kind, Caption: scene.Caption, Advancing: scene.Advancing}
		roles := make(map[int][]string)
		roles[scene.Entry.Index] = append(roles[scene.Entry.Index], "entry")
		for i, frame := range scene.Adjacent {
			roles[frame.Index] = append(roles[frame.Index], fmt.Sprintf("adjacent-%d", i))
		}
		roles[scene.Late.Index] = append(roles[scene.Late.Index], "late")
		for _, frame := range scene.Frames() {
			probe.Frames = append(probe.Frames, workload.ProbeFrame{Index: frame.Index, At: frame.At, Roles: roles[frame.Index]})
		}
		if _, err := probe.Key(); err != nil {
			return nil, err
		}
		result = append(result, probe)
	}
	return result, nil
}
