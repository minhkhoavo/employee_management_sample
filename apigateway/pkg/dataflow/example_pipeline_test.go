package dataflow_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/pkg/dataflow"
)

func TestRefactoredPipeline(t *testing.T) {
	ctx := context.Background()

	// 1. Source
	items := []interface{}{"1,Alice", "2,Bob", "retry,Charlie"}
	source := dataflow.From(ctx, items...)

	// 2. Map: Parse
	type Row struct {
		ID   string
		Name string
	}

	parsed := dataflow.Map(ctx, source, func(msg interface{}) (interface{}, error) {
		s := msg.(string)
		parts := strings.Split(s, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format")
		}
		return Row{ID: parts[0], Name: parts[1]}, nil
	}, dataflow.WithWorkers(2)) // Parallel processing

	// 3. Map with Retry: Save
	var attempts int32
	saved := dataflow.Map(ctx, parsed, func(msg interface{}) (interface{}, error) {
		row := msg.(Row)
		if row.ID == "retry" {
			val := atomic.AddInt32(&attempts, 1)
			if val < 3 {
				return nil, fmt.Errorf("transient error")
			}
		}
		return row, nil
	}, dataflow.WithRetry(3, func(i int) time.Duration { return time.Millisecond }))

	// 4. Sink: Collect
	results := make([]Row, 0)
	err := dataflow.ForEach(ctx, saved, func(msg interface{}) error {
		row := msg.(Row)
		results = append(results, row)
		return nil
	})

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// Verify
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	foundRetry := false
	for _, r := range results {
		if r.ID == "retry" {
			foundRetry = true
		}
	}
	if !foundRetry {
		t.Error("Did not find retried item")
	}
}

func TestFanIn(t *testing.T) {
	ctx := context.Background()

	s1 := dataflow.From(ctx, 1)
	s2 := dataflow.From(ctx, 2)

	merged := dataflow.FanIn(ctx, s1, s2)

	sum := 0
	err := dataflow.ForEach(ctx, merged, func(msg interface{}) error {
		sum += msg.(int)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if sum != 3 {
		t.Errorf("Expected sum 3, got %d", sum)
	}
}
