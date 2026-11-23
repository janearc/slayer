package cities

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"testing"
)

// digestOf is a hash of drawn parts: each mesh's name and corners, its
// place, its material's name and colours, and its lamp, in order.
func digestOf(t *testing.T, drawn []Rendered, trees []Tree) string {
	t.Helper()
	sum := sha256.New()
	for _, one := range drawn {
		fmt.Fprint(sum, one.Mesh.Name, one.Place, one.Material.Name,
			one.Material.Colour, one.Material.Glow, one.Lamp)
		for _, triangle := range one.Mesh.Triangles {
			write(t, sum, triangle)
		}
	}
	for _, tree := range trees {
		fmt.Fprint(sum, tree)
	}
	return fmt.Sprintf("%x", sum.Sum(nil))
}

// write writes a value to a hash in its bytes.
func write(t *testing.T, sum hash.Hash, value any) {
	t.Helper()
	if err := binary.Write(sum, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}
