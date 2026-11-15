//go:build draw

package hillside

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCPUCommandDoesNotClaimGPUWork(t *testing.T) {
	args := only(map[string]string{})
	args = append(args, "-w", "1", "-h", "1", "-n", "1", "-out",
		filepath.Join(t.TempDir(), "cpu.png"))
	var said, wrong bytes.Buffer
	if code := run(args, &said, &wrong); code != 0 || wrong.Len() != 0 {
		t.Fatalf("real one-pixel CPU render failed: code%d, %s", code, wrong.String())
	}
	for _, unwanted := range []string{"native closest:", "native hierarchy:", "smoke lighting:"} {
		if strings.Contains(said.String(), unwanted) {
			t.Fatalf("CPU command claimed an absent GPU stage %q: %s", unwanted, said.String())
		}
	}
}
