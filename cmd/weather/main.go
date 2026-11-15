// Command weather runs a planet's weather from still air and draws what
// it makes: its vorticity, and its bands' air as the flow has torn
// them, each a png, latitude and longitude straight across, every so
// many seconds of its time. It is the named scene weather (package
// scene/named), which a job draws through slayer draw.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/scene/named"
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

// run runs the weather as asked, drawing it every so often: the named
// scene weather, its scene seconds the weather's own.
func run(given asked) error {
	if !(given.every > 0) || math.IsInf(given.every, 0) ||
		!(given.seconds >= 0) || math.IsInf(given.seconds, 0) {
		return fmt.Errorf("invalid time: duration %g s, interval %g s",
			given.seconds, given.every)
	}
	made, err := mapper(given)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(given.out, 0o755); err != nil {
		return err
	}
	for at := given.every; at <= given.seconds; at += given.every {
		maps, err := made.At(seconds(at))
		if err != nil {
			return err
		}
		for _, one := range maps {
			if err := save(one.Picture(), given.out, one.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}

// mapper is the named scene weather as the flags ask, on the renderer
// they say, through the cache where one is named, saying where it is on
// the standard output.
func mapper(given asked) (named.Mapper, error) {
	scene, err := named.GetMaps("weather")
	if err != nil {
		return nil, err
	}
	renderer := named.CPU
	if given.onGPU {
		renderer = named.GPU
	}
	ask, err := named.AskMaps(renderer, true, given.cache,
		seconds(given.every), os.Stdout)
	if err != nil {
		return nil, err
	}
	return scene.Build(map[string]any{"across": given.across,
		"rows": given.rows, "limit": given.limit, "radius": given.radius,
		"day": given.day, "drag": given.drag, "relax": given.relax,
		"rate": given.rate, "storm-width": given.wide,
		"strength": given.strength, "bands": given.bands,
		"seed": given.seed, "pace": time.Second}, ask)
}

// seconds is so many of the weather's seconds as scene time, at the
// pace of a second a second.
func seconds(weather float64) time.Duration {
	return time.Duration(weather * float64(time.Second))
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
