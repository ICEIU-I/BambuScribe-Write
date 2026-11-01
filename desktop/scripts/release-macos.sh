#!/bin/bash
# Build only a committed source snapshot; leave the running app and its data alone.
set -euo pipefail

version=${1:-}
release_ref=${2:-HEAD}
if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'Usage: desktop/scripts/release-macos.sh VERSION [GIT_REF]' >&2
  exit 2
fi
if [[ $(uname -s) != Darwin || $(uname -m) != arm64 ]]; then
  echo 'This release script requires an Apple Silicon Mac.' >&2
  exit 2
fi
repo=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
source_commit=$(git -C "$repo" rev-parse --verify "$release_ref^{commit}")
wails_bin=${WAILS_BIN:-$(command -v wails || true)}
if [[ -z $wails_bin && -x "$HOME/go/bin/wails" ]]; then
  wails_bin="$HOME/go/bin/wails"
fi
if [[ ! -x $wails_bin ]]; then
  echo 'Install Wails v2.16.0 or set WAILS_BIN to its absolute path.' >&2
  exit 2
fi
for tool in git go node npm hdiutil codesign xattr ditto lipo rg; do
  command -v "$tool" >/dev/null
done
release_dir="$repo/desktop/build/releases/$version"
if [[ -e $release_dir ]]; then
  echo "Release directory already exists: $release_dir" >&2
  exit 2
fi
workspace=$(mktemp -d "${TMPDIR:-/tmp}/bambuscribe-release.XXXXXX")
trap 'rm -rf "$workspace"' EXIT
mkdir "$workspace/source" "$workspace/artifacts"
git -C "$repo" archive "$source_commit" | tar -x -C "$workspace/source"
snapshot="$workspace/source"
node - "$snapshot/desktop/wails.json" "$version" <<'NODE'
const fs = require('node:fs')
const [path, version] = process.argv.slice(2)
const config = JSON.parse(fs.readFileSync(path, 'utf8'))
if (config.info.productVersion !== version) {
  throw new Error('Commit the matching wails.json productVersion before releasing')
}
NODE
if ! rg -q "^const Version = \"$version\"$" "$snapshot/desktop/internal/buildinfo/version.go"; then
  echo 'Commit the matching buildinfo.Version before releasing' >&2
  exit 2
fi
export MACOSX_DEPLOYMENT_TARGET=13.0
(
  cd "$snapshot/desktop"
  "$wails_bin" build -platform darwin/arm64 -trimpath
)
bash "$snapshot/desktop/scripts/package-macos.sh" \
  "$snapshot/desktop/build/bin/BambuScribe Write.app" \
  "$workspace/artifacts" "$version" "$snapshot"
git -C "$repo" archive --format=zip \
  --prefix="BambuScribe-Write-$version-source/" \
  --output="$workspace/artifacts/BambuScribe-Write-$version-source.zip" "$source_commit"
node - "$workspace/artifacts/build-info.json" "$version" "$source_commit" <<'NODE'
const fs = require('node:fs')
const cp = require('node:child_process')
const [path, version, sourceCommit] = process.argv.slice(2)
fs.writeFileSync(path, JSON.stringify({
  version, sourceCommit, platform: 'darwin/arm64', minimumMacOS: '13.0',
  signing: 'ad-hoc', notarized: false,
  go: cp.execFileSync('go', ['version'], {encoding: 'utf8'}).trim(),
  node: process.version,
}, null, 2) + '\n')
NODE
cp "$snapshot/desktop/releases/$version.md" "$workspace/artifacts/release-notes.md"
cp "$snapshot/LICENSE" "$workspace/artifacts/LICENSE.txt"
cp "$snapshot/desktop/releases/SOURCE-ACCESS.md" "$workspace/artifacts/SOURCE-ACCESS.md"
cp "$snapshot/THIRD_PARTY_NOTICES.md" "$workspace/artifacts/THIRD_PARTY_NOTICES.md"
(
  cd "$workspace/artifacts"
  shasum -a 256 ./*.dmg ./*.zip build-info.json release-notes.md \
    LICENSE.txt SOURCE-ACCESS.md THIRD_PARTY_NOTICES.md > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
mkdir -p "$(dirname "$release_dir")"
mv "$workspace/artifacts" "$release_dir"
echo "Release artifacts: $release_dir"
