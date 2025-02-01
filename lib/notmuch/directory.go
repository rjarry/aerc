//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <notmuch.h>

*/
import "C"
import "time"

type Directory struct {
	dir *C.notmuch_directory_t
}

func (dir *Directory) SetModifiedTime(t time.Time) error {
	cTime := C.time_t(t.Unix())
	return errorWrap(C.notmuch_directory_set_mtime(dir.dir, cTime))
}

func (dir *Directory) ModifiedTime() time.Time {
	cTime := C.notmuch_directory_get_mtime(dir.dir)
	return time.Unix(int64(cTime), 0)
}

func (dir *Directory) Filenames() []string {
	cFilenames := C.notmuch_directory_get_child_files(dir.dir)
	defer C.notmuch_filenames_destroy(cFilenames)

	filenames := []string{}
	for C.notmuch_filenames_valid(cFilenames) > 0 {
		filename := C.notmuch_filenames_get(cFilenames)
		filenames = append(filenames, C.GoString(filename))
		C.notmuch_filenames_move_to_next(cFilenames)
	}
	return filenames
}

func (dir *Directory) Directories() []string {
	cFilenames := C.notmuch_directory_get_child_directories(dir.dir)
	defer C.notmuch_filenames_destroy(cFilenames)

	filenames := []string{}
	for C.notmuch_filenames_valid(cFilenames) > 0 {
		filename := C.notmuch_filenames_get(cFilenames)
		filenames = append(filenames, C.GoString(filename))
		C.notmuch_filenames_move_to_next(cFilenames)
	}
	return filenames
}

// Delete deletes a directory document from the database and destroys
// the underlying object. Any child directories and files must have been
// deleted first by the caller
func (dir *Directory) Delete() error {
	return errorWrap(C.notmuch_directory_delete(dir.dir))
}

func (dir *Directory) Close() {
	C.notmuch_directory_destroy(dir.dir)
}
