package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// runConcurrentTasks runs worker per id, at most maxConcurrent at once.
// Items are unordered; any error discards all items.
func runConcurrentTasks[T any](
	ctx context.Context,
	ids []string,
	worker func(ctx context.Context, id string) (T, error),
) ([]T, error) {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		items []T
		errs  []error
	)

	// Bounds parallel requests to stay under Robot's per-account rate limits.
	const maxConcurrent = 10

	sem := make(chan struct{}, maxConcurrent)

	for _, id := range ids {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			sem <- struct{}{}

			item, err := worker(ctx, id)

			<-sem

			if err != nil {
				mu.Lock()

				errs = append(errs, err)

				mu.Unlock()

				return
			}

			mu.Lock()

			items = append(items, item)

			mu.Unlock()
		}(id)
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf("error fetching: %w", errors.Join(errs...))
	}

	return items, nil
}
