package cheeptune

import (
	"strings"
)

// studioVoicing is a studio voicing around some channels.
func studioVoicing(channels, extra string) []byte {
	return []byte(`{"name": "v", "machine": "studio", "master": 0.2` +
		extra + `, "channels": [` + channels + `]}`)
}

// held is an envelope that is full while a note is held.
const held = `"envelope": {"attack": 0.005, "decay": 0, "sustain": 1,
	"release": 0.05}`

// fmPair is a lead of one carrier and one modulator at a ratio.
func fmPair(ratio string) string {
	return `{"kind": "fm", "role": "lead", "fm": {"arrangement": "chain",
		"operators": [{"ratio": 1, "level": 1, ` + held + `},
		{"ratio": ` + ratio + `, "level": 0.15, ` + held + `}]}}`
}

// slow is a score at sixty beats a minute, so a beat is a second.
func slow(parts ...string) []byte {
	return []byte(`{"name": "t", "tempo": 60, "meter": 4, "loop": false,
		"parts": [` + strings.Join(parts, ",") + `]}`)
}
