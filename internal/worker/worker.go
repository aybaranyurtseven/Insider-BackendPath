package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type Job interface {
	Execute(ctx context.Context) error
	GetID() string
	GetType() string
}

type JobResult struct {
	JobID string
	Error error
}

type WorkerPool struct {
	workerCount   int
	jobQueue      chan Job
	resultChannel chan JobResult
	workers       []*Worker
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	metrics       *WorkerMetrics
}

type Worker struct {
	id         int
	jobQueue   chan Job
	resultChan chan JobResult
	ctx        context.Context
	metrics    *WorkerMetrics
}

type WorkerMetrics struct {
	JobsProcessed    int64
	JobsSuccessful   int64
	JobsFailed       int64
	JobsInProgress   int64
	TotalProcessTime time.Duration
	mu               sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workerCount:   workerCount,
		jobQueue:      make(chan Job, queueSize),
		resultChannel: make(chan JobResult, queueSize),
		workers:       make([]*Worker, workerCount),
		ctx:           ctx,
		cancel:        cancel,
		metrics:       &WorkerMetrics{},
	}
}

// Start initializes and starts all workers
func (wp *WorkerPool) Start() {
	log.Info().Int("worker_count", wp.workerCount).Msg("Starting worker pool")

	for i := 0; i < wp.workerCount; i++ {
		worker := &Worker{
			id:         i,
			jobQueue:   wp.jobQueue,
			resultChan: wp.resultChannel,
			ctx:        wp.ctx,
			metrics:    wp.metrics,
		}
		wp.workers[i] = worker

		wp.wg.Add(1)
		go worker.start(&wp.wg)
	}

	// Start result processor
	go wp.processResults()
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() {
	log.Info().Msg("Stopping worker pool")

	// Close job queue to signal workers to stop accepting new jobs
	close(wp.jobQueue)

	// Wait for all workers to finish
	wp.wg.Wait()

	// Cancel context and close result channel
	wp.cancel()
	close(wp.resultChannel)

	log.Info().Msg("Worker pool stopped")
}

// SubmitJob submits a job to the worker pool
func (wp *WorkerPool) SubmitJob(job Job) error {
	select {
	case wp.jobQueue <- job:
		log.Debug().Str("job_id", job.GetID()).Str("job_type", job.GetType()).Msg("Job submitted")
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

// GetMetrics returns worker pool metrics
func (wp *WorkerPool) GetMetrics() WorkerMetrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	return WorkerMetrics{
		JobsProcessed:    atomic.LoadInt64(&wp.metrics.JobsProcessed),
		JobsSuccessful:   atomic.LoadInt64(&wp.metrics.JobsSuccessful),
		JobsFailed:       atomic.LoadInt64(&wp.metrics.JobsFailed),
		JobsInProgress:   atomic.LoadInt64(&wp.metrics.JobsInProgress),
		TotalProcessTime: wp.metrics.TotalProcessTime,
	}
}

// processResults processes job results
func (wp *WorkerPool) processResults() {
	for result := range wp.resultChannel {
		if result.Error != nil {
			log.Error().
				Str("job_id", result.JobID).
				Err(result.Error).
				Msg("Job failed")
			atomic.AddInt64(&wp.metrics.JobsFailed, 1)
		} else {
			log.Debug().
				Str("job_id", result.JobID).
				Msg("Job completed successfully")
			atomic.AddInt64(&wp.metrics.JobsSuccessful, 1)
		}
	}
}

// start starts the worker
func (w *Worker) start(wg *sync.WaitGroup) {
	defer wg.Done()

	log.Debug().Int("worker_id", w.id).Msg("Worker started")

	for {
		select {
		case job, ok := <-w.jobQueue:
			if !ok {
				log.Debug().Int("worker_id", w.id).Msg("Worker stopped - job queue closed")
				return
			}

			w.processJob(job)

		case <-w.ctx.Done():
			log.Debug().Int("worker_id", w.id).Msg("Worker stopped - context cancelled")
			return
		}
	}
}

// processJob processes a single job
func (w *Worker) processJob(job Job) {
	atomic.AddInt64(&w.metrics.JobsInProgress, 1)
	atomic.AddInt64(&w.metrics.JobsProcessed, 1)
	defer atomic.AddInt64(&w.metrics.JobsInProgress, -1)

	startTime := time.Now()

	log.Debug().
		Int("worker_id", w.id).
		Str("job_id", job.GetID()).
		Str("job_type", job.GetType()).
		Msg("Processing job")

	err := job.Execute(w.ctx)

	duration := time.Since(startTime)
	w.metrics.mu.Lock()
	w.metrics.TotalProcessTime += duration
	w.metrics.mu.Unlock()

	result := JobResult{
		JobID: job.GetID(),
		Error: err,
	}

	select {
	case w.resultChan <- result:
	case <-w.ctx.Done():
		log.Warn().Str("job_id", job.GetID()).Msg("Could not send job result - context cancelled")
	}
}

// BatchProcessor processes jobs in batches
type BatchProcessor struct {
	workerPool *WorkerPool
	batchSize  int
	flushTime  time.Duration
	jobs       []Job
	mu         sync.Mutex
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(workerPool *WorkerPool, batchSize int, flushTime time.Duration) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		workerPool: workerPool,
		batchSize:  batchSize,
		flushTime:  flushTime,
		jobs:       make([]Job, 0, batchSize),
		ticker:     time.NewTicker(flushTime),
		ctx:        ctx,
		cancel:     cancel,
	}

	go bp.flushPeriodically()

	return bp
}

// AddJob adds a job to the batch
func (bp *BatchProcessor) AddJob(job Job) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.jobs = append(bp.jobs, job)

	if len(bp.jobs) >= bp.batchSize {
		return bp.flushBatch()
	}

	return nil
}

// flushBatch flushes the current batch of jobs
func (bp *BatchProcessor) flushBatch() error {
	if len(bp.jobs) == 0 {
		return nil
	}

	batchJob := NewBatchJob(bp.jobs)
	bp.jobs = bp.jobs[:0] // Clear the slice but keep capacity

	return bp.workerPool.SubmitJob(batchJob)
}

// flushPeriodically flushes jobs periodically
func (bp *BatchProcessor) flushPeriodically() {
	for {
		select {
		case <-bp.ticker.C:
			bp.mu.Lock()
			bp.flushBatch()
			bp.mu.Unlock()
		case <-bp.ctx.Done():
			bp.ticker.Stop()
			return
		}
	}
}

// Stop stops the batch processor
func (bp *BatchProcessor) Stop() {
	bp.cancel()

	bp.mu.Lock()
	bp.flushBatch()
	bp.mu.Unlock()
}

// BatchJob represents a batch of jobs
type BatchJob struct {
	id   string
	jobs []Job
}

// NewBatchJob creates a new batch job
func NewBatchJob(jobs []Job) *BatchJob {
	return &BatchJob{
		id:   fmt.Sprintf("batch-%d", time.Now().UnixNano()),
		jobs: jobs,
	}
}

// Execute executes all jobs in the batch
func (bj *BatchJob) Execute(ctx context.Context) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(bj.jobs))

	for _, job := range bj.jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			if err := j.Execute(ctx); err != nil {
				errorChan <- fmt.Errorf("job %s failed: %w", j.GetID(), err)
			}
		}(job)
	}

	wg.Wait()
	close(errorChan)

	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch job failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// GetID returns the batch job ID
func (bj *BatchJob) GetID() string {
	return bj.id
}

// GetType returns the job type
func (bj *BatchJob) GetType() string {
	return "batch"
}
