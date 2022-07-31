package maildir

import (
	"fmt"
	"io"

	"github.com/emersion/go-maildir"
	"github.com/emersion/go-message"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
)

// A Message is an individual email inside of a maildir.Dir.
type Message struct {
	dir maildir.Dir
	uid uint32
	key string
}

// NewReader reads a message into memory and returns an io.Reader for it.
func (m Message) NewReader() (io.ReadCloser, error) {
	return m.dir.Open(m.key)
}

// Flags fetches the set of flags currently applied to the message.
func (m Message) Flags() ([]maildir.Flag, error) {
	return m.dir.Flags(m.key)
}

// ModelFlags fetches the set of models.flags currently applied to the message.
func (m Message) ModelFlags() ([]models.Flag, error) {
	flags, err := m.dir.Flags(m.key)
	if err != nil {
		return nil, err
	}
	return translateMaildirFlags(flags), nil
}

// SetFlags replaces the message's flags with a new set.
func (m Message) SetFlags(flags []maildir.Flag) error {
	return m.dir.SetFlags(m.key, flags)
}

// SetOneFlag enables or disables a single message flag on the message.
func (m Message) SetOneFlag(flag maildir.Flag, enable bool) error {
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

// MarkReplied either adds or removes the maildir.FlagReplied flag from the
// message.
func (m Message) MarkReplied(answered bool) error {
	return m.SetOneFlag(maildir.FlagReplied, answered)
}

// Remove deletes the email immediately.
func (m Message) Remove() error {
	return m.dir.Remove(m.key)
}

// MessageInfo populates a models.MessageInfo struct for the message.
func (m Message) MessageInfo() (*models.MessageInfo, error) {
	return lib.MessageInfo(m)
}

// NewBodyPartReader creates a new io.Reader for the requested body part(s) of
// the message.
func (m Message) NewBodyPartReader(requestedParts []int) (io.Reader, error) {
	f, err := m.dir.Open(m.key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msg, err := message.Read(f)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %w", err)
	}
	return lib.FetchEntityPartReader(msg, requestedParts)
}

var maildirToFlag = map[maildir.Flag]models.Flag{
	maildir.FlagReplied: models.AnsweredFlag,
	maildir.FlagSeen:    models.SeenFlag,
	maildir.FlagTrashed: models.DeletedFlag,
	maildir.FlagFlagged: models.FlaggedFlag,
	// maildir.FlagDraft Flag = 'D'
	// maildir.FlagPassed Flag = 'P'
}

var flagToMaildir = map[models.Flag]maildir.Flag{
	models.AnsweredFlag: maildir.FlagReplied,
	models.SeenFlag:     maildir.FlagSeen,
	models.DeletedFlag:  maildir.FlagTrashed,
	models.FlaggedFlag:  maildir.FlagFlagged,
	// maildir.FlagDraft Flag = 'D'
	// maildir.FlagPassed Flag = 'P'
}

func translateMaildirFlags(maildirFlags []maildir.Flag) []models.Flag {
	var flags []models.Flag
	for _, maildirFlag := range maildirFlags {
		if flag, ok := maildirToFlag[maildirFlag]; ok {
			flags = append(flags, flag)
		}
	}
	return flags
}

func translateFlags(flags []models.Flag) []maildir.Flag {
	var maildirFlags []maildir.Flag
	for _, flag := range flags {
		if maildirFlag, ok := flagToMaildir[flag]; ok {
			maildirFlags = append(maildirFlags, maildirFlag)
		}
	}
	return maildirFlags
}

func (m Message) UID() uint32 {
	return m.uid
}

func (m Message) Labels() ([]string, error) {
	return nil, nil
}
