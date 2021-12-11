//go:build notmuch
// +build notmuch

package notmuch

type eventType interface{}

type event struct{}

type updateDirCounts struct {
	event
}
