# Lectures Assistant Server

Backend server for the Lectures Assistant application written in Go.

## Architecture

- **Multi-Tenant Security**: Full data isolation between users. Resources (Exams, Lectures, Jobs) are owned by specific users.
- **API Layer** (`internal/api`): Clean RESTful API handlers using a Stage-and-Bind architecture with JWT-like session management.
- **Job Queue** (`internal/jobs`): Asynchronous background job processing for heavy AI tasks.
- **Database** (`internal/database`): SQLite database with a robust schema and resource boundary enforcement.
- **Transcription** (`internal/transcription`): Multi-segment processing with LLM polishing and variable batch sizes.
- **Documents** (`internal/documents`): PDF/PPTX/DOCX processing and Vision LLM OCR.
- **LLM** (`internal/llm`): Provider-agnostic interface for OpenRouter and Ollama with granular task-specific model selection.

## Logical Configuration (`configuration.yaml`)

The server configuration is organized into task-specific operational settings and global provider credentials.

### Key Sections:

- **`llm`**: Global fallback model and granular model selection for ingestion, generation, polishing, etc.
- **`transcription`**: Provider and model settings for audio processing.
- **`providers`**: Centralized API keys and base URLs for OpenRouter, OpenAI, and Ollama.
- **`safety`**: Global thresholds for maximum cost per job, maximum retries for self-healing loops, and login rate limiting.

## Staged Upload Protocol (Prepare, Append, Stage)

The server employs a robust "Stage-and-Bind" protocol for all binary materials (Recordings, Videos, PDFs). This ensures efficiency for small files and resilience for massive multi-gigabyte uploads.

### Workflow: 0 to Hero

1.  **Prepare**: Initialize a staging session.
    - `POST /api/uploads/prepare`: `{"filename": "lecture.mp4", "file_size_bytes": 1024}`
    - Returns an `upload_id`.
2.  **Append**: Stream binary data (supports resumable chunking).
    - `POST /api/uploads/append?upload_id=XYZ`
    - Body: `[Binary Octet-Stream]`
    - Progress is broadcasted in real-time via WebSockets.
3.  **Stage**: Finalize and lock the asset in the staging area.
    - `POST /api/uploads/stage`: `{"upload_id": "XYZ"}`
    - The asset is now ready for binding.
4.  **Bind**: Create the logical resource (e.g., a Lecture) and bind staged assets.
    - `POST /api/lectures`: `{"exam_id": "...", "media_upload_ids": ["XYZ"]}`
    - Assets are instantly moved to permanent storage and AI jobs are enqueued.

## API Endpoints

### Authentication

- `POST /api/auth/setup`: Create the initial admin user.
- `POST /api/auth/login`: Authenticate and receive a session token.
- `GET /api/auth/status`: Check current session and user details.
- `POST /api/auth/logout`: Invalidate the current session.

### Exams

- `GET /api/exams`: List exams for the current user.
- `POST /api/exams`: Create a new exam (supports AI title/description polishing).
- `GET /api/exams/details?exam_id=...`: Get exam metadata.
- `PATCH /api/exams`: Update exam (supports AI polishing).
- `DELETE /api/exams`: Delete exam and all associated data.

### Lectures

- `GET /api/lectures?exam_id=...`: List lectures for an exam.
- `POST /api/lectures`: Create a lecture and bind staged uploads (supports AI title/description polishing and file extension sanitization).
- `GET /api/lectures/details?lecture_id=...`: Get lecture details.
- `PATCH /api/lectures`: Update lecture.
- `DELETE /api/lectures`: Delete lecture.

### Study Tools

- `GET /api/tools?exam_id=...`: List tools (filter with `?type=guide|flashcard|quiz`).
- `POST /api/tools`: Trigger tool generation with optional model overrides.
- `GET /api/tools/details?tool_id=...`: Get tool content.
- `DELETE /api/tools`: Delete tool.

### Chat Sessions

- `GET /api/chat/sessions?exam_id=...`: List sessions in an exam.
- `POST /api/chat/sessions`: Create session.
- `GET /api/chat/sessions/details?session_id=...`: Get history and context.
- `PATCH /api/chat/sessions/context`: Update context (materials included).
- `POST /api/chat/messages`: Send message and stream AI response.

### Jobs & Settings

- `GET /api/jobs`: List recent background jobs for the current user.
- `GET /api/jobs/details?job_id=...`: Get specific job status.
- `DELETE /api/jobs`: Cancel a job.
- `GET /api/settings`: Get current operational preferences.
- `PATCH /api/settings`: Update preferences (LLM models, providers, UI themes).

## WebSocket Protocol

Connect to `ws://localhost:3000/api/socket` with a valid session token.

### Security

- **Origin Restriction**: Only localhost/127.0.0.1 allowed by default.
- **Job Ownership**: Users can only subscribe to progress updates for their own jobs.

### Channels

- **Subscribe**: `{"type": "subscribe", "channel": "job:<id> | upload:<id> | chat:<id>"}`
- **Job Progress**: Received on `job:<id>` channel.
- **Upload Progress**: Received on `upload:<id>` (from staging session).
- **Chat Token**: Streaming assistant response on `chat:<id>`.

## Development & Testing

### Initial Setup

1. **Configure the server**:
   ```bash
   cd server
   cp configuration.yaml.example configuration.yaml
   ```
   Then edit `configuration.yaml` and add your API keys for OpenRouter and/or OpenAI.

2. **Install dependencies**:
   ```bash
   make deps
   ```

### Building

```bash
make build
```

Or manually:
```bash
go build -o lectures ./cmd/server
```

### Running

```bash
make run
```

Or with auto-reload during development:
```bash
make dev
```

### Testing

**Unit and integration tests**:
```bash
make test
```

**Full pipeline integration test** (requires real API keys and test files):

1. Ensure `configuration.yaml` has valid API keys
2. Place test files in `internal/api/test_input/`:
   - `test_audio.mp3` (any audio file)
   - `test_document.pdf` (any PDF)
3. Run:
   ```bash
   make test-integration
   ```
4. Check results in `internal/api/test_integration_pipeline_results/`

The integration suite tests multi-user isolation, safety thresholds, and the complete AI pipeline from upload to PDF export.
