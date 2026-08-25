package watchers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Both the maildir and notmuch workers only use watcher events as a "something
// changed, rescan now" trigger, so any change inside a watched directory must
// produce an event. Without this, new mail never shows up until the folder is
// reopened.
func TestWatcherReportsChanges(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Skipf("no watcher for this platform: %v", err)
	}
	dir := t.TempDir()
	if err := w.Configure(dir); err != nil {
		t.Fatalf("Configure(%q): %v", dir, err)
	}
	// fsevents streams are not live immediately after Start()
	time.Sleep(time.Second)

	if err := os.WriteFile(filepath.Join(dir, "message"), []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-w.Events():
		if ev == nil {
			t.Fatal("nil event")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("no event after creating a file in the watched directory")
	}
}
