// Package workingview draws native scene meshes on the app's resident GPU device.
package workingview

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
	"image"
	"math"
	"time"
)

//go:embed shader.wgsl
var shaderSource string

const uniformBytes = 144

// largestSide is the longest side, in pixels, the view draws. A larger
// window is drawn at this size with its shape kept, and the preview pane
// scales the picture up to fill it.
const largestSide = 4096

// readbackDeadline is how long Draw waits for the GPU to hand a frame back.
const readbackDeadline = 2 * time.Second

// ErrReadbackLate reports a frame the GPU did not hand back in time. The
// view abandoned that readback and stays usable: draw again next tick.
var ErrReadbackLate = errors.New("working GPU readback was late")

// Statistics reports actual resident resources and completed GPU draws.
// Misses counts readbacks abandoned because the GPU handed them back late.
type Statistics struct {
	Meshes, Triangles, Uploads, Draws, Misses, Width, Height int
	Elapsed                                                  time.Duration
}
type mesh struct {
	vertices, indices, edges *wgpu.Buffer
	instances                *wgpu.Buffer
	instanceBytes            int
	triangles, lines         uint32
}
type object struct {
	uniform *wgpu.Buffer
	group   *wgpu.BindGroup
}

// drawObject pairs one resident mesh with the uniform that places it.
type drawObject struct {
	mesh   *mesh
	object *object
	count  uint32
}

type pendingFrame struct {
	mapRequest       *wgpu.MapPending
	scene            *placement.DisplayScene
	camera           matrix
	width, height    int
	mode             string
	background, edge [4]float32
	started          time.Time
}

// View borrows the app device and retains its own GPU resources between draws.
// All methods run on the owning application's UI/render thread.
type View struct {
	device                  *wgpu.Device
	queue                   *wgpu.Queue
	layout                  *wgpu.BindGroupLayout
	pipelineLayout          *wgpu.PipelineLayout
	shader                  *wgpu.ShaderModule
	solid, lines, depthOnly *wgpu.RenderPipeline
	meshes                  map[*placement.DisplayMesh]*mesh
	objects                 map[string]*object
	colour, depth           *wgpu.Texture
	colourView, depthView   *wgpu.TextureView
	readback                *wgpu.Buffer
	width, height, rowBytes int
	background, edge        [4]float32
	stats                   Statistics
	mode                    string
	// picture is the frame Draw returns, reused while the size holds.
	picture *image.NRGBA
	// late is how many coming readbacks are given up on purpose.
	late int
	// pending retains a batch frame while its GPU readback is in flight.
	pending *pendingFrame
}

// New attaches after the app initializes its native GPU context.
func New(provider gpucontext.DeviceProvider) (*View, error) {
	if provider == nil || provider.Device().IsNil() {
		return nil, fmt.Errorf("working view: native GPU device is not ready")
	}
	device := wgpu.DeviceFromHandle(provider.Device())
	if device == nil {
		return nil, fmt.Errorf("working view: no wgpu device")
	}
	view := &View{device: device, queue: device.Queue(), meshes: map[*placement.DisplayMesh]*mesh{}, objects: map[string]*object{}, mode: "edges", background: [4]float32{.03, .04, .055, 1}, edge: [4]float32{.08, .11, .15, 1}}
	if err := view.createPipelines(); err != nil {
		view.Close()
		return nil, err
	}
	return view, nil
}

// SetPalette accepts the host's checked Chroma background and edge colours.
func (view *View) SetPalette(background, edge [4]float32) {
	view.background = background
	view.edge = edge
}

// SetMode chooses opaque solids, solids with visible edges, or occlusion-correct wireframe.
func (view *View) SetMode(mode string) error {
	if mode != "solid" && mode != "edges" && mode != "wire" {
		return fmt.Errorf("unknown working view mode %q", mode)
	}
	view.mode = mode
	return nil
}

// ForceLateReadbacks makes the next count readbacks late on purpose: each
// asks for its map and gives up at once, as a slow GPU would make it. It
// exists so a proof can show the view recovering from a late frame.
func (view *View) ForceLateReadbacks(count int) {
	view.late = max(0, count)
}

// Stats reports counters rather than inferred progress or frame-rate claims.
func (view *View) Stats() Statistics {
	result := view.stats
	result.Meshes = len(view.meshes)
	result.Width = view.width
	result.Height = view.height
	return result
}

