#!/bin/bash
set -e

cd website
npm install
npm run build
cd ..

rm -rf server/internal/api/web/dist
mkdir -p server/internal/api/web/dist
cp -r website/build/* server/internal/api/web/dist/

cd server
mkdir -p dist
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/lectures-mac-arm64 ./cmd/server
    BINARY="dist/lectures-mac-arm64"
else
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/lectures-mac-amd64 ./cmd/server
    BINARY="dist/lectures-mac-amd64"
fi
cd ..

APP_NAME="Lectures Assistant"
APP_DIR="$APP_NAME.app"
CONTENTS="$APP_DIR/Contents"
MACOS="$CONTENTS/MacOS"
RESOURCES="$CONTENTS/Resources"
BIN_DEST_RELATIVE="$RESOURCES/bin"

rm -rf "$APP_DIR"
mkdir -p "$MACOS"
mkdir -p "$RESOURCES"
mkdir -p "$BIN_DEST_RELATIVE"

# Get absolute path for BIN_DEST
BIN_DEST="$(pwd)/$BIN_DEST_RELATIVE"

# Copy binary and resources
cp "server/$BINARY" "$MACOS/lectures"
cp -r server/prompts "$RESOURCES/"
cp server/xelatex-template.tex "$RESOURCES/"

# Bundling Dependencies
echo "Bundling dependencies (FFmpeg, Pandoc, Tectonic, Ghostscript)..."

# Determine URLs based on architecture
if [ "$ARCH" = "arm64" ]; then
    PANDOC_URL="https://github.com/jgm/pandoc/releases/download/3.9/pandoc-3.9-arm64-macOS.zip"
    TECTONIC_URL="https://github.com/tectonic-typesetting/tectonic/releases/download/tectonic%400.15.0/tectonic-0.15.0-aarch64-apple-darwin.tar.gz"
    FFMPEG_URL="https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip"
    FFPROBE_URL="https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip"
    GHOSTSCRIPT_URL="https://github.com/ArtifexSoftware/ghostpdl-downloads/releases/download/gs10040/ghostscript-10.04.0-macos-arm64.tar.gz"
else
    PANDOC_URL="https://github.com/jgm/pandoc/releases/download/3.9/pandoc-3.9-x86_64-macOS.zip"
    TECTONIC_URL="https://github.com/tectonic-typesetting/tectonic/releases/download/tectonic%400.15.0/tectonic-0.15.0-x86_64-apple-darwin.tar.gz"
    FFMPEG_URL="https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip"
    FFPROBE_URL="https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip"
    GHOSTSCRIPT_URL="https://github.com/ArtifexSoftware/ghostpdl-downloads/releases/download/gs10040/ghostscript-10.04.0-macos-x86_64.tar.gz"
fi

download_and_extract() {
    local url=$1
    local name=$2
    local output_file="temp_$name"
    if [[ "$url" == *"/zip" ]]; then output_file="temp_$name.zip"; fi
    if [[ "$url" == *".zip" ]]; then output_file="temp_$name.zip"; fi
    if [[ "$url" == *".tar.gz" ]]; then output_file="temp_$name.tar.gz"; fi
    
    echo "Downloading $name from $url..."
    curl -L "$url" -o "$output_file"
    
    if [[ "$output_file" == *".tar.gz" ]]; then
        tar -xzf "$output_file"
    elif [[ "$output_file" == *".zip" ]]; then
        unzip -q -o "$output_file"
    fi
    rm "$output_file"
}

# Download and move to bin folder
mkdir -p temp_build
cd temp_build

download_and_extract "$PANDOC_URL" "pandoc"
find . -name "pandoc" -type f -exec cp {} "$BIN_DEST/" \;

download_and_extract "$TECTONIC_URL" "tectonic"
find . -name "tectonic" -type f -exec cp {} "$BIN_DEST/" \;

download_and_extract "$FFMPEG_URL" "ffmpeg"
find . -name "ffmpeg" -type f -exec cp {} "$BIN_DEST/" \;

download_and_extract "$FFPROBE_URL" "ffprobe"
find . -name "ffprobe" -type f -exec cp {} "$BIN_DEST/" \;

download_and_extract "$GHOSTSCRIPT_URL" "ghostscript"
find . -name "gs" -type f -exec cp {} "$BIN_DEST/" \;

cd ..
rm -rf temp_build
chmod +x "$BIN_DEST"/*

# Generate Icon
echo "Generating macOS icon..."
ICON_SVG="website/src/lib/assets/favicon.svg"
ICONSET_DIR="icon.iconset"
mkdir -p "$ICONSET_DIR"

gen_png() {
    rsvg-convert -w "$1" -h "$1" "$ICON_SVG" -o "$ICONSET_DIR/icon_$2.png"
}

gen_png 16 "16x16"
gen_png 32 "16x16@2x"
gen_png 32 "32x32"
gen_png 64 "32x32@2x"
gen_png 128 "128x128"
gen_png 256 "128x128@2x"
gen_png 256 "256x256"
gen_png 512 "256x256@2x"
gen_png 512 "512x512"
gen_png 1024 "512x512@2x"

iconutil -c icns "$ICONSET_DIR" -o "$RESOURCES/icon.icns"
rm -rf "$ICONSET_DIR"

# Create launcher script
cat > "$MACOS/launch.sh" << 'EOF'
#!/bin/bash
APP_DIR="$(cd "$(dirname "$0")/../.." && pwd)"

# Use AppleScript to run in Terminal
osascript <<APPLESCRIPT
tell application "Terminal"
    activate
    do script "cd '$APP_DIR' && export BINARY_PATH='$APP_DIR/Contents/MacOS/lectures' && export RESOURCES_PATH='$APP_DIR/Contents/Resources' && bash -c '
# The server handles default configuration and browser opening automatically
mkdir -p data

[ ! -e prompts ] && cp -r \"\$RESOURCES_PATH/prompts\" prompts
[ ! -e xelatex-template.tex ] && cp \"\$RESOURCES_PATH/xelatex-template.tex\" xelatex-template.tex

\"\$BINARY_PATH\"
'"
end tell
APPLESCRIPT
EOF

chmod +x "$MACOS/launch.sh"
chmod +x "$MACOS/lectures"

cat > "$CONTENTS/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>launch.sh</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.lectures.assistant</string>
    <key>CFBundleName</key>
    <string>Lectures Assistant</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF
