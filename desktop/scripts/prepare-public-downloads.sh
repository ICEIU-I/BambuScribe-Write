#!/bin/bash
# All release artifacts, including corresponding source, are public from 1.0.0.
set -euo pipefail
version=${1:-}
if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'Usage: desktop/scripts/prepare-public-downloads.sh VERSION' >&2
  exit 2
fi
repo=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
release_dir="$repo/desktop/build/releases/$version"
(cd "$release_dir" && shasum -a 256 -c SHA256SUMS)
echo "Publish all verified files from: $release_dir"
