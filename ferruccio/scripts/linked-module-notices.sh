#!/bin/sh
# Emit upstream notice texts for the modules linked into the app executables.
set -eu
app=$(CDPATH= cd -- "$1" && pwd)
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root"
export GOWORK=off LC_ALL=C
cache=$(go env GOMODCACHE)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT

# Recognize the in-tree replacements and the separately checked GoGPU patch.
# Refuse any new replacement until its source and notices are reviewed.
modules=$(
    for executable in Contents/MacOS/ferruccio Contents/MacOS/ferruccio-stage \
        Contents/Resources/engine/slayer Contents/Resources/engine/azazel Contents/Resources/engine/game5; do
        info=$(go version -m "$app/$executable") || exit 1
        printf '%s\n' "$info" | awk '
            $1 == "mod" { main = 1 }
            $1 == "dep" || $1 == "mod" { module=$2; version=$3; sum=$4 }
            $1 == "=>" {
                if (module == "github.com/janearc/slayer" && $2 == "../..") next
                if (module == "github.com/janearc/slayer/ferruccio" && $2 == "..") next
                if (module == "github.com/gogpu/gogpu" && $2 == "./third_party/gogpu") next
                print "linked-module-notices: unreviewed replacement for " module > "/dev/stderr"
                exit 1
            }
            ($1 == "dep" || $1 == "mod") && module !~ /^github[.]com\/janearc\/slayer(\/|$)/ {
                print module, version, sum
            }
            END { if (!main) exit 1 }
        ' || exit 1
    done
)
modules=$(printf '%s\n' "$modules" | sort -u)
# The patched local replacement has no h1 in executable build info.
gogpu_sum=h1:xVJD3H9a1EcPYp1bUC0IbkFPN4KNWHRon2r8rXBkfVo=
printf '%s\n' "$modules" | while read -r module version sum; do
    if [ -z "$version" ] || [ "$version" = '(devel)' ]; then
        echo "linked-module-notices: no published version for $module" >&2
        exit 1
    fi
    go mod download "$module@$version"
    if [ -z "$sum" ] && [ "$module@$version" = github.com/gogpu/gogpu@v0.54.1 ]; then
        sum=$gogpu_sum
    fi
    # Go escapes upper-case module path characters with ! and lower case.
    escaped=$(printf '%s\n' "$module" | awk '{
        for (i=1; i<=length; i++) {
            c=substr($0,i,1)
            printf "%s", c ~ /[A-Z]/ ? "!" tolower(c) : c
        }
    }')
    archive="$scratch/$escaped@$version.zip"
    mkdir -p "$(dirname "$archive")"
    cp "$cache/cache/download/$escaped/@v/$version.zip" "$archive"
    printf '%s|%s\n' "$sum" "$archive" >> "$scratch/checks"
    printf '%s|%s|%s|%s\n' "$module" "$version" "$escaped" "$archive" >> "$scratch/modules"
done
go run ./tools/modulezip < "$scratch/checks"

# Downloads above use Go's usual proxy/cache settings. With GOPROXY=off
# the tool requires the pinned archives to be present in the local cache.
printf 'Go modules linked into Ferruccio and its helpers\n'
printf 'Each entry gives the pinned module source archive and its upstream notices.\n'
while IFS='|' read -r module version escaped archive; do
    prefix="$module@$version/"
    printf '\n=== %s %s ===\n' "$module" "$version"
    printf 'Source archive: https://proxy.golang.org/%s/@v/%s.zip\n' "$escaped" "$version"
    if [ "$module" = github.com/gogpu/gogpu ]; then
        printf 'Ferruccio builds a local replacement from this archive with these patches.\n'
        patches="$root/ferruccio/native/patches"
        printf '%s\n' "gogpu-$version-background-startup.patch" "gogpu-$version-command-text.patch"
        if [ "${2:-}" = --check-built-source ]; then
            unzip -q "$archive" -d "$scratch/patched"
            expected="$scratch/patched/$module@$version"
            actual="$root/ferruccio/native/third_party/gogpu"
            (cd "$expected" && patch -p1 --quiet < "$patches/gogpu-$version-background-startup.patch")
            (cd "$expected" && patch -p1 --quiet < "$patches/gogpu-$version-command-text.patch")
            touch "$expected/.patched-$version" "$expected/.patched-$version-command-text"
            if [ -n "$(find "$actual" -type l -print)" ] || ! diff -qr "$expected" "$actual" >&2; then
                echo "linked-module-notices: local GoGPU differs from verified source plus committed patches" >&2
                exit 1
            fi
        fi
    fi
    names=$(unzip -Z1 "$archive")
    texts=$(printf '%s\n' "$names" | awk -v prefix="$prefix" '
        index($0, prefix) == 1 {
            file=substr($0, length(prefix)+1)
            if (file !~ /\// && file ~ /^(LICENSE|COPYING|COPYRIGHT|NOTICE|PATENTS)/) print file
        }
    ' | sort)
    if ! printf '%s\n' "$texts" | grep -Eq '^(LICENSE|COPYING)'; then
        echo "linked-module-notices: no root license for $module@$version" >&2
        exit 1
    fi
    case "$module" in
        github.com/cloudflare/circl) texts="$texts
ecc/p384/LICENSE" ;;
        github.com/gogpu/naga) texts="$texts
third_party/gfx-rs-naga/LICENSE.MIT" ;;
    esac
    printf '%s\n' "$texts" | while IFS= read -r text; do
        printf '\n--- %s ---\n' "$text"
        unzip -p "$archive" "$prefix$text"
        printf '\n'
    done
done < "$scratch/modules"
