package oakland

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

const geometryPolicyVersion = 2

type document struct {
	Asset       assetDefinition `yaml:"asset"`
	Scene       *int            `yaml:"scene"`
	Scenes      []sceneDef      `yaml:"scenes"`
	Nodes       []nodeDef       `yaml:"nodes"`
	Meshes      []meshDef       `yaml:"meshes"`
	Materials   []materialDef   `yaml:"materials"`
	Buffers     []bufferDef     `yaml:"buffers"`
	BufferViews []bufferViewDef `yaml:"bufferViews"`
	Accessors   []accessorDef   `yaml:"accessors"`
}

type assetDefinition struct {
	Version string `yaml:"version"`
}

type sceneDef struct {
	Nodes []int `yaml:"nodes"`
}

type nodeDef struct {
	Name        string    `yaml:"name"`
	Mesh        *int      `yaml:"mesh"`
	Children    []int     `yaml:"children"`
	Translation []float64 `yaml:"translation"`
	Rotation    []float64 `yaml:"rotation"`
	Scale       []float64 `yaml:"scale"`
}

type meshDef struct {
	Primitives []primitiveDef `yaml:"primitives"`
}

type primitiveDef struct {
	Attributes map[string]int `yaml:"attributes"`
	Indices    *int           `yaml:"indices"`
	Material   *int           `yaml:"material"`
	Mode       *int           `yaml:"mode"`
}

type materialDef struct {
	Name string `yaml:"name"`
}

type bufferDef struct {
	URI        *string `yaml:"uri"`
	ByteLength int64   `yaml:"byteLength"`
}

type bufferViewDef struct {
	Buffer     int    `yaml:"buffer"`
	ByteOffset int64  `yaml:"byteOffset"`
	ByteLength int64  `yaml:"byteLength"`
	ByteStride *int64 `yaml:"byteStride"`
}

type accessorDef struct {
	BufferView int       `yaml:"bufferView"`
	ByteOffset int64     `yaml:"byteOffset"`
	Component  int       `yaml:"componentType"`
	Count      int       `yaml:"count"`
	Type       string    `yaml:"type"`
	Normalized bool      `yaml:"normalized"`
	Min        []float64 `yaml:"min"`
	Max        []float64 `yaml:"max"`
}

type primitive struct {
	geometry *mesh.Mesh
	slot     string
}

type placedPrimitive struct {
	primitive int
	transform affine
}

// Read loads one supported static glTF scene and its complete source closure.
func Read(path string, policy Policy) (*Asset, error) {
	if !validNumber(policy.TargetHeight) || policy.TargetHeight < 0 {
		return nil, fmt.Errorf("oakland: target height must be finite and non-negative")
	}
	if policy.TargetHeight > 0 && policy.HeightUnit != "m" && policy.HeightUnit != "ft" && policy.HeightUnit != "in" {
		return nil, fmt.Errorf("oakland: target height unit must be m, ft, or in")
	}
	if policy.TargetHeight == 0 && policy.HeightUnit != "" {
		return nil, fmt.Errorf("oakland: height unit requires a target height")
	}
	reader, input, mainPath, err := readInput(path, policy)
	if err != nil {
		return nil, err
	}
	schema, err := makeGltfSchema()
	if err != nil {
		return nil, fmt.Errorf("oakland: compile glTF subset schema: %w", err)
	}
	node, err := schema.Read(mainPath, input.json)
	if err != nil {
		return nil, err
	}
	var doc document
	if err := node.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: decode glTF subset: %w", mainPath, err)
	}
	if doc.Asset.Version != "2.0" {
		return nil, fmt.Errorf("%s: glTF asset version must be 2.0", mainPath)
	}
	limits := policy.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	if len(doc.Nodes) > limits.Nodes {
		return nil, fmt.Errorf("%s: node count exceeds %d", mainPath, limits.Nodes)
	}
	if err := validateNodeReferences(doc); err != nil {
		return nil, err
	}
	buffers, err := loadBuffers(reader, mainPath, input.bin, doc.Buffers, limits.BufferBytes)
	if err != nil {
		return nil, err
	}
	if err := validateGraph(doc.Nodes, limits); err != nil {
		return nil, err
	}
	selected, err := selectedScene(doc, policy.Scene)
	if err != nil {
		return nil, err
	}
	primitives, err := loadPrimitives(doc, buffers, limits)
	if err != nil {
		return nil, err
	}
	placed, err := selectedPrimitives(doc, selected, primitives, limits)
	if err != nil {
		return nil, err
	}
	if len(placed) == 0 {
		return nil, fmt.Errorf("oakland: selected scene contains no triangle instances")
	}
	var normalized *HeightNormalization
	if policy.TargetHeight > 0 {
		factor, minY, maxY, err := normalizeHeight(placed, primitives, policy.TargetHeight, limits)
		if err != nil {
			return nil, err
		}
		verticalOffset := -minY * factor
		if !validNumber(verticalOffset) {
			return nil, fmt.Errorf("oakland: normalized vertical offset is not finite")
		}
		normalized = &HeightNormalization{sourceMinY: minY, sourceMaxY: maxY,
			targetHeight: policy.TargetHeight, heightUnit: policy.HeightUnit,
			factor: factor, verticalOffset: verticalOffset}
		for i := range placed {
			placed[i].transform = placed[i].transform.scaled(factor)
			placed[i].transform.offset.Y += verticalOffset
		}
	}
	instances, slots, err := instantiateGeometry(placed, primitives, limits)
	if err != nil {
		return nil, err
	}
	options, err := optionsKey(reader.sourceRecords(), policy, selected)
	if err != nil {
		return nil, err
	}
	return &Asset{sources: reader.sourceRecords(), options: options,
		normalized: normalized,
		slots:      slots, instances: instances}, nil
}

