package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"os"
	"path/filepath"
	"slices"

	"github.com/janearc/slayer/internal/appicon"
)

//go:embed page.html
var pageRaw string

//go:embed page.css
var pageCSS []byte

// Groups supplies a stable filter list without changing the card order.
//
//ninja:vow waste html/template resolves this exported method by name
func (book book) Groups() []string {
	seen := map[string]bool{}
	var groups []string
	for _, card := range book.Cards {
		if !seen[card.Entry.Group] {
			groups = append(groups, card.Entry.Group)
			seen[card.Entry.Group] = true
		}
	}
	slices.Sort(groups)
	return groups
}

// write keeps pending and failed entries visible when a run stops early.
func (book book) write(directory string) error {
	page, err := template.New("catalogue").Parse(pageRaw)
	if err != nil {
		return err
	}
	var output bytes.Buffer
	if err := page.Execute(&output, book); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "index.html"),
		output.Bytes(), artifactMode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "catalogue.css"),
		pageCSS, artifactMode); err != nil {
		return err
	}
	return writeJSON(filepath.Join(directory, "catalogue.json"), book)
}

// writeIcons keeps the book's artwork independent of the installed binary.
func writeIcons(directory string) error {
	for _, dark := range []bool{false, true} {
		name := "icon-light.png"
		if dark {
			name = "icon-dark.png"
		}
		raw, err := appicon.PNG(dark)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(directory, name),
			raw, artifactMode); err != nil {
			return err
		}
	}
	return nil
}
