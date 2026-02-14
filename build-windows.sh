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
GOOS=windows GOARCH=amd64 go build -o "dist/Lectures Assistant.exe" ./cmd/server
cd ..

PACKAGE_DIR="Lectures Assistant"
rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR/bin"

# Absolute path for bin dest
BIN_DEST="$(pwd)/$PACKAGE_DIR/bin"

cp "server/dist/Lectures Assistant.exe" "$PACKAGE_DIR/"
cp -r server/prompts "$PACKAGE_DIR/"
cp server/xelatex-template.tex "$PACKAGE_DIR/"

# Bundling Dependencies
echo "Bundling dependencies (FFmpeg, Pandoc, Tectonic)..."

PANDOC_URL="https://github.com/jgm/pandoc/releases/download/3.9/pandoc-3.9-windows-x86_64.zip"
TECTONIC_URL="https://github.com/tectonic-typesetting/tectonic/releases/download/tectonic%400.15.0/tectonic-0.15.0-x86_64-pc-windows-msvc.zip"
FFMPEG_URL="https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"

download_and_extract() {
    local url=$1
    local name=$2
    local output_file="temp_$name.zip"
    echo "Downloading $name from $url..."
    curl -L "$url" -o "$output_file"
    unzip -q -o "$output_file"
    rm "$output_file"
}

# Download and move to bin folder
mkdir -p temp_build
cd temp_build

download_and_extract "$PANDOC_URL" "pandoc"
find . -name "pandoc.exe" -type f -exec cp {} "$BIN_DEST/" \;

download_and_extract "$TECTONIC_URL" "tectonic"
find . -name "tectonic.exe" -type f -exec cp {} "$BIN_DEST/" \;

download_and_extract "$FFMPEG_URL" "ffmpeg"
find . -name "ffmpeg.exe" -type f -exec cp {} "$BIN_DEST/" \;
find . -name "ffprobe.exe" -type f -exec cp {} "$BIN_DEST/" \;

cd ..
rm -rf temp_build

# Generate Windows Icon
echo "Generating Windows icon..."
ICON_SVG="website/src/lib/assets/favicon.svg"
convert -background none "$ICON_SVG" -define icon:auto-resize=256,128,64,48,32,16 "$PACKAGE_DIR/icon.ico"

cat > "$PACKAGE_DIR/start.bat" << 'EOF'
@echo off
setlocal

set CONFIGURATION_FILE=configuration.yaml

if not exist "%CONFIGURATION_FILE%" (
    (
        echo server:
        echo     host: 0.0.0.0
        echo     port: 3000
        echo storage:
        echo     data_directory: ./data
        echo     bin_directory: ./bin
        echo security:
        echo     auth:
        echo         type: session
        echo         session_timeout_hours: 72
        echo         password_hash: ""
        echo         require_https: false
        echo llm:
        echo     provider: openrouter
        echo     language: en-US
        echo     enable_documents_matching: false
        echo     models:
        echo         recording_transcription:
        echo             model: google/gemini-2.5-flash-lite
        echo         documents_ingestion:
        echo             model: google/gemini-2.5-flash-lite
        echo         documents_matching:
        echo             model: google/gemini-2.5-flash-lite
        echo         outline_creation:
        echo             model: google/gemini-3-flash-preview
        echo         content_generation:
        echo             model: google/gemini-3-flash-preview
        echo         content_verification:
        echo             model: google/gemini-3-flash-preview
        echo         content_polishing:
        echo             model: google/gemini-2.5-flash-lite
        echo     model: anthropic/claude-3.5-sonnet
        echo transcription:
        echo     provider: openrouter
        echo     audio_chunk_length_seconds: 300
        echo     refining_batch_size: 3
        echo providers:
        echo     openrouter:
        echo         api_key: ""
        echo     ollama:
        echo         base_url: http://localhost:11434
        echo     google:
        echo         client_id: ""
        echo         client_secret: ""
        echo documents:
        echo     render_dots_per_inch: 200
        echo     maximum_pages: 1000
        echo     supported_formats:
        echo         - pdf
        echo         - pptx
        echo         - docx
        echo uploads:
        echo     media:
        echo         maximum_file_size_megabytes: 5120
        echo         maximum_files_per_lecture: 10
        echo         supported_formats:
        echo             video:
        echo                 - mp4
        echo                 - mkv
        echo                 - mov
        echo                 - webm
        echo             audio:
        echo                 - mp3
        echo                 - wav
        echo                 - m4a
        echo                 - flac
        echo         chunked_upload_threshold_megabytes: 100
        echo     documents:
        echo         maximum_file_size_megabytes: 500
        echo         maximum_files_per_lecture: 50
        echo         maximum_pages_per_document: 500
        echo         supported_formats:
        echo             - pdf
        echo             - pptx
        echo             - docx
        echo safety:
        echo     maximum_cost_per_job: 15
        echo     maximum_login_attempts_per_hour: 10
        echo     maximum_retries: 3
    ) > "%CONFIGURATION_FILE%"
)

if not exist "data" mkdir data

timeout /t 2 /nobreak >nul
start http://localhost:3000

"Lectures Assistant.exe" -configuration "%CONFIGURATION_FILE%"
EOF

cat > "$PACKAGE_DIR/README.txt" << 'EOF'
Lectures Assistant for Windows

To start the application:
1. Double-click "start.bat"
2. Your browser will open to http://localhost:3000
3. On first run, you'll be prompted to set up your account and API key

To stop the server:
- Close the command window or press Ctrl+C

All data is stored in the "data" folder.
Configuration is stored in "configuration.yaml".
An application icon "icon.ico" is provided in the folder.
EOF
