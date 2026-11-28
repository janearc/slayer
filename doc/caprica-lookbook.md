# Caprica look book

`game build caprica-lookbook` builds a local Pzm tool. Once all twelve
images and raw light files exist, use:

```
bin/caprica-lookbook make --manifest /absolute/path/lookbook.json --out /absolute/path/lookbook.html
```

The UTF-8 JSON manifest has `Title`, `Introduction`, `ArtDirection`, a
nonempty `Sources` array of `{ "Title": ..., "URL": "https://..." }`,
and exactly twelve `Frames`. Each frame gives `Title`, `Caption`,
`Directory`, `Image` and `Raw`. Directory may be absolute or relative to
the manifest; image and raw are paths inside it. The page links to each
existing file relative to the HTML's directory and escapes text and links.
The tool refuses missing files or incomplete captions rather than inventing
facts; the author must describe artistic choices and cite the actual sources.

This generator creates one page only. Add its completed page to the shared
gallery using the established publisher after checking its actual paths;
the generator does not alter `latest.html` or publish render files.
