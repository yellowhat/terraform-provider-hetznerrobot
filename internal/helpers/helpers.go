package helpers

import (
	"context"
	"fmt"
	"sync"
)

func RunConcurrentTasks(
	ctx context.Context,
	ids []string,
	worker func(ctx context.Context, id string) error,
) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	sem := make(chan struct{}, 10)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			err := worker(ctx, id)
			<-sem
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
		}(id)
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("error fetching: %v", errs)
	}

	return nil
}
