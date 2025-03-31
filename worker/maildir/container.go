package maildir

import (
	"fmt"
	"sort"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
)

// A Container is a directory which contains other directories which adhere to
// the Maildir spec
type Container struct {
	Store      *lib.MaildirStore
	recentUIDS map[models.UID]struct{} // used to set the recent flag
}

// NewContainer creates a new container at the specified directory
func NewContainer(dir string, maildirpp bool) (*Container, error) {
	store, err := lib.NewMaildirStore(dir, maildirpp)
	if err != nil {
		return nil, err
	}
	return &Container{
		Store:      store,
		recentUIDS: make(map[models.UID]struct{}),
	}, nil
}

// SyncNewMail adds emails from new to cur, tracking them
func (c *Container) SyncNewMail(dir maildir.Dir) error {
	unseen, err := dir.Unseen()
	if err != nil {
		return err
	}
	for _, msg := range unseen {
		c.recentUIDS[models.UID(msg.Key())] = struct{}{}
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
func (c *Container) IsRecent(uid models.UID) bool {
	_, ok := c.recentUIDS[uid]
	return ok
}

// ClearRecentFlag removes the Recent flag from the message with the given uid
func (c *Container) ClearRecentFlag(uid models.UID) {
	delete(c.recentUIDS, uid)
}

// UIDs fetches the unique message identifiers for the maildir
func (c *Container) UIDs(d maildir.Dir) ([]models.UID, error) {
	// messages, err := d.Keys()
	messages, err := d.Messages()
	if err != nil && len(messages) == 0 {
		return nil, fmt.Errorf("could not get keys for %s: %w", d, err)
	}
	if err != nil {
		log.Errorf("could not get all keys for %s: %s", d, err.Error())
	}
	var keyList []string
	for _, msg := range messages {
		keyList = append(keyList, msg.Key())
	}
	sort.Strings(keyList)
	var uids []models.UID
	for _, key := range keyList {
		uids = append(uids, models.UID(key))
	}
	return uids, err
}

// Message returns a Message struct for the given UID and maildir
func (c *Container) Message(d maildir.Dir, uid models.UID) (*Message, error) {
	return &Message{
		dir: d,
		uid: uid,
		key: string(uid),
	}, nil
}

// DeleteAll deletes a set of messages by UID and returns the subset of UIDs
// which were successfully deleted, stopping upon the first error.
func (c *Container) DeleteAll(d maildir.Dir, uids []models.UID) ([]models.UID, error) {
	var success []models.UID
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
	dest maildir.Dir, src maildir.Dir, uids []models.UID,
) error {
	for _, uid := range uids {
		if err := c.copyMessage(dest, src, uid); err != nil {
			return fmt.Errorf("could not copy message %s: %w", uid, err)
		}
	}
	return nil
}

func (c *Container) copyMessage(
	dest maildir.Dir, src maildir.Dir, uid models.UID,
) error {
	msg, err := src.MessageByKey(string(uid))
	if err != nil {
		return fmt.Errorf("failed to retrieve message %q: %w", uid, err)
	}
	_, err = msg.CopyTo(dest)
	return err
}

func (c *Container) MoveAll(dest maildir.Dir, src maildir.Dir, uids []models.UID) ([]models.UID, error) {
	var success []models.UID
	for _, uid := range uids {
		if err := c.moveMessage(dest, src, uid); err != nil {
			return success, fmt.Errorf("could not move message %s: %w", uid, err)
		}
		success = append(success, uid)
	}
	return success, nil
}

func (c *Container) moveMessage(dest maildir.Dir, src maildir.Dir, uid models.UID) error {
	msg, err := src.MessageByKey(string(uid))
	if err != nil {
		return fmt.Errorf("failed to retrieve message %q: %w", uid, err)
	}
	return msg.MoveTo(dest)
}
