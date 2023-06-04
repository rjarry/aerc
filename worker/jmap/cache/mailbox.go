package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func (c *JMAPCache) GetMailbox(id jmap.ID) (*mailbox.Mailbox, error) {
	buf, err := c.get(mailboxKey(id))
	if err != nil {
		return nil, err
	}
	m := new(mailbox.Mailbox)
	err = unmarshal(buf, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *JMAPCache) PutMailbox(id jmap.ID, m *mailbox.Mailbox) error {
	buf, err := marshal(m)
	if err != nil {
		return err
	}
	return c.put(mailboxKey(id), buf)
}

func (c *JMAPCache) DeleteMailbox(id jmap.ID) error {
	return c.delete(mailboxKey(id))
}

func mailboxKey(id jmap.ID) string {
	return "mailbox/" + string(id)
}