// createPipelines builds the shader module, the bind group layout and the
// three render pipelines: solid, edges and depth.
func (view *View) createPipelines() error {
	var err error
	view.shader, err = view.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "Ferruccio working mesh", WGSL: shaderSource})
	if err != nil {
		return fmt.Errorf("working view shader: %w", err)
	}
	view.layout, err = view.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{Label: "working camera and placement", Entries: []gputypes.BindGroupLayoutEntry{{Binding: 0, Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, MinBindingSize: uniformBytes}}}})
	if err != nil {
		return err
	}
	view.pipelineLayout, err = view.device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{BindGroupLayouts: []*wgpu.BindGroupLayout{view.layout}})
	if err != nil {
		return err
	}
	vertex := []gputypes.VertexBufferLayout{
		{ArrayStride: 36, StepMode: gputypes.VertexStepModeVertex, Attributes: []gputypes.VertexAttribute{{Format: gputypes.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0}, {Format: gputypes.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1}, {Format: gputypes.VertexFormatFloat32x3, Offset: 24, ShaderLocation: 2}}},
		{ArrayStride: 64, StepMode: gputypes.VertexStepModeInstance, Attributes: []gputypes.VertexAttribute{{Format: gputypes.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 3}, {Format: gputypes.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 4}, {Format: gputypes.VertexFormatFloat32x4, Offset: 32, ShaderLocation: 5}, {Format: gputypes.VertexFormatFloat32x4, Offset: 48, ShaderLocation: 6}}},
	}
	makePipeline := func(mode string) (*wgpu.RenderPipeline, error) {
		topology := gputypes.PrimitiveTopologyTriangleList
		entry, fragment := "solid_vertex", "solid_fragment"
		if mode == "lines" {
			topology = gputypes.PrimitiveTopologyLineList
			entry, fragment = "edge_vertex", "edge_fragment"
		}
		mask := gputypes.ColorWriteMaskAll
		if mode == "depth" {
			mask = gputypes.ColorWriteMaskNone
		}
		return view.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{Label: "Ferruccio opaque working view", Layout: view.pipelineLayout, Vertex: wgpu.VertexState{Module: view.shader, EntryPoint: entry, Buffers: vertex}, Primitive: gputypes.PrimitiveState{Topology: topology, FrontFace: gputypes.FrontFaceCCW, CullMode: gputypes.CullModeNone}, DepthStencil: &wgpu.DepthStencilState{Format: wgpu.TextureFormatDepth32Float, DepthWriteEnabled: mode != "lines", DepthCompare: gputypes.CompareFunctionLessEqual}, Multisample: gputypes.MultisampleState{Count: 1, Mask: 0xffffffff}, Fragment: &wgpu.FragmentState{Module: view.shader, EntryPoint: fragment, Targets: []gputypes.ColorTargetState{{Format: wgpu.TextureFormatRGBA8Unorm, WriteMask: mask}}}})
	}
	view.solid, err = makePipeline("solid")
	if err != nil {
		return fmt.Errorf("working triangles: %w", err)
	}
	view.lines, err = makePipeline("lines")
	if err != nil {
		return fmt.Errorf("working edges: %w", err)
	}
	view.depthOnly, err = makePipeline("depth")
	if err != nil {
		return fmt.Errorf("working depth: %w", err)
	}
	return nil
}

// buffer creates a device buffer holding data. It fails when data is empty.
func (view *View) buffer(label string, data []byte, usage gputypes.BufferUsage) (*wgpu.Buffer, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("working view: empty %s", label)
	}
	buffer, err := view.device.CreateBuffer(&wgpu.BufferDescriptor{Label: label, Size: uint64(len(data)), Usage: usage | wgpu.BufferUsageCopyDst})
	if err != nil {
		return nil, err
	}
	if err = view.queue.WriteBuffer(buffer, 0, data); err != nil {
		buffer.Release()
		return nil, err
	}
	return buffer, nil
}

