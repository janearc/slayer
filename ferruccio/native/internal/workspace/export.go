package workspace

import (
	"fmt"
	"github.com/gogpu/gg"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/gogpu/wgpu"
	"image"
	"strings"
)

// Drawn says what drew an export. Backend is the GPU API, "metal" here, or
// "cpu" when gg's accelerator had no GPU and gg rasterized on the CPU.
// Shared is true when the export itself handed the window's device to gg,
// because the window had not drawn a frame that would have.
type Drawn struct {
	Backend, Adapter, Problem string
	Shared                    bool
}

// Export draws this window's actual widget tree to a standalone image on
// the window's own GPU device. It is an offscreen widget export, not a
// screenshot of the OS compositor. ExportDrawn says what drew it.
func (work *Workspace) Export() (image.Image, error) {
	return work.exportWith(work.host.GPUContextProvider())
}

// ExportDrawn reports what drew the most recent export.
func (work *Workspace) ExportDrawn() Drawn { return work.drawn }

// exportWith draws the widget tree with gg's accelerator on the provider's
// device. A nil provider leaves gg as it is, on the CPU unless the window
// has already given it a device.
func (work *Workspace) exportWith(provider gpucontext.DeviceProvider) (image.Image, error) {
	work.ui.Window().Frame()
	size := work.ui.Window().WindowSize()
	width, height := int(size.Width), int(size.Height)
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("window has no layout yet")
	}
	shared, problem := shareDevice(provider)
	dc := gg.NewContext(width, height)
	defer dc.Close()
	widget.MarkRedrawInTree(work.root)
	work.ui.Window().DrawTo(render.NewCanvas(dc, width, height))
	if err := dc.FlushGPU(); err != nil {
		return nil, err
	}
	work.drawn = drawnBy(provider)
	work.drawn.Shared, work.drawn.Problem = shared, problem
	return dc.Image(), nil
}

// shareDevice gives gg's accelerator the provider's device when it has no
// GPU of its own yet. gg's own start-up asks only for Vulkan, which macOS
// lacks, so an export made before the window's first frame would otherwise
// draw on the CPU. A device already in use is left alone: replacing it
// would rebuild the pipelines the window's canvas holds.
func shareDevice(provider gpucontext.DeviceProvider) (bool, string) {
	if provider == nil || provider.Device().IsNil() || gg.AcceleratorCanRenderDirect() {
		return false, ""
	}
	if err := gg.SetAcceleratorDeviceProvider(provider); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// drawnBy names the device gg's accelerator drew on. On macOS gg reaches a
// GPU only through a provider, since its own start-up asks for Vulkan, so a
// ready accelerator is drawing on the provider's adapter. With no provider
// the GPU was given by someone else and stays unnamed; a provider that
// hides its adapter is named by the summary it gives instead.
func drawnBy(provider gpucontext.DeviceProvider) Drawn {
	if !gg.AcceleratorCanRenderDirect() {
		return Drawn{Backend: "cpu"}
	}
	if provider == nil {
		return Drawn{Backend: "gpu"}
	}
	adapter := wgpu.AdapterFromHandle(provider.Adapter())
	if adapter == nil {
		return Drawn{Backend: "gpu", Adapter: provider.AdapterInfo().Name}
	}
	info := adapter.Info()
	return Drawn{Backend: strings.ToLower(info.Backend.String()), Adapter: info.Name}
}
