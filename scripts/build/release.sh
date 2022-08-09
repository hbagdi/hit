#!/bin/bash -x

GO_VERSION=1.19.0
if [[ -z "$VERSION" ]];
then
  echo "VERSION env var is not set"
  exit 1
fi

if [[ -z "$COMMIT" ]];
then
  echo "VERSION env var is not set"
  exit 1
fi

echo "Building for version $VERSION with commit sha $COMMIT..."
# working directory for all build output
OUTPUT_DIR="out"
XGO_OUTPUT_DIR="$OUTPUT_DIR/xgo-builds"

mkdir $OUTPUT_DIR

xgo \
  -go $GO_VERSION \
  -tags netgo,osusergo \
  -trimpath \
  -ldflags="-s -w -X github.com/hbagdi/hit/pkg/version.Version=v$VERSION -X github.com/hbagdi/hit/pkg/version.CommitHash=$COMMIT" \
  --targets 'linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64' \
  -out "hit-v$VERSION" \
  -dest ${XGO_OUTPUT_DIR} \
  .

makefat $OUTPUT_DIR/hit-v"${VERSION}"-darwin-all \
  $XGO_OUTPUT_DIR/hit-v"${VERSION}"-darwin-10.16-amd64 \
  $XGO_OUTPUT_DIR/hit-v"${VERSION}"-darwin-10.16-arm64

# tar darwin
cp $OUTPUT_DIR/hit-v"${VERSION}"-darwin-all hit
tar czvf $OUTPUT_DIR/hit_"${VERSION}"_darwin_all.tar.gz hit
rm hit

# tar windows
cp $XGO_OUTPUT_DIR/hit-v"${VERSION}"-windows-4.0-amd64.exe hit.exe
tar czvf $OUTPUT_DIR/hit_"${VERSION}"_windows_amd64.tar.gz hit.exe
rm hit.exe

# tar linux amd64
cp $XGO_OUTPUT_DIR/hit-v"${VERSION}"-linux-amd64 hit
tar czvf $OUTPUT_DIR/hit_"${VERSION}"_linux_amd64.tar.gz hit
rm hit

# tar linux arm64
cp $XGO_OUTPUT_DIR/hit-v"${VERSION}"-linux-arm64 hit
tar czvf $OUTPUT_DIR/hit_"${VERSION}"_linux_arm64.tar.gz hit
rm hit
