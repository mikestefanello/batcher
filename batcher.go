package batcher

import (
	"sync"
	"time"
)

// Batcher allows for the grouping and batching of items to be processed
type Batcher[T any] struct {
	// config stores configuration for the Batcher
	config Config[T]

	// queue stores the job for each group
	queue map[string]*job[T]

	// itemCount tracks the total amount of items across all groups
	itemCount int

	// jobs sends the jobs to the workers to be processed
	jobs chan *job[T]

	// results allows the workers to tell the Batcher that a job was complete
	results chan struct{}

	// ticker triggers the processing of the queue periodically
	ticker *time.Ticker

	// addMutex prevents concurrent additions to the queue
	addMutex sync.Mutex

	// processingMutex prevents processing the queue when it is already being processed
	processingMutex sync.Mutex

	// processing indicates if the queue is currently being processed
	processing bool

	// shutdown indicates if the Batcher has been shutdown
	shutdown bool
}

// Add adds an item to a given group and blocks until it has been accepted but not processed
func (b *Batcher[T]) Add(group string, item T) {
	b.addMutex.Lock()
	defer b.addMutex.Unlock()

	if _, exists := b.queue[group]; !exists {
		b.queue[group] = &job[T]{
			group: group,
			items: make([]T, 0, 1),
		}
	}

	b.queue[group].items = append(b.queue[group].items, item)
	b.itemCount++

	if b.isQueueFull() {
		b.processQueue()
	}
}

// isQueueFull indicates if the queue is full either based on group or item count
func (b *Batcher[T]) isQueueFull() bool {
	switch {
	case b.config.GroupCountThreshold > 0 && len(b.queue) >= b.config.GroupCountThreshold:
		return true
	case b.config.ItemCountThreshold > 0 && b.itemCount >= b.config.ItemCountThreshold:
		return true
	}
	return false
}

// processQueue processes the jobs in the queue concurrently and blocks until complete
func (b *Batcher[T]) processQueue() {
	b.processingMutex.Lock()
	b.processing = true

	defer func() {
		b.ticker.Reset(b.config.DelayThreshold)
		b.processing = false
		b.itemCount = 0
		b.processingMutex.Unlock()
	}()

	count := len(b.queue)
	if count == 0 {
		return
	}

	// Pass the jobs to the worker pool
	for k, j := range b.queue {
		b.jobs <- j
		delete(b.queue, k)
	}

	// Wait until all workers are done
	for i := 0; i < count; i++ {
		<-b.results
	}
}

// jobWorker consumes jobs from the channel and passes them to the processor
func (b *Batcher[T]) jobWorker() {
	for j := range b.jobs {
		b.config.Processor(j.group, j.items)

		// Tell the queue processor that we're done
		b.results <- struct{}{}
	}
}

// tickerListener listens to ticks from the ticker in order to process the queue periodically
func (b *Batcher[T]) tickerListener() {
	for {
		select {
		case <-b.ticker.C:
			if !b.processing {
				b.addMutex.Lock()
				b.processQueue()
				b.addMutex.Unlock()
			}
		}
	}
}

// Shutdown shuts down the Batcher and prevents further processing
func (b *Batcher[T]) Shutdown() {
	b.ticker.Stop()
	b.processQueue()
	b.processingMutex.Lock()
	close(b.jobs)
	close(b.results)
	b.shutdown = true
}

// NewBatcher creates a new Batcher from given configuration
func NewBatcher[T any](cfg Config[T]) (*Batcher[T], error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	b := &Batcher[T]{
		config:  cfg,
		queue:   make(map[string]*job[T], cfg.GroupCountThreshold),
		jobs:    make(chan *job[T], cfg.NumGoroutines),
		results: make(chan struct{}, cfg.GroupCountThreshold),
		ticker:  time.NewTicker(cfg.DelayThreshold),
	}

	go b.tickerListener()

	for i := 0; i < cfg.NumGoroutines; i++ {
		go b.jobWorker()
	}

	return b, nil
}
