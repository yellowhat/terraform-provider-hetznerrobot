package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRunConcurrentTasks(t *testing.T) {
	tests := []struct {
		name     string
		ids      []string
		worker   func(ctx context.Context, id string) error
		wantErrs []string
	}{
		{
			name: "All success",
			ids:  []string{"1", "2", "3", "4", "5"},
			worker: func(ctx context.Context, id string) error {
				return nil
			},
			wantErrs: nil,
		},
		{
			name: "Some failures",
			ids:  []string{"1", "2", "3", "4", "5"},
			worker: func(ctx context.Context, id string) error {
				if id == "2" || id == "4" {
					return fmt.Errorf("failed for id: %s", id)
				}
				return nil
			},
			wantErrs: []string{"failed for id: 2", "failed for id: 4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			err := runConcurrentTasks(ctx, tt.ids, tt.worker)

			switch {
			case tt.wantErrs == nil && err == nil:
				return
			case tt.wantErrs == nil && err != nil:
				t.Errorf("unexpected error: %v", err)
			case err == nil:
				t.Errorf("expected error but got nil")
			}

			for _, wantErr := range tt.wantErrs {
				if !strings.Contains(err.Error(), wantErr) {
					t.Errorf("expected error but got nil %s %s", err, wantErr)
				}
			}
		})
	}
}
