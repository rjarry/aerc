package cache

import (
	"reflect"

	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
)

type FolderContents struct {
	MailboxID  jmap.ID
	QueryState string
	Filter     *email.FilterCondition
	Sort       []*email.SortComparator
	MessageIDs []jmap.ID
}

func (c *JMAPCache) GetFolderContents(mailboxId jmap.ID) (*FolderContents, error) {
	buf, err := c.get(folderContentsKey(mailboxId))
	if err != nil {
		return nil, err
	}
	m := new(FolderContents)
	err = unmarshal(buf, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *JMAPCache) PutFolderContents(mailboxId jmap.ID, m *FolderContents) error {
	buf, err := marshal(m)
	if err != nil {
		return err
	}
	return c.put(folderContentsKey(mailboxId), buf)
}

func (c *JMAPCache) DeleteFolderContents(mailboxId jmap.ID) error {
	return c.delete(folderContentsKey(mailboxId))
}

func folderContentsKey(mailboxId jmap.ID) string {
	return "foldercontents/" + string(mailboxId)
}

func (f *FolderContents) NeedsRefresh(
	filter *email.FilterCondition, sort []*email.SortComparator,
) bool {
	if f.QueryState == "" || f.Filter == nil || len(f.Sort) != len(sort) {
		return true
	}

	for i := 0; i < len(sort) && i < len(f.Sort); i++ {
		if !reflect.DeepEqual(sort[i], f.Sort[i]) {
			return true
		}
	}

	return !reflect.DeepEqual(filter, f.Filter)
}
