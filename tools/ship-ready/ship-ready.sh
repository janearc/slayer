#!/bin/sh
# ship-ready: the last check before a slayer release is tagged.
# tools/ship-ready/README.md says what each stage is for.
#
#   ship-ready.sh [--runner OWNER/REPO]
# shellcheck disable=SC2329 # stage runs the stage functions by name
set -u

usage() {
    echo "usage: ship-ready.sh [--runner OWNER/REPO]" >&2
    exit 2
}

runner=
while [ "$#" -gt 0 ]; do
    case $1 in
        --runner) [ "$#" -ge 2 ] || usage; runner=$2; [ -n "$runner" ] || usage; shift 2 ;;
        --runner=*) runner=${1#--runner=}; [ -n "$runner" ] || usage; shift ;;
        -h | --help) usage ;;
        *) echo "ship-ready: unknown argument: $1" >&2; usage ;;
    esac
done
# OWNER/REPO as GitHub names them, and nothing else: no URL, no .git
case $runner in
    '') ;;
    *[!A-Za-z0-9_./-]* | */*/* | /* | */ | *.git)
        echo "ship-ready: --runner takes OWNER/REPO, such as janearc/ferruccio-dryrun" >&2
        exit 2 ;;
    */*) ;;
    *) echo "ship-ready: --runner takes OWNER/REPO, such as janearc/ferruccio-dryrun" >&2; exit 2 ;;
esac

root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$root" || exit 2
if [ -n "$(git status --porcelain)" ]; then
    echo "ship-ready: the tree has changes; it tests the commit, so commit or stash them first" >&2
    exit 2
fi
game=${GAME:-game}
for tool in "$game" go git shuriken actionlint shellcheck hdiutil lipo; do
    command -v "$tool" >/dev/null || { echo "ship-ready: $tool is not on PATH" >&2; exit 2; }
done
# --runner force-pushes over main, so it is checked against the
# repository GitHub resolves the name to, after any rename or transfer
if [ -n "$runner" ]; then
    command -v gh >/dev/null || { echo "ship-ready: --runner needs gh on PATH" >&2; exit 2; }
    resolved=$(gh repo view "$runner" --json nameWithOwner -q .nameWithOwner) ||
        { echo "ship-ready: GitHub has no repository $runner that gh can see" >&2; exit 2; }
    case $(printf '%s' "$resolved" | tr '[:upper:]' '[:lower:]') in
        janearc/slayer)
            echo "ship-ready: $runner is janearc/slayer; --runner force-pushes over main and will not use it" >&2
            exit 2 ;;
        */*-dryrun) runner=$resolved ;;
        *)
            echo "ship-ready: --runner force-pushes over main, so it uses only a repository whose name ends in -dryrun; $resolved does not" >&2
            exit 2 ;;
    esac
fi

head=$(git rev-parse --short HEAD)
logs="$root/bin/ship-ready/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$logs"
summary="$logs/summary.txt"
tmpbase=${TMPDIR:-/tmp}
work=$(mktemp -d "${tmpbase%/}/ship-ready.XXXXXX") || exit 2
dist="$work/dist"
failed=0
cleanfailed=0
slowpackages=

note() {
    printf '%s\n' "$*" | tee -a "$summary"
}

line() {
    note "$(printf '%-4s  %-14s %5s s  %s' "$1" "$2" "$3" "$4")"
}

# stage NAME COMMAND...: runs the command with its output in NAME.log,
# says pass or FAIL with the time taken, and shows a failure's last lines
stage() {
    name=$1
    shift
    log="$logs/$name.log"
    start=$(date +%s)
    if "$@" >"$log" 2>&1; then
        result=pass
    else
        result=FAIL
        failed=1
    fi
    line "$result" "$name" "$(($(date +%s) - start))" "$log"
    if [ "$result" = FAIL ]; then
        tail -n 8 "$log" | sed 's/^/        /' | tee -a "$summary"
    fi
    [ "$result" = pass ]
}

# skip NAME WHY: a stage that could not run because an earlier one failed
skip() {
    line skip "$1" - "$2"
}

# report NAME COMMAND...: like stage, for counts that are read and not
# judged; it fails the run only when the command could not run at all
report() {
    name=$1
    shift
    log="$logs/$name.log"
    start=$(date +%s)
    "$@" >"$log" 2>&1
    code=$?
    result=info
    if [ "$code" -ge 2 ]; then
        result=FAIL
        failed=1
    fi
    line "$result" "$name" "$(($(date +%s) - start))" "$log"
}

