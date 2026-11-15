package audition

// pageHTML is the page, with the rows filled in by the template.
const pageHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Tune library</title>
<style>{{.Style}}</style>
</head>
<body>
<header>
<h1>Tune library</h1>
<p>{{.Count}} short tunes for videos, all chip music because that is what
the synth makes. Play one, then keep it or toss it. Press j to go to the
next tune you have not voted on, and it starts playing.</p>
<p id="counts" aria-live="polite"></p>
<p id="where"></p>
<p><button id="copy" type="button" hidden>Copy my votes as text</button>
<span id="copied" aria-live="polite"></span></p>
</header>
<main>
<section id="queue"><h2>Your queue</h2>
<ol id="queue-list"></ol><p id="queue-empty">Nothing kept yet.</p></section>
{{range .Groups}}
<section class="group"><h2>{{.Name}}</h2>
{{range .Rows}}
<article class="tune" id="tune-{{.Name}}" data-name="{{.Name}}" tabindex="0">
<h3>{{.Name}} <small>{{.Length}}, {{.Tempo}}</small></h3>
<p class="state" role="status"></p>
<video controls preload="metadata" src="{{.Name}}.mp4"></video>
<p><strong>{{.Mood}}.</strong> {{.Feeling}}</p>
<p>For: {{.Intent}}</p>
<p>Listen for: {{.Listen}}</p>
<div class="line" aria-hidden="true">{{range .Parts}}<span style="width:{{.Share}}%"></span>{{end}}</div>
<ul class="parts">{{range .Parts}}
<li><button type="button" class="seek" data-at="{{.Seconds}}">{{.At}} {{.Name}}</button> {{.Feels}}</li>{{end}}
</ul>
<p class="votes"><button type="button" class="keep">&#128077; Keep</button>
<button type="button" class="toss">&#128078; Toss</button></p>
</article>
{{end}}</section>
{{end}}
<section id="tossed-group"><details><summary>Tossed (<span id="tossed-count">0</span>)</summary>
<ul id="tossed-list"></ul></details></section>
</main>
<script>{{.Script}}</script>
</body>
</html>
`

// pageStyle is the page's look. Dark ground, light ink in the house
// contrast band, 24 pixel body text, nothing under 16, sizes in rem.
const pageStyle = `
:root{--ground:#14131a;--ink:#c4c1cf;--card:#1d1b26;--line:#46435a;
--keep:#2d5a3f;--toss:#6a2f3d;--focus:#fff0c8}
html{font-size:16px}
body{margin:0 auto;max-width:56rem;padding:0 1rem 6rem;background:var(--ground);
color:var(--ink);font:1.5rem/1.5 system-ui,-apple-system,"Segoe UI",sans-serif}
h1{font-size:2.5rem;margin:1.5rem 0 .5rem}
h2{font-size:2rem;margin:3rem 0 1rem;border-bottom:1px solid var(--line)}
h3{font-size:1.75rem;margin:0 0 .5rem}
small{font-size:1rem;font-weight:400}
p,ul,ol{margin:.5rem 0}
button{font:inherit;font-size:1.25rem;color:var(--ink);background:var(--card);
border:2px solid var(--line);border-radius:.5rem;padding:.5rem 1rem;cursor:pointer}
button:focus-visible,.tune:focus-visible{outline:3px solid var(--focus);outline-offset:2px}
.tune{background:var(--card);border:2px solid var(--line);border-radius:.75rem;
padding:1rem 1.5rem;margin:1.5rem 0}
.tune.kept{border-color:#5fbf86}.tune.tossed{display:none}
.state{font-size:1.125rem;min-height:1.5rem;margin:0}
.tune.kept .state::before{content:"Kept: in your queue"}
video{width:100%;max-height:22rem;background:#000;border-radius:.5rem}
.line{display:flex;height:.75rem;gap:2px;margin:1rem 0 .5rem}
.line span{background:var(--line);border-radius:2px}
.line span:nth-child(odd){background:#8a87a8}
.parts{list-style:none;padding:0;font-size:1.25rem}
.parts li{margin:.5rem 0}
.seek{font-size:1.125rem;padding:.25rem .75rem;margin-right:.5rem}
.votes{display:flex;gap:4rem;margin-top:1.5rem}
.votes button{font-size:1.75rem;min-width:11rem;min-height:4.5rem}
.keep{background:var(--keep)}.toss{background:var(--toss)}
#queue ol{padding-left:2.5rem}
li button{font-size:1.125rem;margin-left:.75rem}
details summary{cursor:pointer;font-size:1.5rem}
`
