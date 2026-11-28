package session

import "testing"

func TestExactSamplesRefuseUnsupportedRendererPaths(t *testing.T) {
	for name, given := range map[string]asked{
		"wire":   {wire: true},
		"restir": {restir: true}, "sky looks": {skyLooks: true},
		"fast lighting": {fastLighting: true}, "cached lighting": {cachedLighting: true},
	} {
		t.Run(name, func(t *testing.T) {
			if err := exactSamplesRefusal(given); err == nil {
				t.Fatal("unsupported drawing path was accepted")
			}
		})
	}
	for _, given := range []asked{{}, {onGPU: true},
		{onGPU: true, nativeMeshes: true}, {onGPU: true, nativeShadows: true}} {
		if err := exactSamplesRefusal(given); err != nil {
			t.Fatalf("supported exact draw was refused: %v", err)
		}
	}
}
