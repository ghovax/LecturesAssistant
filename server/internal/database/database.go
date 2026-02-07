package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Initialize creates and initializes the SQLite database
func Initialize(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create schema
	if err := createSchema(database); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return database, nil
}

func createSchema(database *sql.DB) error {
	schema := `
	-- Users
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT CHECK(role IN ('admin', 'user')) DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Root: Exams (now owned by a user)
	CREATE TABLE IF NOT EXISTS exams (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Lectures belong to Exams
	CREATE TABLE IF NOT EXISTS lectures (
		id TEXT PRIMARY KEY,
		exam_id TEXT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT CHECK(status IN ('processing', 'ready', 'failed')) DEFAULT 'processing',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Media Files: Audio/Video recordings (one or more per lecture, ordered)
	CREATE TABLE IF NOT EXISTS lecture_media (
		id TEXT PRIMARY KEY,
		lecture_id TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
		media_type TEXT CHECK(media_type IN ('audio', 'video')) NOT NULL,
		sequence_order INTEGER NOT NULL,
		duration_milliseconds INTEGER,
		file_path TEXT NOT NULL,
		original_filename TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(lecture_id, sequence_order)
	);

	-- Unified Transcript (generated from combining all lecture_media files)
	CREATE TABLE IF NOT EXISTS transcripts (
		id TEXT PRIMARY KEY,
		lecture_id TEXT NOT NULL UNIQUE REFERENCES lectures(id) ON DELETE CASCADE,
		language TEXT,
		status TEXT CHECK(status IN ('pending', 'processing', 'completed', 'failed')) DEFAULT 'pending',
		confidence REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Transcript segments with reference to original media file
	CREATE TABLE IF NOT EXISTS transcript_segments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		transcript_id TEXT NOT NULL REFERENCES transcripts(id) ON DELETE CASCADE,
		media_id TEXT REFERENCES lecture_media(id) ON DELETE SET NULL,
		start_millisecond INTEGER NOT NULL,
		end_millisecond INTEGER NOT NULL,
		original_start_milliseconds INTEGER,
		original_end_milliseconds INTEGER,
		text TEXT NOT NULL,
		confidence REAL,
		speaker TEXT
	);

	-- Reference Documents: PDFs, PowerPoints, etc. (zero or more per lecture)
	CREATE TABLE IF NOT EXISTS reference_documents (
		id TEXT PRIMARY KEY,
		lecture_id TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
		document_type TEXT CHECK(document_type IN ('pdf', 'pptx', 'docx', 'other')) NOT NULL,
		title TEXT NOT NULL,
		file_path TEXT NOT NULL,
		original_filename TEXT,
		page_count INTEGER NOT NULL,
		extraction_status TEXT CHECK(extraction_status IN ('pending', 'processing', 'completed', 'failed')) DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Pages/Slides extracted from reference documents
	CREATE TABLE IF NOT EXISTS reference_pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		document_id TEXT NOT NULL REFERENCES reference_documents(id) ON DELETE CASCADE,
		page_number INTEGER NOT NULL,
		image_path TEXT NOT NULL,
		extracted_text TEXT,
		UNIQUE(document_id, page_number)
	);

	-- Generated tools (study guides, flashcards, etc., scoped to Exam)
	CREATE TABLE IF NOT EXISTS tools (
		id TEXT PRIMARY KEY,
		exam_id TEXT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
		type TEXT CHECK(type IN ('guide', 'flashcard', 'quiz', 'custom')) NOT NULL,
		title TEXT NOT NULL,
		content JSON NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tool_source_refs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tool_id TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
		source_type TEXT CHECK(source_type IN ('transcript', 'document')) NOT NULL,
		source_id TEXT NOT NULL,
		metadata JSON
	);

	-- Chat sessions (scoped to an Exam)
	CREATE TABLE IF NOT EXISTS chat_sessions (
		id TEXT PRIMARY KEY,
		exam_id TEXT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
		title TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
		role TEXT CHECK(role IN ('user', 'assistant', 'system')) NOT NULL,
		content TEXT NOT NULL,
		model_used TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chat_citations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id TEXT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
		source_type TEXT CHECK(source_type IN ('transcript', 'slide', 'tool')) NOT NULL,
		source_id TEXT NOT NULL,
		location_type TEXT CHECK(location_type IN ('segment_range', 'page', 'section')),
		location_data JSON,
		snippet TEXT NOT NULL
	);

	-- Chat context: which lectures' materials to include in the session
	CREATE TABLE IF NOT EXISTS chat_context_configuration (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL UNIQUE REFERENCES chat_sessions(id) ON DELETE CASCADE,
		included_lecture_ids JSON,
		included_tool_ids JSON
	);

	-- Background jobs
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type TEXT CHECK(type IN ('TRANSCRIBE_MEDIA', 'INGEST_DOCUMENTS', 'BUILD_MATERIAL', 'PUBLISH_MATERIAL')) NOT NULL,
		status TEXT CHECK(status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')) DEFAULT 'PENDING',
		progress INTEGER DEFAULT 0,
		progress_message_text TEXT,
		payload JSON NOT NULL,
		metadata JSON,
		result JSON,
		error TEXT,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		estimated_cost REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME
	);

	-- User settings (can be global or user-specific if we added user_id, but keeping as is for global defaults)
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value JSON NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Authentication sessions
	CREATE TABLE IF NOT EXISTS auth_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);

	-- Create indexes for common queries
	CREATE INDEX IF NOT EXISTS index_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS index_exams_user_id ON exams(user_id);
	CREATE INDEX IF NOT EXISTS index_lectures_exam_id ON lectures(exam_id);
	CREATE INDEX IF NOT EXISTS index_lecture_media_lecture_id ON lecture_media(lecture_id);
	CREATE INDEX IF NOT EXISTS index_transcripts_lecture_id ON transcripts(lecture_id);
	CREATE INDEX IF NOT EXISTS index_transcript_segments_transcript_id ON transcript_segments(transcript_id);
	CREATE INDEX IF NOT EXISTS index_reference_documents_lecture_id ON reference_documents(lecture_id);
	CREATE INDEX IF NOT EXISTS index_reference_pages_document_id ON reference_pages(document_id);
	CREATE INDEX IF NOT EXISTS index_tools_exam_id ON tools(exam_id);
	CREATE INDEX IF NOT EXISTS index_chat_sessions_exam_id ON chat_sessions(exam_id);
	CREATE INDEX IF NOT EXISTS index_chat_messages_session_id ON chat_messages(session_id);
	CREATE INDEX IF NOT EXISTS index_jobs_user_id ON jobs(user_id);
	CREATE INDEX IF NOT EXISTS index_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS index_auth_sessions_user_id ON auth_sessions(user_id);
	`

	if _, err := database.Exec(schema); err != nil {
		return err
	}

	// Run migrations for schema updates
	migrations := []string{
		// Add original_filename column to lecture_media if it doesn't exist
		`ALTER TABLE lecture_media ADD COLUMN original_filename TEXT`,
	}

	for _, migration := range migrations {
		// SQLite doesn't have IF NOT EXISTS for ALTER TABLE, so we ignore errors if column already exists
		database.Exec(migration)
	}

	return nil
}
