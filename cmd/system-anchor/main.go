// Command system-anchor reports the authored binary's positions at its
// selected still time. It does not advance or alter the scene.
package main

import (
	"fmt"
	"os"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

var build, built, who, sessions string

func main() { os.Exit(tool().Run(os.Args[1:], os.Stdout, os.Stderr)) }

func tool() *pzm.Tool {
	made := pzm.New("system-anchor", "report a binary's still positions",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	made.Verb("locate", verb.NoShort, "report the first binary's authored still positions", page, "Example_locate", facts, locate)
	return made
}

const page = `# locate

locate SET

Read SET and report its first binary's primary, companion and drop in world
coordinates, and any generated drop-orbiting bodies in map coordinates,
at its authored time. Positions use the set's world units; this does not
construct a new orbit or claim dynamical stability.
`

func locate(call pzm.Call) error {
	args := call.Args()
	if len(args) != 1 {
		return fmt.Errorf("locate: say one SET")
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
	if !has || world.Unit() <= 0 || len(world.Binaries()) == 0 {
		return fmt.Errorf("locate: needs a world with a binary")
	}
	pair, middle, err := world.Binaries()[0].Placed(world.Unit())
	if err != nil {
		return err
	}
	primary, companion := middle.Add(pair.Primary.At.Scale(1/world.Unit())), middle.Add(pair.Companion.At.Scale(1/world.Unit()))
	fmt.Fprintf(call.Out(), "primary_world=%+.12g,%+.12g,%+.12g\n", primary.X, primary.Y, primary.Z)
	fmt.Fprintf(call.Out(), "companion_world=%+.12g,%+.12g,%+.12g\n", companion.X, companion.Y, companion.Z)
	if pair.Drop != nil {
		drop := middle.Add(pair.Drop.At.Scale(1 / world.Unit()))
		fmt.Fprintf(call.Out(), "drop_world=%+.12g,%+.12g,%+.12g\n", drop.X, drop.Y, drop.Z)
		for _, body := range world.Bodies() {
			if body.Circles() == "" {
				continue
			}
			planet, ok := body.Generated()
			if !ok {
				continue
			}
			planet.Sun = generate.Sun{Mass: pair.Drop.Shine.Mass, Luminosity: pair.Drop.Shine.Luminosity.InSolarLuminosities()}
			distance, err := planet.Distance()
			if err != nil {
				return err
			}
			orbit := thing.Orbit{Centre: drop, Radius: distance.InMetres() / world.Unit(), Phase: body.Phase() * maths.Degree}
			at := orbit.At(0).Origin
			fmt.Fprintf(call.Out(), "body=%s map_east_north_up=%+.12g,%+.12g,%+.12g\n", body.Name(), at.X, -at.Z, at.Y)
		}
	}
	fmt.Fprintf(call.Out(), "primary_luminosity_solar=%g\n", pair.Primary.Shine.Luminosity.InSolarLuminosities())
	return nil
}
