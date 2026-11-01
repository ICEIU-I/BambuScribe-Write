#!/bin/bash
# Package an app into a DMG and ZIP, preserving its bundle and signature.
set -euo pipefail
if [[ $# != 4 ]]; then
  echo 'Usage: package-macos.sh APP_PATH OUTPUT_DIR VERSION SOURCE_DIR' >&2
  exit 2
fi
app_path=$1
output_dir=$2
version=$3
source_dir=$4
bundle_name='BambuScribe Write.app'
file_name="BambuScribe-Write-$version-macos-arm64"
workspace=$(mktemp -d "${TMPDIR:-/tmp}/bambuscribe-package.XXXXXX")
trap 'rm -rf "$workspace"' EXIT
mkdir "$workspace/image"
mkdir -p "$output_dir"
ditto "$app_path" "$workspace/image/$bundle_name"
bundle="$workspace/image/$bundle_name"
binary="$bundle/Contents/MacOS/BambuScribe Write"
lipo -verify_arch arm64 "$binary"
if [[ $version != 0.1.0 && $("$binary" --version) != "$version" ]]; then
  echo 'The executable update version does not match the package version.' >&2
  exit 2
fi
/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$bundle/Contents/Info.plist" | \
  node -e 'let s="";process.stdin.on("data",b=>s+=b);process.stdin.on("end",()=>{if(s.trim()!==process.argv[1])process.exit(1)})' "$version"
mkdir -p "$bundle/Contents/Resources"
cp "$source_dir/LICENSE" "$bundle/Contents/Resources/LICENSE.txt"
cp "$source_dir/THIRD_PARTY_NOTICES.md" "$bundle/Contents/Resources/THIRD_PARTY_NOTICES.md"
xattr -cr "$bundle"
codesign --force --deep --sign - "$bundle"
codesign --verify --deep --strict "$bundle"
cp "$source_dir/LICENSE" "$workspace/image/LICENSE.txt"
cp "$source_dir/desktop/releases/INSTALL.txt" "$workspace/image/安装说明.txt"
ln -s /Applications "$workspace/image/Applications"
hdiutil create -volname "BambuScribe Write $version" \
  -srcfolder "$workspace/image" -format UDZO "$output_dir/$file_name.dmg"
hdiutil verify "$output_dir/$file_name.dmg"
archive_dir="$workspace/$file_name"
mkdir "$archive_dir"
ditto "$bundle" "$archive_dir/$bundle_name"
cp "$workspace/image/LICENSE.txt" "$workspace/image/安装说明.txt" "$archive_dir/"
ditto -c -k --sequesterRsrc --keepParent "$archive_dir" "$output_dir/$file_name.zip"
