package telemetry

/*
#include <mach/mach.h>
#include <stdint.h>

static int azazel_cpu_ticks(uint32_t *ticks) {
    host_cpu_load_info_data_t info;
    mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
    mach_port_t host = mach_host_self();
    kern_return_t result = host_statistics(host, HOST_CPU_LOAD_INFO,
        (host_info_t)&info, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (result != KERN_SUCCESS || count < HOST_CPU_LOAD_INFO_COUNT) return -1;
    for (int i = 0; i < CPU_STATE_MAX; i++) ticks[i] = info.cpu_ticks[i];
    return 0;
}
*/
import "C"

import (
	"fmt"
	"sync"
	"time"

	"github.com/janearc/slayer/internal/observability"
)

// cpuMeter compares successive whole-host ticks. A single snapshot cannot
// establish utilization. Its mutex serializes concurrent Prometheus scrapes.
type cpuMeter struct {
	sync.Mutex
	previous [4]uint32
	when     time.Time
}

// sample turns the tick readings from read into a busy percent. It needs a
// previous reading at least a second old; until then busy is not known.
func (m *cpuMeter) sample(now time.Time, read func() ([4]uint32, error)) []observability.Sample {
	m.Lock()
	defer m.Unlock()
	ticks, err := read()
	available := err == nil
	known := available && !m.when.IsZero() && now.Sub(m.when) >= time.Second
	var total, busy uint64
	if known {
		for i := range ticks {
			delta := uint64(ticks[i] - m.previous[i])
			total += delta
			if i != 2 {
				busy += delta
			} // Darwin CPU_STATE_IDLE
		}
		known = total > 0
	}
	out := []observability.Sample{{Name: "azazel_host_cpu_ticks_available", Value: flag(available)}, {Name: "azazel_host_cpu_busy_known", Value: flag(known)}}
	if known {
		out = append(out, observability.Sample{Name: "azazel_host_cpu_busy_percent", Value: 100 * float64(busy) / float64(total)}, observability.Sample{Name: "azazel_host_cpu_sample_window_seconds", Value: now.Sub(m.when).Seconds()})
	}
	if available && (m.when.IsZero() || known) {
		m.previous, m.when = ticks, now
	}
	return out
}

// nativeCPUTicks reads the cumulative CPU tick counters from the kernel.
func nativeCPUTicks() ([4]uint32, error) {
	var raw [4]C.uint32_t
	if C.azazel_cpu_ticks(&raw[0]) != 0 {
		return [4]uint32{}, fmt.Errorf("CPU ticks unavailable")
	}
	return [4]uint32{uint32(raw[0]), uint32(raw[1]), uint32(raw[2]), uint32(raw[3])}, nil
}
