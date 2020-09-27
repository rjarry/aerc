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

// SetFlag adds or removes a flag from the message.
// Notmuch doesn't support all the flags, and for those this errors.
func (m *Message) SetFlag(flag models.Flag, enable bool) error {
	// Translate the flag into a notmuch tag, ignoring no-op flags.
	var tag string
	switch flag {
	case models.SeenFlag:
		// Note: Inverted properly later
		tag = "unread"
	case models.AnsweredFlag:
		tag = "replied"
	case models.FlaggedFlag:
		tag = "flagged"
	default:
		return fmt.Errorf("Notmuch doesn't support flag %v", flag)
	}

	// Get the current state of the flag.
	// Note that notmuch handles models.SeenFlag in an inverted sense.
	oldState := false
	tags, err := m.Tags()
	if err != nil {
		return err
	}
	for _, t := range tags {
		if t == tag {
			oldState = true
			break
		}
	}
	if flag == models.SeenFlag {
		oldState = !oldState
	}

	// Skip if flag already in correct state.
	if oldState == enable {
		return nil
	}

	if !enable {
		if flag == models.SeenFlag {
			return m.AddTag("unread")
		} else {
			return m.RemoveTag(tag)
		}
	} else {
		if flag == models.SeenFlag {
			return m.RemoveTag("unread")
		} else {
			return m.AddTag(tag)
		}
	}
}

// MarkAnswered either adds or removes the "replied" tag from the message.
func (m *Message) MarkAnswered(answered bool) error {
	return m.SetFlag(models.AnsweredFlag, answered)
}

// MarkRead either adds or removes the maildir.FlagSeen flag from the message.
func (m *Message) MarkRead(seen bool) error {
	return m.SetFlag(models.SeenFlag, seen)
}

// tags returns the notmuch tags of a message
func (m *Message) Tags() ([]string, error) {
	return m.db.MsgTags(m.key)
}

func (m *Message) Labels() ([]string, error) {
	return m.Tags()
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
