//go:build !darwin || !arm64

package manufacture

import "github.com/janearc/slayer/maths"

func shootNormal(v maths.Vec) maths.Vec { return v.Unit() }
