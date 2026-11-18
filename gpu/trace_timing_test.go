package gpu

import (
	"testing"
	"time"
)

func TestTraceDeviceTimeExcludesEnvironmentAndLighting(t *testing.T) {
	second := time.Second
	for _, check := range []struct {
		name                                 string
		after, before, lighting, environment time.Duration
		want                                 time.Duration
	}{
		{"no bakes", 18 * second, 10 * second, 0, 0, 8 * second},
		{"both bakes", 18 * second, 10 * second, 3 * second, 2 * second, 3 * second},
		{"new kernel with acceleration", 18 * second, 4 * second, 3 * second, 2 * second, 9 * second},
	} {
		t.Run(check.name, func(t *testing.T) {
			got := traceDeviceTime(check.after, check.before, check.lighting, check.environment)
			if got != check.want {
				t.Fatalf("trace device time = %v, want %v", got, check.want)
			}
		})
	}
}
