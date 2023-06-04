package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
)

type IDList struct {
	IDs []jmap.ID
}

func (c *JMAPCache) GetMailboxList() ([]jmap.ID, error) {
	buf, err := c.get(mailboxListKey)
	if err != nil {
		return nil, err
	}
	var list IDList
	err = unmarshal(buf, &list)
	if err != nil {
		return nil, err
	}
	return list.IDs, nil
}

func (c *JMAPCache) PutMailboxList(list []jmap.ID) error {
	buf, err := marshal(&IDList{IDs: list})
	if err != nil {
		return err
	}
	return c.put(mailboxListKey, buf)
}

const mailboxListKey = "mailbox/list"