# check --cover, where a test that runs past go test's limit is slow, not
# failed: some draw tests take longer than ten minutes. go test stops the
# whole package at the limit, so that package's later tests did not run,
# and the summary says which packages. It is FAIL when any test fails or
# any package fails other than by that limit.
check_cover() {
    name=check-cover
    log="$logs/$name.log"
    start=$(date +%s)
    if "$game" check --cover >"$log" 2>&1; then
        result=pass
    else
        timeouts=$(grep -c '^panic: test timed out' "$log")
        packages=$(grep -c '^FAIL[[:space:]]' "$log")
        tests=$(grep -c '^[[:space:]]*--- FAIL' "$log")
        if [ "$timeouts" -gt 0 ] && [ "$tests" -eq 0 ] && [ "$packages" -eq "$timeouts" ]; then
            result=slow
            slowpackages=$(awk '/^FAIL[[:space:]]/ { sub(/^github.com\/janearc\/slayer\/?/, "", $2); print ($2 == "" ? "." : $2) }' "$log" | tr '\n' ' ')
            slowpackages=${slowpackages% }
        else
            result=FAIL
            failed=1
        fi
    fi
    line "$result" "$name" "$(($(date +%s) - start))" "$log"
    case $result in
        slow)
            # go test names each test still running when the limit came
            awk '/^panic: test timed out/ { print "        " $0; on = 1; next }
                on && /^[[:space:]]+running tests:/ { next }
                on && /^[[:space:]]+Test/ { sub(/^[[:space:]]+/, ""); print "          " $0; next }
                { on = 0 }' "$log" | tee -a "$summary"
            note "        no test failed before the limit; the rest of $slowpackages did not run" ;;
        FAIL)
            grep -E '^[[:space:]]*--- FAIL|^FAIL[[:space:]]|^panic:' "$log" | head -n 12 | sed 's/^/        /' | tee -a "$summary"
            tail -n 3 "$log" | sed 's/^/        /' | tee -a "$summary" ;;
    esac
}
signature() { "$game" signature --write; }
sweep() { "$game" sweep; }

# the go version a go.mod names
go_version_of() {
    awk '$1 == "go" { print $2; exit }' "$1"
}

# newer A B: whether go version A is newer than B, as 1.27.4 is than 1.27
newer() {
    awk -v a="$1" -v b="$2" 'BEGIN {
        n = split(a, x, "."); m = split(b, y, ".")
        for (i = 1; i <= (n > m ? n : m); i++) {
            if (x[i] + 0 > y[i] + 0) exit 0
            if (x[i] + 0 < y[i] + 0) exit 1
        }
        exit 1
    }'
}