// upload returns the device copy of source, uploading it on first use.
func (view *View) upload(source *placement.DisplayMesh) (*mesh, error) {
	if ready := view.meshes[source]; ready != nil {
		return ready, nil
	}
	if source == nil || len(source.Vertices) == 0 || len(source.Indices)%3 != 0 {
		return nil, fmt.Errorf("working view: invalid native mesh")
	}
	vertices := make([]byte, len(source.Vertices)*36)
	for index, vertex := range source.Vertices {
		values := []float32{float32(vertex.Position.X), float32(vertex.Position.Y), float32(vertex.Position.Z), float32(vertex.Normal.X), float32(vertex.Normal.Y), float32(vertex.Normal.Z), float32(vertex.Colour.R), float32(vertex.Colour.G), float32(vertex.Colour.B)}
		for field, value := range values {
			binary.LittleEndian.PutUint32(vertices[index*36+field*4:], math.Float32bits(value))
		}
	}
	indices := func(values []uint32) ([]byte, error) {
		result := make([]byte, len(values)*4)
		for index, value := range values {
			if int(value) >= len(source.Vertices) {
				return nil, fmt.Errorf("working mesh index out of bounds")
			}
			binary.LittleEndian.PutUint32(result[index*4:], value)
		}
		return result, nil
	}
	triangles, err := indices(source.Indices)
	if err != nil {
		return nil, err
	}
	edges := source.Edges
	if len(edges) == 0 {
		for index := 0; index < len(source.Indices); index += 3 {
			a, b, c := source.Indices[index], source.Indices[index+1], source.Indices[index+2]
			edges = append(edges, a, b, b, c, c, a)
		}
	}
	lineData, err := indices(edges)
	if err != nil {
		return nil, err
	}
	resident := &mesh{triangles: uint32(len(source.Indices)), lines: uint32(len(edges))}
	resident.vertices, err = view.buffer("native local vertices", vertices, wgpu.BufferUsageVertex)
	if err != nil {
		return nil, err
	}
	resident.indices, err = view.buffer("native triangles", triangles, wgpu.BufferUsageIndex)
	if err != nil {
		resident.vertices.Release()
		return nil, err
	}
	resident.edges, err = view.buffer("native edges", lineData, wgpu.BufferUsageIndex)
	if err != nil {
		resident.vertices.Release()
		resident.indices.Release()
		return nil, err
	}
	view.meshes[source] = resident
	view.stats.Uploads++
	view.stats.Triangles += len(source.Indices) / 3
	return resident, nil
}

// object returns the device object for id, building it on first use.
func (view *View) object(id string) (*object, error) {
	if ready := view.objects[id]; ready != nil {
		return ready, nil
	}
	uniform, err := view.device.CreateBuffer(&wgpu.BufferDescriptor{Label: "working object transform", Size: uniformBytes, Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst})
	if err != nil {
		return nil, err
	}
	group, err := view.device.CreateBindGroup(&wgpu.BindGroupDescriptor{Layout: view.layout, Entries: []wgpu.BindGroupEntry{{Binding: 0, Buffer: uniform, Size: uniformBytes}}})
	if err != nil {
		uniform.Release()
		return nil, err
	}
	ready := &object{uniform: uniform, group: group}
	view.objects[id] = ready
	return ready, nil
}

// Draw changes camera/placement uniforms and renders the resident native meshes.
// Its image is a GPU readback for UI integration, never a CPU scene raster.
// A window past largestSide is drawn smaller with its shape kept. The image
// belongs to the view and is reused by the next Draw of the same size. A
// late readback returns ErrReadbackLate and leaves the view ready to retry.
func (view *View) Draw(scene *placement.DisplayScene, width, height int) (*image.NRGBA, error) {
	if view == nil || view.device == nil || scene == nil {
		return nil, fmt.Errorf("working view is not initialized")
	}
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("working view dimensions must be positive")
	}
	width, height = fit(width, height)
	if view.pending != nil {
		pending := view.pending
		if pending.scene == scene && pending.width == width && pending.height == height && pending.camera == cameraMatrix(scene, width, height) && pending.mode == view.mode && pending.background == view.background && pending.edge == view.edge {
			return view.finishPending()
		}
		if err := view.cancelPending(); err != nil {
			return nil, err
		}
	}
	if err := view.resize(width, height); err != nil {
		return nil, err
	}
	started := time.Now()
	draws, err := view.place(scene, width, height)
	if err != nil {
		return nil, err
	}
	commands, err := view.encode(draws, width, height)
	if err != nil {
		return nil, err
	}
	if _, err = view.queue.Submit(commands); err != nil {
		commands.Release()
		return nil, err
	}
	if len(scene.Batches) > 0 {
		pending, err := view.readback.MapAsync(wgpu.MapModeRead, 0, uint64(view.rowBytes*height))
		if err != nil {
			return nil, fmt.Errorf("working GPU readback: %w", err)
		}
		view.pending = &pendingFrame{mapRequest: pending, scene: scene, camera: cameraMatrix(scene, width, height), width: width, height: height, mode: view.mode, background: view.background, edge: view.edge, started: started}
		return view.finishPending()
	}
	picture, err := view.read(width, height)
	if err != nil {
		return nil, err
	}
	view.stats.Draws++
	view.stats.Elapsed = time.Since(started)
	return picture, nil
}

