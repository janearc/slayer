// Package geometry holds construction shared by unrelated model families.
package geometry

import "github.com/janearc/slayer/maths"

// AreaNormals sums the oriented triangle areas at each vertex.
func AreaNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := make([]maths.Vec, len(points))
	for _, face := range faces {
		origin := points[face[0]]
		area := points[face[1]].Sub(origin).Cross(points[face[2]].Sub(origin))
		for _, index := range face {
			normals[index] = normals[index].Add(area)
		}
	}
	return normals
}

// UnitNormals normalizes used vertices and leaves unused vertices at zero.
func UnitNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := AreaNormals(points, faces)
	for i, normal := range normals {
		if normal.Len() > 0 {
			normals[i] = normal.Unit()
		}
	}
	return normals
}

// SmoothNormals uses an upward fallback for unreferenced vertices.
func SmoothNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := AreaNormals(points, faces)
	for i, normal := range normals {
		if normal == (maths.Vec{}) {
			normals[i] = maths.Up
		}
	}
	return normals
}
