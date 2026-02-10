#!/bin/bash
set -e

# Build the native binary first
./build.sh

APP_NAME="Lectures Assistant"
APP_DIR="$APP_NAME.app"
CONTENTS="$APP_DIR/Contents"
MACOS="$CONTENTS/MacOS"
RESOURCES="$CONTENTS/Resources"

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    BINARY="server/dist/lectures-mac-arm64"
else
    BINARY="server/dist/lectures-mac-amd64"
fi

# Create app bundle structure
rm -rf "$APP_DIR"
mkdir -p "$MACOS"
mkdir -p "$RESOURCES"

# Copy binary and resources
cp "$BINARY" "$MACOS/lectures"
cp -r server/prompts "$RESOURCES/"
cp server/xelatex-template.tex "$RESOURCES/"

# Create launcher script that uses Terminal.app
cat > "$MACOS/launch.sh" << 'EOF'
#!/bin/bash
APP_DIR="$(cd "$(dirname "$0")/../.." && pwd)"

# Use AppleScript to run in Terminal
osascript <<APPLESCRIPT
tell application "Terminal"
    activate
    do script "cd '$APP_DIR' && export BINARY_PATH='$APP_DIR/Contents/MacOS/lectures' && export RESOURCES_PATH='$APP_DIR/Contents/Resources' && bash -c '
CONFIGURATION_FILE=\"configuration.yaml\"

if [ ! -f \"\$CONFIGURATION_FILE\" ]; then
    cat > \"\$CONFIGURATION_FILE\" << CONFIGEOF
server:
    host: 0.0.0.0
    port: 3000
storage:
    data_directory: ./data
security:
    auth:
        type: session
        session_timeout_hours: 72
        password_hash: \"\"
        require_https: false
llm:
    provider: openrouter
    language: en-US
    enable_documents_matching: false
    models:
        recording_transcription:
            model: google/gemini-2.5-flash-lite
        documents_ingestion:
            model: google/gemini-2.5-flash-lite
        documents_matching:
            model: google/gemini-2.5-flash-lite
        outline_creation:
            model: google/gemini-3-flash-preview
        content_generation:
            model: google/gemini-3-flash-preview
        content_verification:
            model: google/gemini-3-flash-preview
        content_polishing:
            model: google/gemini-2.5-flash-lite
    model: anthropic/claude-3.5-sonnet
transcription:
    provider: openrouter
    audio_chunk_length_seconds: 300
    refining_batch_size: 3
providers:
    openrouter:
        api_key: \"\"
    ollama:
        base_url: http://localhost:11434
    google:
        client_id: \"\"
        client_secret: \"\"
documents:
    render_dots_per_inch: 200
    maximum_pages: 1000
    supported_formats:
        - pdf
        - pptx
        - docx
uploads:
    media:
        maximum_file_size_megabytes: 5120
        maximum_files_per_lecture: 10
        supported_formats:
            video:
                - mp4
                - mkv
                - mov
                - webm
            audio:
                - mp3
                - wav
                - m4a
                - flac
        chunked_upload_threshold_megabytes: 100
    documents:
        maximum_file_size_megabytes: 500
        maximum_files_per_lecture: 50
        maximum_pages_per_document: 500
        supported_formats:
            - pdf
            - pptx
            - docx
safety:
    maximum_cost_per_job: 15
    maximum_login_attempts_per_hour: 10
    maximum_retries: 3
CONFIGEOF
    echo \"Created configuration.yaml\"
fi

mkdir -p data

# Create symlinks to resources if they don't exist
[ ! -e prompts ] && ln -s \"\$RESOURCES_PATH/prompts\" prompts
[ ! -e xelatex-template.tex ] && ln -s \"\$RESOURCES_PATH/xelatex-template.tex\" xelatex-template.tex

echo \"Starting Lectures Assistant...\"
echo \"Server will be available at http://localhost:3000\"
echo \"Press Ctrl+C to stop the server\"
echo \"\"

sleep 2
open \"http://localhost:3000\"

\"\$BINARY_PATH\" -configuration \"\$CONFIGURATION_FILE\"
'"
end tell
APPLESCRIPT
EOF

chmod +x "$MACOS/launch.sh"
chmod +x "$MACOS/lectures"

# Create Info.plist
cat > "$CONTENTS/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>launch.sh</string>
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

echo "Created $APP_DIR"
echo "Double-click $APP_DIR to run"
