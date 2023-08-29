//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <stdlib.h>
#include <notmuch.h>

*/
import "C"

// Threads is an iterator over a set of threads.
type Threads struct {
	thread  *C.notmuch_thread_t
	threads *C.notmuch_threads_t
}

// Next advances the Threads iterator to the next thread. Next returns false if
// no more threads are available
func (t *Threads) Next() bool {
	if C.notmuch_threads_valid(t.threads) == 0 {
		return false
	}
	t.thread = C.notmuch_threads_get(t.threads)
	C.notmuch_threads_move_to_next(t.threads)
	return true
}

// Thread returns the current thread in the iterator
func (t *Threads) Thread() Thread {
	return Thread{
		thread: t.thread,
	}
}

// Close frees memory associated with a Threads iterator. This method is not
// strictly necessary to call, as the resources will be freed when the Query
// associated with the Threads object is freed.
func (t *Threads) Close() {
	C.notmuch_threads_destroy(t.threads)
}
