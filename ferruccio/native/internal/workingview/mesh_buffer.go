package workingview

import (
	"fmt"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

// mappedMeshBuffer initializes an immutable mesh buffer in Metal shared memory.
// The caller must keep the bytes local to fill and never retain the mapping.
func (view *View) mappedMeshBuffer(label string, size uint64, usage gputypes.BufferUsage, fill func([]byte)) (_ *wgpu.Buffer, resultErr error) {
	if size == 0 || size%4 != 0 {
		return nil, fmt.Errorf("working view: invalid %s size %d", label, size)
	}
	buffer, err := view.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: label, Size: size, Usage: usage, MappedAtCreation: true,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			_ = buffer.Unmap()
			buffer.Release()
		}
	}()
	mapped, err := buffer.MappedRange(0, size)
	if err != nil {
		return nil, err
	}
	bytes := mapped.BytesMut()
	if uint64(len(bytes)) != size {
		mapped.Release()
		return nil, fmt.Errorf("working view: incomplete %s mapping", label)
	}
	fill(bytes)
	mapped.Release()
	if err := buffer.Unmap(); err != nil {
		return nil, err
	}
	return buffer, nil
}
