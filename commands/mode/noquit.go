package mode

import "sync/atomic"

// noquit is a counter for goroutines that requested the no-quit mode
var noquit int32

// NoQuit enters no-quit mode where aerc cannot be exited (unless the force
// option is used)
func NoQuit() {
	atomic.AddInt32(&noquit, 1)
}

// NoQuitDone leaves the no-quit mode
func NoQuitDone() {
	atomic.AddInt32(&noquit, -1)
}

// QuitAllowed checks if aerc can exit normally (only when all goroutines that
// requested a no-quit mode were done and called the NoQuitDone() function)
func QuitAllowed() bool {
	return atomic.LoadInt32(&noquit) <= 0
}
