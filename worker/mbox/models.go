package mboxer

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
)

type mailboxContainer struct {
	mailboxes map[string]*container
}

func (md *mailboxContainer) Names() []string {
	files := make([]string, 0)
	for file := range md.mailboxes {
		files = append(files, file)
	}
	return files
}

func (md *mailboxContainer) Mailbox(f string) (*container, bool) {
	mb, ok := md.mailboxes[f]
	return mb, ok
}

func (md *mailboxContainer) Create(file string) *container {
	md.mailboxes[file] = &container{filename: file}
	return md.mailboxes[file]
}

func (md *mailboxContainer) Remove(file string) error {
	delete(md.mailboxes, file)
	return nil
}

func (md *mailboxContainer) DirectoryInfo(file string) *models.DirectoryInfo {
	var exists int
	if md, ok := md.Mailbox(file); ok {
		exists = len(md.Uids())
	}
	return &models.DirectoryInfo{
		Name:           file,
		Flags:          []string{},
		ReadOnly:       false,
		Exists:         exists,
		Recent:         0,
		Unseen:         0,
		AccurateCounts: false,
		Caps: &models.Capabilities{
			Sort:   true,
			Thread: false,
		},
	}
}

func (md *mailboxContainer) Copy(dest, src string, uids []uint32) error {
	srcmbox, ok := md.Mailbox(src)
	if !ok {
		return fmt.Errorf("source %s not found", src)
	}
	destmbox, ok := md.Mailbox(dest)
	if !ok {
		return fmt.Errorf("destination %s not found", dest)
	}
	for _, uidSrc := range srcmbox.Uids() {
		found := false
		for _, uid := range uids {
			if uid == uidSrc {
				found = true
				break
			}
		}
		if found {
			msg, err := srcmbox.Message(uidSrc)
			if err != nil {
				return fmt.Errorf("could not get message with uid %d from folder %s", uidSrc, src)
			}
			r, err := msg.NewReader()
			if err != nil {
				return fmt.Errorf("could not get reader for message with uid %d", uidSrc)
			}
			flags, err := msg.ModelFlags()
			if err != nil {
				return fmt.Errorf("could not get flags for message with uid %d", uidSrc)
			}
			destmbox.Append(r, flags)
		}
	}
	md.mailboxes[dest] = destmbox
	return nil
}

type container struct {
	filename string
	messages []lib.RawMessage
}

func (f *container) Uids() []uint32 {
	uids := make([]uint32, len(f.messages))
	for i, m := range f.messages {
		uids[i] = m.UID()
	}
	return uids
}

func (f *container) Message(uid uint32) (lib.RawMessage, error) {
	for _, m := range f.messages {
		if uid == m.UID() {
			return m, nil
		}
	}
	return &message{}, fmt.Errorf("uid [%d] not found", uid)
}

func (f *container) Delete(uids []uint32) (deleted []uint32) {
	newMessages := make([]lib.RawMessage, 0)
	for _, m := range f.messages {
		del := false
		for _, uid := range uids {
			if m.UID() == uid {
				del = true
				break
			}
		}
		if del {
			deleted = append(deleted, m.UID())
		} else {
			newMessages = append(newMessages, m)
		}
	}
	f.messages = newMessages
	return
}

func (f *container) newUid() (next uint32) {
	for _, m := range f.messages {
		if uid := m.UID(); uid > next {
			next = uid
		}
	}
	next++
	return
}

func (f *container) Append(r io.Reader, flags []models.Flag) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	f.messages = append(f.messages, &message{
		uid:     f.newUid(),
		flags:   flags,
		content: data,
	})
	return nil
}

// message implements the lib.RawMessage interface
type message struct {
	uid     uint32
	flags   []models.Flag
	content []byte
}

func (m *message) NewReader() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(m.content)), nil
}

func (m *message) ModelFlags() ([]models.Flag, error) {
	return m.flags, nil
}

func (m *message) Labels() ([]string, error) {
	return nil, nil
}

func (m *message) UID() uint32 {
	return m.uid
}

func (m *message) SetFlag(flag models.Flag, state bool) error {
	flagSet := make(map[models.Flag]bool)
	flags, err := m.ModelFlags()
	if err != nil {
		return err
	}
	for _, f := range flags {
		flagSet[f] = true
	}
	flagSet[flag] = state
	newFlags := make([]models.Flag, 0)
	for flag, isSet := range flagSet {
		if isSet {
			newFlags = append(newFlags, flag)
		}
	}
	m.flags = newFlags
	return nil
}
