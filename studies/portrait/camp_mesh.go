package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
)

const campModelCredit = "SMPL-X sample from Meshcapade and the Max Planck Institute for Intelligent Systems (Pavlakos et al. 2019), CC BY 4.0 https://smpl.is.tue.mpg.de/bodylicense.html. Slayer triangulated and posed the sample as a standing scale figure and supplies a bronze material and 2 m normalization in the camp scene."

// writeCampMesh exports the standing study as a single static Oakland asset.
func writeCampMesh(source, destination string) error {
	body, eyes := readObj(source)
	carry := standing(body).Carry
	points := make([]maths.Vec, len(body.pts))
	for i, point := range body.pts {
		points[i] = carry(point)
	}
	faces := append(body.faces, eyes.faces...)
	data, positions, elements, low, high := campGeometry(points, faces)
	const float32Type, uint32Type = 5126, 5125
	view := func(start, size int) map[string]int {
		return map[string]int{"buffer": 0, "byteOffset": start, "byteLength": size}
	}
	accessor := func(index, component, count int, kind string) map[string]any {
		return map[string]any{"bufferView": index, "componentType": component, "count": count, "type": kind}
	}
	pointsAccessor := accessor(0, float32Type, len(points), "VEC3")
	pointsAccessor["min"], pointsAccessor["max"] = low, high
	document := map[string]any{
		"asset":  map[string]string{"version": "2.0", "generator": "Slayer portrait study", "copyright": campModelCredit},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []int{0}}},
		"nodes":  []any{map[string]int{"mesh": 0}},
		"meshes": []any{map[string]any{"primitives": []any{map[string]any{
			"attributes": map[string]int{"POSITION": 0, "NORMAL": 1}, "indices": 2, "material": 0,
		}}}},
		"materials":   []any{map[string]string{"name": "visitor"}},
		"buffers":     []any{map[string]int{"byteLength": len(data)}},
		"bufferViews": []any{view(0, positions), view(positions, positions), view(2*positions, elements)},
		"accessors":   []any{pointsAccessor, accessor(1, float32Type, len(points), "VEC3"), accessor(2, uint32Type, len(faces)*3, "SCALAR")},
	}
	header, err := json.Marshal(document)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, campGLB(header, data), 0644)
}

// campGeometry declares bounds in the same float32 coordinates it writes.
func campGeometry(points []maths.Vec, faces [][3]int) ([]byte, int, int, [3]float64, [3]float64) {
	const vectorBytes = 3 * 4
	size := len(points) * vectorBytes
	elements := len(faces) * 3 * 4
	data := make([]byte, 0, 2*size+elements)
	point0 := points[0]
	low := [3]float64{float64(float32(point0.X)), float64(float32(point0.Y)), float64(float32(point0.Z))}
	high := low
	for _, p := range points {
		for i, value := range []float32{float32(p.X), float32(p.Y), float32(p.Z)} {
			low[i], high[i] = min(low[i], float64(value)), max(high[i], float64(value))
			data = binary.LittleEndian.AppendUint32(data, math.Float32bits(value))
		}
	}
	for _, n := range geometry.AreaNormals(points, faces) {
		if n.Len() == 0 {
			n = maths.YAxis
		}
		for _, value := range []float64{n.X, n.Y, n.Z} {
			data = binary.LittleEndian.AppendUint32(data, math.Float32bits(float32(value)))
		}
	}
	for _, face := range faces {
		for _, i := range face {
			data = binary.LittleEndian.AppendUint32(data, uint32(i))
		}
	}
	return data, size, elements, low, high
}

// campGLB is a GLB 2.0 container with one JSON and one binary chunk.
func campGLB(header, data []byte) []byte {
	header = append(header, bytes.Repeat([]byte{' '}, (4-len(header)%4)%4)...)
	total := 12 + 8 + len(header) + 8 + len(data)
	result := binary.LittleEndian.AppendUint32([]byte("glTF"), 2)
	result = binary.LittleEndian.AppendUint32(result, uint32(total))
	result = binary.LittleEndian.AppendUint32(result, uint32(len(header)))
	result = append(result, "JSON"...)
	result = append(result, header...)
	result = binary.LittleEndian.AppendUint32(result, uint32(len(data)))
	result = append(result, "BIN\x00"...)
	return append(result, data...)
}
