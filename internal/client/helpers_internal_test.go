package client

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestRunConcurrentTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ids       []string
		worker    func(ctx context.Context, id string) (string, error)
		wantItems []string
		wantErrs  []string
	}{
		{
			name: "All success",
			ids:  []string{"1", "2", "3"},
			worker: func(_ context.Context, id string) (string, error) {
				return "item-" + id, nil
			},
			wantItems: []string{"item-1", "item-2", "item-3"},
			wantErrs:  nil,
		},
		{
			name: "Errors",
			ids:  []string{"1", "2", "3", "4", "5"},
			worker: func(_ context.Context, id string) (string, error) {
				if id == "2" || id == "4" {
					return "", fmt.Errorf("failed for id: %s", id)
				}

				return "item-" + id, nil
			},
			wantItems: nil,
			wantErrs:  []string{"failed for id: 2", "failed for id: 4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			items, err := runConcurrentTasks(ctx, tt.ids, tt.worker)

			if !haveSameItems(tt.wantItems, items) {
				t.Errorf("unexpected items, want %s, got %s", tt.wantItems, items)
			}

			switch {
			case tt.wantErrs == nil && err == nil:
				return
			case tt.wantErrs == nil && err != nil:
				t.Errorf("unexpected error: %v", err)
			case err == nil:
				t.Fatalf("want errors %v, got nil", tt.wantErrs)
			}

			for _, wantErr := range tt.wantErrs {
				if !strings.Contains(err.Error(), wantErr) {
					t.Errorf("unexpected errors, want %s, got %s", wantErr, err)
				}
			}
		})
	}
}

func haveSameItems(a, b []string) bool {
	sort.Strings(a)
	sort.Strings(b)

	return reflect.DeepEqual(a, b)
}

func TestRunConcurrentTasksEmpty(t *testing.T) {
	t.Parallel()

	items, err := runConcurrentTasks(
		context.Background(),
		nil,
		func(_ context.Context, id string) (string, error) {
			t.Errorf("worker called for %s", id)

			return "", nil
		},
	)
	if items != nil || err != nil {
		t.Errorf("want nil, nil; got %v, %v", items, err)
	}
}

// Every failure is kept and unwrappable, not just the first one.
func TestRunConcurrentTasksAllFail(t *testing.T) {
	t.Parallel()

	const count = 25

	ids := make([]string, count)
	for i := range ids {
		ids[i] = strconv.Itoa(i)
	}

	errFail := errors.New("failed")

	_, err := runConcurrentTasks(
		context.Background(),
		ids,
		func(_ context.Context, id string) (string, error) {
			return "", fmt.Errorf("id %s: %w", id, errFail)
		},
	)
	if !errors.Is(err, errFail) {
		t.Fatalf("error: want %v, got %v", errFail, err)
	}

	for _, id := range ids {
		if !strings.Contains(err.Error(), "id "+id+":") {
			t.Errorf("error missing id %s: %v", id, err)
		}
	}
}

// Exactly maxConcurrent run at once. In synctest, fake time advances only when
// all goroutines block, so every startable worker starts before any finishes.
func TestRunConcurrentTasksLimit(t *testing.T) {
	t.Parallel()

	const count = 50

	ids := make([]string, count)
	for i := range ids {
		ids[i] = strconv.Itoa(i)
	}

	synctest.Test(t, func(t *testing.T) {
		var running, peak atomic.Int32

		start := time.Now()

		items, err := runConcurrentTasks(
			context.Background(),
			ids,
			func(_ context.Context, id string) (string, error) {
				now := running.Add(1)
				defer running.Add(-1)

				for {
					old := peak.Load()
					if now <= old || peak.CompareAndSwap(old, now) {
						break
					}
				}

				time.Sleep(time.Second)

				return id, nil
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(items) != count {
			t.Errorf("items: want %d, got %d", count, len(items))
		}

		if got := peak.Load(); got != maxConcurrent {
			t.Errorf("peak concurrency: want %d, got %d", maxConcurrent, got)
		}

		// count/maxConcurrent batches of one second each.
		if got, want := time.Since(start), count/maxConcurrent*time.Second; got != want {
			t.Errorf("elapsed: want %v, got %v", want, got)
		}
	})
}