// selectedScene requires an explicit choice when the document has several scenes.
func selectedScene(doc document, requested *int) (int, error) {
	if requested != nil {
		if *requested < 0 || *requested >= len(doc.Scenes) {
			return 0, fmt.Errorf("oakland: selected scene index is out of range")
		}
		return *requested, nil
	}
	if len(doc.Scenes) != 1 {
		return 0, fmt.Errorf("oakland: select a scene when the document has multiple scenes")
	}
	return 0, nil
}

// validateGraph checks child links, hierarchy cycles and depth before traversal.
func validateGraph(nodes []nodeDef, limits Limits) error {
	parents := make([]int, len(nodes))
	for parent, node := range nodes {
		if len(node.Children) > limits.Nodes {
			return fmt.Errorf("oakland: node %d has too many children", parent)
		}
		for _, child := range node.Children {
			if child < 0 || child >= len(nodes) {
				return fmt.Errorf("oakland: node %d references node %d out of range", parent, child)
			}
			parents[child]++
			if parents[child] > 1 {
				return fmt.Errorf("oakland: node %d has multiple parents", child)
			}
		}
	}
	state := make([]uint8, len(nodes))
	var visit func(int, int) error
	visit = func(at, depth int) error {
		if depth > limits.Depth {
			return fmt.Errorf("oakland: node hierarchy exceeds depth %d", limits.Depth)
		}
		if state[at] == 1 {
			return fmt.Errorf("oakland: node hierarchy contains a cycle at node %d", at)
		}
		if state[at] == 2 {
			return nil
		}
		state[at] = 1
		for _, child := range nodes[at].Children {
			if err := visit(child, depth+1); err != nil {
				return err
			}
		}
		state[at] = 2
		return nil
	}
	for i := range nodes {
		if err := visit(i, 1); err != nil {
			return err
		}
	}
	return nil
}

// validateNodeReferences checks node, mesh and scene references before loading.
func validateNodeReferences(doc document) error {
	if doc.Scene != nil && (*doc.Scene < 0 || *doc.Scene >= len(doc.Scenes)) {
		return fmt.Errorf("oakland: document default scene index is out of range")
	}
	for index, node := range doc.Nodes {
		if _, err := nodeAffine(node); err != nil {
			return fmt.Errorf("oakland: node %d: %w", index, err)
		}
		if node.Mesh != nil && (*node.Mesh < 0 || *node.Mesh >= len(doc.Meshes)) {
			return fmt.Errorf("oakland: node %d mesh index is out of range", index)
		}
	}
	for sceneIndex, scene := range doc.Scenes {
		for _, root := range scene.Nodes {
			if root < 0 || root >= len(doc.Nodes) {
				return fmt.Errorf("oakland: scene %d root node %d is out of range", sceneIndex, root)
			}
		}
	}
	return nil
}

