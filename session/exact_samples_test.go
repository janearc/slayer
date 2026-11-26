package session

import "testing"

func TestExactSamplesRefuseUnsupportedRendererPaths(t *testing.T) {
	for name, given := range map[string]asked{
		"wire":   {wire: true},
		"restir": {restir: true}, "sky looks": {skyLooks: true},
		"fast lighting": {fastLighting: true}, "cached lighting": {cachedLighting: true},
		"native meshes": {nativeMeshes: true}, "native shadows": {nativeShadows: true},
	} {
		t.Run(name, func(t *testing.T) {
			if err := exactSamplesRefusal(given); err == nil {
				t.Fatal("unsupported drawing path was accepted")
			}
		})
	}
	if err := exactSamplesRefusal(asked{}); err != nil {
		t.Fatalf("plain CPU draw was refused: %v", err)
	}
}
