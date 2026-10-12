#!/bin/sh
# Sets the Fly app's runtime secrets from an env file (default ./.env):
#   DATABASE_URL, JWT_SECRET, CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY, CLOUDINARY_API_SECRET
# Only those keys are sent. Values go to `flyctl secrets import` on stdin, so they never
# show up in shell history or `ps`. With -n it checks the file and prints what it
# would set (names only) without calling Fly. Run it via `make fly-secrets` /
# `make fly-secrets-dry-run`. Refuses unless the app is in the boraquest org (fly-app.sh).
set -eu

dry_run=false
if [ "${1:-}" = "-n" ]; then
  dry_run=true
  shift
fi
file="${1:-./.env}"
case "$file" in */*) ;; *) file="./$file" ;; esac
[ -f "$file" ] || { echo "$file not found" >&2; exit 1; }

keys="DATABASE_URL JWT_SECRET CLOUDINARY_CLOUD_NAME CLOUDINARY_API_KEY CLOUDINARY_API_SECRET"

# Read only the file: drop anything inherited (e.g. the Makefile's dev JWT_SECRET).
for k in $keys; do unset "$k"; done
set -a
. "$file"
set +a

fail=false
for k in $keys; do
  eval "v=\${$k:-}"
  [ -n "$v" ] || { echo "$k is missing or empty in $file" >&2; fail=true; }
done
case "${DATABASE_URL:-}" in
  *localhost* | *127.0.0.1*) echo "DATABASE_URL points at a local database" >&2; fail=true ;;
esac
case "${JWT_SECRET:-}" in
  boraquest-dev-secret) echo "JWT_SECRET is the dev default; generate one with: openssl rand -base64 48" >&2; fail=true ;;
esac
if [ -n "${JWT_SECRET:-}" ] && [ "${#JWT_SECRET}" -lt 32 ]; then
  echo "JWT_SECRET is shorter than 32 characters; generate one with: openssl rand -base64 48" >&2
  fail=true
fi
$fail && exit 1

if $dry_run; then
  echo "would set on Fly (from $file):"
  for k in $keys; do echo "  $k"; done
  exit 0
fi

app=$(./scripts/fly-app.sh)
for k in $keys; do
  eval "printf '%s=%s\n' \"\$k\" \"\$$k\""
done | flyctl secrets import -a "$app"