// loadBuffers reads embedded and external buffers under source and byte limits.
func loadBuffers(reader *sourceReader, main string, glb []byte, definitions []bufferDef,
	maxBytes int64,
) ([][]byte, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("oakland: document has no buffers")
	}
	var total int64
	loaded := make([][]byte, len(definitions))
	glbUsed := false
	mainDir := filepath.ToSlash(filepath.Dir(main))
	for index, definition := range definitions {
		if definition.ByteLength <= 0 || definition.ByteLength > maxBytes-total {
			return nil, fmt.Errorf("oakland: buffer byte lengths exceed %d", maxBytes)
		}
		total += definition.ByteLength
		if definition.URI == nil {
			if len(glb) == 0 || glbUsed || index != 0 || int64(len(glb)) < definition.ByteLength || int64(len(glb))-definition.ByteLength > 3 {
				return nil, fmt.Errorf("oakland: invalid GLB buffer declaration")
			}
			for _, padding := range glb[definition.ByteLength:] {
				if padding != 0 {
					return nil, fmt.Errorf("oakland: GLB BIN padding must be zero")
				}
			}
			loaded[index] = glb[:int(definition.ByteLength)]
			glbUsed = true
			continue
		}
		if len(glb) != 0 && !glbUsed {
			return nil, fmt.Errorf("oakland: GLB BIN chunk cannot accompany an external buffer")
		}
		uri, err := externalURI(*definition.URI)
		if err != nil {
			return nil, err
		}
		if mainDir != "." {
			uri = filepath.ToSlash(filepath.Join(filepath.FromSlash(mainDir), filepath.FromSlash(uri)))
		}
		_, bytes, err := reader.read(uri)
		if err != nil {
			return nil, err
		}
		if int64(len(bytes)) != definition.ByteLength {
			return nil, fmt.Errorf("oakland: buffer %q has %d bytes, declared %d", uri, len(bytes), definition.ByteLength)
		}
		loaded[index] = bytes
	}
	if len(glb) != 0 && !glbUsed {
		return nil, fmt.Errorf("oakland: unused GLB BIN chunk")
	}
	return loaded, nil
}

// optionsKey hashes canonical decode choices and the exact source closure.
func optionsKey(sources []Source, policy Policy, selected int) ([32]byte, error) {
	limits := policy.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	canonical := struct {
		GeometryPolicy int
		Scene          int
		TargetHeight   uint64
		HeightUnit     string
		Limits         Limits
		Sources        []struct {
			Path   string
			Size   int64
			Digest [32]byte
		}
	}{GeometryPolicy: geometryPolicyVersion, Scene: selected, TargetHeight: math.Float64bits(policy.TargetHeight),
		HeightUnit: policy.HeightUnit, Limits: limits}
	for _, source := range sources {
		canonical.Sources = append(canonical.Sources, struct {
			Path   string
			Size   int64
			Digest [32]byte
		}{source.path, source.size, source.sha256})
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return [32]byte{}, fmt.Errorf("oakland: encode asset key: %w", err)
	}
	return sha256.Sum256(raw), nil
}

