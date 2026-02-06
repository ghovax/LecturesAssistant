# Lectures Assistant Server

Backend server for the Lectures Assistant application written in Go.

## Architecture

- **API Layer** (`internal/api`): RESTful API handlers and routing
- **Job Queue** (`internal/jobs`): Asynchronous background job processing with sequential execution for heavy tasks
- **Database** (`internal/database`): SQLite database with full schema management and foreign key constraints
- **Models** (`internal/models`): Shared data structures and constants
- **Transcription** (`internal/transcription`): Whisper-based audio processing and FFmpeg integration
- **Documents** (`internal/documents`): PDF/PPTX/DOCX processing and Vision LLM OCR
- **LLM** (`internal/llm`): Provider-agnostic interface for OpenRouter and Ollama

## Configuration

The server is designed to be "zero-config" for basic usage but remains highly customizable via a `configuration.yaml` file.

### Storage and Paths

- **Default Location**: `~/.lectures/configuration.yaml`
- **Override Path**: Use the `-configuration /path/to/file` flag when starting the server.
- **Data Directory**: The base directory for all files (database, logs, uploads) defaults to `~/.lectures`. This can be overridden by setting the `STORAGE_DATA_DIRECTORY` environment variable.

### Automatic Generation

If no configuration file is found on startup, the server automatically generates a default one with sensible values (SQLite local storage, OpenRouter provider, and local Whisper transcription) and saves it to disk.

## Startup Sequence

When the executable is launched, the following initialization steps occur:

1.  **Configuration Loading**: Reads the YAML file or initializes defaults.
2.  **Filesystem Setup**: Ensures the data directory and subfolders exist:
    - `/files/lectures/`: Stores uploaded media and documents.
    - `/files/exports/`: Stores generated study guides and PDFs.
    - `/models/`: Target for Whisper models.
3.  **Logging**: Initializes JSON logging to both `stdout` and a `server.log` file in the data directory.
4.  **Database Initialization**: Opens the SQLite database and executes schema migrations to ensure the table structure is up to date.
5.  **Dependency Verification**: Checks for required system binaries in the PATH:
    - `ffmpeg` & `ffprobe` (Media extraction).
    - `gs` (Ghostscript) & `soffice` (Document processing).
    - `pandoc` & `tectonic` (PDF generation).
    - `whisper` (Transcription).
6.  **Provider Registry**: Initializes AI backend connections (OpenRouter/Ollama) and loads prompt templates.
7.  **Job Queue**: Starts the asynchronous worker pool (default: 4 workers) to handle long-running tasks.
8.  **Server Bind**: Starts the HTTP server and WebSocket Hub on the configured host and port.

## Building

```bash
cd server
go mod download
go build -o lectures ./cmd/server
```

## Running

```bash
./lectures
```

Or specify a custom configuration:

```bash
./lectures -configuration /path/to/configuration.yaml
```

## Authentication

All API endpoints (except `/api/health`, `/api/auth/setup`, `/api/auth/login`, and `/api/auth/status`) require authentication.

### Methods

1.  **Session Cookie**: Set automatically upon successful login.
2.  **Bearer Token**: Include `Authorization: Bearer <token>` in the request headers.

### Flow

- `POST /api/auth/setup`: Set the initial administrator password (only allowed on first run).
- `POST /api/auth/login`: Authenticate with password to receive a session token.
- `POST /api/auth/logout`: Invalidate the current session.

## API Endpoints

### Exams

- `GET /api/exams`: List all exams.
- `POST /api/exams`: Create a new exam.
  - Body: `{"title": "string", "description": "string"}`
- `GET /api/exams/:id`: Get exam details.
- `PATCH /api/exams/:id`: Update exam.
- `DELETE /api/exams/:id`: Delete exam and all associated data.

### Lectures

- `GET /api/exams/:exam_id/lectures`: List lectures for an exam.
- `POST /api/exams/:exam_id/lectures`: Create a lecture with files (Multipart).
  - Parameters: `upload_id` (query, optional for progress tracking)
  - Form Fields: `title`, `description`, `media` (files), `documents` (files)
- `GET /api/exams/:exam_id/lectures/:lecture_id`: Get lecture details.
- `PATCH /api/exams/:exam_id/lectures/:lecture_id`: Update lecture.
- `DELETE /api/exams/:exam_id/lectures/:lecture_id`: Delete lecture.

### Chunked Uploads

- `POST /api/exams/:exam_id/lectures/upload/initialize`: Start a chunked upload.
  - Body: `{"filename": "string", "file_size_bytes": number, "media_type": "media"|"document"}`
- `POST /api/exams/:exam_id/lectures/upload/{uploadId}/chunk`: Upload a file chunk.
- `POST /api/exams/:exam_id/lectures/upload/{uploadId}/complete`: Finalize upload and create lecture.

### Study Tools

- `GET /api/exams/:exam_id/tools`: List tools (filter with `?type=guide|flashcard|quiz`).
- `POST /api/exams/:exam_id/tools`: Trigger tool generation.
  - Body: `{"lecture_id": "uuid", "type": "guide", "length": "medium", "language_code": "en-US"}`
- `GET /api/exams/:exam_id/tools/:tool_id`: Get tool content.

### Chat Sessions

- `POST /api/exams/:exam_id/chat/sessions`: Create session.
- `GET /api/exams/:exam_id/chat/sessions/:session_id`: Get history and context.
- `PATCH /api/exams/:exam_id/chat/sessions/:session_id/context`: Update included materials.
- `POST /api/exams/:exam_id/chat/sessions/:session_id/messages`: Send message (triggers WS stream).

### Jobs & Settings

- `GET /api/jobs`: List recent background jobs.
- `GET /api/jobs/:id`: Get job status.
- `DELETE /api/jobs/:id`: Cancel a running job.
- `GET /api/settings`: Get current preferences.
- `PATCH /api/settings`: Update preferences.

## WebSocket Protocol

Connect to `ws://localhost:3000/api/socket` with a valid session token.

### Messages

- **Subscribe**: `{"type": "subscribe", "channel": "job:<id> | upload:<id> | chat:<id>"}`
- **Job Progress**: Received on `job:<id>` channel.
- **Upload Progress**: Received on `upload:<id>` channel.
- **Chat Token**: Streaming assistant response on `chat:<id>` channel.

## Development

### Project Structure

```
server/
├── cmd/server/          # Entry point
├── internal/
│   ├── api/             # HTTP & WebSocket handlers
│   ├── configuration/   # Config management
│   ├── database/        # SQLite & Schema
│   ├── documents/       # Document processing logic
│   ├── jobs/            # Queue & Workers
│   ├── llm/             # AI Provider implementations
│   ├── markdown/        # AST Parser & Reconstructor
│   ├── models/          # Data models
│   ├── prompts/         # Prompt management
│   ├── tools/           # Study tool generators
│   └── transcription/   # Whisper & FFmpeg logic
└── prompts/             # Markdown prompt templates
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run specific API integration tests
go test -v ./internal/api -run TestIntegration_EndToEndPipeline
```
