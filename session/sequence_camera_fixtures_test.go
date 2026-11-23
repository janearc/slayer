package session

import (
	_ "embed"
)

// sequenceZero is defined as no displacement, successful exit or no items.
const sequenceZero = 0

// sequenceOne is defined as the first milestone and one sample per pixel.
const sequenceOne = 1

//go:embed testdata/camera.yaml
var cameraFixture string
