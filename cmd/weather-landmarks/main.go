// Command weather-landmarks selects body-space weather features before a
// paired-camera observation. It reads an existing checkpoint, never runs it.
package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
	"github.com/janearc/slayer/weather"
)

var build, built, who, sessions string

func main() { os.Exit(tool().Run(os.Args[1:], os.Stdout, os.Stderr)) }

func tool() *pzm.Tool {
	made := pzm.New("weather-landmarks", "select weather features for a camera comparison",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	made.Verb("scan", verb.NoShort, "read a kept weather and select two features", page, "Example_scan", facts, scan)
	return made
}

const page = `# scan

scan SET

Reads SET's first generated flowing body, its cached GPU weather checkpoint,
and its disc and second cameras. Samples spin on a one-degree latitude and
longitude grid, chooses the strongest two points at least twenty degrees
apart and comfortably visible and sunlit from both cameras, then independently
projects their positions with pinhole geometry. Writes nothing. The selected
points are candidates, not proof of visible or coherent structures.
`

type candidate struct {
	lat, lon, spin, origin float64
	normal                 maths.Vec
	xy                     [2][2]float64
}

func scan(call pzm.Call) error {
	args := call.Args()
	if len(args) != 1 {
		return fmt.Errorf("scan: say one SET")
	}
	source, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	read, err := set.Read(args[0], source)
	if err != nil {
		return err
	}
	world, has := read.World()
	if !has || world.Unit() != 1 || len(world.Bodies()) != 1 {
		return fmt.Errorf("scan: requires one body in a metre world")
	}
	each := world.Bodies()[0]
	input, generated := each.Generated()
	optical, flowing := each.Flowing()
	if !generated || !flowing || each.Circles() != "" {
		return fmt.Errorf("scan: needs one fixed generated flowing body")
	}
	derived, err := generate.Derive(input)
	if err != nil {
		return err
	}
	radius := derived.Radius.InKilometres() * 1000
	written, has := optical.Weather()
	if !has {
		return fmt.Errorf("scan: no weather")
	}
	cells := written.Grid()
	grid, err := weather.NewGrid(cells, cells/2, 80*math.Pi/180, radius)
	if err != nil {
		return err
	}
	rate, wide, strength := written.Storms()
	stir, err := weather.NewStirring(rate, wide, strength)
	if err != nil {
		return err
	}
	day, drag, relax := written.Turning()
	sky, err := weather.New(grid, day, drag, relax, stir, written.Seed())
	if err != nil {
		return err
	}
	cache, err := weather.CacheDir(os.LookupEnv)
	if err != nil {
		return err
	}
	moment, _ := written.Moment()
	if err := sky.LoadReached(cache, moment, true); err != nil {
		return fmt.Errorf("scan: kept GPU checkpoint: %w", err)
	}
	first, err := read.Shot("disc", nil)
	if err != nil {
		return err
	}
	second, err := read.Shot("second", nil)
	if err != nil {
		return err
	}
	eyes := [2]maths.Vec{first.Eye().Origin.Sub(each.Middle()), second.Eye().Origin.Sub(each.Middle())}
	film, ok := read.Film()
	if !ok {
		return fmt.Errorf("scan: no film")
	}
	width, height, _ := film.Size()
	if width != height || width <= 0 || film.Wide() <= 0 {
		return fmt.Errorf("scan: needs a square film with a stated horizontal field of view")
	}
	sun, ok := read.Sunlit()
	if !ok {
		return fmt.Errorf("scan: no scene sun")
	}
	light := stage.Skyward(sun.Where())
	tilt := input.Tilt.InRadians()
	if override, has := each.Tilt(); has {
		tilt = override * maths.Degree
	}
	rotation := maths.Rotation(maths.XAxis, tilt)
	pole, prime := rotation.Out(maths.Up), rotation.Out(maths.ZAxis)
	east := pole.Cross(prime).Unit()
	focal := float64(width) / (2 * math.Tan(film.Wide()*maths.Degree/2))
	var found []candidate
	for lat := -60; lat <= 60; lat++ {
		for lon := 0; lon < 360; lon++ {
			latitude, longitude := float64(lat)*maths.Degree, float64(lon)*maths.Degree
			n := prime.Scale(math.Cos(latitude) * math.Cos(longitude)).Add(east.Scale(math.Cos(latitude) * math.Sin(longitude))).Add(pole.Scale(math.Sin(latitude)))
			if n.Dot(light) < 0.35 {
				continue
			}
			point := n.Scale(radius)
			var xy [2][2]float64
			visible := true
			for i, eye := range eyes {
				if n.Dot(eye.Sub(point).Unit()) < 0.4 {
					visible = false
					break
				}
				forward := eye.Neg().Unit()
				right := forward.Cross(maths.Up).Unit()
				up := right.Cross(forward).Unit()
				delta := point.Sub(eye)
				depth := delta.Dot(forward)
				if depth <= 0 {
					visible = false
					break
				}
				xy[i] = [2]float64{float64(width)/2 + focal*delta.Dot(right)/depth, float64(height)/2 - focal*delta.Dot(up)/depth}
				if xy[i][0] < 20 || xy[i][0] > float64(width-20) || xy[i][1] < 20 || xy[i][1] > float64(height-20) {
					visible = false
					break
				}
			}
			if !visible {
				continue
			}
			origin, spin, _ := sky.At(latitude, longitude)
			found = append(found, candidate{latitude, longitude, spin, origin, n, xy})
		}
	}
	sort.Slice(found, func(i, j int) bool { return math.Abs(found[i].spin) > math.Abs(found[j].spin) })
	if len(found) == 0 {
		return fmt.Errorf("scan: no shared sunlit candidates")
	}
	chosen := []candidate{found[0]}
	for _, item := range found[1:] {
		if item.normal.Dot(chosen[0].normal) < math.Cos(20*maths.Degree) {
			chosen = append(chosen, item)
			break
		}
	}
	if len(chosen) < 2 {
		return fmt.Errorf("scan: fewer than two separated candidates")
	}
	fmt.Fprintf(call.Out(), "body=%s radius_m=%.17g cells=%d seed=%d requested_s=%.17g checkpoint_s=%.17g field=GPU\n", each.Name(), radius, cells, written.Seed(), moment, sky.Clock())
	for i, item := range chosen {
		fmt.Fprintf(call.Out(), "candidate=%d lat_deg=%.9g lon_deg=%.9g spin=%.17g origin_deg=%.17g disc_px=(%.3f,%.3f) second_px=(%.3f,%.3f)\n", i+1, item.lat/maths.Degree, item.lon/maths.Degree, item.spin, item.origin/maths.Degree, item.xy[0][0], item.xy[0][1], item.xy[1][0], item.xy[1][1])
	}
	return nil
}
