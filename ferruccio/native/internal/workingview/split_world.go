package workingview

import (
	_ "embed"
	"encoding/binary"
	"math"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

//go:embed split_world.wgsl
var splitWorldShader string

const vertexBytes = 36
const splitVertexBytes = 48

type splitWorldPipelines struct {
	tried               bool
	shader              *wgpu.ShaderModule
	solid, lines, depth *wgpu.RenderPipeline
}

func splitWorldFits(limits gputypes.Limits, count int) bool {
	return limits.MaxVertexAttributes >= 4 && limits.MaxVertexBufferArrayStride >= splitVertexBytes &&
		uint64(count) <= limits.MaxBufferSize/splitVertexBytes
}

// prepareSplitWorld leaves the normal pipelines usable even when the device
// cannot build a separate vertex layout for a dispersed cached world.
func (view *View) prepareSplitWorld() bool {
	if view.split.tried {
		return view.split.solid != nil
	}
	pipelines := &view.split
	pipelines.tried = true
	shader, err := view.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "Ferruccio split world geometry", WGSL: shaderSource + splitWorldShader,
	})
	if err != nil {
		return false
	}
	pipelines.shader = shader
	vertex := []gputypes.VertexBufferLayout{{
		ArrayStride: splitVertexBytes, StepMode: gputypes.VertexStepModeVertex,
		Attributes: []gputypes.VertexAttribute{
			{Format: gputypes.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
			{Format: gputypes.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1},
			{Format: gputypes.VertexFormatFloat32x3, Offset: 24, ShaderLocation: 2},
			{Format: gputypes.VertexFormatFloat32x3, Offset: 36, ShaderLocation: 3},
		},
	}}
	for _, pipe := range []struct {
		mode string
		slot **wgpu.RenderPipeline
	}{{"solid", &pipelines.solid}, {"lines", &pipelines.lines}, {"depth", &pipelines.depth}} {
		made, err := view.meshPipeline(shader, vertex, pipe.mode, "split_")
		if err != nil {
			pipelines.release()
			return false
		}
		*pipe.slot = made
	}
	return true
}

func (pipelines *splitWorldPipelines) release() {
	for _, pipeline := range []*wgpu.RenderPipeline{pipelines.solid, pipelines.lines, pipelines.depth} {
		if pipeline != nil {
			pipeline.Release()
		}
	}
	if pipelines.shader != nil {
		pipelines.shader.Release()
	}
	*pipelines = splitWorldPipelines{tried: true}
}

// splitPosition preserves the residual from narrowing each native coordinate.
func splitPosition(position maths.Vec) (high, low [3]float32) {
	for axis, coordinate := range [3]float64{position.X, position.Y, position.Z} {
		high[axis] = float32(coordinate)
		low[axis] = float32(coordinate - float64(high[axis]))
	}
	return high, low
}

func vertexData(vertices []placement.DisplayVertex, split bool) []byte {
	stride := vertexBytes
	if split {
		stride = splitVertexBytes
	}
	data := make([]byte, len(vertices)*stride)
	packVertices(data, vertices, stride)
	return data
}

func packVertices(data []byte, vertices []placement.DisplayVertex, stride int) {
	for index, vertex := range vertices {
		position := vertex.Position
		values := [9]float32{
			float32(position.X), float32(position.Y), float32(position.Z),
			float32(vertex.Normal.X), float32(vertex.Normal.Y), float32(vertex.Normal.Z),
			float32(vertex.Colour.R), float32(vertex.Colour.G), float32(vertex.Colour.B),
		}
		for field, value := range values {
			binary.LittleEndian.PutUint32(data[index*stride+field*4:], math.Float32bits(value))
		}
		if stride == splitVertexBytes {
			_, low := splitPosition(position)
			for field, value := range low {
				binary.LittleEndian.PutUint32(data[index*stride+vertexBytes+field*4:], math.Float32bits(value))
			}
		}
	}
}