// loadPrimitives validates and constructs bounded native triangle meshes.
func loadPrimitives(doc document, buffers [][]byte, limits Limits) ([]primitive, error) {
	primitives := make([]primitive, 0)
	vertices, indices := 0, 0
	for meshIndex, definition := range doc.Meshes {
		for primitiveIndex, source := range definition.Primitives {
			mode := 4
			if source.Mode != nil {
				mode = *source.Mode
			}
			if mode != 4 {
				return nil, fmt.Errorf("%w: mesh %d primitive %d mode %d", errUnsupported, meshIndex, primitiveIndex, mode)
			}
			for semantic := range source.Attributes {
				if semantic != "POSITION" && semantic != "NORMAL" {
					return nil, fmt.Errorf("%w: vertex attribute %s", errUnsupported, semantic)
				}
			}
			positionIndex, ok := source.Attributes["POSITION"]
			if !ok {
				return nil, fmt.Errorf("oakland: mesh %d primitive %d has no POSITION", meshIndex, primitiveIndex)
			}
			if positionIndex < 0 || positionIndex >= len(doc.Accessors) ||
				doc.Accessors[positionIndex].Count > limits.Vertices-vertices {
				return nil, fmt.Errorf("oakland: total vertices exceed %d", limits.Vertices)
			}
			points, err := readVec3Accessor(doc, buffers, positionIndex, "POSITION")
			if err != nil {
				return nil, err
			}
			vertices += len(points)
			if vertices > limits.Vertices {
				return nil, fmt.Errorf("oakland: total vertices exceed %d", limits.Vertices)
			}
			var normals []maths.Vec
			if normalIndex, exists := source.Attributes["NORMAL"]; exists {
				if normalIndex < 0 || normalIndex >= len(doc.Accessors) ||
					doc.Accessors[normalIndex].Count > limits.Vertices {
					return nil, fmt.Errorf("oakland: normal vertex count exceeds %d", limits.Vertices)
				}
				normals, err = readVec3Accessor(doc, buffers, normalIndex, "NORMAL")
				if err != nil {
					return nil, err
				}
				if len(normals) != len(points) {
					return nil, fmt.Errorf("oakland: NORMAL count does not match POSITION")
				}
			}
			indexCount := len(points)
			if source.Indices != nil {
				if *source.Indices < 0 || *source.Indices >= len(doc.Accessors) ||
					doc.Accessors[*source.Indices].Count > limits.Indices-indices {
					return nil, fmt.Errorf("oakland: total indices exceed %d", limits.Indices)
				}
				indexCount = doc.Accessors[*source.Indices].Count
			}
			if indexCount > limits.Indices-indices {
				return nil, fmt.Errorf("oakland: total indices exceed %d", limits.Indices)
			}
			faces, err := readIndices(doc, buffers, source.Indices, len(points))
			if err != nil {
				return nil, err
			}
			if len(faces) > (limits.Indices-indices)/3 {
				return nil, fmt.Errorf("oakland: total indices exceed %d", limits.Indices)
			}
			indices += len(faces) * 3
			if indices > limits.Indices {
				return nil, fmt.Errorf("oakland: total indices exceed %d", limits.Indices)
			}
			made, err := mesh.New(points, faces, normals, normals != nil)
			if err != nil {
				return nil, fmt.Errorf("oakland: mesh %d primitive %d: %w", meshIndex, primitiveIndex, err)
			}
			slot, err := materialSlot(source.Material, doc.Materials)
			if err != nil {
				return nil, err
			}
			primitives = append(primitives, primitive{geometry: made, slot: slot})
		}
	}
	return primitives, nil
}

// materialSlot gives each source material a stable slot, including the default.
func materialSlot(index *int, materials []materialDef) (string, error) {
	if index == nil {
		return "material/default", nil
	}
	if *index < 0 || *index >= len(materials) {
		return "", fmt.Errorf("oakland: material index %d is out of range", *index)
	}
	name := materials[*index].Name
	return fmt.Sprintf("material/%d/%s", *index, name), nil
}

