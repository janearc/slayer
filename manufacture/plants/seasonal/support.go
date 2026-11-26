package seasonal

import (
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
)

const (
	spaceZero = 0
	spaceOne  = 1
	spaceHalf = .5
	fullTurn  = 2 * math.Pi
	picking   = 0
	growing   = 1
)

type span struct{ least, most float64 }

func (s span) pick(v float64) float64 { return s.least + (s.most-s.least)*v }
func (s span) count(v float64) int    { return maths.CountBetween(s.least, s.most, v) }
func positive(v float64) bool         { return v > 0 && !math.IsInf(v, 0) }

type face = [3]maths.Vec
type surface []face

func (s surface) placed(p maths.Subspace) surface         { return geometry.Placed(s, p) }
func (s surface) meshed() (*mesh.Mesh, error)             { return geometry.FaceMesh(s, nil) }
func enclosing(a, b maths.Box) maths.Box                  { return geometry.Enclosing(a, b) }
func transformed(a maths.Box, b maths.Subspace) maths.Box { return geometry.Transformed(a, b) }
func quad(s surface, a, b, c, d maths.Vec) surface        { return geometry.Quad(s, a, b, c, d) }

type cut struct{ around int }