// finishPending polls without submitting the same frame a second time. Batch
// scenes are immutable while pending; a changed scene or camera cancels it.
func (view *View) finishPending() (*image.NRGBA, error) {
	pending := view.pending
	if view.late > 0 {
		view.late--
		return nil, ErrReadbackLate
	}
	view.device.Poll(wgpu.PollPoll)
	ready, err := pending.mapRequest.Status()
	if !ready {
		return nil, ErrReadbackLate
	}
	pending.mapRequest.Release()
	view.pending = nil
	if err != nil {
		return nil, fmt.Errorf("working GPU readback: %w", err)
	}
	picture, err := view.copyReadback(pending.width, pending.height)
	if err != nil {
		return nil, err
	}
	view.stats.Draws++
	view.stats.Elapsed = time.Since(pending.started)
	return picture, nil
}

// cancelPending discards an obsolete readback and replaces its buffer.
func (view *View) cancelPending() error {
	if view.pending == nil {
		return nil
	}
	pending := view.pending
	view.pending = nil
	err := view.readback.Unmap()
	pending.mapRequest.Release()
	if err != nil && !errors.Is(err, wgpu.ErrMapNotMapped) {
		return fmt.Errorf("working GPU readback: cancel: %w", err)
	}
	view.readback.Release()
	view.readback, err = view.newReadback(view.width, view.height)
	return err
}

// fit keeps a target within largestSide on its longer side and keeps its
// shape, never letting a side round down to nothing.
func fit(width, height int) (int, int) {
	longest := max(width, height)
	if longest <= largestSide {
		return width, height
	}
	scale := float64(largestSide) / float64(longest)
	scaled := func(side int) int {
		return max(1, int(math.Round(float64(side)*scale)))
	}
	return scaled(width), scaled(height)
}

