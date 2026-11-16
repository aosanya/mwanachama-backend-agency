#!/usr/bin/env bash
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
hub="${1:-$repo/../projects/hub-steward/code/wakala-api/modules/agency}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

copy_files() {
  local from="$1" to="$2" depth="$3"
  mkdir -p "$to"
  (cd "$from" && find . -maxdepth "$depth" \( -name '*.go' -o -name '*.json' \) -not -path './.git/*' -not -path './scripts/*' -not -path './testdata/*' -print0 | tar -cf - --null -T -) | tar -xf - -C "$to"
}

copy_files "$hub" "$work/hub" 1
mkdir -p "$work/backend"
copy_files "$repo" "$work/backend" 1
for package in mcp models routes; do
  copy_files "$hub/$package" "$work/hub/$package" 1
  copy_files "$repo/$package" "$work/backend/$package" 1
done

find "$work/hub" -name '*.go' -print0 | xargs -0 perl -pi -e '
  s{"wakalaserver/modules/shared/}{"github.com/aosanya/mwanachama-backend-shared/}g;
  s{"wakalaserver/modules/agency}{"github.com/aosanya/mwanachama-backend-agency}g;
'

(cd "$repo" && go run ./scripts/hubdiff "$work/backend")
(cd "$repo" && go run ./scripts/hubdiff "$work/hub")

cd "$work"
diff -r backend hub | sed "s#^diff -r backend/#diff #"
