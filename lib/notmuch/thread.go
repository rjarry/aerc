//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <stdlib.h>
#include <notmuch.h>

*/
import "C"
import "time"

type Thread struct {
	thread *C.notmuch_thread_t
}

// ID returns the thread ID
func (t *Thread) ID() string {
	cID := C.notmuch_thread_get_thread_id(t.thread)
	return C.GoString(cID)
}

// TotalMessages returns the total number of messages in the thread
func (t *Thread) TotalMessages() int {
	return int(C.notmuch_thread_get_total_messages(t.thread))
}

// TotalMessages returns the total number of files in the thread
func (t *Thread) TotalFiles() int {
	return int(C.notmuch_thread_get_total_files(t.thread))
}

// TopLevelMessages returns an iterator over the top level messages in the
// thread. Messages are sorted oldest-first
func (t *Thread) TopLevelMessages() Messages {
	cMessages := C.notmuch_thread_get_toplevel_messages(t.thread)
	return Messages{
		messages: cMessages,
	}
}

// Messages returns an iterator over the messages in the thread. Messages are
// sorted oldest-first
func (t *Thread) Messages() Messages {
	cMessages := C.notmuch_thread_get_messages(t.thread)
	return Messages{
		messages: cMessages,
	}
}

// Matches returns the number of messages in the thread that matched the query
func (t *Thread) Matches() int {
	return int(C.notmuch_thread_get_matched_messages(t.thread))
}

// Returns a string of authors of the thread
func (t *Thread) Authors() string {
	cAuthors := C.notmuch_thread_get_authors(t.thread)
	return C.GoString(cAuthors)
}

// Returns the subject of the thread
func (t *Thread) Subject() string {
	cSubject := C.notmuch_thread_get_subject(t.thread)
	return C.GoString(cSubject)
}

// Returns the sent-date of the oldest message in the thread
func (t *Thread) OldestDate() time.Time {
	cTime := C.notmuch_thread_get_oldest_date(t.thread)
	return time.Unix(int64(cTime), 0)
}

// Returns the sent-date of the newest message in the thread
func (t *Thread) NewestDate() time.Time {
	cTime := C.notmuch_thread_get_newest_date(t.thread)
	return time.Unix(int64(cTime), 0)
}

// Tags returns a slice of all tags in the thread
func (t *Thread) Tags() []string {
	cTags := C.notmuch_thread_get_tags(t.thread)
	defer C.notmuch_tags_destroy(cTags)

	tags := []string{}
	for C.notmuch_tags_valid(cTags) > 0 {
		tag := C.notmuch_tags_get(cTags)
		tags = append(tags, C.GoString(tag))
		C.notmuch_tags_move_to_next(cTags)
	}
	return tags
}

func (t *Thread) Close() {
	C.notmuch_thread_destroy(t.thread)
}
