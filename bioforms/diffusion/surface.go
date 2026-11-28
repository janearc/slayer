package diffusion

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Surface describes a posed, indexed mesh and its unit-albedo diffuse
// lighting at each vertex. Radius gives the three Gaussian widths in the
// same length units as Points.
type Surface struct {
	Points   []maths.Vec
	Faces    [][3]int
	Incident []maths.Light
	Radius   [3]float64
}

type link struct {
	to     int
	length float64
}

// SurfaceProbe contains the portion of filtered illumination that came from
// other vertices and the weight assigned to those vertices, per RGB channel.
// The remaining weight is reserved for local lighting sampled at the hit.
type SurfaceProbe struct {
	Nearby, Share maths.Vec
}

// SurfaceDelta returns filtered minus local illumination per vertex. It
// weights the Gaussian by triangle area and follows mesh edges, keeping
// geometrically adjacent but disconnected regions independent. The caller
// scales the correction by diffuse reflectance and the dermal share.
func SurfaceDelta(input Surface) ([]maths.Vec, error) {
	probes, err := SurfaceProbes(input)
	if err != nil {
		return nil, err
	}
	result := make([]maths.Vec, len(probes))
	for vertex, probe := range probes {
		local := input.Incident[vertex]
		result[vertex] = probe.Nearby.Sub(maths.V(probe.Share.X*local.R,
			probe.Share.Y*local.G, probe.Share.Z*local.B))
	}
	return result, nil
}

// SurfaceProbes normalizes an area-weighted Gaussian in each connected
// vertex neighbourhood. Its three-width cutoff is applied per channel.
func SurfaceProbes(input Surface) ([]SurfaceProbe, error) {
	for _, radius := range input.Radius {
		if !(radius > 0 && radius <= math.MaxFloat64/3) {
			return nil, fmt.Errorf("diffusion: surface radii must be positive and finite")
		}
	}
	if len(input.Points) != len(input.Incident) {
		return nil, fmt.Errorf("diffusion: surface needs one incident value per point")
	}
	for _, light := range input.Incident {
		if !(light.R >= 0 && light.R <= math.MaxFloat64 &&
			light.G >= 0 && light.G <= math.MaxFloat64 &&
			light.B >= 0 && light.B <= math.MaxFloat64) {
			return nil, fmt.Errorf("diffusion: surface illumination must be finite and nonnegative")
		}
	}

	neighbors := make([][]link, len(input.Points))
	areas := make([]float64, len(input.Points))
	for _, face := range input.Faces {
		for _, index := range face {
			if index < 0 || index >= len(input.Points) {
				return nil, fmt.Errorf("diffusion: face index %d is outside its points", index)
			}
		}
		first, second, third := input.Points[face[0]], input.Points[face[1]], input.Points[face[2]]
		area := second.Sub(first).Cross(third.Sub(first)).Len() / 6
		if !(area <= math.MaxFloat64) {
			return nil, fmt.Errorf("diffusion: surface area is not finite")
		}
		if area == 0 {
			continue
		}
		for i, from := range face {
			to := face[(i+1)%3]
			length := input.Points[to].Sub(input.Points[from]).Len()
			neighbors[from] = append(neighbors[from], link{to, length})
			neighbors[to] = append(neighbors[to], link{from, length})
			areas[from] += area
		}
	}

	result := make([]SurfaceProbe, len(input.Points))
	distance := make([]float64, len(input.Points))
	seen := make([]int, len(input.Points))
	var queue []int
	limit := 3 * max(input.Radius[0], input.Radius[1], input.Radius[2])
	for root, area := range areas {
		if area == 0 {
			continue
		}
		if !(area <= math.MaxFloat64) {
			return nil, fmt.Errorf("diffusion: accumulated surface area is not finite")
		}
		stamp := root + 1
		queue = append(queue[:0], root)
		seen[root], distance[root] = stamp, 0
		for i := 0; i < len(queue); i++ {
			from := queue[i]
			for _, edge := range neighbors[from] {
				away := distance[from] + edge.length
				if away > limit || (seen[edge.to] == stamp && away >= distance[edge.to]) {
					continue
				}
				distance[edge.to], seen[edge.to] = away, stamp
				queue = append(queue, edge.to)
			}
		}
		var sum, weight [3]float64
		for _, point := range queue {
			if point == root || seen[point] != stamp {
				continue
			}
			seen[point] = -stamp
			incident := input.Incident[point]
			light := [3]float64{incident.R, incident.G, incident.B}
			for channel, radius := range input.Radius {
				widths := distance[point] / radius
				if widths > 3 {
					continue
				}
				w := areas[point] * math.Exp(-.5*widths*widths)
				sum[channel] += w * light[channel]
				weight[channel] += w
			}
		}
		var nearby, share [3]float64
		for channel := range weight {
			total := area + weight[channel]
			if !(total <= math.MaxFloat64 && sum[channel] <= math.MaxFloat64) {
				return nil, fmt.Errorf("diffusion: surface response is not representable")
			}
			nearby[channel] = sum[channel] / total
			share[channel] = weight[channel] / total
		}
		result[root] = SurfaceProbe{
			Nearby: maths.V(nearby[0], nearby[1], nearby[2]),
			Share:  maths.V(share[0], share[1], share[2]),
		}
	}
	return result, nil
}
