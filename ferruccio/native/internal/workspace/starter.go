package workspace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// starterReceipt binds a real pre-rendered view to its exact editable source.
type starterReceipt struct {
	SourceSHA256   string `json:"source_sha256" yaml:"source_sha256"`
	ImageSHA256    string `json:"image_sha256" yaml:"image_sha256"`
	ProducerSHA256 string `json:"producer_sha256" yaml:"producer_sha256"`
	Width          int    `json:"width" yaml:"width"`
	Height         int    `json:"height" yaml:"height"`
}

// openStarter gives the opening world real pixels before staging.
func (work *Workspace) openStarter(directory string) error {
	raw, err := os.ReadFile(filepath.Join(directory, "receipt.json"))
	if err != nil {
		return err
	}
	var receipt starterReceipt
	if err := yamlfile.Read("starting view receipt", raw, &receipt); err != nil {
		return err
	}
	source := sha256.Sum256([]byte(work.state.document.Text()))
	if receipt.SourceSHA256 != hex.EncodeToString(source[:]) || receipt.ProducerSHA256 == "" {
		return fmt.Errorf("starting view does not match the editable scene")
	}
	picturePath := filepath.Join(directory, "preview.png")
	info, err := os.Stat(picturePath)
	if err != nil {
		return err
	}
	if info.Size() > 16*1024*1024 {
		return fmt.Errorf("starting picture exceeds its size budget")
	}
	pixels, err := os.ReadFile(picturePath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(pixels)
	if receipt.ImageSHA256 != hex.EncodeToString(hash[:]) {
		return fmt.Errorf("starting picture differs from its receipt")
	}
	config, err := png.DecodeConfig(bytes.NewReader(pixels))
	if err != nil {
		return err
	}
	if config.Width != receipt.Width || config.Height != receipt.Height || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 {
		return fmt.Errorf("starting picture has unexpected dimensions")
	}
	if err := work.showImage(picturePath); err != nil {
		return err
	}
	work.play.renderedText = work.state.document.Text()
	work.output = picturePath
	work.acceptedGeneration, work.acceptedPath, work.acceptedText = work.generation, work.state.path, work.state.document.Text()
	work.autoPreview = false
	work.status = "Larson playground · ready to explore"
	work.caption = "Pre-rendered starting view · " + chooseWorkingView
	return nil
}
