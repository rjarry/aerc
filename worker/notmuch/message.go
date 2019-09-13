//+build notmuch

package notmuch

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/lib"
	notmuch "git.sr.ht/~sircmpwn/aerc/worker/notmuch/lib"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
)

type Message struct {
	uid uint32
	key string
	db  *notmuch.DB
}

// NewReader reads a message into memory and returns an io.Reader for it.
func (m *Message) NewReader() (io.Reader, error) {
	name, err := m.Filename()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// MessageInfo populates a models.MessageInfo struct for the message.
func (m *Message) MessageInfo() (*models.MessageInfo, error) {
	return lib.MessageInfo(m)
}

// NewBodyPartReader creates a new io.Reader for the requested body part(s) of
// the message.
func (m *Message) NewBodyPartReader(requestedParts []int) (io.Reader, error) {
	name, err := m.Filename()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msg, err := message.Read(f)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %v", err)
	}
	return lib.FetchEntityPartReader(msg, requestedParts)
}

// MarkRead either adds or removes the maildir.FlagSeen flag from the message.
func (m *Message) MarkRead(seen bool) error {
	haveUnread := false
	tags, err := m.Tags()
	if err != nil {
		return err
	}
	for _, t := range tags {
		if t == "unread" {
			haveUnread = true
			break
		}
	}
	if (haveUnread && !seen) || (!haveUnread && seen) {
		// we already have the desired state
		return nil
	}

	if haveUnread {
		err := m.RemoveTag("unread")
		if err != nil {
			return err
		}
		return nil
	}

	err = m.AddTag("unread")
	if err != nil {
		return err
	}
	return nil
}

// tags returns the notmuch tags of a message
func (m *Message) Tags() ([]string, error) {
	return m.db.MsgTags(m.key)
}

func (m *Message) ModelFlags() ([]models.Flag, error) {
	var flags []models.Flag
	seen := true
	tags, err := m.Tags()
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		switch tag {
		case "replied":
			flags = append(flags, models.AnsweredFlag)
		case "flagged":
			flags = append(flags, models.FlaggedFlag)
		case "unread":
			seen = false
		default:
			continue
		}
	}
	if seen {
		flags = append(flags, models.SeenFlag)
	}
	return flags, nil
}

func (m *Message) UID() uint32 {
	return m.uid
}

func (m *Message) Filename() (string, error) {
	return m.db.MsgFilename(m.key)
}

//AddTag adds a single tag.
//Consider using *Message.ModifyTags for multiple additions / removals
//instead of looping over a tag array
func (m *Message) AddTag(tag string) error {
	return m.ModifyTags([]string{tag}, nil)
}

//RemoveTag removes a single tag.
//Consider using *Message.ModifyTags for multiple additions / removals
//instead of looping over a tag array
func (m *Message) RemoveTag(tag string) error {
	return m.ModifyTags(nil, []string{tag})
}

func (m *Message) ModifyTags(add, remove []string) error {
	return m.db.MsgModifyTags(m.key, add, remove)
}
