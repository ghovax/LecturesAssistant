@echo off

set CONFIGURATION_FILE=configuration.yaml
set BINARY=server\dist\lectures-windows-amd64.exe

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
        password_hash: ''''
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
        api_key: ''''
    ollama:
        base_url: http://localhost:11434
    google:
        client_id: ''''
        client_secret: ''''
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

docker --version >nul 2>&1
if %errorlevel% equ 0 (
    docker-compose up --build
    exit /b 0
)

if exist "%BINARY%" (
    "%BINARY%" -configuration "%CONFIGURATION_FILE%"
) else (
    echo No Docker found and no binary built.
    echo Install Docker from https://docker.com or run build.sh
    pause
    exit /b 1
)
