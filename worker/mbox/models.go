package mboxer

import (
	"bytes"
	"fmt"
	"io"

	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/models"
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
		Name:   file,
		Exists: exists,
		Recent: 0,
		Unseen: 0,
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
			err = destmbox.Append(r, flags)
			if err != nil {
				return fmt.Errorf("could not append data to mbox: %w", err)
			}
		}
	}
	md.mailboxes[dest] = destmbox
	return nil
}

type container struct {
	filename string
	messages []rfc822.RawMessage
}

func (f *container) Uids() []uint32 {
	uids := make([]uint32, len(f.messages))
	for i, m := range f.messages {
		uids[i] = m.UID()
	}
	return uids
}

func (f *container) Message(uid uint32) (rfc822.RawMessage, error) {
	for _, m := range f.messages {
		if uid == m.UID() {
			return m, nil
		}
	}
	return &message{}, fmt.Errorf("uid [%d] not found", uid)
}

func (f *container) Delete(uids []uint32) (deleted []uint32) {
	newMessages := make([]rfc822.RawMessage, 0)
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

func (f *container) Append(r io.Reader, flags models.Flags) error {
	data, err := io.ReadAll(r)
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
	flags   models.Flags
	content []byte
}

func (m *message) NewReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.content)), nil
}

func (m *message) ModelFlags() (models.Flags, error) {
	return m.flags, nil
}

func (m *message) Labels() ([]string, error) {
	return nil, nil
}

func (m *message) UID() uint32 {
	return m.uid
}

func (m *message) SetFlag(flag models.Flags, state bool) error {
	if state {
		m.flags |= flag
	} else {
		m.flags &^= flag
	}
	return nil
}
