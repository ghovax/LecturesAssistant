#!/bin/bash
set -e

# Build Windows binary
./build.sh

DIST_DIR="Lectures Assistant"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Copy binary
cp server/dist/lectures-windows-amd64.exe "$DIST_DIR/Lectures Assistant.exe"

# Create launcher batch file
cat > "$DIST_DIR/Lectures Assistant.bat" << 'EOF'
@echo off
cd /d "%~dp0"

set CONFIGURATION_FILE=configuration.yaml

if not exist "%CONFIGURATION_FILE%" (
    powershell -Command "& {Set-Content -Path '%CONFIGURATION_FILE%' -Value @'
server:
    host: 0.0.0.0
    port: 3000
storage:
    data_directory: ./data
security:
    auth:
        type: session
        session_timeout_hours: 72
        password_hash: ''
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
        api_key: ''
    ollama:
        base_url: http://localhost:11434
    google:
        client_id: ''
        client_secret: ''
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
'@}"
)

if not exist "data" mkdir data

start "" "http://localhost:3000"
"Lectures Assistant.exe" -configuration "%CONFIGURATION_FILE%"
EOF

# Create README
cat > "$DIST_DIR/README.txt" << 'EOF'
LECTURES ASSISTANT

Double-click "Lectures Assistant.bat" to run the application.

First time:
1. The launcher will create configuration.yaml
2. Edit it and add your OpenRouter API key (get one free at https://openrouter.ai)
3. Run "Lectures Assistant.bat" again

The application will open in your web browser at http://localhost:3000

To stop: Press Ctrl+C in the console window
EOF

echo "Created $DIST_DIR/"
echo "Copy this folder to share. Users just double-click 'Lectures Assistant.bat'"
