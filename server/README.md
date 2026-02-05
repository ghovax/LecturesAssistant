# Lectures Assistant Server

Backend server for the Lectures Assistant application written in Go.

## Architecture

- **API Layer** (`internal/api`): RESTful API handlers and routing
- **Job Queue** (`internal/jobs`): Asynchronous background job processing with multiple workers
- **Database** (`internal/database`): SQLite database with schema management
- **Models** (`internal/models`): Data structures and types
- **Config** (`internal/config`): Configuration management with YAML support

## Features

### Implemented

- ✅ Exam management (CRUD operations)
- ✅ Lecture management (CRUD operations)
- ✅ Media file upload and storage
- ✅ Asynchronous job queue with multiple concurrent workers
- ✅ SQLite database with foreign key constraints
- ✅ Configuration file management
- ✅ Health check endpoint
- ✅ Job monitoring and cancellation

### In Progress

- 🚧 Transcription integration (Whisper)
- 🚧 Document processing (PDF, PPTX, DOCX)
- 🚧 LLM integration (OpenRouter, Ollama)
- 🚧 Chat sessions and messaging
- 🚧 Study tool generation
- 🚧 WebSocket real-time updates
- 🚧 Authentication and sessions

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

Or specify a custom configuration file:

```bash
./lectures -configuration /path/to/configuration.yaml
```

The server will:

1. Create the configuration file if it doesn't exist (`~/.lectures/configuration.yaml`)
2. Initialize the SQLite database (`~/.lectures/database.db`)
3. Start the job queue with 4 concurrent workers
4. Start the HTTP server on `localhost:3000`

## Configuration

The default configuration is created at `~/.lectures/configuration.yaml`:

```yaml
server:
  host: 127.0.0.1
  port: 3000

storage:
  data_directory: ~/.lectures

llm:
  provider: openrouter
  openrouter:
    api_key: ""
    default_model: "anthropic/claude-3.5-sonnet"
  ollama:
    base_url: "http://localhost:11434"
    default_model: "llama3.2"

transcription:
  provider: whisper-local
  whisper:
    model: base
    device: auto
```

## API Endpoints

### Exams

- `POST /api/exams` - Create exam
- `GET /api/exams` - List exams
- `GET /api/exams/:id` - Get exam
- `PATCH /api/exams/:id` - Update exam
- `DELETE /api/exams/:id` - Delete exam

### Lectures

- `POST /api/exams/:exam_id/lectures` - Create lecture
- `GET /api/exams/:exam_id/lectures` - List lectures
- `GET /api/exams/:exam_id/lectures/:lecture_id` - Get lecture
- `PATCH /api/exams/:exam_id/lectures/:lecture_id` - Update lecture
- `DELETE /api/exams/:exam_id/lectures/:lecture_id` - Delete lecture

### Media

- `POST /api/exams/:exam_id/lectures/:lecture_id/media/upload` - Upload media
- `GET /api/exams/:exam_id/lectures/:lecture_id/media` - List media
- `DELETE /api/exams/:exam_id/lectures/:lecture_id/media/:media_id` - Delete media

### Transcripts

- `POST /api/exams/:exam_id/lectures/:lecture_id/transcribe` - Start transcription
- `GET /api/exams/:exam_id/lectures/:lecture_id/transcript` - Get transcript

### Jobs

- `GET /api/jobs` - List jobs
- `GET /api/jobs/:id` - Get job status
- `DELETE /api/jobs/:id` - Cancel job

### System

- `GET /api/health` - Health check
- `GET /api/settings` - Get settings
- `PATCH /api/settings` - Update settings

## Job Queue

The job queue supports multiple concurrent workers and handles:

- Transcription jobs (`TRANSCRIBE_MEDIA`)
- Document ingestion (`INGEST_DOCUMENTS`)
- Tool generation (`BUILD_MATERIAL`)
- Export generation (`PUBLISH_MATERIAL`)

Jobs have the following states:

- `PENDING` - Waiting to be processed
- `RUNNING` - Currently being processed
- `COMPLETED` - Successfully finished
- `FAILED` - Encountered an error
- `CANCELLED` - Manually cancelled

## Database Schema

The SQLite database includes:

- `exams` - Course/exam groupings
- `lectures` - Individual lessons
- `lecture_media` - Audio/video files
- `transcripts` - Unified transcripts
- `transcript_segments` - Individual segments
- `reference_documents` - PDFs, PowerPoints, etc.
- `reference_pages` - Extracted pages
- `tools` - Generated study materials
- `chat_sessions` - Conversation sessions
- `chat_messages` - Individual messages
- `jobs` - Background tasks
- `settings` - User preferences
- `auth_sessions` - Authentication sessions

All tables have proper foreign key constraints and cascading deletes.

## Development

### Project Structure

```
server/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── api/                 # HTTP handlers
│   ├── config/              # Configuration
│   ├── database/            # Database setup
│   ├── jobs/                # Job queue
│   └── models/              # Data models
└── go.mod
```

### Adding a New Job Handler

```go
// Register handler in main.go
jobQueue.RegisterHandler(models.JobTypeTranscribeMedia, func(context context.Context, job *models.Job, updateFn func(int, string)) error {
    updateFn(50, "Processing audio...")
    // Do work here
    updateFn(100, "Complete")
    return nil
})
```

### Testing

```bash
# Create an exam
curl -X POST http://localhost:3000/api/exams \
  -H "Content-Type: application/json" \
  -d '{"title":"Computer Science 101","description":"Introduction to CS"}'

# List exams
curl http://localhost:3000/api/exams

# Health check
curl http://localhost:3000/api/health
```
