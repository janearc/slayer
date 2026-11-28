#!/bin/sh
# Print asset and toolchain notices; linked-module-notices.sh covers Go modules.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root"
export GOWORK=off
if [ "$#" -gt 0 ]; then
    goroot=$(GOTOOLCHAIN="$1" go env GOROOT)
else
    goroot=$(go env GOROOT)
fi
cat <<NOTICES
OFL-Cormorant.txt|$root/ferruccio/internal/assets/OFL-Cormorant.txt
OFL-Inter.txt|$root/ferruccio/native/internal/appearance/OFL-Inter.txt
Chroma.txt|$root/ferruccio/native/internal/appearance/themes/LICENSE.chroma
Go-LICENSE.txt|$goroot/LICENSE
Go-PATENTS.txt|$goroot/PATENTS
Mission-Image-Credits.txt|$root/ferruccio/internal/assets/MISSION-IMAGE-CREDITS.txt
Embedded-Data-Credits.txt|$root/ferruccio/internal/assets/EMBEDDED-DATA-CREDITS.txt
NOTICES
