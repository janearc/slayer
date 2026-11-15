package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/compare"
)

var build, built, who, sessions string

// main runs the omnipositor tool on the command line, with stretchArgs first
// letting the stretch verb be left out, and exits with the status it returns.
func main() {
	os.Exit(tool().Run(stretchArgs(os.Args[1:]), os.Stdout, os.Stderr))
}

// tool builds the omnipositor command: the compare verb, then the verbs that
// declareStretch and declareScene add. The version fields come from the
// variables set at link time.
func tool() *pzm.Tool {
	made := pzm.New("omnipositor", "composite stretches and compare rendered images",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("an optional new difference PNG"),
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	made.Verb("compare", 'c', "compare ORACLE and CANDIDATE PNG or PFM files",
		comparePage, "Example_compare", facts, compareFiles,
		pzm.Value("difference", verb.NoShort, "new contrast-stretched difference PNG"),
		pzm.Switch("check", verb.NoShort, "require 99.9% within one code value and PSNR at least 45 dB; PNG only"))
	declareStretch(made)
	declareScene(made)
	return made
}

const comparePage = `# compare

compare ORACLE CANDIDATE [--difference PATH] [--check]

Both inputs must have the same type: .png or colour .pfm. Reports image
size, pixel count, per-channel maximum absolute differences, fraction of
pixels within one code value, PSNR, and all 256 histogram bins counting
RGB channel samples. PFM maxima include pixel coordinates; its within-one
tolerance is peak/255, PSNR uses the peak absolute input sample, and its
histogram bins are contrast-stretched relative to the largest difference.

--difference writes a new contrast-stretched PNG, never
replacing an existing file. --check is PNG-only and fails below 99.9%
of pixels within one code value in every RGB channel or below 45 dB PSNR.
--dry-run reports the plan without reading images or writing files.
`

// compareFiles is the compare verb. It takes two files that are both PNG or
// both PFM, prints their size, pixel count and largest differences, then the
// metrics. It writes a new difference PNG if asked, and with check it fails
// below 99.9% within one code value or below 45 dB PSNR; check is PNG only.
// The difference file is written before the check is judged.
func compareFiles(call pzm.Call) error {
	args := call.Args()
	if len(args) != 2 {
		return errors.New("compare: name ORACLE and CANDIDATE PNG or PFM files")
	}
	kind := strings.ToLower(filepath.Ext(args[0]))
	if (kind != ".png" && kind != ".pfm") || kind != strings.ToLower(filepath.Ext(args[1])) {
		return errors.New("compare: ORACLE and CANDIDATE must both be PNG or both be PFM")
	}
	if call.On("check") && kind != ".png" {
		return errors.New("compare: --check is PNG only")
	}
	destination := call.Value("difference")
	if call.Given("difference") && destination == "" {
		return errors.New("compare: difference path must not be empty")
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "would compare %q against %q; difference PNG: %q; check: %t\n", args[0], args[1], destination, call.On("check"))
		return err
	}
	var difference image.Image
	var fraction, psnr float64
	var histogram [256]uint64
	if kind == ".png" {
		result, err := compare.ComparePNG(args[0], args[1], "")
		if err != nil {
			return err
		}
		difference, fraction, psnr, histogram = result.Difference, result.FractionWithinOne, result.PSNR, result.Histogram
		if _, err := fmt.Fprintf(call.Out(), "size: %dx%d\npixels: %d\nmax difference RGB: %v\n", result.Width, result.Height, result.Pixels, result.MaxDifference); err != nil {
			return err
		}
	} else {
		result, err := compare.CompareRaw(args[0], args[1])
		if err != nil {
			return err
		}
		difference, fraction, psnr, histogram = result.Difference, result.FractionWithinOne, result.PSNR, result.Histogram
		if _, err := fmt.Fprintf(call.Out(), "size: %dx%d\npixels: %d\nmax difference RGB: %v\nmax at RGB: %v\n", result.Width, result.Height, result.Pixels, result.MaxDifference, result.MaxAt); err != nil {
			return err
		}
	}
	if err := writeMetrics(call.Out(), fraction, psnr, histogram); err != nil {
		return err
	}
	if destination != "" {
		if err := writeDifference(destination, difference); err != nil {
			return err
		}
	}
	if call.On("check") && (fraction < 0.999 || psnr < 45) {
		return errors.New("compare: check failed: require at least 99.9% within one code value and PSNR at least 45 dB")
	}
	return nil
}

// writeMetrics writes the fraction within one code value, as a fraction and a
// percentage, then the PSNR in dB, then the 256 histogram bins one per line.
func writeMetrics(out io.Writer, fraction, psnr float64, histogram [256]uint64) error {
	if _, err := fmt.Fprintf(out, "fraction within one: %.9f (%.6f%%)\nPSNR: %.9g dB\nhistogram (absolute RGB channel differences; PFM contrast-stretched):\n", fraction, fraction*100, psnr); err != nil {
		return err
	}
	for bin, count := range histogram {
		if _, err := fmt.Fprintf(out, "%d: %d\n", bin, count); err != nil {
			return err
		}
	}
	return nil
}

// writeDifference writes img as a PNG to a new file at path with mode 0600. It
// refuses a path that exists, and removes the file if writing it fails.
func writeDifference(path string, img image.Image) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("compare: create difference PNG: %w", err)
	}
	encodeErr := png.Encode(file, img)
	closeErr := file.Close()
	if err := errors.Join(encodeErr, closeErr); err != nil {
		return fmt.Errorf("compare: write difference PNG: %w", errors.Join(err, os.Remove(path)))
	}
	return nil
}
