#!/bin/sh
# Update Formula/molebar.rb to the published GitHub Release source tarball.
# The release asset must already exist; this script does not invent checksums.
#
# Usage:
#   scripts/update-homebrew-formula.sh 0.1.3
#   scripts/update-homebrew-formula.sh v0.1.3
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
formula="$root/Formula/molebar.rb"

if [ ! -f "$formula" ]; then
  echo "update-homebrew-formula: missing $formula" >&2
  exit 1
fi

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 0.1.3" >&2
  echo "Publish dist/molebar-<version>.tar.gz on the GitHub Release first." >&2
  exit 2
fi

version=${1#v}
homepage=$(sed -n 's/^  homepage "\(.*\)"$/\1/p' "$formula" | head -n 1)
if [ -z "$homepage" ]; then
  echo "update-homebrew-formula: could not read homepage from $formula" >&2
  exit 1
fi

url="${homepage}/releases/download/v${version}/molebar-${version}.tar.gz"
tmp=$(mktemp)
updated=$(mktemp)
trap 'rm -f "$tmp" "$updated"' EXIT

if ! curl -fsSL "$url" -o "$tmp"; then
  echo "update-homebrew-formula: failed to download $url" >&2
  echo "Publish the GitHub Release source tarball before updating the formula." >&2
  exit 1
fi

sha256=$(shasum -a 256 "$tmp" | awk '{print $1}')
if [ -z "$sha256" ] || [ "${#sha256}" -ne 64 ]; then
  echo "update-homebrew-formula: could not compute sha256 for $url" >&2
  exit 1
fi

awk -v url="$url" -v sha="$sha256" '
  BEGIN { u = 0; s = 0 }
  /^  url "/ && !u { print "  url \"" url "\""; u = 1; next }
  /^  sha256 "/ && !s { print "  sha256 \"" sha "\""; s = 1; next }
  { print }
' "$formula" > "$updated"

if ! grep -q "$sha256" "$updated"; then
  echo "update-homebrew-formula: failed to write sha256 into $formula" >&2
  exit 1
fi

mv "$updated" "$formula"
trap 'rm -f "$tmp"' EXIT

echo "Updated $formula"
echo "  url    $url"
echo "  sha256 $sha256"
