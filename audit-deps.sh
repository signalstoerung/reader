#!/usr/bin/env bash
#
# audit-deps.sh - periodic dependency and toolchain check for reader.
#
# Run this every 3-6 months. It answers two questions:
#   1. Is the Go toolchain still receiving security fixes?
#   2. Does the binary I actually ship call any known-vulnerable code?
#
# It changes nothing. It only reports. Usage: ./audit-deps.sh

set -uo pipefail
cd "$(dirname "$0")" || exit 1

BOLD=$(tput bold 2>/dev/null || true)
RESET=$(tput sgr0 2>/dev/null || true)
action_needed=0

hr() { printf '%s\n' "------------------------------------------------------------"; }

# ---------------------------------------------------------------- toolchain
hr
printf '%sGo toolchain%s\n' "$BOLD" "$RESET"
hr

pinned=$(sed -n 's/^toolchain //p' go.mod)
[ -z "$pinned" ] && pinned=$(sed -n 's/^go /go/p' go.mod)

# go.dev returns exactly the two major lines that still get security fixes.
supported=$(curl -sS --max-time 20 'https://go.dev/dl/?mode=json' \
  | grep -o '"version": "go[0-9.]*"' | sed 's/.*"go/go/;s/"//' | sort -u)

if [ -z "$supported" ]; then
  echo "WARNING: could not reach go.dev to check release status."
  action_needed=1
else
  pinned_line=${pinned%.*}      # go1.27.1 -> go1.27
  echo "  pinned in go.mod : $pinned"
  echo "  still supported  : $(echo "$supported" | tr '\n' ' ')"
  if echo "$supported" | grep -q "^${pinned_line}\."; then
    latest_in_line=$(echo "$supported" | grep "^${pinned_line}\." | tail -1)
    if [ "$pinned" = "$latest_in_line" ]; then
      echo "  -> OK: on a supported line, at the latest patch."
    else
      echo "  -> UPDATE: $latest_in_line is out. Run:"
      echo "       go mod edit -toolchain=$latest_in_line"
      action_needed=1
    fi
  else
    newest=$(echo "$supported" | tail -1)
    echo "  -> ACT: ${pinned_line}.x no longer receives security fixes. Run:"
    echo "       go mod edit -toolchain=$newest"
    action_needed=1
  fi
fi

# ------------------------------------------------------------ vulnerability
echo
hr
printf '%sReachable vulnerabilities%s\n' "$BOLD" "$RESET"
hr
echo "(scanning the built binary: covers stdlib + deps, and whether you CALL them)"
echo

if [ ! -x ./reader ]; then
  echo "  ./reader not found - building it so the scan reflects what you ship."
  go build -o ./reader . || { echo "  BUILD FAILED"; exit 1; }
fi

# The binary on disk may predate the current go.mod. Say so loudly: a stale
# binary is the thing actually running, so its findings are the real ones.
built_with=$(go version -m ./reader 2>/dev/null | head -1 | awk '{print $2}')
echo "  ./reader was built with : $built_with"
echo "  go.mod pins toolchain   : $pinned"
if [ "$built_with" != "$pinned" ]; then
  echo "  -> NOTE: binary is stale. Findings below reflect the RUNNING build."
  echo "     Rebuild (go build -o reader .) and re-run to see the fixed state."
  action_needed=1
fi
echo

# `go run` masks govulncheck's exit code 3, so read the summary line instead.
scan=$(go run golang.org/x/vuln/cmd/govulncheck@latest -mode=binary ./reader 2>&1)
echo "$scan"
echo
summary=$(echo "$scan" | grep -m1 'Your code is affected by')
if [ -z "$summary" ]; then
  echo "  -> scan did not complete; check the output above."
  action_needed=1
elif echo "$summary" | grep -q 'affected by 0 vulnerabilities'; then
  echo "  -> OK: nothing you call is vulnerable."
else
  echo "  -> ACT: $summary"
  echo "     Only entries under '=== Symbol Results ===' are reachable."
  echo "     Anything reported as 'not called' can wait."
  action_needed=1
fi

# ------------------------------------------------------------------ summary
echo
hr
if [ "$action_needed" -eq 0 ]; then
  printf '%sNothing to do.%s Re-run in 3-6 months.\n' "$BOLD" "$RESET"
else
  printf '%sAction needed - see above.%s\n' "$BOLD" "$RESET"
  echo
  echo "After any change:  go build ./... && go vet ./... && ./audit-deps.sh"
  echo "Then rebuild and restart:  go build -o reader . && systemctl --user restart reader"
fi
hr
exit "$action_needed"
