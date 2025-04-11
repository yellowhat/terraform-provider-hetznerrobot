package client

import (
	"context"
	"fmt"
	"sync"
)

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
		return nil, fmt.Errorf("error fetching: %v", errs)
	}

	return items, nil
}
