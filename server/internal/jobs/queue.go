package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
)

// Queue manages background job processing
type Queue struct {
	database         *sql.DB
	workers          int
	context          context.Context
	cancel           context.CancelFunc
	waitGroup        sync.WaitGroup
	handlers         map[string]JobHandler
	subscribers      map[string][]chan JobUpdate
	subscribersMutex sync.RWMutex
}

// JobMetrics contains token usage and cost information
type JobMetrics struct {
	InputTokens   int
	OutputTokens  int
	EstimatedCost float64
}

// JobHandler is a function that processes a specific job type
type JobHandler func(context context.Context, job *models.Job, updateProgress func(progress int, message string, metadata any, metrics JobMetrics)) error

// JobUpdate represents a job progress update
type JobUpdate struct {
	JobID               string
	Status              string
	Progress            int
	ProgressMessageText string
	Metadata            any
	Error               string
	Result              string
	InputTokens         int
	OutputTokens        int
	EstimatedCost       float64
}

// NewQueue creates a new job queue
func NewQueue(database *sql.DB, workers int) *Queue {
	context, cancel := context.WithCancel(context.Background())
	return &Queue{
		database:    database,
		workers:     workers,
		context:     context,
		cancel:      cancel,
		handlers:    make(map[string]JobHandler),
		subscribers: make(map[string][]chan JobUpdate),
	}
}

// RegisterHandler registers a handler for a specific job type
func (queue *Queue) RegisterHandler(jobType string, handler JobHandler) {
	queue.handlers[jobType] = handler
}

// Start begins processing jobs
func (queue *Queue) Start() {
	for i := 0; i < queue.workers; i++ {
		queue.waitGroup.Add(1)
		go queue.worker(i)
	}
	slog.Info("Job queue started", "workers", queue.workers)
}

// Stop gracefully shuts down the job queue
func (queue *Queue) Stop() {
	slog.Info("Stopping job queue...")
	queue.cancel()
	queue.waitGroup.Wait()
	slog.Info("Job queue stopped")
}

