package main

import (
	"fmt"
	"testing"
	"time"
)

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

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		batch.Add(fmt.Sprint(b.N%5), n)
	}
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

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		batch.Add(fmt.Sprint(b.N%5), n)
	}
}
