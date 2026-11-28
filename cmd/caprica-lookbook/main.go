// Command caprica-lookbook lays out twelve completed local renders from a
// manifest. It does not render or publish the shared gallery.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

type source struct{ Title, URL string }
type frame struct {
	Title, Caption, Directory, Image, Raw string
	ImageLink, RawLink                    string `json:"-"`
}
type book struct {
	Title, Introduction, ArtDirection string
	Sources                           []source
	Frames                            []frame
}

func main() { os.Exit(tool().Run(os.Args[1:], os.Stdout, os.Stderr)) }

func tool() *pzm.Tool {
	t := pzm.New("caprica-lookbook", "arrange twelve completed local views",
		pzm.Build(""), pzm.Built(""), pzm.Who(""), pzm.Sessions(""))
	facts := verb.NewFacts(verb.WritesTo("the specified HTML page"), verb.Changes,
		verb.ThisMachine, verb.NoOne, verb.Light)
	t.Verb("make", verb.NoShort, "make a local look book", "# make\n\nRead a JSON manifest of twelve completed image and raw pairs; write an HTML page with relative links.\n", "Example_make", facts, makeBook,
		pzm.Value("manifest", verb.NoShort, "JSON manifest with twelve views"),
		pzm.Value("out", verb.NoShort, "HTML page to write"))
	return t
}

func makeBook(call pzm.Call) error {
	if len(call.Args()) != 0 || !call.Given("manifest") || !call.Given("out") {
		return fmt.Errorf("lookbook: make needs --manifest and --out")
	}
	manifest, err := filepath.Abs(call.Value("manifest"))
	if err != nil {
		return err
	}
	out, err := filepath.Abs(call.Value("out"))
	if err != nil {
		return err
	}
	if manifest == out {
		return fmt.Errorf("lookbook: output must not replace the manifest")
	}
	input, err := os.Open(manifest)
	if err != nil {
		return err
	}
	defer input.Close()
	var data book
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("lookbook: %w", err)
	}
	if strings.TrimSpace(data.Title) == "" || strings.TrimSpace(data.Introduction) == "" || strings.TrimSpace(data.ArtDirection) == "" || len(data.Frames) != 12 || len(data.Sources) == 0 {
		return fmt.Errorf("lookbook: title, introduction, art direction, sources and exactly twelve frames are required")
	}
	for i := range data.Sources {
		s := data.Sources[i]
		u, err := url.Parse(s.URL)
		if strings.TrimSpace(s.Title) == "" || err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("lookbook: source %d needs a title and HTTPS URL", i+1)
		}
	}
	for i := range data.Frames {
		f := &data.Frames[i]
		if strings.TrimSpace(f.Title) == "" || strings.TrimSpace(f.Caption) == "" || f.Directory == "" || f.Image == "" || f.Raw == "" {
			return fmt.Errorf("lookbook: frame %d needs title, caption, directory, image and raw", i+1)
		}
		dir := f.Directory
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(manifest), dir)
		}
		if !filepath.IsLocal(f.Image) || !filepath.IsLocal(f.Raw) {
			return fmt.Errorf("lookbook: frame %d image and raw must be paths within its directory", i+1)
		}
		f.ImageLink, err = localLink(out, filepath.Join(dir, f.Image))
		if err != nil {
			return fmt.Errorf("lookbook: frame %d image: %w", i+1, err)
		}
		f.RawLink, err = localLink(out, filepath.Join(dir, f.Raw))
		if err != nil {
			return fmt.Errorf("lookbook: frame %d raw: %w", i+1, err)
		}
	}
	if call.DryRun() {
		_, err = fmt.Fprintf(call.Out(), "would write twelve-view look book to %s\n", out)
		return err
	}
	var page bytes.Buffer
	if err := layout.Execute(&page, data); err != nil {
		return err
	}
	if err := os.WriteFile(out, page.Bytes(), 0o644); err != nil {
		return err
	}
	_, err = fmt.Fprintf(call.Out(), "wrote %s\n", out)
	return err
}

func localLink(out, file string) (string, error) {
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", file)
	}
	absolute, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(filepath.Dir(out), absolute)
	if err != nil {
		return "", err
	}
	return (&url.URL{Path: filepath.ToSlash(relative)}).String(), nil
}

var layout = template.Must(template.New("lookbook").Funcs(template.FuncMap{"indexNumber": func(i int) int { return i + 1 }}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}}</title><style>
:root{color-scheme:dark;--ink:#f0ede5;--muted:#bbb8af;--line:#41434a;--accent:#e1b995}
*{box-sizing:border-box}body{margin:0;background:#101217;color:var(--ink);font:16px/1.55 system-ui,-apple-system,sans-serif}
main{max-width:1500px;margin:auto;padding:clamp(24px,5vw,72px)}header{max-width:780px;margin:3rem 0 5rem}
.eyebrow{text-transform:uppercase;letter-spacing:.2em;color:var(--accent);font-size:.72rem;font-weight:700}
h1{font:normal clamp(3.3rem,8vw,7rem)/.98 Georgia,serif;letter-spacing:-.04em;margin:.5rem 0 1.8rem}
header p{color:var(--muted);font-size:1.1rem} .grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:4rem 2rem}
figure{margin:0}figure>a{display:block;background:#1b1d23}img{display:block;width:100%;height:auto}
figcaption{border-top:1px solid var(--line);padding-top:1rem;margin-top:1rem}.number{color:var(--accent);font-size:.75rem;letter-spacing:.12em}
h2{font:normal 1.7rem/1.2 Georgia,serif;margin:.35rem 0 .5rem}figcaption p{color:var(--muted);margin:.4rem 0 1rem}
a{color:inherit}a:hover{color:var(--accent)}.links{display:flex;gap:1.4rem;font-size:.85rem}.notes{border-top:1px solid var(--line);margin-top:5rem;padding-top:2rem;max-width:850px;color:var(--muted)}
@media(max-width:760px){main{padding:24px}header{margin:2rem 0 3rem}.grid{grid-template-columns:1fr;gap:3rem}}
</style></head><body><main><header><div class="eyebrow">A system in twelve perspectives</div><h1>{{.Title}}</h1><p>{{.Introduction}}</p></header>
<section class="grid" aria-label="Twelve views">{{range $i, $f := .Frames}}<figure><a href="{{$f.ImageLink}}"><img src="{{$f.ImageLink}}" alt="{{$f.Title}}" loading="lazy"></a><figcaption><span class="number">VIEW {{printf "%02d" (indexNumber $i)}}</span><h2>{{$f.Title}}</h2><p>{{$f.Caption}}</p><div class="links"><a href="{{$f.ImageLink}}">Full image ↗</a><a href="{{$f.RawLink}}">Raw light ↗</a></div></figcaption></figure>{{end}}</section>
<section class="notes"><h2>Art direction</h2><p>{{.ArtDirection}}</p>{{if .Sources}}<h2>Sources</h2><ul>{{range .Sources}}<li><a href="{{.URL}}">{{.Title}}</a></li>{{end}}</ul>{{end}}</section></main></body></html>`))
