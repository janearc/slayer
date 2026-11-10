package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
)

type producer struct {
	BinarySHA256 string `json:"binary-sha256"`
	Revision     string `json:"compiled-revision"`
	Modified     string `json:"compiled-tree-modified"`
	GoVersion    string `json:"go-version"`
}

type receipt struct {
	Card            card          `json:"specimen"`
	Producer        *producer     `json:"producer"`
	Profile         gpu.Profile   `json:"gpu-profile"`
	DurationUnit    string        `json:"duration-unit"`
	Bounds          maths.Box     `json:"physical-bounds-metres"`
	StoredFaces     int           `json:"stored-faces"`
	Seed            uint64        `json:"seed"`
	ImageSHA256     string        `json:"image-sha256"`
	SettingsSHA256  string        `json:"settings-sha256"`
	PreparationWall time.Duration `json:"preparation-wall"`
}

// compiledProducer records the executing bytes, not a nearby Git checkout.
// Missing build stamps remain explicitly unknown, including source exports.
func compiledProducer() (*producer, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	fingerprint, err := hashFile(path)
	if err != nil {
		return nil, err
	}
	made := producer{BinarySHA256: hex.EncodeToString(fingerprint),
		Revision: "unknown", Modified: "unknown", GoVersion: "unknown"}
	if info, available := debug.ReadBuildInfo(); available {
		made.GoVersion = info.GoVersion
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				made.Revision = setting.Value
			case "vcs.modified":
				made.Modified = setting.Value
			}
		}
	}
	return &made, nil
}

// hashFile holds the bytes actually written, rather than a caller's filename.
func hashFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fingerprint := sha256.New()
	if _, err := io.Copy(fingerprint, file); err != nil {
		return nil, err
	}
	return fingerprint.Sum(nil), nil
}
