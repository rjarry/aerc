package maildir

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/lib/uidstore"
)

// uidReg matches filename encoded UIDs in maildirs synched with mbsync or
// OfflineIMAP
var uidReg = regexp.MustCompile(`,U=\d+`)

// A Container is a directory which contains other directories which adhere to
// the Maildir spec
type Container struct {
	dir        string
	uids       *uidstore.Store
	recentUIDS map[uint32]struct{} // used to set the recent flag
	maildirpp  bool                // whether to use Maildir++ directory layout
}

// NewContainer creates a new container at the specified directory
func NewContainer(dir string, maildirpp bool) (*Container, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !s.IsDir() {
		return nil, fmt.Errorf("Given maildir '%s' not a directory", dir)
	}
	return &Container{
		dir: dir, uids: uidstore.NewStore(),
		recentUIDS: make(map[uint32]struct{}), maildirpp: maildirpp,
	}, nil
}

// ListFolders returns a list of maildir folders in the container
func (c *Container) ListFolders() ([]string, error) {
	folders := []string{}
	if c.maildirpp {
		// In Maildir++ layout, INBOX is the root folder
		folders = append(folders, "INBOX")
	}
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("Invalid path '%s': error: %w", path, err)
		}
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

		// Drop dirs that lack {new,tmp,cur} subdirs
		for _, sub := range []string{"new", "tmp", "cur"} {
			if _, err := os.Stat(filepath.Join(path, sub)); os.IsNotExist(err) {
				return nil
			}
		}

		if c.maildirpp {
			// In Maildir++ layout, mailboxes are stored in a single directory
			// and prefixed with a dot, and subfolders are separated by dots.
			if !strings.HasPrefix(dirPath, ".") {
				return filepath.SkipDir
			}
			dirPath = strings.TrimPrefix(dirPath, ".")
			dirPath = strings.ReplaceAll(dirPath, ".", "/")
			folders = append(folders, dirPath)

			// Since all mailboxes are stored in a single directory, don't
			// recurse into subdirectories
			return filepath.SkipDir
		}

		folders = append(folders, dirPath)
		return nil
	})
	return folders, err
}

// SyncNewMail adds emails from new to cur, tracking them
func (c *Container) SyncNewMail(dir maildir.Dir) error {
	keys, err := dir.Unseen()
	if err != nil {
		return err
	}
	for _, key := range keys {
		uid := c.uids.GetOrInsert(key)
		c.recentUIDS[uid] = struct{}{}
	}
	return nil
}

// OpenDirectory opens an existing maildir in the container by name, moves new
// messages into cur, and registers the new keys in the UIDStore.
func (c *Container) OpenDirectory(name string) (maildir.Dir, error) {
	dir := c.Dir(name)
	if err := c.SyncNewMail(dir); err != nil {
		return dir, err
	}
	return dir, nil
}

// Dir returns a maildir.Dir with the specified name inside the container
func (c *Container) Dir(name string) maildir.Dir {
	if c.maildirpp {
		// Use Maildir++ layout
		if name == "INBOX" {
			return maildir.Dir(c.dir)
		}
		return maildir.Dir(filepath.Join(c.dir, "."+strings.ReplaceAll(name, "/", ".")))
	}
	return maildir.Dir(filepath.Join(c.dir, name))
}

// IsRecent returns if a uid has the Recent flag set
func (c *Container) IsRecent(uid uint32) bool {
	_, ok := c.recentUIDS[uid]
	return ok
}

// ClearRecentFlag removes the Recent flag from the message with the given uid
func (c *Container) ClearRecentFlag(uid uint32) {
	delete(c.recentUIDS, uid)
}

// UIDs fetches the unique message identifiers for the maildir
func (c *Container) UIDs(d maildir.Dir) ([]uint32, error) {
	keys, err := d.Keys()
	if err != nil {
		return nil, fmt.Errorf("could not get keys for %s: %w", d, err)
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
	dest maildir.Dir, src maildir.Dir, uids []uint32,
) error {
	for _, uid := range uids {
		if err := c.copyMessage(dest, src, uid); err != nil {
			return fmt.Errorf("could not copy message %d: %w", uid, err)
		}
	}
	return nil
}

func (c *Container) copyMessage(
	dest maildir.Dir, src maildir.Dir, uid uint32,
) error {
	key, ok := c.uids.GetKey(uid)
	if !ok {
		return fmt.Errorf("could not find key for message id %d", uid)
	}
	_, err := src.Copy(dest, key)
	return err
}

func (c *Container) MoveAll(dest maildir.Dir, src maildir.Dir, uids []uint32) ([]uint32, error) {
	var success []uint32
	for _, uid := range uids {
		if err := c.moveMessage(dest, src, uid); err != nil {
			return success, fmt.Errorf("could not move message %d: %w", uid, err)
		}
		success = append(success, uid)
	}
	return success, nil
}

func (c *Container) moveMessage(dest maildir.Dir, src maildir.Dir, uid uint32) error {
	key, ok := c.uids.GetKey(uid)
	if !ok {
		return fmt.Errorf("could not find key for message id %d", uid)
	}
	path, err := src.Filename(key)
	if err != nil {
		return fmt.Errorf("could not find path for message id %d", uid)
	}
	// Remove encoded UID information from the key to prevent sync issues
	name := uidReg.ReplaceAllString(filepath.Base(path), "")
	destPath := filepath.Join(string(dest), "cur", name)
	return os.Rename(path, destPath)
}
