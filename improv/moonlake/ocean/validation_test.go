package ocean

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/oceantex"
)

func TestReadPFMRejectsInvalid(t *testing.T) {
	for _, header := range []string{
		"PF\n0 1\n-1\n", "PF\n-1 1\n-1\n", "PF\n9223372036854775807 2\n-1\n",
		"PF\n1000000000 1000000000\n-1\n", "PF\n1 1\n0\n", "PF\n1 1\nNaN\n",
		"PF\n1 1\n+Inf\n", "PF\n1 1\n-Inf\n", "PF\n1 1\n-1x\n", "PF\n1 1\n-1", "PF\n1 1 extra\n-1\n",
	} {
		t.Run(header, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.pfm")
			if err := os.WriteFile(path, []byte(header), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadPFM(path); err == nil {
				t.Fatal("accepted invalid header")
			}
		})
	}
	for _, length := range []int{0, 11, 13} {
		path := filepath.Join(t.TempDir(), "payload.pfm")
		payload := append([]byte("PF\n1 1\n-1\n"), make([]byte, length)...)
		if err := os.WriteFile(path, payload, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadPFM(path); err == nil {
			t.Fatalf("accepted payload length %d", length)
		}
	}
}

func TestReadPFMSpecialFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := ReadPFM(path); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("accepted FIFO")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked opening FIFO")
	}
	if _, err := ReadPFM(t.TempDir()); err == nil {
		t.Fatal("accepted directory")
	}
}

func TestReadPFMCRLFAndScale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sky.pfm")
	var data bytes.Buffer
	data.WriteString("PF\r\n1 1\r\n-2\r\n")
	if err := binary.Write(&data, binary.LittleEndian, []float32{1, -2, 3}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	image, err := ReadPFM(path)
	if err != nil {
		t.Fatal(err)
	}
	if image.Light[0] != 1 || image.Light[1] != -2 || image.Light[2] != 3 {
		t.Fatalf("changed scale semantics: %v", image.Light)
	}
}

func TestRGBRejectsDimensionsOverLimit(t *testing.T) {
	image := RGB{Width: 22369622, Height: 1}
	if err := image.validate(); err == nil || !strings.Contains(err.Error(), "excessive") {
		t.Fatalf("oversized image error: %v", err)
	}
}

func TestNewSkyRejectsInvalid(t *testing.T) {
	image := RGB{Width: 1, Height: 1, Light: []float32{1, 2, 3}}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Focal = 0 }, func(c *Config) { c.Focal = -1 },
		func(c *Config) { c.Focal = math.Inf(1) }, func(c *Config) { c.LookDown = math.NaN() },
		func(c *Config) { c.LookDown = 90 }, func(c *Config) { c.MoonAltitude = math.Inf(1) },
		func(c *Config) { c.MoonScale = 0 }, func(c *Config) { c.MoonScale = -1 },
		func(c *Config) { c.MoonScale = math.SmallestNonzeroFloat64 }, func(c *Config) { c.MoonScale = math.MaxFloat64 },
		func(c *Config) { c.FrameHeight = -1 }, func(c *Config) { c.EyeOverLake = 0 },
		func(c *Config) { c.Seconds = math.NaN() }, func(c *Config) { c.Wind = -1 },
		func(c *Config) { c.Width = -1 }, func(c *Config) { c.Height = -1 },
	} {
		config := DefaultConfig()
		mutate(&config)
		if _, err := NewSky(config, image, nil); err == nil {
			t.Errorf("accepted config %+v", config)
		}
	}
	for _, value := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 0} {
		invalid := RGB{Width: 1, Height: 1, Light: []float32{value, value, value}}
		if _, err := NewSky(DefaultConfig(), invalid, nil); err == nil {
			t.Errorf("accepted sky %v", value)
		}
		if _, err := NewSky(DefaultConfig(), image, &invalid); err == nil {
			t.Errorf("accepted moon %v", value)
		}
	}
	black := RGB{Width: 1, Height: 1, Light: []float32{0, 0, 0}}
	if _, err := NewSky(DefaultConfig(), black, &image); err != nil {
		t.Fatalf("valid reference must permit black sky: %v", err)
	}
	large := RGB{Width: 1, Height: 1, Light: []float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}}
	if _, err := NewSky(DefaultConfig(), large, nil); err != nil {
		t.Fatalf("finite bright input: %v", err)
	}
}

func TestWritePFMInvalidPreservesFile(t *testing.T) {
	for _, picture := range []*oceantex.Texture{
		nil, {}, {Width: -1, Height: 1}, {Width: 1, Height: 1},
		{Width: 1, Height: 1, Light: make([][3]float64, 2)},
		{Width: int(^uint(0) >> 1), Height: 2},
		{Width: 1, Height: 1, Light: [][3]float64{{math.NaN(), 0, 0}}},
		{Width: 1, Height: 1, Light: [][3]float64{{math.Inf(1), 0, 0}}},
		{Width: 1, Height: 1, Light: [][3]float64{{math.MaxFloat64, 0, 0}}},
	} {
		path := filepath.Join(t.TempDir(), "existing.pfm")
		want := []byte("keep existing data")
		if err := os.WriteFile(path, want, 0600); err != nil {
			t.Fatal(err)
		}
		if err := WritePFM(path, picture); err == nil {
			t.Error("accepted invalid texture")
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("changed existing file: %q %v", got, err)
		}
	}
	picture := &oceantex.Texture{Width: 1, Height: 1, Light: [][3]float64{{1, 2, 3}}}
	if err := WritePFM(t.TempDir(), picture); err == nil {
		t.Fatal("ignored open error")
	}
}
