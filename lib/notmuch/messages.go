//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <notmuch.h>

*/
import "C"

type Messages struct {
	message  *C.notmuch_message_t
	messages *C.notmuch_messages_t
}

// Next advances the Messages iterator to the next message. Next returns false if
// no more messages are available
func (m *Messages) Next() bool {
	if C.notmuch_messages_valid(m.messages) == 0 {
		return false
	}
	m.message = C.notmuch_messages_get(m.messages)
	C.notmuch_messages_move_to_next(m.messages)
	return true
}

// Message returns the current message in the iterator
func (m *Messages) Message() Message {
	return Message{
		message: m.message,
	}
}

// Close frees memory associated with a Messages iterator. This method is not
// strictly necessary to call, as the resources will be freed when the Query
// associated with the Messages object is freed.
func (m *Messages) Close() {
	C.notmuch_messages_destroy(m.messages)
}

// Tags returns a slice of all tags in the message list. WARNING: After calling
// tags, the message list can no longer be iterated; a new list must be created
// to iterate after calling Tags
func (m *Messages) Tags() []string {
	cTags := C.notmuch_messages_collect_tags(m.messages)
	defer C.notmuch_tags_destroy(cTags)

	tags := []string{}
	for C.notmuch_tags_valid(cTags) > 0 {
		tag := C.notmuch_tags_get(cTags)
		tags = append(tags, C.GoString(tag))
		C.notmuch_tags_move_to_next(cTags)
	}
	return tags
}
