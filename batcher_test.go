package batcher

import (
	"fmt"
	"testing"
	"time"
)

func TestNewBatcher(t *testing.T) {
	// Invalid config
	_, err := NewBatcher[int](Config[int]{})
	if err == nil {
		t.Error("expected error")
	}

	// Valid config
	b, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 1,
		DelayThreshold:      time.Minute,
		NumGoroutines:       1,
		Processor:           func(key string, items []int) {},
	})
	if err != nil {
		t.Error("expected no error")
	}

	b.Shutdown()
}

func TestBatcher_Add(t *testing.T) {
	b, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 10,
		DelayThreshold:      time.Minute,
		NumGoroutines:       1,
		Processor:           func(key string, items []int) {},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer b.Shutdown()

	assertQueue := func(expectedGroups, expectedItems int) {
		if len(b.queue) != expectedGroups {
			t.Errorf("expected %d groups, got %d", expectedGroups, len(b.queue))
		}

		if b.itemCount != expectedItems {
			t.Errorf("expected %d items from itemCount, got %d", expectedItems, b.itemCount)
		}

		var count int
		for _, j := range b.queue {
			count += len(j.items)
		}

		if count != expectedItems {
			t.Errorf("expected %d items from queue, got %d", expectedItems, count)
		}
	}

	assertGroup := func(group string, expectedToExist bool, expectedItems int) {
		j, exists := b.queue[group]

		if exists != expectedToExist {
			t.Errorf("expected existance of group %s failed: expected %v, got %v", group, expectedToExist, exists)
		}

		if exists && (len(j.items) != expectedItems) {
			t.Errorf("expected %d items in %s group, got %d", expectedItems, group, len(j.items))
		}
	}

	b.Add("group1", 1)
	assertQueue(1, 1)

	b.Add("group1", 1)
	assertQueue(1, 2)

	b.Add("group2", 1)
	assertQueue(2, 3)

	b.Add("group2", 1)
	assertQueue(2, 4)

	assertGroup("group1", true, 2)
	assertGroup("group2", true, 2)
	assertGroup("group3", false, 0)
}

func TestBatcher_DelayTrigger(t *testing.T) {
	var counter int

	b, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 3,
		ItemCountThreshold:  5,
		DelayThreshold:      10 * time.Millisecond,
		NumGoroutines:       1,
		Processor: func(key string, items []int) {
			counter++
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %d", len(items))
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer b.Shutdown()

	b.Add("group1", 1)
	b.Add("group1", 2)
	b.Add("group2", 1)
	b.Add("group2", 2)

	time.Sleep(11 * time.Millisecond)

	if len(b.queue) != 0 || counter != 2 {
		t.Error("queue did not process after the delay elapsed")
	}
}

func TestBatcher_GroupTrigger(t *testing.T) {
	var counter int

	b, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 3,
		ItemCountThreshold:  5,
		DelayThreshold:      10 * time.Minute,
		NumGoroutines:       2,
		Processor: func(key string, items []int) {
			counter++
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer b.Shutdown()

	b.Add("group1", 1)
	b.Add("group1", 2)
	b.Add("group2", 1)
	b.Add("group2", 2)

	if len(b.queue) == 0 || counter > 0 {
		t.Error("queue processed before the group threshold was met")
	}

	b.Add("group3", 2)

	if len(b.queue) != 0 || counter != 3 {
		t.Error("queue did not process after the group threshold was met")
	}
}

func TestBatcher_ItemTrigger(t *testing.T) {
	var counter int

	b, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 3,
		ItemCountThreshold:  3,
		DelayThreshold:      10 * time.Minute,
		NumGoroutines:       2,
		Processor: func(key string, items []int) {
			counter++
			if len(items) != 3 {
				t.Errorf("expected 3 items, got %d", len(items))
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer b.Shutdown()

	b.Add("group1", 1)
	b.Add("group1", 2)

	if len(b.queue) == 0 || counter > 0 {
		t.Error("queue processed before the item threshold was met")
	}

	b.Add("group1", 3)

	if len(b.queue) != 0 || counter != 1 {
		t.Error("queue did not process after the item threshold was met")
	}
}

func TestBatcher_Shutdown(t *testing.T) {
	var counter int

	b, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 3,
		ItemCountThreshold:  3,
		DelayThreshold:      10 * time.Millisecond,
		NumGoroutines:       1,
		Processor: func(key string, items []int) {
			counter++
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	b.Add("group1", 1)
	b.Shutdown()

	// Shutting down should process the queue
	if counter != 1 {
		t.Error("queue did not process on shutdown")
	}

	// Add another item and wait for the delay to ensure the queue is no longer processing
	b.Add("group2", 2)
	time.Sleep(11 * time.Millisecond)

	if counter != 1 {
		t.Error("queue is still processing after shutdown")
	}
}

func BenchmarkBatcher_Add(b *testing.B) {
	batch, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: b.N + 1,
		ItemCountThreshold:  b.N + 1,
		DelayThreshold:      100 * time.Minute,
		NumGoroutines:       1,
		Processor:           func(key string, items []int) {},
	})

	if err != nil {
		b.Fatal(err)
	}

	defer batch.Shutdown()

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		batch.Add(fmt.Sprint(b.N%10), n)
	}
}

func BenchmarkBatcher_processQueue(b *testing.B) {
	batch, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 5,
		ItemCountThreshold:  5,
		DelayThreshold:      100 * time.Minute,
		NumGoroutines:       1,
		Processor:           func(key string, items []int) {},
	})

	if err != nil {
		b.Fatal(err)
	}

	defer batch.Shutdown()

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		batch.Add(fmt.Sprint(b.N%5), n)
	}

	batch.Shutdown()
}

func BenchmarkBatcher_processQueue_concurrent(b *testing.B) {
	batch, err := NewBatcher[int](Config[int]{
		GroupCountThreshold: 5,
		ItemCountThreshold:  5,
		DelayThreshold:      100 * time.Minute,
		NumGoroutines:       3,
		Processor:           func(key string, items []int) {},
	})

	if err != nil {
		b.Fatal(err)
	}

	defer batch.Shutdown()

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		batch.Add(fmt.Sprint(b.N%5), n)
	}
}
