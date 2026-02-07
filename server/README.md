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
- **`providers`**: Centralized API keys and base URLs for OpenRouter and Ollama.
- **`safety`**: Global thresholds for maximum cost per job, maximum retries for self-healing loops, and login rate limiting.

## Staged Upload Protocol (Prepare, Append, Stage, Import)

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
4.  **Import**: Trigger an asynchronous import from an external provider (e.g., Google Drive).
    - `POST /api/uploads/import`: `{"source": "google_drive", "data": {"file_id": "...", "oauth_token": "..."}}`
    - Returns a `job_id` to track the download and staging.
5.  **Bind**: Create the logical resource (e.g., a Lecture) and bind staged assets.
    - `POST /api/lectures`: (Multipart Form) `exam_id`, `title`, `description`, `specified_date`, `media_upload_ids[]`, `document_upload_ids[]`.
    - Assets are instantly moved to permanent storage and AI jobs are enqueued.

## API Endpoints

### Public

- `GET /api/health`: Check server health and version.
- `POST /api/auth/setup`: Create the initial admin user (only works if no users exist).
- `POST /api/auth/login`: Authenticate and receive a session token (via cookie and JSON).
- `GET /api/auth/status`: Check current session and user details.

### Authentication (Requires Session)

- `POST /api/auth/logout`: Invalidate the current session.

### Exams

- `GET /api/exams`: List exams for the current user.
- `POST /api/exams`: Create a new exam.
- `GET /api/exams/details?exam_id=...`: Get exam metadata.
- `PATCH /api/exams`: Update exam. Body: `{"exam_id": "...", "title": "...", "description": "..."}`.
- `DELETE /api/exams`: Delete exam and all associated data. Body: `{"exam_id": "..."}`.

### Lectures & Materials

- `GET /api/lectures?exam_id=...`: List lectures for an exam.
- `POST /api/lectures`: Create a lecture and bind staged uploads (Supports direct multipart file upload as well).
- `GET /api/lectures/details?lecture_id=...&exam_id=...`: Get lecture details.
- `PATCH /api/lectures`: Update lecture. Body: `{"lecture_id": "...", "exam_id": "...", "title": "...", "description": "...", "specified_date": "..."}`.
- `DELETE /api/lectures`: Delete lecture. Body: `{"lecture_id": "...", "exam_id": "..."}`.
- `GET /api/media?lecture_id=...`: List ordered media files for a lecture.
- `GET /api/transcripts?lecture_id=...`: Retrieve the unified, cleaned transcript.

### Reference Documents

- `GET /api/documents?lecture_id=...`: List all reference documents for a lecture.
- `GET /api/documents/details?document_id=...&lecture_id=...`: Get document metadata.
- `GET /api/documents/pages?document_id=...&lecture_id=...`: List extracted pages and their AI-interpreted content.
- `GET /api/documents/pages/image?document_id=...&lecture_id=...&page_number=...`: Serve the rendered image of a specific page.

### Study Tools

- `GET /api/tools?exam_id=...`: List tools (filter with `?type=guide|flashcard|quiz`).
- `POST /api/tools`: Trigger tool generation.
- `GET /api/tools/details?tool_id=...&exam_id=...`: Get tool content.
- `DELETE /api/tools`: Delete tool. Body: `{"tool_id": "...", "exam_id": "..."}`.
- `POST /api/tools/export`: Trigger an export job (PDF, Docx, MD) for a tool. Body: `{"tool_id": "...", "exam_id": "...", "format": "..."}`.
- `GET /api/exports/download?path=...`: Download a generated export file.

### Chat Sessions

- `GET /api/chat/sessions?exam_id=...`: List sessions in an exam.
- `POST /api/chat/sessions`: Create session.
- `GET /api/chat/sessions/details?session_id=...&exam_id=...`: Get history and context configuration.
- `PATCH /api/chat/sessions/context`: Update active context. Body: `{"session_id": "...", "included_lecture_ids": [...], "included_tool_ids": [...]}`.
- `DELETE /api/chat/sessions`: Delete session. Body: `{"session_id": "...", "exam_id": "..."}`.
- `POST /api/chat/messages`: Send user message and trigger async AI response.

### Jobs & Settings

- `GET /api/jobs`: List recent background jobs for the current user.
- `GET /api/jobs/details?job_id=...`: Get specific job status and results.
- `DELETE /api/jobs`: Request job cancellation. Body: `{"job_id": "..."}`.
- `GET /api/settings`: Get current operational preferences.
- `PATCH /api/settings`: Update preferences (LLM models, themes).

## WebSocket Protocol

Connect to `ws://localhost:3000/api/socket` with a valid session token (Cookie or Authorization header).

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
   Then edit `configuration.yaml` and add your API key for OpenRouter.

2. **Install dependencies**:
   ```bash
   make deps
   ```

### Building

```bash
make build
```

### Running

```bash
make run
```

### Testing

**Unit and integration tests**:
```bash
make test
```

**Full pipeline integration test**:
```bash
make test-integration
```