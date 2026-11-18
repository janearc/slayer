// Package headless opens a Metal device with no window, for tests that must
// draw on the hardware Ferruccio itself draws on. It never falls back to
// another backend: a test that opens one draws on Metal or not at all.
package headless

import (
	"fmt"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	// The Metal backend registers itself with wgpu when imported.
	_ "github.com/gogpu/wgpu/hal/metal"
)

// Device is a windowless Metal device. It is a gpucontext.DeviceProvider,
// so gg's accelerator and the working view both accept it.
type Device struct {
	instance *wgpu.Instance
	adapter  *wgpu.Adapter
	device   *wgpu.Device
}

// Open asks wgpu for a Metal adapter and a device on it.
func Open() (*Device, error) {
	instance, err := wgpu.CreateInstance(&wgpu.InstanceDescriptor{
		Backends: wgpu.BackendsMetal,
	})
	if err != nil {
		return nil, fmt.Errorf("headless metal instance: %w", err)
	}
	opened := &Device{instance: instance}
	opened.adapter, err = instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		PowerPreference: gputypes.PowerPreferenceHighPerformance,
	})
	if err != nil {
		opened.Close()
		return nil, fmt.Errorf("headless metal adapter: %w", err)
	}
	if backend := opened.adapter.Info().Backend; backend != gputypes.BackendMetal {
		opened.Close()
		return nil, fmt.Errorf("headless adapter is %s, not Metal", backend)
	}
	opened.device, err = opened.adapter.RequestDevice(&wgpu.DeviceDescriptor{
		Label: "Ferruccio headless Metal",
	})
	if err != nil {
		opened.Close()
		return nil, fmt.Errorf("headless metal device: %w", err)
	}
	return opened, nil
}

// Device returns the wgpu device as gpucontext's opaque handle.
func (opened *Device) Device() gpucontext.Device {
	return wgpu.DeviceToHandle(opened.device)
}

// Queue returns the device's queue as gpucontext's opaque handle.
func (opened *Device) Queue() gpucontext.Queue {
	return wgpu.QueueToHandle(opened.device.Queue())
}

// SurfaceFormat is undefined: there is no window and so no surface.
func (opened *Device) SurfaceFormat() gputypes.TextureFormat {
	return gputypes.TextureFormatUndefined
}

// Adapter returns the Metal adapter as gpucontext's opaque handle.
func (opened *Device) Adapter() gpucontext.Adapter {
	return wgpu.AdapterToHandle(opened.adapter)
}

// AdapterInfo names the adapter and says whether it is a real GPU.
func (opened *Device) AdapterInfo() gpucontext.AdapterInfo {
	info := opened.adapter.Info()
	kind := gpucontext.AdapterTypeUnknown
	switch info.DeviceType {
	case gputypes.DeviceTypeDiscreteGPU:
		kind = gpucontext.AdapterTypeDiscrete
	case gputypes.DeviceTypeIntegratedGPU:
		kind = gpucontext.AdapterTypeIntegrated
	case gputypes.DeviceTypeCPU:
		kind = gpucontext.AdapterTypeSoftware
	}
	return gpucontext.AdapterInfo{Name: info.Name, Type: kind}
}

// Features reports the optional features the device was given.
func (opened *Device) Features() gputypes.Features {
	return opened.device.Features()
}

// DownlevelCapabilities reports what the adapter can do, compute included.
func (opened *Device) DownlevelCapabilities() gputypes.DownlevelCapabilities {
	return opened.adapter.DownlevelCapabilities()
}

// Close releases the device, the adapter and the instance, in that order,
// skipping any that were never opened.
func (opened *Device) Close() {
	if opened.device != nil {
		opened.device.Release()
		opened.device = nil
	}
	if opened.adapter != nil {
		opened.adapter.Release()
		opened.adapter = nil
	}
	if opened.instance != nil {
		opened.instance.Release()
		opened.instance = nil
	}
}
