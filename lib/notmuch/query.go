//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <stdlib.h>
#include <notmuch.h>

*/
import "C"
import "unsafe"

type ExcludeMode int

const (
	EXCLUDE_FLAG  ExcludeMode = C.NOTMUCH_EXCLUDE_FLAG
	EXCLUDE_TRUE  ExcludeMode = C.NOTMUCH_EXCLUDE_TRUE
	EXCLUDE_FALSE ExcludeMode = C.NOTMUCH_EXCLUDE_FALSE
	EXCLUDE_ALL   ExcludeMode = C.NOTMUCH_EXCLUDE_ALL
)

type SortMode int

const (
	SORT_OLDEST_FIRST SortMode = C.NOTMUCH_SORT_OLDEST_FIRST
	SORT_NEWEST_FIRST SortMode = C.NOTMUCH_SORT_NEWEST_FIRST
	SORT_MESSAGE_ID   SortMode = C.NOTMUCH_SORT_MESSAGE_ID
	SORT_UNSORTED     SortMode = C.NOTMUCH_SORT_UNSORTED
)

type Query struct {
	query *C.notmuch_query_t
}

// Close frees resources associated with a query. Closing a query release all
// resources associated with any underlying search (Threads, Messages, etc)
func (q *Query) Close() {
	C.notmuch_query_destroy(q.query)
}

// Return the string of the query
func (q *Query) String() string {
	return C.GoString(C.notmuch_query_get_query_string(q.query))
}

// Returns the Database associated with the query. The Path, Config, and Profile
// values will not be set on the returned valued
func (q *Query) Database() Database {
	db := C.notmuch_query_get_database(q.query)
	return Database{
		db: db,
	}
}

// Exclude sets the exclusion mode.
func (q *Query) Exclude(val ExcludeMode) {
	cVal := C.notmuch_exclude_t(val)
	C.notmuch_query_set_omit_excluded(q.query, cVal)
}

// Sort sets the sort order of the results
func (q *Query) Sort(sort SortMode) {
	cVal := C.notmuch_sort_t(sort)
	C.notmuch_query_set_sort(q.query, cVal)
}

// SortMode returns the current sort order of the results
func (q *Query) SortMode() SortMode {
	return SortMode(C.notmuch_query_get_sort(q.query))
}

// ExcludeTag adds a tag to exclude from the results
func (q *Query) ExcludeTag(tag string) error {
	cTag := C.CString(tag)
	defer C.free(unsafe.Pointer(cTag))
	return errorWrap(C.notmuch_query_add_tag_exclude(q.query, cTag))
}

// Threads returns an iterator over the threads that match the query
func (q *Query) Threads() (Threads, error) {
	var cThreads *C.notmuch_threads_t
	err := errorWrap(C.notmuch_query_search_threads(q.query, &cThreads)) //nolint:gocritic // see note in notmuch.go
	if err != nil {
		return Threads{}, err
	}
	threads := Threads{
		threads: cThreads,
	}
	return threads, nil
}

// Messages returns an iterator over the messages that match the query
func (q *Query) Messages() (Messages, error) {
	var cMessages *C.notmuch_messages_t
	err := errorWrap(C.notmuch_query_search_messages(q.query, &cMessages)) //nolint:gocritic // see note in notmuch.go
	if err != nil {
		return Messages{}, err
	}
	messages := Messages{
		messages: cMessages,
	}
	return messages, nil
}

// CountMessages returns the number of messages matching the query
func (q *Query) CountMessages() (int, error) {
	var count C.uint
	err := errorWrap(C.notmuch_query_count_messages(q.query, &count))
	return int(count), err
}

// CountThreads returns the number of threads matching the query
func (q *Query) CountThreads() (int, error) {
	var count C.uint
	err := errorWrap(C.notmuch_query_count_threads(q.query, &count))
	return int(count), err
}
