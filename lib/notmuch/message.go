//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <stdlib.h>
#include <notmuch.h>

*/
import "C"

import (
	"time"
	"unsafe"
)

type Message struct {
	message *C.notmuch_message_t
}

// Close frees resources associated with the message
func (m *Message) Close() {
	C.notmuch_message_destroy(m.message)
}

// ID returns the message ID
func (m *Message) ID() string {
	cID := C.notmuch_message_get_message_id(m.message)
	return C.GoString(cID)
}

// ThreadID returns the thread ID of the message
func (m *Message) ThreadID() string {
	cID := C.notmuch_message_get_thread_id(m.message)
	return C.GoString(cID)
}

func (m *Message) Replies() Messages {
	cMessages := C.notmuch_message_get_replies(m.message)
	return Messages{
		messages: cMessages,
	}
}

func (m *Message) TotalFiles() int {
	return int(C.notmuch_message_count_files(m.message))
}

// Filename returns a single filename associated with the message. If the
// message has multiple filenames, the return value will be arbitrarily chosen
func (m *Message) Filename() string {
	cFilename := C.notmuch_message_get_filename(m.message)
	return C.GoString(cFilename)
}

func (m *Message) Filenames() []string {
	cFilenames := C.notmuch_message_get_filenames(m.message)
	defer C.notmuch_filenames_destroy(cFilenames)

	filenames := []string{}
	for C.notmuch_filenames_valid(cFilenames) > 0 {
		filename := C.notmuch_filenames_get(cFilenames)
		filenames = append(filenames, C.GoString(filename))
		C.notmuch_filenames_move_to_next(cFilenames)
	}
	return filenames
}

// TODO is this needed?
// func (m *Message) Reindex() error {
//
// }

type Flag int

const (
	MESSAGE_FLAG_MATCH Flag = iota
	MESSAGE_FLAG_EXCLUDED
	MESSAGE_FLAG_GHOST
)

func (m *Message) Flag(flag Flag) (bool, error) {
	var ok C.notmuch_bool_t
	cFlag := C.notmuch_message_flag_t(flag)
	err := errorWrap(C.notmuch_message_get_flag_st(m.message, cFlag, &ok))
	if err != nil {
		return false, err
	}
	if ok == 0 {
		return false, nil
	}
	return true, nil
}

// TODO why does this exist??
// func (m *Message) SetFlag(flag Flag) {
//
// }

func (m *Message) Date() time.Time {
	cTime := C.notmuch_message_get_date(m.message)
	return time.Unix(int64(cTime), 0)
}

func (m *Message) Header(field string) string {
	cField := C.CString(field)
	defer C.free(unsafe.Pointer(cField))
	cHeader := C.notmuch_message_get_header(m.message, cField)
	return C.GoString(cHeader)
}

func (m *Message) Tags() []string {
	cTags := C.notmuch_message_get_tags(m.message)
	defer C.notmuch_tags_destroy(cTags)

	tags := []string{}
	for C.notmuch_tags_valid(cTags) > 0 {
		tag := C.notmuch_tags_get(cTags)
		tags = append(tags, C.GoString(tag))
		C.notmuch_tags_move_to_next(cTags)
	}
	return tags
}

func (m *Message) AddTag(tag string) error {
	cTag := C.CString(tag)
	defer C.free(unsafe.Pointer(cTag))

	return errorWrap(C.notmuch_message_add_tag(m.message, cTag))
}

func (m *Message) RemoveTag(tag string) error {
	cTag := C.CString(tag)
	defer C.free(unsafe.Pointer(cTag))

	return errorWrap(C.notmuch_message_remove_tag(m.message, cTag))
}

func (m *Message) RemoveAllTags() error {
	return errorWrap(C.notmuch_message_remove_all_tags(m.message))
}

// SyncTagsToMaildirFlags adds/removes the appropriate tags to the maildir
// filename
func (m *Message) SyncTagsToMaildirFlags() error {
	return errorWrap(C.notmuch_message_tags_to_maildir_flags(m.message))
}

// SyncMaildirFlagsToTags syncs the current maildir flags to the notmuch tags
func (m *Message) SyncMaildirFlagsToTags() error {
	return errorWrap(C.notmuch_message_maildir_flags_to_tags(m.message))
}

func (m *Message) HasMaildirFlag(flag rune) (bool, error) {
	var ok C.notmuch_bool_t
	err := errorWrap(C.notmuch_message_has_maildir_flag_st(m.message, C.char(flag), &ok))
	if err != nil {
		return false, err
	}
	if ok == 0 {
		return false, nil
	}
	return true, nil
}

func (m *Message) Freeze() error {
	return errorWrap(C.notmuch_message_freeze(m.message))
}

func (m *Message) Thaw() error {
	return errorWrap(C.notmuch_message_thaw(m.message))
}

func (m *Message) Property(key string) (string, error) {
	var (
		cKey   *C.char
		cValue *C.char
	)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cValue))
	cKey = C.CString(key)
	err := errorWrap(C.notmuch_message_get_property(m.message, cKey, &cValue)) //nolint:gocritic // see note in notmuch.go
	if err != nil {
		return "", err
	}
	return C.GoString(cValue), nil
}

func (m *Message) AddProperty(key string, value string) error {
	var (
		cKey   *C.char
		cValue *C.char
	)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cValue))
	cKey = C.CString(key)
	cValue = C.CString(value)
	return errorWrap(C.notmuch_message_add_property(m.message, cKey, cValue))
}

func (m *Message) RemoveProperty(key string, value string) error {
	var (
		cKey   *C.char
		cValue *C.char
	)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cValue))
	cKey = C.CString(key)
	cValue = C.CString(value)
	return errorWrap(C.notmuch_message_remove_property(m.message, cKey, cValue))
}

func (m *Message) RemoveAllProperties(key string) error {
	var cKey *C.char
	defer C.free(unsafe.Pointer(cKey))
	cKey = C.CString(key)
	return errorWrap(C.notmuch_message_remove_all_properties(m.message, cKey))
}

func (m *Message) RemoveAllPropertiesWithPrefix(prefix string) error {
	var cPrefix *C.char
	defer C.free(unsafe.Pointer(cPrefix))
	cPrefix = C.CString(prefix)
	return errorWrap(C.notmuch_message_remove_all_properties_with_prefix(m.message, cPrefix))
}

func (m *Message) Properties(key string, exact bool) *Properties {
	var (
		cKey   *C.char
		cExact C.int
	)
	defer C.free(unsafe.Pointer(cKey))
	if exact {
		cExact = 1
	}

	cKey = C.CString(key)
	props := C.notmuch_message_get_properties(m.message, cKey, cExact)

	return &Properties{
		properties: props,
	}
}

func (m *Message) CountProperties(key string) (int, error) {
	var (
		cKey   *C.char
		cCount C.uint
	)
	defer C.free(unsafe.Pointer(cKey))
	cKey = C.CString(key)
	err := errorWrap(C.notmuch_message_count_properties(m.message, cKey, &cCount))
	if err != nil {
		return 0, err
	}
	return int(cCount), nil
}
