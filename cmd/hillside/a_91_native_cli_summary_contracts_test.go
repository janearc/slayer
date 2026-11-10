//go:build darwin

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnePixelGPUCommandsReportTheirActualTraversalModes(t *testing.T) {
	for _, mode := range []string{"gpu", "gpu-native", "gpu-native-shadows"} {
		t.Run(mode, func(t *testing.T) {
			path := onFixture(t, "", "")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "scene:", "scene:\n  renderer: "+mode, 1)
			text = strings.Replace(text, "width: 24", "width: 1", 1)
			text = strings.Replace(text, "height: 16", "height: 1", 1)
			text = strings.Replace(text, "alight: true", "alight: false", 1)
			if err := os.WriteFile(path, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			var said, wrong bytes.Buffer
			code := run([]string{"-set", path, "-out", filepath.Join(t.TempDir(), "native.png")}, &said, &wrong)
			if code != 0 || wrong.Len() != 0 {
				t.Fatalf("real one-pixel %s render failed: code%d, %s", mode, code, wrong.String())
			}
			hierarchy := strings.Contains(said.String(), "native hierarchy:")
			if hierarchy != (mode != "gpu") {
				t.Fatalf("requested %s lost or invented hierarchy work: %s", mode, said.String())
			}
			closest := strings.Contains(said.String(), "native closest:")
			if closest != (mode == "gpu-native") {
				t.Fatalf("requested %s lost or invented primary queries: %s", mode, said.String())
			}
			if mode == "gpu-native" &&
				(!strings.Contains(said.String(), "native closest: 1 queries,") ||
					!strings.Contains(said.String(), "software fallbacks")) {
				t.Fatalf("single actual closest query was hidden: %s", said.String())
			}
		})
	}
}