// place uploads each mesh once and batches its placements into one draw.
func (view *View) place(scene *placement.DisplayScene, width, height int) ([]drawObject, error) {
	camera := cameraMatrix(scene, width, height)
	uniform, err := view.object("staged-camera")
	if err != nil {
		return nil, err
	}
	if err := view.queue.WriteBuffer(uniform.uniform, 0, uniformData(camera, matrix{}, view.edge)); err != nil {
		return nil, err
	}
	type batch struct {
		mesh  *mesh
		data  []byte
		count uint32
	}
	live := make(map[*placement.DisplayMesh]bool)
	for _, source := range scene.Objects {
		if source.Mesh != nil && len(source.Mesh.Indices) != 0 {
			live[source.Mesh] = true
		}
	}
	for _, source := range scene.Batches {
		if source.Mesh != nil && len(source.Mesh.Indices) != 0 && len(source.Places) != 0 {
			live[source.Mesh] = true
		}
	}
	for source, resident := range view.meshes {
		if !live[source] {
			resident.release()
			delete(view.meshes, source)
			view.stats.Triangles -= len(source.Indices) / 3
		}
	}
	var batches []batch
	known := make(map[*placement.DisplayMesh]int)
	frustum := viewFrustum(scene, width, height)
	bounds := make(map[*placement.DisplayMesh]sphere)
	count := func(source *placement.DisplayMesh, places []maths.Subspace, staged *placement.StagedBatch) ([]uint64, error) {
		if source == nil || len(source.Indices) == 0 || len(places) == 0 {
			return nil, nil
		}
		sphere, found := bounds[source]
		if !found {
			var ok bool
			sphere, ok = meshSphere(source)
			if !ok {
				return nil, fmt.Errorf("working view: mesh has no bounds")
			}
			bounds[source] = sphere
		}
		visible := 0
		mask := make([]uint64, (len(places)+63)/64)
		for i := 0; i < len(places); {
			if staged != nil && i%placement.PlacementBlockSize == 0 {
				if box, ok := staged.BlockBounds(i / placement.PlacementBlockSize); ok && !frustum.boxVisible(box) {
					i += placement.PlacementBlockSize
					continue
				}
			}
			place := places[i]
			if frustum.visible(sphere, place) {
				visible++
				mask[i/64] |= uint64(1) << uint(i%64)
			}
			i++
		}
		if visible == 0 {
			return nil, nil
		}
		index, found := known[source]
		if !found {
			geometry, err := view.upload(source)
			if err != nil {
				return nil, err
			}
			index = len(batches)
			known[source] = index
			batches = append(batches, batch{mesh: geometry})
		}
		current := &batches[index]
		if uint64(current.count)+uint64(visible) > math.MaxUint32 {
			return nil, fmt.Errorf("working view has too many instances of one mesh")
		}
		current.count += uint32(visible)
		return mask, nil
	}
	objectMasks := make([][]uint64, len(scene.Objects))
	for i, source := range scene.Objects {
		var err error
		objectMasks[i], err = count(source.Mesh, []maths.Subspace{source.At}, nil)
		if err != nil {
			return nil, err
		}
	}
	batchMasks := make([][]uint64, len(scene.Batches))
	for i, source := range scene.Batches {
		if !frustum.batchVisible(source) {
			continue
		}
		var err error
		batchMasks[i], err = count(source.Mesh, source.Places, &scene.Batches[i])
		if err != nil {
			return nil, err
		}
	}
	for i := range batches {
		batches[i].data = make([]byte, 0, int(batches[i].count)*64)
	}
	add := func(source *placement.DisplayMesh, places []maths.Subspace, mask []uint64) {
		if mask == nil {
			return
		}
		index, found := known[source]
		if !found {
			return
		}
		current := &batches[index]
		for i, place := range places {
			if mask[i/64]&(uint64(1)<<uint(i%64)) == 0 {
				continue
			}
			for _, value := range frameMatrix(place, false) {
				current.data = binary.LittleEndian.AppendUint32(current.data, math.Float32bits(value))
			}
		}
	}
	for i, source := range scene.Objects {
		add(source.Mesh, []maths.Subspace{source.At}, objectMasks[i])
	}
	for i, source := range scene.Batches {
		add(source.Mesh, source.Places, batchMasks[i])
	}
	draws := make([]drawObject, 0, len(batches))
	for _, current := range batches {
		resident := current.mesh
		if resident.instances == nil || resident.instanceBytes < len(current.data) {
			buffer, err := view.buffer("working instance transforms", current.data, wgpu.BufferUsageVertex)
			if err != nil {
				return nil, err
			}
			if resident.instances != nil {
				resident.instances.Release()
			}
			resident.instances, resident.instanceBytes = buffer, len(current.data)
		} else if err := view.queue.WriteBuffer(resident.instances, 0, current.data); err != nil {
			return nil, err
		}
		draws = append(draws, drawObject{mesh: resident, object: uniform, count: current.count})
	}
	return draws, nil
}

// release frees a resident mesh and its instance buffer.
func (resident *mesh) release() {
	resident.vertices.Release()
	resident.indices.Release()
	resident.edges.Release()
	if resident.instances != nil {
		resident.instances.Release()
	}
}

// uniformData packs the camera, the model frame and the edge colour in the
// layout shader.wgsl reads.
func uniformData(camera, model matrix, edge [4]float32) []byte {
	data := make([]byte, uniformBytes)
	for index, value := range camera {
		binary.LittleEndian.PutUint32(data[index*4:], math.Float32bits(value))
	}
	for index, value := range model {
		binary.LittleEndian.PutUint32(data[64+index*4:], math.Float32bits(value))
	}
	for index, value := range edge {
		binary.LittleEndian.PutUint32(data[128+index*4:], math.Float32bits(value))
	}
	return data
}

