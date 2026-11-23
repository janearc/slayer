//go:build draw && darwin

package gpu

import (
	"fmt"
	"math"
)

// geometryPage holds one or more complete meshes in the existing packed ABI.
// Indices inside its bytes remain global: callers need the first offsets to
// find records, and cannot pass the page alone to the current trace kernel.
// The visitor owns the slices it receives; retaining them retains the memory.
type geometryPage struct {
	firstMesh, meshes                   int
	firstTriangle, firstBend, firstNode int
	triangles, bends, nodes             []byte
}

// sizedPage is what a mesh page holds, as pageSize counts it, and the bytes
// it packs to.
type sizedPage struct{ triangles, bends, nodes, bytes int }

// pageMeshSize checks the same face and smooth-normal limits as treeWithin,
// and accounts for each mesh's actual packed bytes before allocating them.
func pageMeshSize(m pendingMesh) (sizedPage, error) {
	if !m.tree.finite() {
		return sizedPage{}, unsupported{"a mesh face outside finite float32 coordinates"}
	}
	size := sizedPage{triangles: m.tree.faceCount(), bends: m.tree.bendCount(),
		nodes: m.tree.nodeCount()}
	var err error
	size.bytes, err = packedPageBytes(size)
	return size, err
}

// packedPageBytes is how many bytes a mesh page packs to: its triangles,
// bends and nodes at their struct sizes. It refuses a size that overflows.
func packedPageBytes(size sizedPage) (int, error) {
	total := 0
	for _, pair := range [][2]int{{size.triangles, triangleStruct.size()}, {size.bends, bendStruct.size()}, {size.nodes, nodeStruct.size()}} {
		if pair[0] > (math.MaxInt-total)/pair[1] {
			return 0, fmt.Errorf("gpu: mesh page size overflows")
		}
		total += pair[0] * pair[1]
	}
	return total, nil
}

// addPageCounts checks the global indices and smooth-normal precision,
// even though no single page holds the whole scene.
func addPageCounts(held, added sizedPage) (sizedPage, error) {
	if added.triangles > math.MaxInt32-held.triangles || added.nodes > math.MaxInt32-held.nodes {
		return sizedPage{}, fmt.Errorf("gpu: geometry page indices exceed the signed native range")
	}
	if added.bends > mostBends-held.bends {
		return sizedPage{}, unsupported{fmt.Sprintf("more than %d faces shaded smooth", mostBends)}
	}
	held.triangles += added.triangles
	held.bends += added.bends
	held.nodes += added.nodes
	return held, nil
}

// checkPageSizes refuses a list of mesh sizes in which one needs more
// packed bytes than the page budget, or whose counts together overflow.
func checkPageSizes(sizes []sizedPage, budget int) error {
	var counts sizedPage
	for index, size := range sizes {
		if size.bytes > budget {
			return fmt.Errorf("gpu: mesh %d needs %d packed bytes, exceeding page budget %d", index, size.bytes, budget)
		}
		var err error
		counts, err = addPageCounts(counts, size)
		if err != nil {
			return err
		}
	}
	return nil
}

// visitGeometryPages packs the encoder's pending mesh pool in original order.
// Preflight rejects an invalid or oversized mesh before any page is handed
// out. Each emitted page's three packed buffers together fit the given budget.
// It does not bound the source trees, visitor-retained pages, other scene
// buffers, acceleration structures or device memory.
func visitGeometryPages(waiting []pendingMesh, budget int, visit func(geometryPage) error) error {
	if budget <= 0 {
		return fmt.Errorf("gpu: geometry page budget must be positive")
	}
	if visit == nil {
		return fmt.Errorf("gpu: geometry page visitor is nil")
	}
	sizes := make([]sizedPage, len(waiting))
	for index, m := range waiting {
		size, err := pageMeshSize(m)
		if err != nil {
			return err
		}
		sizes[index] = size
	}
	if err := checkPageSizes(sizes, budget); err != nil {
		return err
	}
	var faces, bends, nodes int
	for start := 0; start < len(waiting); {
		var total sizedPage
		end := start
		for end < len(waiting) {
			size := sizes[end]
			if size.bytes > budget-total.bytes {
				break
			}
			total.triangles += size.triangles
			total.bends += size.bends
			total.nodes += size.nodes
			total.bytes += size.bytes
			end++
		}
		page := geometryPage{firstMesh: start, meshes: end - start, firstTriangle: faces, firstBend: bends, firstNode: nodes}
		packed := newScene()
		packed.triangles.bytes = make([]byte, 0, total.triangles*triangleStruct.size())
		packed.bends.bytes = make([]byte, 0, total.bends*bendStruct.size())
		packed.nodes.bytes = make([]byte, 0, total.nodes*nodeStruct.size())
		for _, m := range waiting[start:end] {
			packed.packTree(m.tree, pageSize{triangles: faces, bends: bends,
				nodes: nodes})
			faces += m.tree.faceCount()
			bends += m.tree.bendCount()
			nodes += m.tree.nodeCount()
		}
		page.triangles, page.bends, page.nodes = packed.triangles.bytes, packed.bends.bytes, packed.nodes.bytes
		if err := visit(page); err != nil {
			return err
		}
		start = end
	}
	return nil
}
