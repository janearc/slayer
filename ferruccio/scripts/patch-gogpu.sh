#!/bin/sh
# Ferruccio builds against gogpu v0.54.1 with patches on top: on macOS an
# app can start in the background, without taking focus, so proof and
# regression windows do not interrupt whoever is at the machine. The patch
# is offered upstream; until gogpu releases it, the build applies it here.
#
# This copies gogpu from Go's module cache into ferruccio/native/third_party/
# gogpu, which git ignores and go.mod's replace names, and applies the patch.
# It does nothing when the copy is already patched at this version.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
native="$root/ferruccio/native"
version=v0.54.1
patch="$native/patches/gogpu-$version-background-startup.patch"
copy="$native/third_party/gogpu"
mark="$copy/.patched-$version"
text_patch="$native/patches/gogpu-$version-command-text.patch"
text_mark="$copy/.patched-$version-command-text"

if [ -f "$mark" ]; then
    if [ ! -f "$text_mark" ]; then
        (cd "$copy" && patch -p1 --quiet < "$text_patch")
        touch "$text_mark"
    fi
    exit 0
fi
source=$(cd "$native" && GOWORK=off GOFLAGS=-mod=mod go mod download -json "github.com/gogpu/gogpu@$version" |
    sed -n 's/^[[:space:]]*"Dir": "\(.*\)",$/\1/p')
if [ -z "$source" ] || [ ! -d "$source" ]; then
    echo "patch-gogpu: go mod download did not give gogpu $version" >&2
    exit 1
fi
rm -rf "$copy.new"
mkdir -p "$(dirname "$copy")"
cp -R "$source" "$copy.new"
chmod -R u+w "$copy.new"
(cd "$copy.new" && patch -p1 --quiet < "$patch")
(cd "$copy.new" && patch -p1 --quiet < "$text_patch")
touch "$copy.new/.patched-$version"
touch "$copy.new/.patched-$version-command-text"
rm -rf "$copy"
mv "$copy.new" "$copy"
