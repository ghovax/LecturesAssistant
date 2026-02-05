package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
)

// Queue manages background job processing
type Queue struct {
	db          *sql.DB
	workers     int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	handlers    map[string]JobHandler
	subscribers map[string][]chan JobUpdate
	subMutex    sync.RWMutex
}

// JobHandler is a function that processes a specific job type
type JobHandler func(ctx context.Context, job *models.Job, updateFn func(progress int, message string)) error

// JobUpdate represents a job progress update
type JobUpdate struct {
	JobID               string
	Status              string
	Progress            int
	ProgressMessageText string
	Error               string
	Result              string
}

// NewQueue creates a new job queue
func NewQueue(db *sql.DB, workers int) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{
		db:          db,
		workers:     workers,
		ctx:         ctx,
		cancel:      cancel,
		handlers:    make(map[string]JobHandler),
		subscribers: make(map[string][]chan JobUpdate),
	}
}

// RegisterHandler registers a handler for a specific job type
func (q *Queue) RegisterHandler(jobType string, handler JobHandler) {
	q.handlers[jobType] = handler
}

// Start begins processing jobs
func (q *Queue) Start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
	log.Printf("Job queue started with %d workers", q.workers)
}

// Stop gracefully shuts down the job queue
func (q *Queue) Stop() {
	log.Println("Stopping job queue...")
	q.cancel()
	q.wg.Wait()
	log.Println("Job queue stopped")
}

