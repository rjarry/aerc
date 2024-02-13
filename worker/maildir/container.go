package maildir

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/uidstore"
	"git.sr.ht/~rjarry/aerc/worker/lib"
)

// A Container is a directory which contains other directories which adhere to
// the Maildir spec
type Container struct {
	Store      *lib.MaildirStore
	uids       *uidstore.Store
	recentUIDS map[uint32]struct{} // used to set the recent flag
}

// NewContainer creates a new container at the specified directory
func NewContainer(dir string, maildirpp bool) (*Container, error) {
	store, err := lib.NewMaildirStore(dir, maildirpp)
	if err != nil {
		return nil, err
	}
	return &Container{
		Store: store, uids: uidstore.NewStore(),
		recentUIDS: make(map[uint32]struct{}),
	}, nil
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
	dir := c.Store.Dir(name)
	if err := c.SyncNewMail(dir); err != nil {
		return dir, err
	}
	return dir, nil
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
	if err != nil && len(keys) == 0 {
		return nil, fmt.Errorf("could not get keys for %s: %w", d, err)
	}
	if err != nil {
		log.Errorf("could not get all keys for %s: %s", d, err.Error())
	}
	sort.Strings(keys)
	var uids []uint32
	for _, key := range keys {
		uids = append(uids, c.uids.GetOrInsert(key))
	}
	return uids, err
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

func (c *Container) MessageFromKey(d maildir.Dir, key string) *Message {
	uid := c.uids.GetOrInsert(key)
	return &Message{
		dir: d,
		uid: uid,
		key: key,
	}
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
	name := lib.StripUIDFromMessageFilename(filepath.Base(path))
	destPath := filepath.Join(string(dest), "cur", name)
	return os.Rename(path, destPath)
}
