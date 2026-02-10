#!/bin/bash
cd "$(dirname "$0")"

CONFIGURATION_FILE="configuration.yaml"

if [ ! -f "$CONFIGURATION_FILE" ]; then
    cat > "$CONFIGURATION_FILE" << 'EOF'
server:
    host: 0.0.0.0
    port: 3000
storage:
    data_directory: ./data
security:
    auth:
        type: session
        session_timeout_hours: 72
        password_hash: ""
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
        api_key: ""
    ollama:
        base_url: http://localhost:11434
    google:
        client_id: ""
        client_secret: ""
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
EOF
    echo "Created configuration.yaml"
fi

mkdir -p data

# Detect architecture and use the right binary
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    BINARY="server/dist/lectures-mac-arm64"
else
    BINARY="server/dist/lectures-mac-amd64"
fi

echo "Starting Lectures Assistant..."
echo "Server will be available at http://localhost:3000"
echo "Press Ctrl+C to stop"
echo ""

sleep 2
open "http://localhost:3000"

"$BINARY" -configuration "$CONFIGURATION_FILE"