// encode records the opaque pass and the copy into the readback buffer. An
// encoder that fails is discarded, so none is left neither submitted nor
// released, as wgpu requires.
func (view *View) encode(draws []drawObject, width, height int) (*wgpu.CommandBuffer, error) {
	encoder, err := view.device.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "working viewport"})
	if err != nil {
		return nil, err
	}
	if err = view.pass(encoder, draws); err != nil {
		encoder.DiscardEncoding()
		return nil, err
	}
	encoder.CopyTextureToBuffer(view.colour, view.readback, []wgpu.BufferTextureCopy{{TextureBase: wgpu.ImageCopyTexture{Texture: view.colour}, BufferLayout: wgpu.ImageDataLayout{BytesPerRow: uint32(view.rowBytes), RowsPerImage: uint32(height)}, Size: wgpu.Extent3D{Width: uint32(width), Height: uint32(height), DepthOrArrayLayers: 1}}})
	commands, err := encoder.Finish()
	if err != nil {
		// Finish discards a failed encoder itself; this makes it certain.
		encoder.DiscardEncoding()
		return nil, err
	}
	return commands, nil
}

// pass draws the resident meshes as opaque solids, edges or wire.
func (view *View) pass(encoder *wgpu.CommandEncoder, draws []drawObject) error {
	pass, err := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{Label: "opaque native solids and edges", ColorAttachments: []wgpu.RenderPassColorAttachment{{View: view.colourView, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore, ClearValue: gputypes.Color{R: float64(view.background[0]), G: float64(view.background[1]), B: float64(view.background[2]), A: 1}}}, DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{View: view.depthView, DepthLoadOp: gputypes.LoadOpClear, DepthStoreOp: gputypes.StoreOpStore, DepthClearValue: 1}})
	if err != nil {
		return err
	}
	fill := view.solid
	if view.mode == "wire" {
		fill = view.depthOnly
	}
	for _, item := range draws {
		pass.SetPipeline(fill)
		pass.SetBindGroup(0, item.object.group, nil)
		pass.SetVertexBuffer(0, item.mesh.vertices, 0)
		pass.SetVertexBuffer(1, item.mesh.instances, 0)
		pass.SetIndexBuffer(item.mesh.indices, gputypes.IndexFormatUint32, 0)
		pass.DrawIndexed(gputypes.DrawIndexedArgs{IndexCount: item.mesh.triangles, InstanceCount: item.count})
	}
	if view.mode != "solid" {
		for _, item := range draws {
			pass.SetPipeline(view.lines)
			pass.SetBindGroup(0, item.object.group, nil)
			pass.SetVertexBuffer(0, item.mesh.vertices, 0)
			pass.SetVertexBuffer(1, item.mesh.instances, 0)
			pass.SetIndexBuffer(item.mesh.edges, gputypes.IndexFormatUint32, 0)
			pass.DrawIndexed(gputypes.DrawIndexedArgs{IndexCount: item.mesh.lines, InstanceCount: item.count})
		}
	}
	return pass.End()
}

// read maps the copied frame and copies it into the reused picture. A map
// that misses its deadline is abandoned and reported as ErrReadbackLate.
func (view *View) read(width, height int) (*image.NRGBA, error) {
	size := uint64(view.rowBytes * height)
	err := view.mapReadback(size)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, view.abandon()
	}
	if err != nil {
		return nil, fmt.Errorf("working GPU readback: %w", err)
	}
	return view.copyReadback(width, height)
}

// copyReadback copies an already-mapped GPU frame into the reusable image.
func (view *View) copyReadback(width, height int) (*image.NRGBA, error) {
	size := uint64(view.rowBytes * height)
	mapped, err := view.readback.MappedRange(0, size)
	if err != nil {
		view.readback.Unmap()
		return nil, err
	}
	pixels := mapped.Bytes()
	picture := view.frame(width, height)
	for row := 0; row < height; row++ {
		copy(picture.Pix[row*picture.Stride:row*picture.Stride+width*4], pixels[row*view.rowBytes:row*view.rowBytes+width*4])
	}
	mapped.Release()
	view.readback.Unmap()
	return picture, nil
}

// mapReadback waits up to readbackDeadline for the frame. A readback forced
// late asks for its map and gives up at once, leaving the map pending.
func (view *View) mapReadback(size uint64) error {
	if view.late > 0 {
		view.late--
		pending, err := view.readback.MapAsync(wgpu.MapModeRead, 0, size)
		if err != nil {
			return err
		}
		pending.Release()
		return context.DeadlineExceeded
	}
	ctx, cancel := context.WithTimeout(context.Background(), readbackDeadline)
	defer cancel()
	return view.readback.Map(ctx, wgpu.MapModeRead, 0, size)
}