// Enqueue creates a new job and adds it to the queue
func (queue *Queue) Enqueue(jobType string, payload interface{}) (string, error) {
	jobID := uuid.New().String()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = queue.database.Exec(`
		INSERT INTO jobs (id, type, status, progress, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, jobType, models.JobStatusPending, 0, string(payloadJSON), time.Now())

	if err != nil {
		return "", fmt.Errorf("failed to insert job: %w", err)
	}

	slog.Info("Enqueued job", "jobID", jobID, "type", jobType)
	return jobID, nil
}

// Subscribe subscribes to updates for a specific job
func (queue *Queue) Subscribe(jobID string) <-chan JobUpdate {
	queue.subscribersMutex.Lock()
	defer queue.subscribersMutex.Unlock()

	channel := make(chan JobUpdate, 10)
	queue.subscribers[jobID] = append(queue.subscribers[jobID], channel)
	return channel
}

// Unsubscribe removes a subscription
func (queue *Queue) Unsubscribe(jobID string, channel <-chan JobUpdate) {
	queue.subscribersMutex.Lock()
	defer queue.subscribersMutex.Unlock()

	subscribersList := queue.subscribers[jobID]
	for i, subscriber := range subscribersList {
		if subscriber == channel {
			queue.subscribers[jobID] = append(subscribersList[:i], subscribersList[i+1:]...)
			close(subscriber)
			break
		}
	}

	if len(queue.subscribers[jobID]) == 0 {
		delete(queue.subscribers, jobID)
	}
}

// publishUpdate sends an update to all subscribers of a job
func (queue *Queue) publishUpdate(update JobUpdate) {
	queue.subscribersMutex.RLock()
	defer queue.subscribersMutex.RUnlock()

	if subscribersList, ok := queue.subscribers[update.JobID]; ok {
		for _, channel := range subscribersList {
			select {
			case channel <- update:
			default:
				// Channel full, skip
			}
		}
	}
}

// worker processes jobs from the queue
func (queue *Queue) worker(id int) {
	defer queue.waitGroup.Done()
	slog.Debug("Worker started", "workerID", id)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-queue.context.Done():
			slog.Debug("Worker stopping", "workerID", id)
			return
		case <-ticker.C:
			queue.processNextJob(id)
		}
	}
}

// processNextJob picks up and processes the next pending job
func (queue *Queue) processNextJob(workerID int) {
	transaction, err := queue.database.Begin()
	if err != nil {
		slog.Error("Worker failed to begin transaction", "workerID", workerID, "error", err)
		return
	}
	defer transaction.Rollback()

	// Find and lock a pending job
	var job models.Job
	var metadataJSON sql.NullString
	err = transaction.QueryRow(`
		SELECT id, type, status, progress, progress_message_text, payload, metadata, created_at
		FROM jobs
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT 1
	`, models.JobStatusPending).Scan(
		&job.ID, &job.Type, &job.Status, &job.Progress,
		&job.ProgressMessageText, &job.Payload, &metadataJSON, &job.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return // No pending jobs
	}
	if err != nil {
		slog.Error("Worker failed to query job", "workerID", workerID, "error", err)
		return
	}

	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &job.Metadata)
	}

	// Mark job as running
	now := time.Now()
	_, err = transaction.Exec(`
		UPDATE jobs
		SET status = ?, started_at = ?
		WHERE id = ?
	`, models.JobStatusRunning, now, job.ID)

	if err != nil {
		slog.Error("Worker failed to update job status", "workerID", workerID, "error", err)
		return
	}

	if err := transaction.Commit(); err != nil {
		slog.Error("Worker failed to commit transaction", "workerID", workerID, "error", err)
		return
	}

	job.Status = models.JobStatusRunning
	job.StartedAt = &now

	slog.Info("Worker processing job", "workerID", workerID, "jobID", job.ID, "type", job.Type)

	// Publish initial update
	queue.publishUpdate(JobUpdate{
		JobID:    job.ID,
		Status:   models.JobStatusRunning,
		Progress: 0,
	})

	// Execute job
	queue.executeJob(&job)
}

// executeJob runs the job handler and updates the database
func (queue *Queue) executeJob(job *models.Job) {
	handler, ok := queue.handlers[job.Type]
	if !ok {
		queue.failJob(job.ID, fmt.Sprintf("no handler registered for job type: %s", job.Type))
		return
	}

	// Create update function
	updateProgress := func(progress int, message string, metadata any, metrics JobMetrics) {
		var metadataJSON []byte
		if metadata != nil {
			metadataJSON, _ = json.Marshal(metadata)
		}

		_, err := queue.database.Exec(`
			UPDATE jobs
			SET progress = ?, progress_message_text = ?, metadata = ?, input_tokens = ?, output_tokens = ?, estimated_cost = ?
			WHERE id = ?
		`, progress, message, string(metadataJSON), metrics.InputTokens, metrics.OutputTokens, metrics.EstimatedCost, job.ID)

		if err != nil {
			slog.Error("Failed to update job progress", "error", err)
		}

		queue.publishUpdate(JobUpdate{
			JobID:               job.ID,
			Status:              models.JobStatusRunning,
			Progress:            progress,
			ProgressMessageText: message,
			Metadata:            metadata,
			InputTokens:         metrics.InputTokens,
			OutputTokens:        metrics.OutputTokens,
			EstimatedCost:       metrics.EstimatedCost,
		})
	}

	// Execute handler
	context, cancel := context.WithCancel(queue.context)
	defer cancel()

	err := handler(context, job, updateProgress)

	if err != nil {
		queue.failJob(job.ID, err.Error())
		return
	}

	queue.completeJob(job.ID, job.Result)
}

// completeJob marks a job as completed
func (queue *Queue) completeJob(jobID, result string) {
	now := time.Now()
	_, err := queue.database.Exec(`
		UPDATE jobs
		SET status = ?, progress = 100, completed_at = ?, result = ?
		WHERE id = ?
	`, models.JobStatusCompleted, now, result, jobID)

	if err != nil {
		slog.Error("Failed to mark job as completed", "error", err)
		return
	}

	slog.Info("Job completed successfully", "jobID", jobID)

	queue.publishUpdate(JobUpdate{
		JobID:    jobID,
		Status:   models.JobStatusCompleted,
		Progress: 100,
		Result:   result,
	})
}

// failJob marks a job as failed
func (queue *Queue) failJob(jobID, errorMsg string) {
	now := time.Now()
	_, err := queue.database.Exec(`
		UPDATE jobs
		SET status = ?, completed_at = ?, error = ?
		WHERE id = ?
	`, models.JobStatusFailed, now, errorMsg, jobID)

	if err != nil {
		slog.Error("Failed to mark job as failed", "error", err)
		return
	}

	slog.Error("Job failed", "jobID", jobID, "error", errorMsg)

	queue.publishUpdate(JobUpdate{
		JobID:  jobID,
		Status: models.JobStatusFailed,
		Error:  errorMsg,
	})
}

// GetJob retrieves a job by ID
func (queue *Queue) GetJob(jobID string) (*models.Job, error) {
	var job models.Job
	var startedAt, completedAt sql.NullTime
	var metadataJSON sql.NullString

	err := queue.database.QueryRow(`
		SELECT id, type, status, progress, progress_message_text, payload, result, error, metadata,
		       input_tokens, output_tokens, estimated_cost, created_at, started_at, completed_at
		FROM jobs
		WHERE id = ?
	`, jobID).Scan(
		&job.ID, &job.Type, &job.Status, &job.Progress, &job.ProgressMessageText,
		&job.Payload, &job.Result, &job.Error, &metadataJSON, &job.InputTokens, &job.OutputTokens, &job.EstimatedCost,
		&job.CreatedAt, &startedAt, &completedAt,
	)

	if err != nil {
		return nil, err
	}

	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &job.Metadata)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

// CancelJob cancels a running or pending job
func (queue *Queue) CancelJob(jobID string) error {
	_, err := queue.database.Exec(`
		UPDATE jobs
		SET status = ?, completed_at = ?
		WHERE id = ? AND status IN (?, ?)
	`, models.JobStatusCancelled, time.Now(), jobID, models.JobStatusPending, models.JobStatusRunning)

	if err != nil {
		return err
	}

	queue.publishUpdate(JobUpdate{
		JobID:  jobID,
		Status: models.JobStatusCancelled,
	})

	return nil
}
