package maildir

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~sircmpwn/aerc/lib/uidstore"
)

// A Container is a directory which contains other directories which adhere to
// the Maildir spec
type Container struct {
	dir  string
	log  *log.Logger
	uids *uidstore.Store
}

// NewContainer creates a new container at the specified directory
// TODO: return an error if the provided directory is not accessible
func NewContainer(dir string, l *log.Logger) *Container {
	return &Container{dir: dir, uids: uidstore.NewStore(), log: l}
}

// ListFolders returns a list of maildir folders in the container
func (c *Container) ListFolders() ([]string, error) {
	folders := []string{}
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		// Skip maildir's default directories
		n := info.Name()
		if n == "new" || n == "tmp" || n == "cur" {
			return filepath.SkipDir
		}

		// Get the relative path from the parent directory
		dirPath, err := filepath.Rel(c.dir, path)
		if err != nil {
			return err
		}

		// Skip the parent directory
		if dirPath == "." {
			return nil
		}

		folders = append(folders, dirPath)
		return nil
	})
	return folders, err
}

// OpenDirectory opens an existing maildir in the container by name, moves new
// messages into cur, and registers the new keys in the UIDStore.
func (c *Container) OpenDirectory(name string) (maildir.Dir, error) {
	dir := c.Dir(name)
	keys, err := dir.Unseen()
	if err != nil {
		return dir, err
	}
	for _, key := range keys {
		c.uids.GetOrInsert(key)
	}
	return dir, nil
}

// Dir returns a maildir.Dir with the specified name inside the container
func (c *Container) Dir(name string) maildir.Dir {
	return maildir.Dir(filepath.Join(c.dir, name))
}

// UIDs fetches the unique message identifiers for the maildir
func (c *Container) UIDs(d maildir.Dir) ([]uint32, error) {
	keys, err := d.Keys()
	if err != nil {
		return nil, fmt.Errorf("could not get keys for %s: %v", d, err)
	}
	sort.Strings(keys)
	var uids []uint32
	for _, key := range keys {
		uids = append(uids, c.uids.GetOrInsert(key))
	}
	return uids, nil
}

// Message returns a Message struct for the given UID and maildir
func (c *Container) Message(d maildir.Dir, uid uint32) (*Message, error) {
	if key, ok := c.uids.GetKey(uid); ok {
		return &Message{
			dir: d,
			uid: uid,
			key: key,
		}, nil
	}
	return nil, fmt.Errorf("could not find message with uid %d in maildir %s",
		uid, d)
}

// DeleteAll deletes a set of messages by UID and returns the subset of UIDs
// which were successfully deleted, stopping upon the first error.
func (c *Container) DeleteAll(d maildir.Dir, uids []uint32) ([]uint32, error) {
	var success []uint32
	for _, uid := range uids {
		msg, err := c.Message(d, uid)
		if err != nil {
			return success, err
		}
		if err := msg.Remove(); err != nil {
			return success, err
		}
		success = append(success, uid)
	}
	return success, nil
}

func (c *Container) CopyAll(
	dest maildir.Dir, src maildir.Dir, uids []uint32) error {
	for _, uid := range uids {
		if err := c.copyMessage(dest, src, uid); err != nil {
			return fmt.Errorf("could not copy message %d: %v", uid, err)
		}
	}
	return nil
}

func (c *Container) copyMessage(
	dest maildir.Dir, src maildir.Dir, uid uint32) error {
	key, ok := c.uids.GetKey(uid)
	if !ok {
		return fmt.Errorf("could not find key for message id %d", uid)
	}
	_, err := src.Copy(dest, key)
	return err
}