// abandon gives up a readback the GPU has not handed back. Unmap cancels a
// map still pending or undoes one that landed after the deadline. The buffer
// is then replaced, so the old request can never resolve against the next
// frame's copy. Releasing it is safe while the GPU still writes it: wgpu
// keeps a buffer alive until the submission that uses it completes.
func (view *View) abandon() error {
	view.stats.Misses++
	err := view.readback.Unmap()
	if err != nil && !errors.Is(err, wgpu.ErrMapNotMapped) {
		view.releaseTargets()
		return fmt.Errorf("working GPU readback: abandon: %w", err)
	}
	view.readback.Release()
	view.readback, err = view.newReadback(view.width, view.height)
	if err != nil {
		view.releaseTargets()
		return err
	}
	return ErrReadbackLate
}

// frame returns the reused picture, making a new one only on a size change.
func (view *View) frame(width, height int) *image.NRGBA {
	bounds := image.Rect(0, 0, width, height)
	if view.picture == nil || view.picture.Rect != bounds {
		view.picture = image.NewNRGBA(bounds)
	}
	return view.picture
}

// newReadback makes the mappable buffer a frame of this size is copied to.
func (view *View) newReadback(width, height int) (*wgpu.Buffer, error) {
	view.rowBytes = (width*4 + 255) / 256 * 256
	return view.device.CreateBuffer(&wgpu.BufferDescriptor{Label: "working readback", Size: uint64(view.rowBytes * height), Usage: wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst})
}

// resize rebuilds the render targets for a new size. The same size is a
// no-op.
func (view *View) resize(width, height int) error {
	if width == view.width && height == view.height {
		return nil
	}
	view.releaseTargets()
	texture := func(label string, format gputypes.TextureFormat, usage gputypes.TextureUsage) (*wgpu.Texture, error) {
		return view.device.CreateTexture(&wgpu.TextureDescriptor{Label: label, Size: wgpu.Extent3D{Width: uint32(width), Height: uint32(height), DepthOrArrayLayers: 1}, MipLevelCount: 1, SampleCount: 1, Dimension: wgpu.TextureDimension2D, Format: format, Usage: usage})
	}
	var err error
	view.colour, err = texture("working RGBA", wgpu.TextureFormatRGBA8Unorm, wgpu.TextureUsageRenderAttachment|wgpu.TextureUsageCopySrc)
	if err != nil {
		return err
	}
	view.depth, err = texture("working depth", wgpu.TextureFormatDepth32Float, wgpu.TextureUsageRenderAttachment)
	if err != nil {
		return err
	}
	view.colourView, err = view.device.CreateTextureView(view.colour, nil)
	if err != nil {
		return err
	}
	view.depthView, err = view.device.CreateTextureView(view.depth, nil)
	if err != nil {
		return err
	}
	view.readback, err = view.newReadback(width, height)
	if err != nil {
		return err
	}
	view.width, view.height = width, height
	return nil
}

// releaseTargets releases the render targets and the readback buffer.
func (view *View) releaseTargets() {
	if view.readback != nil {
		view.readback.Release()
		view.readback = nil
	}
	if view.colourView != nil {
		view.colourView.Release()
		view.colourView = nil
	}
	if view.depthView != nil {
		view.depthView.Release()
		view.depthView = nil
	}
	if view.colour != nil {
		view.colour.Release()
		view.colour = nil
	}
	if view.depth != nil {
		view.depth.Release()
		view.depth = nil
	}
	view.width, view.height = 0, 0
}

// Close releases only working-view resources; the host retains device ownership.
func (view *View) Close() error {
	if view == nil || view.device == nil {
		return nil
	}
	pendingErr := view.cancelPending()
	view.releaseTargets()
	for _, item := range view.objects {
		item.group.Release()
		item.uniform.Release()
	}
	for _, item := range view.meshes {
		item.release()
	}
	if view.solid != nil {
		view.solid.Release()
	}
	if view.lines != nil {
		view.lines.Release()
	}
	if view.depthOnly != nil {
		view.depthOnly.Release()
	}
	if view.pipelineLayout != nil {
		view.pipelineLayout.Release()
	}
	if view.layout != nil {
		view.layout.Release()
	}
	if view.shader != nil {
		view.shader.Release()
	}
	view.objects = nil
	view.meshes = nil
	view.picture = nil
	view.device = nil
	view.queue = nil
	return pendingErr
}
