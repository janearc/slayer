//go:build draw

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWeatherCacheAndGPUReachTheSameOutputClock(t *testing.T) {
	for _, gpu := range []bool{false, true} {
		for _, cached := range []bool{false, true} {
			t.Run(fmt.Sprintf("gpu%t-cache%t", gpu, cached), func(t *testing.T) {
				dir := t.TempDir()
				args := tinyWeather(dir)
				if gpu {
					args = append(args, "-gpu")
				}
				if cached {
					args = append(args, "-cache", t.TempDir())
				}
				out, err := weatherCommand(t, args...)
				if err != nil || !strings.Contains(out, "3600 s:") {
					t.Fatalf("%v %q", err, out)
				}
				if _, err := os.Stat(filepath.Join(dir, "spun-00003600.png")); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
