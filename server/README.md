# Learning Assistant Server

The backend server for the Learning Assistant application, built with Go. It leverages AI to transform lecture recordings and reference documents into comprehensive study materials, providing a robust REST API and real-time updates via WebSockets.

## Features

- **Multi-modal AI Pipeline**: Automatic transcription of audio/video and intelligent OCR/interpretation of PDF, PPTX, and DOCX files.
- **Study Tool Generation**: Creation of high-fidelity study guides, flashcards, and quizzes with deep grounding in provided materials.
- **Intelligent Chat**: A reading assistant that can answer questions based on the context of multiple lectures and documents.
- **Robust Export Engine**: Export generated study tools to PDF (via XeLaTeX), Docx, and Markdown.
- **Reliable Uploads**: A "Stage-and-Bind" protocol designed for large multi-gigabyte media files.
- **Asynchronous Processing**: Scalable background job queue for heavy AI tasks.
- **Cross-Provider LLM Support**: Native integration with OpenRouter (Cloud) and Ollama (Local).

## Architecture

- **Security**: Multi-tenant isolation with JWT-like session management and CSRF protection.
- **Concurrency**: SQLite in WAL mode with a managed worker pool for background tasks.
- **Observability**: Structured JSON logging using `slog` with automatic file rotation.
- **Scalability**: Decoupled LLM provider interface allowing for granular task-specific model selection.

## Logical Configuration (`configuration.yaml`)

The server uses a hierarchical configuration structure. A default file is created at `~/.lectures/configuration.yaml` on the first run.

### Key Sections

- **`llm`**: Global provider settings and task-specific model routing (e.g., `outline_creation`, `content_generation`).
- **`transcription`**: Chunking strategies and refining batch sizes for audio processing.
- **`uploads`**: File size limits and supported formats for media and documents.
- **`safety`**: Budget controls (max cost per job), retry thresholds, and rate limiting.
- **`storage`**: Data directory paths for database and permanent file storage.

## Staged Upload Protocol

To handle massive file uploads reliably, the server employs a 4-step "Stage-and-Bind" protocol:

1.  **Prepare** (`POST /api/uploads/prepare`): Initialize a session and declare the expected file size.
2.  **Append** (`POST /api/uploads/append`): Stream binary chunks. Supports resumable uploads.
3.  **Stage** (`POST /api/uploads/stage`): Finalize the asset in the staging area.
4.  **Bind** (`POST /api/lectures`): Create a logical resource and move the staged assets to permanent storage.

---

## API Endpoints

### Authentication

- `POST /api/auth/setup`: Create the initial admin user (enabled only if no users exist).
- `POST /api/auth/login`: Authenticate and receive a session token.
- `GET /api/auth/status`: Check current session validity and user details.
- `POST /api/auth/logout`: Invalidate the current session.
- `PATCH /api/auth/password`: Change the authenticated user's password.

### Exams & Management

- `GET | POST /api/exams`: List or create exams.
- `GET /api/exams/details`: Get metadata for a specific exam.
- `PATCH /api/exams`: Update exam title or description.
- `DELETE /api/exams`: Cascading delete of an exam and all associated data.
- `GET /api/exams/search`: Global keyword search across all transcripts and documents in an exam.
- `POST /api/exams/suggest`: Trigger an AI job to suggest improved metadata for the exam.
- `GET /api/exams/concepts`: Retrieve a "concept map" or glossary generated from study tools.

### Lectures & Transcripts

- `GET | POST /api/lectures`: List or create lectures (supports direct multipart or binding staged IDs).
- `GET /api/lectures/details`: Get lecture status and metadata.
- `PATCH /api/lectures`: Update lecture details.
- `DELETE /api/lectures`: Cancel active jobs and delete lecture assets.
- `GET /api/media`: List all audio/video files associated with a lecture.
- `GET /api/transcripts`: Retrieve the unified, polished transcript segments.
- `PATCH /api/transcripts`: Manually refine transcript text.
- `GET /api/transcripts/html`: Retrieve transcript segments converted to HTML.

### Documents & OCR

- `GET /api/documents`: List all reference documents for a lecture.
- `GET /api/documents/details`: Get document extraction status and metadata.
- `GET /api/documents/pages`: List all extracted pages and their AI-interpreted content.
- `GET /api/documents/pages/image`: Serve the rendered PNG image of a specific page.
- `GET /api/documents/pages/html`: Get the interpreted content of a page as HTML.

### Study Tools

- `GET | POST /api/tools`: List tools or trigger the generation of a new study guide, flashcard set, or quiz.
- `GET /api/tools/details`: Get the tool content (JSON).
- `PATCH /api/tools/details`: Update tool title or content.
- `GET /api/tools/html`: Get tool content converted to formatted HTML.
- `POST /api/tools/export`: Trigger an export job (PDF, Docx, MD, Anki).
- `GET /api/exports/download`: Download a generated export file.

### AI Chat

- `GET | POST /api/chat/sessions`: Manage chat sessions scoped to an exam.
- `GET /api/chat/sessions/details`: Get message history and active context configuration.
- `PATCH /api/chat/sessions/context`: Update which lectures are currently "in-scope" for the assistant.
- `POST /api/chat/messages`: Send a message and trigger an asynchronous, streaming AI response.

---

## WebSocket Protocol

Connect to `ws://[host]/api/socket` with a valid session token.

### Handshake & Messaging

- **Subscribe**: `{"type": "subscribe", "channel": "job:<id> | upload:<id> | chat:<id>"}`
- **Heartbeat**: Standard Ping/Pong frames every 30 seconds.

### Event Types

- `upload:progress`: Real-time byte-level progress for staged uploads.
- `job:progress`: Status updates, percentages, and metrics for background tasks.
- `chat:token`: Incremental assistant response tokens for streaming UI.
- `chat:complete`: Final message metadata including token usage and cost.

---

## Deployment & Development

### Using Docker (Recommended)

The server can be deployed with all its system dependencies (FFmpeg, LibreOffice, Pandoc, Tectonic) using the provided Docker configuration.

```bash
docker-compose up --build -d
```

Access the server at `http://localhost:3000`.

### Local Setup

1. **Install System Dependencies**: FFmpeg, Ghostscript, LibreOffice, Pandoc, and Tectonic.
2. **Download Dependencies**: `make deps`
3. **Build**: `make build`
4. **Run**: `make run` or `make dev` (for development with auto-reload)
5. **Clean**: `make clean` to remove build artifacts.

### Testing

- **Unit Tests**: `make test`
- **Integration Tests**: `make test-integration` (Requires real AI provider keys in `configuration.yaml`)
- **Build All Platforms**: `make build-all`
