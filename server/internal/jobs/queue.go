package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"lectures/internal/models"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// Queue manages background job processing
type Queue struct {
	database           *sql.DB
	workers            int
	context            context.Context
	cancel             context.CancelFunc
	waitGroup          sync.WaitGroup
	handlers           map[string]JobHandler
	subscribers        map[string][]chan JobUpdate
	subscribersMutex   sync.RWMutex
	heavyTaskSemaphore chan struct{}
}

// JobHandler is a function that processes a specific job type
type JobHandler func(context context.Context, job *models.Job, updateProgress func(progress int, message string, metadata any, metrics models.JobMetrics)) error

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
	jobContext, cancel := context.WithCancel(context.Background())
	return &Queue{
		database:           database,
		workers:            workers,
		context:            jobContext,
		cancel:             cancel,
		handlers:           make(map[string]JobHandler),
		subscribers:        make(map[string][]chan JobUpdate),
		heavyTaskSemaphore: make(chan struct{}, 1), // Only 1 heavy task at a time
	}
}

// RegisterHandler registers a handler for a specific job type
func (queue *Queue) RegisterHandler(jobType string, handler JobHandler) {
	queue.handlers[jobType] = handler
}

// Start begins processing jobs
func (queue *Queue) Start() {
	queue.recoverJobs()
	for index := 0; index < queue.workers; index++ {
		queue.waitGroup.Add(1)
		go queue.worker(index)
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

// recoverJobs marks jobs that were left in RUNNING state (e.g. server crash) as FAILED
func (queue *Queue) recoverJobs() {
	result, err := queue.database.Exec(`
		UPDATE jobs 
		SET status = ?, error = 'Server restarted while task was running', completed_at = ?
		WHERE status = ?
	`, models.JobStatusFailed, time.Now(), models.JobStatusRunning)

	if err == nil {
		rows, _ := result.RowsAffected()
		if rows > 0 {
			slog.Info("Recovered stuck jobs", "count", rows)
		}
	}
}

// Enqueue creates a new job and adds it to the queue
func (queue *Queue) Enqueue(userID string, jobType string, payload interface{}, courseID, lectureID string) (string, error) {
	jobID, _ := gonanoid.New()

	payloadJSON, marshalingError := json.Marshal(payload)
	if marshalingError != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", marshalingError)
	}

	var courseIDValue interface{} = courseID
	if courseID == "" {
		courseIDValue = nil
	}
	var lectureIDValue interface{} = lectureID
	if lectureID == "" {
		lectureIDValue = nil
	}

	_, executionError := queue.database.Exec(`
		INSERT INTO jobs (id, user_id, course_id, lecture_id, type, status, progress, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, jobID, userID, courseIDValue, lectureIDValue, jobType, models.JobStatusPending, 0, string(payloadJSON), time.Now())

	if executionError != nil {
		return "", fmt.Errorf("failed to insert job: %w", executionError)
	}

	slog.Info("Enqueued job", "jobID", jobID, "type", jobType, "userID", userID, "courseID", courseID, "lectureID", lectureID)
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
	for index, subscriberChannel := range subscribersList {
		if subscriberChannel == channel {
			queue.subscribers[jobID] = append(subscribersList[:index], subscribersList[index+1:]...)
			close(subscriberChannel)
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
func (queue *Queue) worker(workerID int) {
	defer queue.waitGroup.Done()
	slog.Debug("Worker started", "workerID", workerID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-queue.context.Done():
			slog.Debug("Worker stopping", "workerID", workerID)
			return
		case <-ticker.C:
			queue.processNextJob(workerID)
		}
	}
}

// processNextJob picks up and processes the next pending job
func (queue *Queue) processNextJob(workerID int) {
	transaction, transactionError := queue.database.Begin()
	if transactionError != nil {
		// Transient lock errors are normal when multiple workers compete
		if strings.Contains(transactionError.Error(), "database is locked") {
			return // Silently retry next tick
		}
		slog.Error("Worker failed to begin transaction", "workerID", workerID, "error", transactionError)
		return
	}
	defer transaction.Rollback()

	// Find and lock a pending job
	var job models.Job
	var metadataJSON, progressMessageText, courseID, lectureID sql.NullString
	queryError := transaction.QueryRow(`
		SELECT id, user_id, course_id, lecture_id, type, status, progress, progress_message_text, payload, metadata, created_at
		FROM jobs
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT 1
	`, models.JobStatusPending).Scan(
		&job.ID, &job.UserID, &courseID, &lectureID, &job.Type, &job.Status, &job.Progress,
		&progressMessageText, &job.Payload, &metadataJSON, &job.CreatedAt,
	)

	if queryError == sql.ErrNoRows {
		return // No pending jobs
	}
	if queryError != nil {
		// Transient lock errors are normal when multiple workers compete
		if strings.Contains(queryError.Error(), "database is locked") {
			return // Silently retry next tick
		}
		slog.Error("Worker failed to query job", "workerID", workerID, "error", queryError)
		return
	}

	if courseID.Valid {
		job.CourseID = courseID.String
	}
	if lectureID.Valid {
		job.LectureID = lectureID.String
	}

	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &job.Metadata)
	}
	if progressMessageText.Valid {
		job.ProgressMessageText = progressMessageText.String
	}

	// Mark job as running
	now := time.Now()
	_, executionError := transaction.Exec(`
		UPDATE jobs
		SET status = ?, started_at = ?
		WHERE id = ?
	`, models.JobStatusRunning, now, job.ID)

	if executionError != nil {
		// Transient lock errors are normal when multiple workers compete
		if strings.Contains(executionError.Error(), "database is locked") {
			return // Silently retry next tick
		}
		slog.Error("Worker failed to update job status", "workerID", workerID, "error", executionError)
		return
	}

	if commitError := transaction.Commit(); commitError != nil {
		// Transient lock errors are normal when multiple workers compete
		if strings.Contains(commitError.Error(), "database is locked") {
			return // Silently retry next tick
		}
		slog.Error("Worker failed to commit transaction", "workerID", workerID, "error", commitError)
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

	// Handle resource-intensive tasks sequentially
	isHeavyTask := job.Type == models.JobTypeTranscribeMedia || job.Type == models.JobTypeIngestDocuments
	if isHeavyTask {
		select {
		case queue.heavyTaskSemaphore <- struct{}{}:
			defer func() { <-queue.heavyTaskSemaphore }()
		case <-queue.context.Done():
			return
		}
	}

	// Create update function
	updateProgress := func(progress int, message string, metadata any, metrics models.JobMetrics) {
		var metadataJSON []byte
		if metadata != nil {
			metadataJSON, _ = json.Marshal(metadata)
		}

		_, executionError := queue.database.Exec(`
			UPDATE jobs
			SET progress = ?, progress_message_text = ?, metadata = ?, input_tokens = ?, output_tokens = ?, estimated_cost = ?
			WHERE id = ?
		`, progress, message, string(metadataJSON), metrics.InputTokens, metrics.OutputTokens, metrics.EstimatedCost, job.ID)

		if executionError != nil {
			slog.Error("Failed to update job progress", "error", executionError)
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
	jobContext, cancelFunc := context.WithCancel(queue.context)
	defer cancelFunc()

	executionError := handler(jobContext, job, updateProgress)

	if executionError != nil {
		queue.failJob(job.ID, executionError.Error())
		return
	}

	queue.completeJob(job.ID, job.Result)
}

// completeJob marks a job as completed
func (queue *Queue) completeJob(jobID, result string) {
	now := time.Now()
	_, executionError := queue.database.Exec(`
		UPDATE jobs
		SET status = ?, progress = 100, completed_at = ?, result = ?
		WHERE id = ?
	`, models.JobStatusCompleted, now, result, jobID)

	if executionError != nil {
		slog.Error("Failed to mark job as completed", "error", executionError)
		return
	}

	// Get final metrics for logging
	var inputTokens, outputTokens int
	var estimatedCost float64
	queue.database.QueryRow("SELECT input_tokens, output_tokens, estimated_cost FROM jobs WHERE id = ?", jobID).Scan(&inputTokens, &outputTokens, &estimatedCost)

	slog.Info("Job completed successfully",
		"jobID", jobID,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"estimated_cost_usd", estimatedCost,
		"total_tokens", inputTokens+outputTokens)

	queue.publishUpdate(JobUpdate{
		JobID:         jobID,
		Status:        models.JobStatusCompleted,
		Progress:      100,
		Result:        result,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		EstimatedCost: estimatedCost,
	})
}

// failJob marks a job as failed
func (queue *Queue) failJob(jobID, errorMsg string) {
	now := time.Now()
	_, executionError := queue.database.Exec(`
		UPDATE jobs
		SET status = ?, completed_at = ?, error = ?
		WHERE id = ?
	`, models.JobStatusFailed, now, errorMsg, jobID)

	if executionError != nil {
		slog.Error("Failed to mark job as failed", "error", executionError)
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
	var startedAtTime, completedAtTime sql.NullTime
	var metadataJSON, progressMessageText, result, errorMsg, courseID, lectureID sql.NullString

	queryError := queue.database.QueryRow(`
		SELECT id, user_id, course_id, lecture_id, type, status, progress, progress_message_text, payload, result, error, metadata,
		       input_tokens, output_tokens, estimated_cost, created_at, started_at, completed_at
		FROM jobs
		WHERE id = ?
	`, jobID).Scan(
		&job.ID, &job.UserID, &courseID, &lectureID, &job.Type, &job.Status, &job.Progress, &progressMessageText,
		&job.Payload, &result, &errorMsg, &metadataJSON, &job.InputTokens, &job.OutputTokens, &job.EstimatedCost,
		&job.CreatedAt, &startedAtTime, &completedAtTime,
	)

	if queryError != nil {
		return nil, queryError
	}

	if courseID.Valid {
		job.CourseID = courseID.String
	}
	if lectureID.Valid {
		job.LectureID = lectureID.String
	}

	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &job.Metadata)
	}
	if progressMessageText.Valid {
		job.ProgressMessageText = progressMessageText.String
	}
	if result.Valid {
		job.Result = result.String
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}

	if startedAtTime.Valid {
		job.StartedAt = &startedAtTime.Time
	}
	if completedAtTime.Valid {
		job.CompletedAt = &completedAtTime.Time
	}

	return &job, nil
}

// CancelJob cancels a running or pending job
func (queue *Queue) CancelJob(jobID string) error {
	_, executionError := queue.database.Exec(`
		UPDATE jobs
		SET status = ?, completed_at = ?
		WHERE id = ? AND status IN (?, ?)
	`, models.JobStatusCancelled, time.Now(), jobID, models.JobStatusPending, models.JobStatusRunning)

	if executionError != nil {
		return executionError
	}

	queue.publishUpdate(JobUpdate{
		JobID:  jobID,
		Status: models.JobStatusCancelled,
	})

	return nil
}
