#!/usr/bin/env bash
# Build gdoc presence as a macOS app bundle.
#
#   ./bundle.sh <version> <out>
#
# version is the release tag, with or without the v, and `dev` for a build
# that came from no release. The plist carries it as 0.0.0 then, because the
# updater compares the plist against the release it is installing and a build
# from no release matches no release.
#
# out is the .app folder to write. It is removed first: a bundle is a tree,
# and half an old one left under a new one is a bundle nobody can reason
# about.
#
# What comes out is one universal executable, one plist, and the signature.
# No symlink, no Resources folder, nothing else, so the zip the release packs
# unzips into the same bytes and the signature is still valid afterwards.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: bundle.sh <version> <out.app>" >&2
  exit 2
fi

version="$1"
out="$2"
here="$(cd "$(dirname "$0")" && pwd)"

# The one place the version is turned into what a Mac calls a version: the tag
# without its v, and 0.0.0 for a checkout build.
stamp="${version#v}"
if [ "$version" = "dev" ] || [ -z "$stamp" ]; then
  stamp="0.0.0"
fi
if ! printf '%s' "$stamp" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "bundle.sh: $version is not a version a bundle can carry" >&2
  exit 1
fi

executable="GdocPresence"

echo "swift build $executable $stamp, arm64 and x86_64"
cd "$here"
swift build -c release --arch arm64 --arch x86_64 --product "$executable" >&2
built="$(swift build -c release --arch arm64 --arch x86_64 --show-bin-path)/$executable"
if [ ! -f "$built" ]; then
  echo "bundle.sh: the build wrote no $built" >&2
  exit 1
fi

rm -rf "$out"
mkdir -p "$out/Contents/MacOS"
cp "$built" "$out/Contents/MacOS/$executable"
chmod 0755 "$out/Contents/MacOS/$executable"
sed "s/@VERSION@/$stamp/g" "$here/Info.plist" > "$out/Contents/Info.plist"
chmod 0644 "$out/Contents/Info.plist"

# Ad-hoc, because this machine holds no Developer ID and M17 owns that
# question. A bundle a Go program unzips carries no quarantine attribute, so
# an ad-hoc signature is enough for the app to open.
codesign --force --sign - --identifier com.altery.gdoc.presence "$out"
codesign --verify --deep --strict "$out"

archs="$(lipo -archs "$out/Contents/MacOS/$executable")"
case " $archs " in
  *" arm64 "*) ;;
  *) echo "bundle.sh: $archs is not universal" >&2; exit 1 ;;
esac
case " $archs " in
  *" x86_64 "*) ;;
  *) echo "bundle.sh: $archs is not universal" >&2; exit 1 ;;
esac

# A symlink in the bundle is a path the updater refuses by name, so a bundle
# with one in it is a release that cannot be installed.
if [ -n "$(find "$out" -type l)" ]; then
  echo "bundle.sh: $out holds a symlink, which the updater refuses" >&2
  exit 1
fi

echo "$out, version $stamp, $archs"
