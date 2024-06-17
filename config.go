package batcher

import (
	"errors"
	"time"
)

// Config stores configuration for a Batcher
type Config[T any] struct {
	// GroupCountThreshold is the amount of groups that will trigger the Batcher to process the queue.
	// It is required that this value be greater than zero.
	GroupCountThreshold int

	// ItemCountThreshold is the amount of items across all groups that will trigger the Batcher to process the queue
	ItemCountThreshold int

	// DelayThreshold is the amount of time to elapse that will trigger the Batcher to process the queue
	DelayThreshold time.Duration

	// NumGoroutines is the amount of goroutines to use to concurrently process the queue
	NumGoroutines int

	// Processor is a function callback to process the items in each group when the Batcher is processing the queue
	Processor func(group string, items []T)
}

// validate returns an error if the Config is not considered valid
func (c *Config[T]) validate() error {
	switch {
	case c.GroupCountThreshold < 1:
		return errors.New("must be greater than zero: GroupCountThreshold")
	case c.DelayThreshold < 1:
		return errors.New("must be greater than zero: DelayThreshold")
	case c.NumGoroutines < 1:
		return errors.New("must be greater than zero: NumGoroutines")
	case c.Processor == nil:
		return errors.New("missing field: Processor")
	}

	return nil
}