# every check runs, so one run shows each problem
github_parses() {
    status=0
    actionlint .github/workflows/*.yml || status=1
    # SC1007 is CDPATH= cd, which is meant
    shellcheck -s sh -e SC1007 ferruccio/scripts/*.sh tools/ship-ready/ship-ready.sh || status=1
    workflow=$(sed -n "s/^ *go-version: *'\{0,1\}\([0-9.]*\)'\{0,1\} *$/\1/p" .github/workflows/ferruccio-dmg.yml | head -n 1)
    echo "go: the Ferruccio workflow installs ${workflow:-nothing it names}"
    [ -n "$workflow" ] || status=1
    # a module that names a newer go than the workflow installs stops the
    # build there; an older one builds
    for mod in go.mod ferruccio/go.mod ferruccio/native/go.mod; do
        version=$(go_version_of "$mod")
        echo "go: $mod names ${version:-none}"
        if [ -z "$version" ] || newer "$version" "$workflow"; then
            echo "$mod names a newer go than the workflow installs" >&2
            status=1
        fi
    done
    return "$status"
}

# a copy of HEAD with the dist's omissions left out, as its own repository
make_dist() (
    mkdir -p "$dist" || exit 1
    sed -e '/^[[:space:]]*#/d' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' .game5/omissions/*.game |
        grep -v '^$' >"$work/omissions"
    # a path, not a pattern: the check below tests each one as a path
    if grep -n '[][*?:#!]' "$work/omissions" >&2; then
        echo "an omission above is a pattern, not a path; ship-ready cannot check it" >&2
        exit 1
    fi
    set --
    while IFS= read -r path; do
        set -- "$@" ":(exclude)$path"
    done <"$work/omissions"
    git archive --format=tar HEAD -- . "$@" | tar -x -C "$dist" || exit 1
    while IFS= read -r path; do
        if [ -e "$dist/$path" ]; then
            echo "omitted path is in the copy: $path" >&2
            exit 1
        fi
    done <"$work/omissions"
    echo "$# omissions left out of $head"
    cd "$dist" || exit 1
    git init -q && git add -A && git commit -q -m "ship-ready: slayer $head with the dist's omissions" || exit 1
    git log --oneline -1
)

# the Ferruccio workflow's steps, on nothing of this machine's but the Go
# build cache: an empty module cache, the public proxy and nothing else
# (no fallback to git, which would use this machine's credentials), no go
# env file
clean_machine() (
    cd "$dist" || exit 1
    export GOENV=off GOWORK=off GOMAXPROCS=2 GOFLAGS=-modcacherw GIT_TERMINAL_PROMPT=0
    export GOMODCACHE="$work/modcache"
    export GOPROXY=https://proxy.golang.org GOPRIVATE= GONOPROXY= GONOSUMDB=
    unset SLAYER_SHELF GAME_BIN GAME_UNATTENDED
    GAME_HOME="$work/game-home"
    SLAYER_CACHE="$work/slayer-cache"
    export GAME_HOME SLAYER_CACHE
    mkdir -p "$SLAYER_CACHE" "$GAME_HOME/.config/game" "$GOMODCACHE" || exit 1
    printf 'set SLAYER_CACHE %s\n' "$SLAYER_CACHE" >"$GAME_HOME/.config/game/config"
    go tool game --age &&
        go tool game build --unattended ferruccio-dmg &&
        sh ferruccio/scripts/check-dmg.sh bin/Ferruccio.dmg "$work/mount" || exit 1
    echo "modules fetched: $(find "$GOMODCACHE/cache/download" -name '*.zip' | wc -l | tr -d ' ')"
)

# the copy, force-pushed over main of a throwaway repository, and the
# Ferruccio workflow run there by hand and watched to the end
on_runner() (
    cd "$dist" || exit 1
    sha=$(git rev-parse HEAD)
    git push --force --quiet "https://github.com/$runner.git" HEAD:refs/heads/main || exit 1
    gh workflow run ferruccio-dmg.yml -R "$runner" || exit 1
    id=
    tries=0
    while [ -z "$id" ] && [ "$tries" -lt 60 ]; do
        sleep 5
        tries=$((tries + 1))
        id=$(gh run list -R "$runner" -w ferruccio-dmg.yml -e workflow_dispatch -L 10 \
            --json databaseId,headSha -q ".[] | select(.headSha == \"$sha\") | .databaseId" | head -n 1)
    done
    if [ -z "$id" ]; then
        echo "no run of ferruccio-dmg.yml for $sha appeared in five minutes" >&2
        exit 1
    fi
    echo "https://github.com/$runner/actions/runs/$id"
    if gh run watch "$id" -R "$runner" --exit-status --interval 30 >/dev/null; then
        gh run view "$id" -R "$runner"
    else
        gh run view "$id" -R "$runner"
        gh run view "$id" -R "$runner" --log-failed | tail -n 60
        exit 1
    fi
)

note "ship-ready: slayer $head, $(date '+%Y-%m-%d %H:%M %Z')"
note ""
note "the tree"
report lint "$game" lint
tail -n 1 "$logs/lint.log" | sed 's/^ */        /' | tee -a "$summary"
report shuriken shuriken throw
grep -E '^ *the scroll of .*[0-9]+$' "$logs/shuriken.log" | sed 's/^ */        /' | tee -a "$summary"
check_cover
if stage signature signature; then
    changed=$(git status --porcelain | wc -l | tr -d ' ')
    [ "$changed" = 0 ] || note "        signature --write changed $changed files; commit them with the release"
fi
stage sweep sweep
note ""
note "what GitHub parses"
stage github-parses github_parses
note ""
note "a clean machine"
if stage dist-copy make_dist; then
    if stage clean-build clean_machine; then
        if [ -n "$runner" ]; then
            note ""
            note "GitHub's Mac, $runner"
            stage runner on_runner || cleanfailed=1
        fi
    else
        cleanfailed=1
        [ -z "$runner" ] || skip runner "the clean build failed, so nothing was pushed"
    fi
else
    cleanfailed=1
    skip clean-build "the copy was not made"
    [ -z "$runner" ] || skip runner "the copy was not made, so nothing was pushed"
fi
if [ "$cleanfailed" = 0 ]; then
    # patch-gogpu.sh downloads with its own GOFLAGS, so some of the module
    # cache is read-only; a mounted image would also stop the removal
    case $work in
        "${tmpbase%/}"/ship-ready.?*)
            if ! { chmod -R u+w "$work" && rm -rf "$work"; }; then
                note "        $work could not be removed; check for a mounted Ferruccio image (hdiutil info)"
            fi ;;
    esac
else
    note "        the copy and its module cache are kept in $work"
fi

note ""
if [ "$failed" != 0 ]; then
    note "ship-ready: a stage failed; $summary has each one"
elif [ -n "$slowpackages" ]; then
    note "ship-ready: every stage passed; slayer $head is ready to tag; check --cover did not finish in $slowpackages"
else
    note "ship-ready: every stage passed; slayer $head is ready to tag"
fi
exit "$failed"
