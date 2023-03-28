package batcher

// job represents each job that is sent to a batch processor
type job[T any] struct {
	// group is the name of the batch group
	group string

	// items are all items that were batched in to the given group
	items []T
}
