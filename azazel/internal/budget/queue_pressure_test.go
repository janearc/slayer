package budget

import (
	"encoding/binary"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

// TestQueueUsesPressureInsteadOfAccumulatedSwap checks the approved policy.
func TestQueueUsesPressureInsteadOfAccumulatedSwap(t *testing.T) {
	for _, pressure := range []uint32{1, 2, 4} {
		host := NewMockadmissionHostAdapter(gomock.NewController(t))
		disk, _ := queueDisk()
		host.EXPECT().Statfs(gomock.Any()).Return(disk, nil).MinTimes(1)
		host.EXPECT().Now().Return(time.Now()).MinTimes(1)
		host.EXPECT().SysctlRaw("kern.bootsessionuuid").Return(sampleRaw("kern.bootsessionuuid"), nil).MinTimes(2)
		load := sampleRaw("vm.loadavg")
		binary.NativeEndian.PutUint32(load, 2400)
		host.EXPECT().SysctlRaw("vm.loadavg").Return(load, nil)
		swap := sampleRaw("vm.swapusage")
		binary.NativeEndian.PutUint64(swap, 32<<30)
		binary.NativeEndian.PutUint64(swap[8:], 16<<30)
		binary.NativeEndian.PutUint64(swap[16:], 16<<30)
		host.EXPECT().SysctlRaw("vm.swapusage").Return(swap, nil)
		level := sampleRaw("kern.memorystatus_vm_pressure_level")
		binary.NativeEndian.PutUint32(level, pressure)
		host.EXPECT().SysctlRaw("kern.memorystatus_vm_pressure_level").Return(level, nil)
		observation, err := observeQueueHost(t.Context(), t.TempDir(), host)
		if err != nil || observation.Ready != (pressure == 1) || observation.Stop != (pressure != 1) {
			t.Fatal(pressure, observation, err)
		}
	}
}
