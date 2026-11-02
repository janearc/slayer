#!/bin/sh
# generate.sh makes the mocks of everything slayer's tracer reads from its
# scene and film packages, with uber's mockgen, a tool of this module at
# the version go.mod pins: one package of mocks for each, named for it.
# go generate runs it here.
set -eu
slayer=github.com/janearc/slayer

# mock makes NAMEmock/NAME.go from slayer's package at PATH, for the
# interfaces listed, comma between.
mock() {
	go tool mockgen -destination "$1mock/$1.go" -package "$1mock" \
		"$slayer/$2" "$3"
}

mock shape scene/shape Shape,Hitter,Spanner,Normaler
mock film film Sensor,Emitter,Beamer
mock medium scene/medium Medium,Timed,Spanner
mock texture scene/texture Texture,Timed,Footprinted,Checks,Painter
mock sky scene/sky Sky
mock thing scene/thing Motion,Radiant,Lit
mock gas scene/gas Optical
