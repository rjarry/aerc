package maildir

import (
	"context"
	"runtime"
	"sync"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (w *Worker) search(ctx context.Context, criteria *types.SearchCriteria) ([]uint32, error) {
	criteria.PrepareHeader()
	requiredParts := lib.GetRequiredParts(criteria)
	w.worker.Debugf("Required parts bitmask for search: %b", requiredParts)

	keys, err := w.c.UIDs(*w.selected)
	if err != nil {
		return nil, err
	}

	matchedUids := []uint32{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	// Hard limit at 2x CPU cores
	max := runtime.NumCPU() * 2
	limit := make(chan struct{}, max)
	for _, key := range keys {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
			limit <- struct{}{}
			wg.Add(1)
			go func(key uint32) {
				defer log.PanicHandler()
				defer wg.Done()
				success, err := w.searchKey(key, criteria, requiredParts)
				if err != nil {
					// don't return early so that we can still get some results
					w.worker.Errorf("Failed to search key %d: %v", key, err)
				} else if success {
					mu.Lock()
					matchedUids = append(matchedUids, key)
					mu.Unlock()
				}
				<-limit
			}(key)

		}
	}
	wg.Wait()
	return matchedUids, nil
}

// Execute the search criteria for the given key, returns true if search succeeded
func (w *Worker) searchKey(key uint32, criteria *types.SearchCriteria,
	parts lib.MsgParts,
) (bool, error) {
	message, err := w.c.Message(*w.selected, key)
	if err != nil {
		return false, err
	}
	return lib.SearchMessage(message, criteria, parts)
}
