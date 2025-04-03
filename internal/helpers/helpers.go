package helpers

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

func RunConcurrentTasks(
	ctx context.Context,
	ids []int,
	worker func(ctx context.Context, id int) error,
) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	sem := make(chan struct{}, 10)
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
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

func IntSliceToString(ints []int) string {
	out := make([]string, len(ints))
	for i, v := range ints {
		out[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(out, "-")
}
