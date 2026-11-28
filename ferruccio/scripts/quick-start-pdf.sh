#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root"
guide=doc/applications/ferruccio
mkdir -p out
pandoc "$guide/quick-start.md" --from gfm+implicit_figures --to html5 --standalone \
    --embed-resources --resource-path "$guide" --css doc/visualguide/print.css \
    --css "$guide/quick-start-print.css" --metadata 'pagetitle=Ferruccio Quick Start' \
    --output out/ferruccio-quick-start.html
prince --style doc/visualguide/print-fonts.css out/ferruccio-quick-start.html \
    --output "$guide/Ferruccio Quick Start.pdf"
