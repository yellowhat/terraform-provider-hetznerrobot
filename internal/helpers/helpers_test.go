package helpers_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/helpers"
)

func TestRunConcurrentTasks(t *testing.T) {
	tests := []struct {
		name     string
		ids      []int
		worker   func(ctx context.Context, id int) error
		wantErrs []string
	}{
		{
			name: "All success",
			ids:  []int{1, 2, 3, 4, 5},
			worker: func(ctx context.Context, id int) error {
				return nil
			},
			wantErrs: nil,
		},
		{
			name: "Some failures",
			ids:  []int{1, 2, 3, 4, 5},
			worker: func(ctx context.Context, id int) error {
				if id == 2 || id == 4 {
					return fmt.Errorf("failed for id: %d", id)
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
			err := helpers.RunConcurrentTasks(ctx, tt.ids, tt.worker)

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

func TestIntSliceToString(t *testing.T) {
	tests := []struct {
		name string
		ints []int
		want string
	}{
		{
			name: "Empty slice",
			ints: []int{},
			want: "",
		},
		{
			name: "Single element",
			ints: []int{42},
			want: "42",
		},
		{
			name: "Multiple elements",
			ints: []int{1, 2, 3},
			want: "1-2-3",
		},
		{
			name: "Negative values",
			ints: []int{-5, 0, 5},
			want: "-5-0-5",
		},
		{
			name: "Unordered",
			ints: []int{3, 2, 1},
			want: "3-2-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := helpers.IntSliceToString(tc.ints)
			if got != tc.want {
				t.Errorf("IntSliceToString(%v) = %q; want %q", tc.ints, got, tc.want)
			}
		})
	}
}
