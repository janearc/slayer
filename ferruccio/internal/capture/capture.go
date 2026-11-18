// Package capture writes a new screenshot and its provenance together.
package capture

import (
	"encoding/json"
	"image"
	"image/png"
	"io"
	"os"
)

// Write refuses to replace either an existing image or an existing receipt.
func Write(path string, picture image.Image, receipt any) error {
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := exclusive(path+".json", func(writer io.Writer) error { _, err := writer.Write(raw); return err }); err != nil {
		return err
	}
	if err := exclusive(path, func(writer io.Writer) error { return png.Encode(writer, picture) }); err != nil {
		os.Remove(path + ".json")
		return err
	}
	return nil
}

// exclusive removes only its newly created file when writing cannot finish.
func exclusive(path string, write func(io.Writer) error) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := write(file); err != nil {
		file.Close()
		os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}
