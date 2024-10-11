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
	"errors"
	"fmt"
	"unsafe"
)

type Mode int

const (
	MODE_READ_ONLY  Mode = C.NOTMUCH_DATABASE_MODE_READ_ONLY
	MODE_READ_WRITE Mode = C.NOTMUCH_DATABASE_MODE_READ_WRITE
)

type Database struct {
	// The profile to use. If it is the empty string, the profile will be
	// determined from the environment variable NOTMUCH_PROFILE or the
	// default profile will be used instead.
	Profile string

	db   *C.notmuch_database_t
	open bool
}

// Open opens the database with the given mode. Caller must call Close when done
// to commit changes and free resources
func (db *Database) Open(mode Mode) error {
	var (
		cProfile *C.char
		cErr     *C.char
	)
	defer C.free(unsafe.Pointer(cProfile))
	defer C.free(unsafe.Pointer(cErr))

	if db.Profile != "" {
		cProfile = C.CString(db.Profile)
	}
	cmode := C.notmuch_database_mode_t(mode)

	var cdb *C.notmuch_database_t

	// gocritic:dupSubExpr throws an issue here no matter how we call this
	// function
	err := errorWrap(
		C.notmuch_database_open_with_config(
			nil, cmode, nil, cProfile, &cdb, &cErr, //nolint:gocritic // see above
		),
	)
	if err != nil {
		return err
	}
	db.db = cdb
	db.open = true
	return nil
}

func (db *Database) MailRoot() string {
	// notmuch_config_get() returns a `const char *`. No need to free().
	root := C.notmuch_config_get(db.db, C.NOTMUCH_CONFIG_MAIL_ROOT)
	return C.GoString(root)
}

// Reopen an open notmuch database, usually with a different mode
func (db *Database) Reopen(mode Mode) error {
	cmode := C.notmuch_database_mode_t(mode)
	return errorWrap(C.notmuch_database_reopen(db.db, cmode))
}

// Close commits changes and closes the database, freeing any resources
// associated with it
func (db *Database) Close() error {
	if !db.open {
		return nil
	}
	err := errorWrap(C.notmuch_database_close(db.db))
	if err != nil {
		return err
	}
	err = errorWrap(C.notmuch_database_destroy(db.db))
	if err != nil {
		return err
	}
	db.open = false
	return nil
}

// LastStatus returns the last status string for the database
func (db *Database) LastStatus() string {
	cStatus := C.notmuch_database_status_string(db.db)
	defer C.free(unsafe.Pointer(cStatus))
	return C.GoString(cStatus)
}

func (db *Database) Compact(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("must have backup path before compacting")
	}
	var cBackupPath *C.char
	defer C.free(unsafe.Pointer(cBackupPath))
	return errorWrap(C.notmuch_database_compact_db(db.db, cBackupPath, nil, nil))
}

// Return the resolved path to the notmuch database
func (db *Database) ResolvedPath() string {
	cPath := C.notmuch_database_get_path(db.db)
	return C.GoString(cPath)
}

// NeedsUpgrade reports if the database must be upgraded before a write
// operation can be safely performed
func (db *Database) NeedsUpgrade() bool {
	return C.notmuch_database_needs_upgrade(db.db) == 1
}

// Indicate the beginning of an atomic operation
func (db *Database) BeginAtomic() error {
	return errorWrap(C.notmuch_database_begin_atomic(db.db))
}

// Indicate the end of an atomic operation
func (db *Database) EndAtomic() error {
	return errorWrap(C.notmuch_database_end_atomic(db.db))
}

// Returns the UUID and LastMod of the notmuch database
func (db *Database) Revision() (string, uint64) {
	var uuid *C.char
	defer C.free(unsafe.Pointer(uuid))
	lastmod := uint64(C.notmuch_database_get_revision(db.db, &uuid)) //nolint:gocritic // see note in notmuch.go
	return C.GoString(uuid), lastmod
}

// Returns a Directory object relative to the path of the Database
func (db *Database) Directory(relativePath string) (Directory, error) {
	var result Directory

	if relativePath == "" {
		return result, fmt.Errorf("path can't be empty")
	}
	var (
		dir   *C.notmuch_directory_t
		cPath *C.char
	)
	cPath = C.CString(relativePath)
	defer C.free(unsafe.Pointer(cPath))
	err := errorWrap(C.notmuch_database_get_directory(db.db, cPath, &dir)) //nolint:gocritic // see note in notmuch.go
	if err != nil {
		return result, err
	}
	result.dir = dir

	return result, nil
}

// IndexFile indexes a file with path relative to the database path, or an
// absolute path which share a common ancestor as the database path
func (db *Database) IndexFile(path string) (Message, error) {
	var (
		cPath *C.char
		msg   *C.notmuch_message_t
	)
	cPath = C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	err := errorWrap(C.notmuch_database_index_file(db.db, cPath, nil, &msg)) //nolint:gocritic // see note in notmuch.go
	switch {
	case errors.Is(err, STATUS_DUPLICATE_MESSAGE_ID):
		break
	case err != nil:
		return Message{}, err
	}
	message := Message{
		message: msg,
	}
	return message, nil
}

// Remove a file from the database. If this is the last file associated with a
// message, the message will be removed from the database.
func (db *Database) RemoveFile(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	return errorWrap(C.notmuch_database_remove_message(db.db, cPath))
}

// FindMessageByID finds a message by the Message-ID header field value
func (db *Database) FindMessageByID(id string) (Message, error) {
	var (
		cID *C.char
		msg *C.notmuch_message_t
	)
	cID = C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	err := errorWrap(C.notmuch_database_find_message(db.db, cID, &msg)) //nolint:gocritic // see note in notmuch.go
	if err != nil {
		return Message{}, err
	}
	message := Message{
		message: msg,
	}
	return message, nil
}

// FindMessageByFilename finds a message by filename
func (db *Database) FindMessageByFilename(filename string) (Message, error) {
	var (
		cFilename *C.char
		msg       *C.notmuch_message_t
	)
	cFilename = C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))
	err := errorWrap(C.notmuch_database_find_message_by_filename(db.db, cFilename, &msg)) //nolint:gocritic // see note in notmuch.go
	if err != nil {
		return Message{}, err
	}
	if msg == nil {
		return Message{}, fmt.Errorf("couldn't find message by filename: %s", filename)
	}
	message := Message{
		message: msg,
	}
	return message, nil
}

// Tags returns a slice of all tags in the database
func (db *Database) Tags() []string {
	cTags := C.notmuch_database_get_all_tags(db.db)
	defer C.notmuch_tags_destroy(cTags)

	tags := []string{}
	for C.notmuch_tags_valid(cTags) > 0 {
		tag := C.notmuch_tags_get(cTags)
		tags = append(tags, C.GoString(tag))
		C.notmuch_tags_move_to_next(cTags)
	}
	return tags
}

// Create a new Query
func (db *Database) Query(query string) (Query, error) {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))
	nmQuery := C.notmuch_query_create(db.db, cQuery)
	if nmQuery == nil {
		return Query{}, STATUS_OUT_OF_MEMORY
	}
	q := Query{
		query: nmQuery,
	}
	return q, nil
}
