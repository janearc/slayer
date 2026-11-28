// Command modulezip verifies Go module archive content against build sums.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

func main() {
	if err := verify(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify() error {
	rows := bufio.NewScanner(os.Stdin)
	for rows.Scan() {
		checksum, archive, ok := strings.Cut(rows.Text(), "|")
		if !ok || checksum == "" || archive == "" {
			return fmt.Errorf("modulezip: invalid archive verification row")
		}
		actual, err := dirhash.HashZip(archive, dirhash.Hash1)
		if err != nil {
			return fmt.Errorf("modulezip: %w", err)
		}
		if actual != checksum {
			return fmt.Errorf("modulezip: checksum mismatch for %s", archive)
		}
	}
	return rows.Err()
}
