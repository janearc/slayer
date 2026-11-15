package beats

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// CheckAudio binds an edited grid to the original encoded audio bytes.
// Paths may move; a grid from another recording or edit must be refused.
func (grid *Grid) CheckAudio(path string) error {
	if grid == nil || grid.Score.SHA256 == "" {
		return fmt.Errorf("beats: supplied score needs its audio SHA256")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return err
	}
	if fmt.Sprintf("%x", sum.Sum(nil)) != grid.Score.SHA256 {
		return fmt.Errorf("beats: score belongs to different audio bytes")
	}
	return nil
}