// Enqueue creates a new job and adds it to the queue
func (q *Queue) Enqueue(jobType string, payload interface{}) (string, error) {
	jobID := uuid.New().String()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.db.Exec(`
		INSERT INTO jobs (id, type, status, progress, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, jobType, models.JobStatusPending, 0, string(payloadJSON), time.Now())

	if err != nil {
		return "", fmt.Errorf("failed to insert job: %w", err)
	}

	log.Printf("Enqueued job %s of type %s", jobID, jobType)
	return jobID, nil
}

// Subscribe subscribes to updates for a specific job
func (q *Queue) Subscribe(jobID string) <-chan JobUpdate {
	q.subMutex.Lock()
	defer q.subMutex.Unlock()

	ch := make(chan JobUpdate, 10)
	q.subscribers[jobID] = append(q.subscribers[jobID], ch)
	return ch
}

// Unsubscribe removes a subscription
func (q *Queue) Unsubscribe(jobID string, ch <-chan JobUpdate) {
	q.subMutex.Lock()
	defer q.subMutex.Unlock()

	subs := q.subscribers[jobID]
	for i, sub := range subs {
		if sub == ch {
			q.subscribers[jobID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}

	if len(q.subscribers[jobID]) == 0 {
		delete(q.subscribers, jobID)
	}
}

// publishUpdate sends an update to all subscribers of a job
func (q *Queue) publishUpdate(update JobUpdate) {
	q.subMutex.RLock()
	defer q.subMutex.RUnlock()

	if subs, ok := q.subscribers[update.JobID]; ok {
		for _, ch := range subs {
			select {
			case ch <- update:
			default:
				// Channel full, skip
			}
		}
	}
}

// worker processes jobs from the queue
func (q *Queue) worker(id int) {
	defer q.wg.Done()
	log.Printf("Worker %d started", id)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		case <-ticker.C:
			q.processNextJob(id)
		}
	}
}

// processNextJob picks up and processes the next pending job
func (q *Queue) processNextJob(workerID int) {
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("Worker %d: failed to begin transaction: %v", workerID, err)
		return
	}
	defer tx.Rollback()

	// Find and lock a pending job
	var job models.Job
	err = tx.QueryRow(`
		SELECT id, type, status, progress, progress_message_text, payload, created_at
		FROM jobs
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT 1
	`, models.JobStatusPending).Scan(
		&job.ID, &job.Type, &job.Status, &job.Progress,
		&job.ProgressMessageText, &job.Payload, &job.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return // No pending jobs
	}
	if err != nil {
		log.Printf("Worker %d: failed to query job: %v", workerID, err)
		return
	}

	// Mark job as running
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE jobs
		SET status = ?, started_at = ?
		WHERE id = ?
	`, models.JobStatusRunning, now, job.ID)

	if err != nil {
		log.Printf("Worker %d: failed to update job status: %v", workerID, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Worker %d: failed to commit transaction: %v", workerID, err)
		return
	}

	job.Status = models.JobStatusRunning
	job.StartedAt = &now

	log.Printf("Worker %d: processing job %s (type: %s)", workerID, job.ID, job.Type)

	// Publish initial update
	q.publishUpdate(JobUpdate{
		JobID:    job.ID,
		Status:   models.JobStatusRunning,
		Progress: 0,
	})

	// Execute job
	q.executeJob(&job)
}

// executeJob runs the job handler and updates the database
func (q *Queue) executeJob(job *models.Job) {
	handler, ok := q.handlers[job.Type]
	if !ok {
		q.failJob(job.ID, fmt.Sprintf("no handler registered for job type: %s", job.Type))
		return
	}

	// Create update function
	updateFn := func(progress int, message string) {
		_, err := q.db.Exec(`
			UPDATE jobs
			SET progress = ?, progress_message_text = ?
			WHERE id = ?
		`, progress, message, job.ID)

		if err != nil {
			log.Printf("Failed to update job progress: %v", err)
		}

		q.publishUpdate(JobUpdate{
			JobID:               job.ID,
			Status:              models.JobStatusRunning,
			Progress:            progress,
			ProgressMessageText: message,
		})
	}

	// Execute handler
	ctx, cancel := context.WithCancel(q.ctx)
	defer cancel()

	err := handler(ctx, job, updateFn)

	if err != nil {
		q.failJob(job.ID, err.Error())
		return
	}

	q.completeJob(job.ID, job.Result)
}

// completeJob marks a job as completed
func (q *Queue) completeJob(jobID, result string) {
	now := time.Now()
	_, err := q.db.Exec(`
		UPDATE jobs
		SET status = ?, progress = 100, completed_at = ?, result = ?
		WHERE id = ?
	`, models.JobStatusCompleted, now, result, jobID)

	if err != nil {
		log.Printf("Failed to mark job as completed: %v", err)
		return
	}

	log.Printf("Job %s completed successfully", jobID)

	q.publishUpdate(JobUpdate{
		JobID:    jobID,
		Status:   models.JobStatusCompleted,
		Progress: 100,
		Result:   result,
	})
}

// failJob marks a job as failed
func (q *Queue) failJob(jobID, errorMsg string) {
	now := time.Now()
	_, err := q.db.Exec(`
		UPDATE jobs
		SET status = ?, completed_at = ?, error = ?
		WHERE id = ?
	`, models.JobStatusFailed, now, errorMsg, jobID)

	if err != nil {
		log.Printf("Failed to mark job as failed: %v", err)
		return
	}

	log.Printf("Job %s failed: %s", jobID, errorMsg)

	q.publishUpdate(JobUpdate{
		JobID:  jobID,
		Status: models.JobStatusFailed,
		Error:  errorMsg,
	})
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(jobID string) (*models.Job, error) {
	var job models.Job
	var startedAt, completedAt sql.NullTime

	err := q.db.QueryRow(`
		SELECT id, type, status, progress, progress_message_text, payload, result, error,
		       created_at, started_at, completed_at
		FROM jobs
		WHERE id = ?
	`, jobID).Scan(
		&job.ID, &job.Type, &job.Status, &job.Progress, &job.ProgressMessageText,
		&job.Payload, &job.Result, &job.Error, &job.CreatedAt, &startedAt, &completedAt,
	)

	if err != nil {
		return nil, err
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
func (q *Queue) CancelJob(jobID string) error {
	_, err := q.db.Exec(`
		UPDATE jobs
		SET status = ?, completed_at = ?
		WHERE id = ? AND status IN (?, ?)
	`, models.JobStatusCancelled, time.Now(), jobID, models.JobStatusPending, models.JobStatusRunning)

	if err != nil {
		return err
	}

	q.publishUpdate(JobUpdate{
		JobID:  jobID,
		Status: models.JobStatusCancelled,
	})

	return nil
}
