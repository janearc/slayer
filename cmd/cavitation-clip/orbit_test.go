package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

const orbitFixture = "camera-orbit:\n  centre: [-1500, 0, -55.25]\n  radius-m: 85\n  azimuth-deg: 160\n  turns: 1\n  elevation-deg: -10\n  elevation-amplitude-deg: 25\n  elevation-cycles: 1\n  surface-up-m: 0\n"

func TestOrbitRealSnapshotsFirstMiddleLast(t *testing.T) {
	root := t.TempDir()
	p := attachedFixture(t, root, "attached-fizzes: [residual]\n"+orbitFixture+"rays: 4\nvertical-view-deg: 55\n")
	if err := generate(p, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	orbit := cameraOrbit{Centre: [3]float64{-1500, 0, -55.25}, Radius: 85, Azimuth: 160, Turns: 1, Elevation: -10, Amplitude: 25, Cycles: 1}
	for _, i := range []int{0, 2, 3} {
		sceneName := fmt.Sprintf("frame-%06d.yaml", i)
		raw, err := os.ReadFile(filepath.Join(root, "output/scenes", sceneName))
		if err != nil {
			t.Fatal(err)
		}
		var doc yaml.Node
		if err = yaml.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		time := float64(i) / 2
		delta := [3]float64{2 * time, 4 * time, .5 * time}
		eye, aim := orbit.points(time, 2, delta)
		for j, keys := range [][]string{{"location", "at"}, {"orientation", "look_at"}} {
			n, err := nodeAt(&doc, append([]string{"scene", "cameras", "underwater-60m"}, keys...)...)
			if err != nil {
				t.Fatal(err)
			}
			want := eye
			if j == 1 {
				want = aim
			}
			for k := range 3 {
				v, _ := strconv.ParseFloat(n.Content[k].Value, 64)
				if math.Abs(v-want[k]) > 1e-10 {
					t.Fatalf("frame%d coordinate%d=%g want%g", i, k, v, want[k])
				}
			}
		}
		jobName := fmt.Sprintf("frame-%06d.yaml", i)
		job, _ := os.ReadFile(filepath.Join(root, "output/jobs", jobName))
		if !bytes.Contains(job, []byte("rays: 4, tall: 55")) {
			t.Fatal("quality controls not authored")
		}
	}
}

func TestOrbitContinuousPeriodAndElevation(t *testing.T) {
	o := cameraOrbit{Radius: 85, Azimuth: 160, Turns: 1, Elevation: -10, Amplitude: 25, Cycles: 1}
	a, _ := o.points(0, 30, [3]float64{})
	b, _ := o.points(30, 30, [3]float64{})
	for k := range 3 {
		if math.Abs(a[k]-b[k]) > 1e-10 {
			t.Fatal("period does not close at true duration")
		}
	}
	for i := 0; i <= 720; i++ {
		eye, _ := o.points(float64(i)/24, 30, [3]float64{})
		r := math.Sqrt(eye[0]*eye[0] + eye[1]*eye[1] + eye[2]*eye[2])
		if math.Abs(r-85) > 1e-10 {
			t.Fatal("orbit radius changed")
		}
	}
	q, _ := o.points(7.5, 30, [3]float64{})
	if math.Abs(q[2]-85*math.Sin(15*math.Pi/180)) > 1e-10 {
		t.Fatal("vertical variation absent")
	}
}

func TestOrbitRefusalsBeforeAnyWrite(t *testing.T) {
	for _, extra := range []string{
		strings.Replace(orbitFixture, "radius-m: 85", "radius-m: 0", 1),
		strings.Replace(orbitFixture, "elevation-deg: -10", "elevation-deg: 80", 1),
		strings.Replace(orbitFixture, "surface-up-m: 0", "surface-up-m: -40", 1),
		strings.Replace(orbitFixture, "turns: 1", "turns: .inf", 1),
		orbitFixture + "rays: 0\n", orbitFixture + "vertical-view-deg: 180\n",
		strings.Replace(orbitFixture, "  turns: 1", "  magic: 1\n  turns: 1", 1),
	} {
		t.Run(extra, func(t *testing.T) {
			root := t.TempDir()
			p := attachedFixture(t, root, extra)
			if err := generate(p, &bytes.Buffer{}); err == nil {
				t.Fatal("invalid orbit accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "output")); !os.IsNotExist(err) {
				t.Fatal("invalid path wrote output")
			}
		})
	}
	root := t.TempDir()
	p := attachedFixture(t, root, orbitFixture)
	raw, _ := os.ReadFile(p)
	halved := bytes.Replace(raw, []byte("camera-tracking-fraction: 1"), []byte("camera-tracking-fraction: 0.5"), 1)
	os.WriteFile(p, halved, 0644)
	if err := generate(p, &bytes.Buffer{}); err == nil {
		t.Fatal("orbit drift accepted")
	}
}

func TestQualityDefaultsPreserveExactGeneratedBytes(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for i, root := range []string{a, b} {
		extra := ""
		if i == 1 {
			extra = "rays: 1\nvertical-view-deg: 80\n"
		}
		p := attachedFixture(t, root, extra)
		if err := generate(p, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, part := range []string{"jobs/frame-000000.yaml", "jobs/frame-000003.yaml", "scenes/frame-000000.yaml", "scenes/frame-000003.yaml", "jobs/encode.yaml"} {
		x, _ := os.ReadFile(filepath.Join(a, "output", part))
		y, _ := os.ReadFile(filepath.Join(b, "output", part))
		if !bytes.Equal(x, y) {
			t.Fatalf("default changed%s", part)
		}
	}
}

func TestSourceClockOffsetLeavesPositionalTrajectoryUnchanged(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for i, root := range []string{a, b} {
		extra := orbitFixture
		if i == 1 {
			extra += "start-seconds: 0.7\n"
		}
		p := attachedFixture(t, root, extra)
		if err := generate(p, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, i := range []int{0, 2, 3} {
		name := fmt.Sprintf("frame-%06d.yaml", i)
		x, _ := os.ReadFile(filepath.Join(a, "output/scenes", name))
		y, _ := os.ReadFile(filepath.Join(b, "output/scenes", name))
		if !bytes.Equal(x, y) {
			t.Fatal("source clock offset changed positional trajectory")
		}
		job, _ := os.ReadFile(filepath.Join(b, "output/jobs", name))
		want := fmt.Sprintf("at: {value: %.9f, unit: s}", .7+float64(i)/2)
		if !bytes.Contains(job, []byte(want)) {
			t.Fatalf("missing source clock %s", want)
		}
	}
}

func TestSourceClockDefaultAndBounds(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for i, root := range []string{a, b} {
		extra := ""
		if i == 1 {
			extra = "start-seconds: 0\n"
		}
		if err := generate(attachedFixture(t, root, extra), &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"jobs/frame-000000.yaml", "jobs/frame-000003.yaml", "jobs/encode.yaml", "scenes/frame-000003.yaml"} {
		x, _ := os.ReadFile(filepath.Join(a, "output", name))
		y, _ := os.ReadFile(filepath.Join(b, "output", name))
		if !bytes.Equal(x, y) {
			t.Fatal("explicit zero changed omitted-clock output")
		}
	}
	for _, bad := range []string{"-0.1", "86400.1", ".inf", ".nan"} {
		root := t.TempDir()
		if err := generate(attachedFixture(t, root, "start-seconds: "+bad+"\n"), &bytes.Buffer{}); err == nil {
			t.Fatal("invalid clock accepted")
		}
		if _, err := os.Stat(filepath.Join(root, "output")); !os.IsNotExist(err) {
			t.Fatal("invalid clock wrote files")
		}
	}
}
