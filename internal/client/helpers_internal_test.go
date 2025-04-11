package client

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
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
