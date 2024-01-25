//go:build notmuch
// +build notmuch

package notmuch

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	notmuch "git.sr.ht/~rjarry/aerc/worker/notmuch/lib"
)

type Message struct {
	uid uint32
	key string
	db  *notmuch.DB
}

// NewReader returns a reader for a message
func (m *Message) NewReader() (io.ReadCloser, error) {
	name, err := m.Filename()
	if err != nil {
		return nil, err
	}
	return os.Open(name)
}

// MessageInfo populates a models.MessageInfo struct for the message.
func (m *Message) MessageInfo() (*models.MessageInfo, error) {
	info, err := rfc822.MessageInfo(m)
	if err != nil {
		return nil, err
	}
	// if size retrieval fails, just return info and log error
	if name, err := m.Filename(); err != nil {
		log.Errorf("failed to obtain filename: %v", err)
	} else {
		if info.Size, err = lib.FileSize(name); err != nil {
			log.Errorf("failed to obtain file size: %v", err)
		}
	}
	filenames, err := m.db.MsgFilenames(m.key)
	if err == nil {
		info.Filenames = filenames
	}
	return info, nil
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
	msg, err := rfc822.ReadMessage(f)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %w", err)
	}
	return rfc822.FetchEntityPartReader(msg, requestedParts)
}

// SetFlag adds or removes a flag from the message.
// Notmuch doesn't support all the flags, and for those this errors.
func (m *Message) SetFlag(flag models.Flags, enable bool) error {
	// Translate the flag into a notmuch tag, ignoring no-op flags.
	tag, ok := flagToTag[flag]
	if !ok {
		return fmt.Errorf("Notmuch doesn't support flag %v", flag)
	}

	// Get the current state of the flag.
	// Note that notmuch handles some flags in an inverted sense
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

	if flagToInvert[flag] {
		enable = !enable
	}

	switch {
	case oldState == enable:
		return nil
	case enable:
		return m.AddTag(tag)
	default:
		return m.RemoveTag(tag)
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

func (m *Message) ModelFlags() (models.Flags, error) {
	var flags models.Flags = models.SeenFlag
	tags, err := m.Tags()
	if err != nil {
		return 0, err
	}
	for _, tag := range tags {
		flag := tagToFlag[tag]
		if flagToInvert[flag] {
			flags &^= flag
		} else {
			flags |= flag
		}
	}
	return flags, nil
}

func (m *Message) UID() uint32 {
	return m.uid
}

func (m *Message) Filename() (string, error) {
	return m.db.MsgFilename(m.key)
}

// AddTag adds a single tag.
// Consider using *Message.ModifyTags for multiple additions / removals
// instead of looping over a tag array
func (m *Message) AddTag(tag string) error {
	return m.ModifyTags([]string{tag}, nil)
}

// RemoveTag removes a single tag.
// Consider using *Message.ModifyTags for multiple additions / removals
// instead of looping over a tag array
func (m *Message) RemoveTag(tag string) error {
	return m.ModifyTags(nil, []string{tag})
}

func (m *Message) ModifyTags(add, remove []string) error {
	return m.db.MsgModifyTags(m.key, add, remove)
}

func (m *Message) Remove(dir maildir.Dir) error {
	filenames, err := m.db.MsgFilenames(m.key)
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		if dirContains(dir, filename) {
			err := m.db.DeleteMessage(filename)
			if err != nil {
				return err
			}

			if err := os.Remove(filename); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("no matching message file found in %s", string(dir))
}

func (m *Message) Copy(target maildir.Dir) error {
	filename, err := m.Filename()
	if err != nil {
		return err
	}

	source, key := parseFilename(filename)
	if key == "" {
		return fmt.Errorf("failed to parse message filename: %s", filename)
	}

	newKey, err := source.Copy(target, key)
	if err != nil {
		return err
	}
	newFilename, err := target.Filename(newKey)
	if err != nil {
		return err
	}
	_, err = m.db.IndexFile(newFilename)
	return err
}

func (m *Message) Move(srcDir, destDir maildir.Dir) error {
	var src string

	filenames, err := m.db.MsgFilenames(m.key)
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		if dirContains(srcDir, filename) {
			src = filename
			break
		}
	}

	if src == "" {
		return fmt.Errorf("no matching message file found in %s", string(srcDir))
	}

	tags, err := m.Tags()
	if err != nil {
		return err
	}

	// Remove encoded UID information from the key to prevent sync issues
	name := lib.StripUIDFromMessageFilename(filepath.Base(src))
	dest := filepath.Join(string(destDir), "cur", name)

	if err := m.db.DeleteMessage(src); err != nil {
		return err
	}

	if err := os.Rename(src, dest); err != nil {
		return err
	}

	if _, err = m.db.IndexFile(dest); err != nil {
		return err
	}

	if err := m.ModifyTags(tags, nil); err != nil {
		return err
	}

	return nil
}

func parseFilename(filename string) (maildir.Dir, string) {
	base := filepath.Base(filename)
	dir := filepath.Dir(filename)
	dir, curdir := filepath.Split(dir)
	if curdir != "cur" {
		return "", ""
	}
	split := strings.Split(base, ":")
	if len(split) < 2 {
		return maildir.Dir(dir), ""
	}
	key := split[0]
	return maildir.Dir(dir), key
}

func dirContains(dir maildir.Dir, filename string) bool {
	for _, sub := range []string{"cur", "new"} {
		match, _ := filepath.Match(filepath.Join(string(dir), sub, "*"), filename)
		if match {
			return true
		}
	}
	return false
}
