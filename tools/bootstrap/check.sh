#!/usr/bin/env bash
# Controlled tool boundaries: never download, install or draw during this check.
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd -P)
work=$(mktemp -d "${TMPDIR:-/tmp}/slayer-bootstrap-test.XXXXXX")
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/repo" "$work/tools"
cp "$root/bootstrap.sh" "$work/repo/bootstrap.sh"
printf 'module bootstrap.fixture\ngo 1.27.3\n' > "$work/repo/go.mod"
printf 'game v0.5.0\nname slayer\n' > "$work/repo/.game"
export SLAYER_BOOTSTRAP_DIR="$work/setup"
export SLAYER_SHELF="$work/shelf"
export BOOTSTRAP_TRACE="$work/trace"
cat > "$work/tools/go" <<'GO'
#!/usr/bin/env bash
set -eu
if [[ "$*" == version ]]; then echo 'go version go1.27.3 fixture'; exit 0; fi
printf 'go %s\n' "$*" >> "$BOOTSTRAP_TRACE"
[[ "$*" == 'mod download' ]] || exit 98
exit "${FAKE_DOWNLOAD_FAIL:-0}"
GO
cat > "$work/tools/game" <<'GAME'
#!/usr/bin/env bash
set -eu
if [[ "$*" == --age ]]; then echo 'fixture Game'; exit 0; fi
if [[ "${1:-}" == lane ]]; then shift 3; exec "$@"; fi
printf 'game %s\n' "$*" >> "$BOOTSTRAP_TRACE"
if [[ "$*" == 'run sources-verify' ]]; then exit "${FAKE_VERIFY_FAIL:-0}"; fi
GAME
cat > "$work/tools/git" <<'GIT'
#!/usr/bin/env bash
echo 'Unexpected network/install attempt in bootstrap test' >&2
exit 99
GIT
chmod +x "$work/tools/"*
export PATH="$work/tools:$PATH"
export GAME="$work/tools/game"

run() { bash "$work/repo/bootstrap.sh" "$@" > "$work/output" 2>&1; }
run --help
run --check
[[ ! -e "$SLAYER_BOOTSTRAP_DIR" && ! -e "$SLAYER_SHELF" && ! -e "$BOOTSTRAP_TRACE" ]]
if run --unknown; then echo 'Unknown option accepted' >&2; exit 1; fi
grep -q Usage "$work/output"

printf 'replace example.invalid/pin => %s/missing\n' "$work" >> "$work/repo/go.mod"
if run; then echo 'Missing local replacement accepted' >&2; exit 1; fi
grep -q 'Missing local module replacement' "$work/output"
[[ ! -e "$BOOTSTRAP_TRACE" && ! -e "$SLAYER_BOOTSTRAP_DIR" ]]
sed -i.bak '/^replace /d' "$work/repo/go.mod"

run
cat > "$work/expected" <<'EXPECTED'
go mod download
game run sources-fetch
game run sources-verify
game build
EXPECTED
cmp "$work/expected" "$BOOTSTRAP_TRACE"
grep -q '^gomaxprocs 2$' "$SLAYER_BOOTSTRAP_DIR/game-home/.config/game/config"
grep -q 'export GAME_HOME=' "$work/output"

# Repeating setup preserves a stable script-owned home and repeats the gates.
: > "$BOOTSTRAP_TRACE"
run
cmp "$work/expected" "$BOOTSTRAP_TRACE"

# Dependency failure must stop before source fetching or compilation.
: > "$BOOTSTRAP_TRACE"
export FAKE_DOWNLOAD_FAIL=7
if run; then echo 'Dependency failure accepted' >&2; exit 1; fi
printf 'go mod download\n' > "$work/expected-failure"
cmp "$work/expected-failure" "$BOOTSTRAP_TRACE"
unset FAKE_DOWNLOAD_FAIL

# A failed source check must never be followed by a build.
: > "$BOOTSTRAP_TRACE"
export FAKE_VERIFY_FAIL=8
if run; then echo 'Source failure accepted' >&2; exit 1; fi
if grep -q '^game build$' "$BOOTSTRAP_TRACE"; then exit 1; fi
unset FAKE_VERIFY_FAIL

printf 'personal config\n' > "$SLAYER_BOOTSTRAP_DIR/game-home/.config/game/config"
cp "$SLAYER_BOOTSTRAP_DIR/game-home/.config/game/config" "$work/preserved"
: > "$BOOTSTRAP_TRACE"
if run; then echo 'Unmanaged configuration replaced' >&2; exit 1; fi
cmp "$work/preserved" "$SLAYER_BOOTSTRAP_DIR/game-home/.config/game/config"
[[ ! -s "$BOOTSTRAP_TRACE" ]]

# When no runner is available, install only into the script-owned cache.
# The clone and upstream bootstrap are controlled boundaries, not real network.
mv "$work/tools/game" "$work/game-fixture"
export BOOTSTRAP_GAME_FIXTURE="$work/game-fixture"
unset GAME
export PATH="$work/tools:/usr/bin:/bin"
export SLAYER_BOOTSTRAP_DIR="$work/installed"
cat > "$work/tools/git" <<'CLONE'
#!/usr/bin/env bash
set -eu
[[ "$1" == clone && "$2" == https://github.com/janearc/game.git ]] || exit 99
mkdir -p "$3"
cat > "$3/bootstrap.sh" <<'BUILD'
set -eu
printf 'game-bootstrap\n' >> "$BOOTSTRAP_TRACE"
mkdir -p bin
cp "$BOOTSTRAP_GAME_FIXTURE" bin/game
BUILD
CLONE
chmod +x "$work/tools/git"
: > "$BOOTSTRAP_TRACE"
run
[[ -x "$SLAYER_BOOTSTRAP_DIR/game/bin/game" ]]
printf 'game-bootstrap\n' > "$work/install-expected"
cat "$work/expected" >> "$work/install-expected"
cmp "$work/install-expected" "$BOOTSTRAP_TRACE"
: > "$BOOTSTRAP_TRACE"
run
cmp "$work/expected" "$BOOTSTRAP_TRACE"

export SLAYER_BOOTSTRAP_DIR="$work/repo/inside"
if run --check; then echo 'Setup inside checkout accepted' >&2; exit 1; fi
[[ ! -e "$SLAYER_BOOTSTRAP_DIR" ]]
printf 'Bootstrap boundary checks passed (no network or rendering).\n'
