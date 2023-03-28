package batcher

import (
	"fmt"
	"testing"
	"time"
)

func TestConfig_validate(t *testing.T) {
	type testCase struct {
		cfg      Config[int]
		expected bool
	}

	cases := []testCase{
		{
			cfg:      Config[int]{},
			expected: false,
		},
		{
			cfg: Config[int]{
				GroupCountThreshold: 1,
				DelayThreshold:      time.Millisecond,
				NumGoroutines:       1,
				Processor:           func(key string, items []int) {},
			},
			expected: true,
		},
		{
			cfg: Config[int]{
				GroupCountThreshold: 0,
				DelayThreshold:      time.Millisecond,
				NumGoroutines:       1,
				Processor:           func(key string, items []int) {},
			},
			expected: false,
		},
		{
			cfg: Config[int]{
				GroupCountThreshold: 1,
				DelayThreshold:      0,
				NumGoroutines:       1,
				Processor:           func(key string, items []int) {},
			},
			expected: false,
		},
		{
			cfg: Config[int]{
				GroupCountThreshold: 1,
				DelayThreshold:      time.Millisecond,
				NumGoroutines:       0,
				Processor:           func(key string, items []int) {},
			},
			expected: false,
		},
		{
			cfg: Config[int]{
				GroupCountThreshold: 1,
				DelayThreshold:      time.Millisecond,
				NumGoroutines:       1,
			},
			expected: false,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			err := c.cfg.validate()
			if c.expected != (err == nil) {
				t.Errorf("expected %v, got %v: %v", c.expected, err == nil, err)
			}
		})
	}
}
