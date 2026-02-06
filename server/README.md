# Lectures Assistant Server

Backend server for the Lectures Assistant application written in Go.

## Architecture

- **API Layer** (`internal/api`): Clean RESTful API handlers using a Stage-and-Bind architecture.
- **Job Queue** (`internal/jobs`): Asynchronous background job processing for heavy AI tasks.
- **Database** (`internal/database`): SQLite database with robust schema and resource boundary enforcement.
- **Transcription** (`internal/transcription`): Multi-segment Whisper processing with LLM polishing.
- **Documents** (`internal/documents`): PDF/PPTX/DOCX processing and Vision LLM OCR.
- **LLM** (`internal/llm`): Provider-agnostic interface for OpenRouter and Ollama.

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

All resource identifiers are passed via **payload arguments** (POST/PATCH/DELETE) or **query parameters** (GET) to ensure a clean and flexible interface.

### Exams
- `GET /api/exams`: List all exams.
- `POST /api/exams`: Create a new exam.
- `GET /api/exams/details?exam_id=...`: Get exam metadata.
- `PATCH /api/exams`: Update exam (requires `exam_id` in body).
- `DELETE /api/exams`: Delete exam (requires `exam_id` in body).

### Lectures
- `GET /api/lectures?exam_id=...`: List lectures for an exam.
- `POST /api/lectures`: Create a lecture and bind staged uploads.
  - Body: `{"exam_id", "title", "media_upload_ids": [], "document_upload_ids": []}`
- `GET /api/lectures/details?lecture_id=...`: Get lecture details.
- `PATCH /api/lectures`: Update lecture (requires `lecture_id` in body).
- `DELETE /api/lectures`: Delete lecture (requires `lecture_id` in body).

### Study Tools
- `GET /api/tools?exam_id=...`: List tools (filter with `?type=guide|flashcard|quiz`).
- `POST /api/tools`: Trigger tool generation.
  - Body: `{"exam_id", "lecture_id", "type", "length", "language_code"}`
- `GET /api/tools/details?tool_id=...`: Get tool content.
- `DELETE /api/tools`: Delete tool (requires `tool_id` in body).

### Chat Sessions
- `GET /api/chat/sessions?exam_id=...`: List sessions in an exam.
- `POST /api/chat/sessions`: Create session (requires `exam_id`).
- `GET /api/chat/sessions/details?session_id=...`: Get history and context.
- `PATCH /api/chat/sessions/context`: Update context (requires `session_id`, `included_lecture_ids`).
- `POST /api/chat/messages`: Send message (requires `session_id`, `content`).

### Jobs & Settings
- `GET /api/jobs`: List recent background jobs.
- `GET /api/jobs/details?job_id=...`: Get specific job status.
- `DELETE /api/jobs`: Cancel a job (requires `job_id` in body).
- `GET /api/settings`: Get current preferences.
- `PATCH /api/settings`: Update preferences (updates in-memory config immediately).

## WebSocket Protocol

Connect to `ws://localhost:3000/api/socket` with a valid session token.

### Channels
- **Subscribe**: `{"type": "subscribe", "channel": "job:<id> | upload:<id> | chat:<id>"}`
- **Job Progress**: Received on `job:<id>` channel.
- **Upload Progress**: Received on `upload:<id>` (from staging session).
- **Chat Token**: Streaming assistant response on `chat:<id>`.

## Development & Testing

### Building
```bash
cd server
go mod download
go build -o lectures ./cmd/server
```

### Testing
All changes are verified via a comprehensive integration suite that tests the entire pipeline from staging to AI generation.
```bash
cd server
go test -v ./internal/api/...
```