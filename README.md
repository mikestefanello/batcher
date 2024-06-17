# Batcher

[![Go Report Card](https://goreportcard.com/badge/github.com/mikestefanello/batcher)](https://goreportcard.com/report/github.com/mikestefanello/batcher)
[![Test](https://github.com/mikestefanello/batcher/actions/workflows/test.yml/badge.svg)](https://github.com/mikestefanello/batcher/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/mikestefanello/batcher.svg)](https://pkg.go.dev/github.com/mikestefanello/batcher)
[![GoT](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev)

## Overview

_Batcher_ is a Go library that provides a type-safe, easy way to batch together arbitrary groups of items to be automatically and asynchronously processed. An item can be of any type that you want to pass to your processor. Items are queued within groups (using a _string_ as a name). If you do not have a need to separately group items, you can specify the same, or empty, group name for all items. Each group of items is separately sent to the processor callback that you specify.

The queue can be configured to automatically execute the batch operation under any of the following circumstances:
1) A specified duration has elapsed since the last process executed
2) The amount of total items queued (across all groups) has exceeded a given threshold
3) The amount of _groups_ of items has exceeded a given threshold

## Installation

`go get github.com/mikestefanello/batcher`

## Usage

As an example, we'll create a batcher to process items of type `Log` which will be grouped together via their `Level` then written in bulk.

```go
b, err := batcher.NewBatcher[Log](Config[Log]{
    // Process the batch if we've queued 5 different Level values
    GroupCountThreshold: 5,
    // Process the batch if we've queued 100 Log items
    ItemCountThreshold:  100,
    // Process the queue every 30 seconds
    DelayThreshold:      30 * time.Second,
    // Use 3 Goroutines to process the queue groups
    NumGoroutines:       3,
    // Execute this func for each queued group
    Processor:           func(group string, log []Log) {
        writeLogs(logs)
    },
})
```

Add a `Log` to the batch queue.

```go
b.Add(log.Level, log)
```

## Origin

This concept was originally devised to handle some challenges faced with PubSub messages but was made entirely generic for this library so it could be used for any purpose.
1) Group incoming messages by a unique identifier, ie, a user ID, to deduplicate requests and ensure a given pod was only ever executing operations for a given ID in isolation.
2) Group all incoming messages to bulk-export to persistent storage for archiving.

To _roughly_ illustrate an example in code:

```go
func main() {
    b, err := batcher.NewBatcher[*pubsub.Message](Config[*pubsub.Message]{
        GroupCountThreshold: 10,
        ItemCountThreshold:  100,
        DelayThreshold:      10 * time.Second,
        NumGoroutines:       3,
        Processor:           func(group string, items []*pubsub.Message) {
            err := doUserOperation(group)

            for _, m := range items {
                if err != nil {
                    m.Nack()
                } else {
                    m.Ack()
                }
            }
        },
    })


    // Consume the messages from PubSub
    err = subscription.Receive(ctx, func(ctx context.Context, message *pubsub.Message) {
        // For this example, Data is the user ID
        b.Add(string(message.Data), message)
    })
}
```