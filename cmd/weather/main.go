// Command weather runs a planet's weather from still air and draws what
// it makes: its vorticity, and its bands' air as the flow has torn
// them, each a png, latitude and longitude straight across, every so
// many seconds of its time.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/weather"
)

// asked is what the command was asked to run.
type asked struct {
	across, rows                int
	limit, radius, day, drag    float64
	relax, rate, wide, strength float64
	seconds, every, bands       float64
	seed                        uint64
	out, cache                  string
	onGPU                       bool
}

// main runs the weather as the flags ask.
func main() {
	given := asked{}
	flags := flag.NewFlagSet("weather", flag.ExitOnError)
	flags.IntVar(&given.across, "across", 512, "cells round the circle")
	flags.IntVar(&given.rows, "rows", 256, "rows of latitude")
	flags.Float64Var(&given.limit, "limit", 80, "the walls, degrees")
	flags.Float64Var(&given.radius, "radius", 6e7, "metres")
	flags.Float64Var(&given.day, "day", 18000, "seconds the planet turns in")
	flags.Float64Var(&given.drag, "drag", 1e6, "friction's time, seconds")
	flags.Float64Var(&given.relax, "relax", 2e6, "the bands' time, seconds")
	flags.Float64Var(&given.rate, "rate", 3.8e-17,
		"storms a square metre a second")
	flags.Float64Var(&given.wide, "wide", 1.3e6, "a storm's width, metres")
	flags.Float64Var(&given.strength, "strength", 1.5e-5,
		"a storm's spin, radians a second")
	flags.Float64Var(&given.seconds, "seconds", 3e6, "how long to run")
	flags.Float64Var(&given.every, "every", 1e6,
		"draw a picture every so many seconds")
	flags.Float64Var(&given.bands, "bands", 15, "a stripe's width, degrees")
	flags.Uint64Var(&given.seed, "seed", 7, "the storms' seed")
	flags.StringVar(&given.out, "out", "weather", "the pictures' directory")
	flags.BoolVar(&given.onGPU, "gpu", false, "run on the gpu, in float32")
	flags.StringVar(&given.cache, "cache", "",
		"reach each picture's moment through the weather cache here, "+
			"as hillside does, so a shot after reads it")
	_ = flags.Parse(os.Args[1:])
	if err := run(given); err != nil {
		fmt.Fprintln(os.Stderr, "weather:", err)
		os.Exit(1)
	}
}

// run runs the weather as asked, drawing it every so often.
func run(given asked) error {
	if !(given.every > 0) || math.IsInf(given.every, 0) ||
		!(given.seconds >= 0) || math.IsInf(given.seconds, 0) {
		return fmt.Errorf("invalid time: duration %g s, interval %g s",
			given.seconds, given.every)
	}
	degree := math.Pi / 180
	grid, err := weather.NewGrid(given.across, given.rows,
		given.limit*degree, given.radius)
	if err != nil {
		return err
	}
	stirring, err := weather.NewStirring(given.rate, given.wide,
		given.strength)
	if err != nil {
		return err
	}
	made, err := weather.New(grid, given.day, given.drag, given.relax,
		stirring, given.seed)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(given.out, 0o755); err != nil {
		return err
	}
	for at := given.every; at <= given.seconds; at += given.every {
		steps, err := reach(made, given, at)
		if err != nil {
			return err
		}
		fmt.Printf("%.0f s: %d steps, fastest %.1f m/s, jets hold %.2f "+
			"of the energy (jupiter 0.93)\n", made.Clock(), steps,
			made.Fastest(), made.Zonality())
		zonal := made.Zonal()
		for j := 0; j < len(zonal); j += max(1, len(zonal)/32) {
			fmt.Printf(" %+.0f", zonal[j])
		}
		fmt.Println(" (mean east wind, m/s, south to north)")
		if err := draw(made, given, degree); err != nil {
			return err
		}
	}
	return nil
}

// reach carries the weather to a moment as asked: through the cache, or
// run straight, on the gpu or the cpu.
func reach(made *weather.Weather, given asked, at float64) (int, error) {
	switch {
	case given.cache != "":
		return made.Reach(given.cache, at, given.every, given.onGPU)
	case given.onGPU:
		return made.RunOnGPU(at)
	}
	return made.Run(at), nil
}

// cooking is how long air takes to cook, for the age picture, in
// seconds: melancholia's.
const cooking = 1e6

// draw saves the weather's three pictures, named by its clock.
func draw(made *weather.Weather, given asked, degree float64) error {
	name := fmt.Sprintf("%08.0f", made.Clock())
	for prefix, picture := range map[string]image.Image{
		"spun-":  made.Spun(),
		"bands-": made.Bands(given.bands * degree),
		"aged-":  made.Aged(cooking),
	} {
		if err := save(picture, given.out, prefix+name); err != nil {
			return err
		}
	}
	return nil
}

// save writes a picture as a png in a directory, by a name.
func save(picture image.Image, dir, name string) error {
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, picture)
}
