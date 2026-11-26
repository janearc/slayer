package stagewire

import "testing"

func TestFilmPresetsRejectIncompleteOrAmbiguousProducerOutput(t *testing.T) {
	for _, raw := range []string{
		`[]`,
		`[{"name":"x","width":0,"height":2}]`,
		`[{"name":"x","width":1,"height":2},{"name":"x","width":2,"height":3}]`,
		`[{"name":"x","width":1,"height":2}] {}`,
		`[{"name":"x","width":1,"height":2,"unknown":3}]`,
	} {
		if _, err := decodeFilmPresets([]byte(raw)); err == nil {
			t.Errorf("accepted invalid preset output %s", raw)
		}
	}
	got, err := decodeFilmPresets([]byte(`[{"name":"half-portrait","width":1080,"height":960}]`))
	if err != nil || len(got) != 1 || got[0].Name != "half-portrait" || got[0].Width != 1080 || got[0].Height != 960 {
		t.Fatalf("valid preset = %+v, %v", got, err)
	}
}