// selectedPrimitives walks the selected scene and composes its node transforms.
func selectedPrimitives(doc document, sceneIndex int, primitives []primitive,
	limits Limits,
) ([]placedPrimitive, error) {
	var placed []placedPrimitive
	seenNodes := make(map[int]bool)
	meshStart := make([]int, len(doc.Meshes)+1)
	for index, definition := range doc.Meshes {
		meshStart[index+1] = meshStart[index] + len(definition.Primitives)
	}
	var visit func(int, affine, int) error
	visit = func(index int, parent affine, depth int) error {
		if depth > limits.Depth {
			return fmt.Errorf("oakland: selected scene exceeds node depth %d", limits.Depth)
		}
		if seenNodes[index] {
			return fmt.Errorf("oakland: selected scene repeats node %d", index)
		}
		seenNodes[index] = true
		node := doc.Nodes[index]
		local, err := nodeAffine(node)
		if err != nil {
			return fmt.Errorf("oakland: node %d: %w", index, err)
		}
		global := parent.then(local)
		if node.Mesh != nil {
			if *node.Mesh < 0 || *node.Mesh >= len(doc.Meshes) {
				return fmt.Errorf("oakland: node %d mesh index is out of range", index)
			}
			for offset, definition := range doc.Meshes[*node.Mesh].Primitives {
				primitiveIndex := meshStart[*node.Mesh] + offset
				if primitiveIndex < 0 || primitiveIndex >= len(primitives) || definition.Mode != nil && *definition.Mode != 4 {
					return fmt.Errorf("oakland: primitive lookup is inconsistent")
				}
				if len(placed) >= limits.Instances {
					return fmt.Errorf("oakland: selected scene exceeds %d instances", limits.Instances)
				}
				placed = append(placed, placedPrimitive{primitive: primitiveIndex, transform: global})
			}
		}
		for _, child := range node.Children {
			if err := visit(child, global, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	identity := affineIdentity()
	roots := make(map[int]bool)
	for _, root := range doc.Scenes[sceneIndex].Nodes {
		if root < 0 || root >= len(doc.Nodes) {
			return nil, fmt.Errorf("oakland: scene root node %d is out of range", root)
		}
		if roots[root] {
			return nil, fmt.Errorf("oakland: scene repeats root node %d", root)
		}
		roots[root] = true
		if err := visit(root, identity, 1); err != nil {
			return nil, err
		}
	}
	return placed, nil
}

// normalizeHeight measures unique transformed geometry and bounds its vertex work.
func normalizeHeight(placed []placedPrimitive, primitives []primitive, target float64, limits Limits) (float64, float64, float64, error) {
	minY, maxY := math.Inf(1), math.Inf(-1)
	type verticalRange struct{ min, max float64 }
	ranges := make(map[geometryKey]verticalRange)
	var workVertices int64
	for _, instance := range placed {
		key := transformedGeometryKey(instance.transform, instance.primitive)
		rangeY, exists := ranges[key]
		if !exists {
			workVertices += int64(len(primitives[instance.primitive].geometry.Points()))
			if workVertices > int64(limits.Vertices) {
				return 0, 0, 0, fmt.Errorf("oakland: normalization work exceeds %d vertices", limits.Vertices)
			}
			rangeY = verticalRange{min: math.Inf(1), max: math.Inf(-1)}
			for _, point := range primitives[instance.primitive].geometry.Points() {
				local := instance.transform.direction(point)
				if !validVec(local) {
					return 0, 0, 0, fmt.Errorf("oakland: transformed bound is not finite")
				}
				rangeY.min = math.Min(rangeY.min, local.Y)
				rangeY.max = math.Max(rangeY.max, local.Y)
			}
			ranges[key] = rangeY
		}
		minY = math.Min(minY, rangeY.min+instance.transform.offset.Y)
		maxY = math.Max(maxY, rangeY.max+instance.transform.offset.Y)
	}
	height := maxY - minY
	if !validNumber(height) || height <= 0 {
		return 0, 0, 0, fmt.Errorf("oakland: selected scene has no finite positive height")
	}
	factor := target / height
	if !validNumber(factor) || factor <= 0 {
		return 0, 0, 0, fmt.Errorf("oakland: target height scale is not finite")
	}
	return factor, minY, maxY, nil
}

// instantiateGeometry shares source or canonical baked meshes across placements.
func instantiateGeometry(placed []placedPrimitive, primitives []primitive, limits Limits) ([]Instance, []string, error) {
	if err := boundBakedGeometry(placed, primitives, limits); err != nil {
		return nil, nil, err
	}
	type cacheEntry struct {
		mesh *mesh.Mesh
	}
	cache := map[geometryKey]cacheEntry{}
	instances := make([]Instance, 0, len(placed))
	slotSet := map[string]struct{}{}
	for _, item := range placed {
		prototype := primitives[item.primitive]
		key, placement, shareSource, err := geometryPlacement(item.transform, item.primitive)
		if err != nil {
			return nil, nil, err
		}
		geometry := prototype.geometry
		if existing, ok := cache[key]; ok {
			geometry = existing.mesh
		} else {
			if !shareSource {
				geometry, _, _, err = geometryFor(prototype.geometry, item.transform, item.primitive)
				if err != nil {
					return nil, nil, err
				}
			}
			cache[key] = cacheEntry{mesh: geometry}
		}
		instances = append(instances, Instance{geometry: geometry,
			materialSlot: prototype.slot, placement: placement})
		slotSet[prototype.slot] = struct{}{}
	}
	slots := make([]string, 0, len(slotSet))
	for slot := range slotSet {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	return instances, slots, nil
}

// boundBakedGeometry preflights unique transformed mesh allocation totals.
func boundBakedGeometry(placed []placedPrimitive, primitives []primitive, limits Limits) error {
	seen := make(map[geometryKey]struct{}, len(placed))
	var vertices, indices int64
	for _, item := range placed {
		key, _, shared, err := geometryPlacement(item.transform, item.primitive)
		if err != nil {
			return err
		}
		if shared {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		geometry := primitives[item.primitive].geometry
		vertices += int64(len(geometry.Points()))
		indices += int64(geometry.FaceCount()) * 3
		if vertices > int64(limits.Vertices) || indices > int64(limits.Indices) {
			return fmt.Errorf("oakland: baked geometry exceeds %d vertices or %d indices", limits.Vertices, limits.Indices)
		}
	}
	return nil
}

// Instantiate binds every source slot to an explicit native material.
func (asset *Asset) Instantiate(bindings map[string]*thing.Material) ([]thing.Thing, error) {
	if asset == nil {
		return nil, fmt.Errorf("oakland: no asset")
	}
	for _, slot := range asset.slots {
		if bindings[slot] == nil {
			return nil, fmt.Errorf("oakland: material slot %q has no native binding", slot)
		}
	}
	for slot := range bindings {
		index := sort.SearchStrings(asset.slots, slot)
		if index == len(asset.slots) || asset.slots[index] != slot {
			return nil, fmt.Errorf("oakland: extra material binding %q", slot)
		}
	}
	made := make([]thing.Thing, 0, len(asset.instances))
	for _, instance := range asset.instances {
		made = append(made, thing.Thing{Shape: instance.geometry,
			Material: bindings[instance.materialSlot], Motion: thing.Still(instance.placement)})
	}
	return made, nil
}

// readVec3Accessor reads a checked float VEC3 attribute into Slayer vectors.
func readVec3Accessor(doc document, buffers [][]byte, index int, semantic string) ([]maths.Vec, error) {
	if index < 0 || index >= len(doc.Accessors) {
		return nil, fmt.Errorf("oakland: %s accessor index is out of range", semantic)
	}
	accessor := doc.Accessors[index]
	if accessor.Component != 5126 || accessor.Type != "VEC3" || accessor.Normalized {
		return nil, fmt.Errorf("%w: %s must be an unnormalized float32 VEC3", errUnsupported, semantic)
	}
	if accessor.Min != nil && len(accessor.Min) != 3 || accessor.Max != nil && len(accessor.Max) != 3 {
		return nil, fmt.Errorf("oakland: %s accessor bounds must have three components", semantic)
	}
	if semantic == "POSITION" && (accessor.Min == nil || accessor.Max == nil) {
		return nil, fmt.Errorf("oakland: POSITION accessor requires min and max bounds")
	}
	data, stride, err := accessorBytes(doc, buffers, accessor, 12, 4)
	if err != nil {
		return nil, err
	}
	points := make([]maths.Vec, accessor.Count)
	for i := range points {
		at := i * stride
		values := [3]float64{}
		for axis := range values {
			values[axis] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[at+axis*4 : at+axis*4+4])))
			if !validNumber(values[axis]) {
				return nil, fmt.Errorf("oakland: %s accessor contains a non-finite value", semantic)
			}
		}
		points[i] = maths.V(values[0], values[1], values[2])
	}
	if accessor.Min != nil && accessor.Max != nil {
		for axis := range 3 {
			if !validNumber(accessor.Min[axis]) || !validNumber(accessor.Max[axis]) || accessor.Min[axis] > accessor.Max[axis] {
				return nil, fmt.Errorf("oakland: %s bounds are invalid", semantic)
			}
			if semantic == "POSITION" {
				for index, point := range points {
					value := component(point, axis)
					if value < accessor.Min[axis] || value > accessor.Max[axis] {
						return nil, fmt.Errorf("oakland: POSITION %d is outside its declared bounds", index)
					}
				}
			}
		}
	}
	return points, nil
}

// readIndices checks scalar indices and groups triangle lists into faces.
func readIndices(doc document, buffers [][]byte, index *int, vertices int) ([][3]int, error) {
	if index == nil {
		if vertices%3 != 0 {
			return nil, fmt.Errorf("oakland: non-indexed TRIANGLES vertex count is not divisible by three")
		}
		faces := make([][3]int, vertices/3)
		for i := range faces {
			faces[i] = [3]int{3 * i, 3*i + 1, 3*i + 2}
		}
		return faces, nil
	}
	if *index < 0 || *index >= len(doc.Accessors) {
		return nil, fmt.Errorf("oakland: index accessor is out of range")
	}
	accessor := doc.Accessors[*index]
	if accessor.Type != "SCALAR" || accessor.Normalized || accessor.Component != 5121 && accessor.Component != 5123 && accessor.Component != 5125 {
		return nil, fmt.Errorf("%w: indices must be unsigned SCALAR values", errUnsupported)
	}
	componentBytes := 0
	switch accessor.Component {
	case 5121:
		componentBytes = 1
	case 5123:
		componentBytes = 2
	case 5125:
		componentBytes = 4
	}
	if accessor.BufferView >= 0 && accessor.BufferView < len(doc.BufferViews) &&
		doc.BufferViews[accessor.BufferView].ByteStride != nil {
		return nil, fmt.Errorf("oakland: index bufferView cannot have a byteStride")
	}
	data, stride, err := accessorBytes(doc, buffers, accessor, componentBytes, componentBytes)
	if err != nil {
		return nil, err
	}
	if accessor.Count%3 != 0 {
		return nil, fmt.Errorf("oakland: index count is not divisible by three")
	}
	faces := make([][3]int, accessor.Count/3)
	for i := 0; i < accessor.Count; i++ {
		at := i * stride
		var value uint32
		switch accessor.Component {
		case 5121:
			value = uint32(data[at])
		case 5123:
			value = uint32(binary.LittleEndian.Uint16(data[at : at+2]))
		case 5125:
			value = binary.LittleEndian.Uint32(data[at : at+4])
		}
		if uint64(value) >= uint64(vertices) {
			return nil, fmt.Errorf("oakland: index %d names vertex %d of %d", i, value, vertices)
		}
		faces[i/3][i%3] = int(value)
	}
	return faces, nil
}

// accessorBytes verifies alignment and bounds before returning attribute bytes.
func accessorBytes(doc document, buffers [][]byte, accessor accessorDef,
	elementBytes, componentBytes int,
) ([]byte, int, error) {
	if accessor.BufferView < 0 || accessor.BufferView >= len(doc.BufferViews) {
		return nil, 0, fmt.Errorf("oakland: accessor bufferView is out of range")
	}
	view := doc.BufferViews[accessor.BufferView]
	if view.Buffer < 0 || view.Buffer >= len(buffers) || accessor.ByteOffset < 0 || view.ByteOffset < 0 || view.ByteLength < 0 {
		return nil, 0, fmt.Errorf("oakland: invalid accessor buffer range")
	}
	stride := elementBytes
	if view.ByteStride != nil {
		if *view.ByteStride < int64(elementBytes) || *view.ByteStride > 252 || *view.ByteStride%4 != 0 {
			return nil, 0, fmt.Errorf("oakland: invalid glTF bufferView stride")
		}
		stride = int(*view.ByteStride)
	}
	if accessor.Count <= 0 || accessor.ByteOffset > view.ByteLength {
		return nil, 0, fmt.Errorf("oakland: accessor range is empty or outside its view")
	}
	if (view.ByteOffset%int64(componentBytes)+accessor.ByteOffset%int64(componentBytes))%int64(componentBytes) != 0 {
		return nil, 0, fmt.Errorf("oakland: accessor is not component-aligned")
	}
	count := int64(accessor.Count - 1)
	if count > (math.MaxInt64-int64(elementBytes))/int64(stride) {
		return nil, 0, fmt.Errorf("oakland: accessor byte range overflows")
	}
	last := count*int64(stride) + int64(elementBytes)
	if last < 0 || accessor.ByteOffset > view.ByteLength-last {
		return nil, 0, fmt.Errorf("oakland: accessor exceeds its bufferView")
	}
	if view.ByteOffset > int64(len(buffers[view.Buffer]))-view.ByteLength {
		return nil, 0, fmt.Errorf("oakland: bufferView exceeds its buffer")
	}
	start := view.ByteOffset + accessor.ByteOffset
	end := view.ByteOffset + view.ByteLength
	if end < start || end > int64(len(buffers[view.Buffer])) {
		return nil, 0, fmt.Errorf("oakland: bufferView range overflows its buffer")
	}
	return buffers[view.Buffer][int(start):int(end)], stride, nil
}
