package maildir

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
)

// A Message is an individual email inside of a maildir.Dir.
type Message struct {
	dir maildir.Dir
	uid models.UID
	key string
	msg *maildir.Message
	hdr *models.MessageInfo
}

// loadMsg lazy-loads and caches the underlying maildir.Message.
func (m *Message) loadMsg() (*maildir.Message, error) {
	if m.msg != nil {
		return m.msg, nil
	}
	msg, err := m.dir.MessageByKey(m.key)
	if err != nil {
		return nil, fmt.Errorf("failed to find message with key %q: %w", m.key, err)
	}
	m.msg = msg
	return msg, nil
}

// NewReader reads a message into memory and returns an io.Reader for it.
func (m *Message) NewReader() (io.ReadCloser, error) {
	msg, err := m.loadMsg()
	if err != nil {
		return nil, err
	}
	return msg.Open()
}

// Flags fetches the set of flags currently applied to the message.
func (m *Message) Flags() ([]maildir.Flag, error) {
	msg, err := m.loadMsg()
	if err != nil {
		return nil, err
	}
	return msg.Flags(), nil
}

// ModelFlags fetches the set of models.flags currently applied to the message.
func (m *Message) ModelFlags() (models.Flags, error) {
	msg, err := m.loadMsg()
	if err != nil {
		return 0, err
	}
	return lib.FromMaildirFlags(msg.Flags()), nil
}

// SetFlags replaces the message's flags with a new set.
func (m *Message) SetFlags(flags []maildir.Flag) error {
	msg, err := m.loadMsg()
	if err != nil {
		return err
	}
	err = msg.SetFlags(flags)
	m.msg = nil
	return err
}

// SetOneFlag enables or disables a single message flag on the message.
func (m *Message) SetOneFlag(flag maildir.Flag, enable bool) error {
	flags, err := m.Flags()
	if err != nil {
		return fmt.Errorf("could not read previous flags: %w", err)
	}
	if enable {
		flags = append(flags, flag)
		return m.SetFlags(flags)
	}
	var newFlags []maildir.Flag
	for _, oldFlag := range flags {
		if oldFlag != flag {
			newFlags = append(newFlags, oldFlag)
		}
	}
	return m.SetFlags(newFlags)
}

// MarkForwarded either adds or removes the maildir.FlagForwarded flag
// from the message.
func (m *Message) MarkForwarded(forwarded bool) error {
	return m.SetOneFlag(maildir.FlagPassed, forwarded)
}

// MarkReplied either adds or removes the maildir.FlagReplied flag from the
// message.
func (m *Message) MarkReplied(answered bool) error {
	return m.SetOneFlag(maildir.FlagReplied, answered)
}

// Remove deletes the email immediately.
func (m *Message) Remove() error {
	msg, err := m.loadMsg()
	if err != nil {
		return err
	}
	err = msg.Remove()
	m.msg = nil
	return err
}

// MessageInfo populates a models.MessageInfo struct for the message.
func (m *Message) MessageInfo(dir string) (*models.MessageInfo, error) {
	info, err := rfc822.MessageInfo(m)
	if err != nil {
		return nil, err
	}
	info.Size, err = m.Size()
	info.Directory = dir
	if err != nil {
		// don't care if size retrieval fails
		log.Debugf("message size: %v", err)
	}
	info.InternalDate = m.InternalDate()
	return info, nil
}

func (m *Message) Size() (uint32, error) {
	msg, err := m.loadMsg()
	if err != nil {
		return 0, err
	}
	size, err := lib.FileSize(msg.Filename())
	if err != nil {
		return 0, fmt.Errorf("failed to get filesize: %w", err)
	}
	return size, nil
}

func (m *Message) InternalDate() time.Time {
	msg, err := m.loadMsg()
	if err != nil {
		return time.Time{}
	}
	info, err := os.Stat(msg.Filename())
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// MessageHeaders populates a models.MessageInfo struct for the message with
// minimal information, used for sorting and threading.
func (m *Message) MessageHeaders() (*models.MessageInfo, error) {
	if m.hdr != nil {
		return m.hdr, nil
	}

	info, err := rfc822.MessageHeaders(m)
	if err != nil {
		return nil, err
	}
	info.Size, err = m.Size()
	if err != nil {
		// don't care if size retrieval fails
		log.Debugf("message size failed: %v", err)
	}
	info.InternalDate = m.InternalDate()
	m.hdr = info
	return info, nil
}

// NewBodyPartReader creates a new io.Reader for the requested body part(s) of
// the message.
func (m *Message) NewBodyPartReader(requestedParts []int) (io.Reader, error) {
	msgWrapper, err := m.loadMsg()
	if err != nil {
		return nil, err
	}
	f, err := msgWrapper.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msg, err := rfc822.ReadMessage(f)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %w", err)
	}
	return rfc822.FetchEntityPartReader(msg, requestedParts)
}

func (m *Message) UID() models.UID {
	return m.uid
}

func (m *Message) Labels() ([]string, error) {
	return nil, nil
}
