package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
)

func (c *JMAPCache) HasEmail(id jmap.ID) bool {
	_, err := c.get(emailKey(id))
	return err == nil
}

func (c *JMAPCache) GetEmail(id jmap.ID) (*email.Email, error) {
	buf, err := c.get(emailKey(id))
	if err != nil {
		return nil, err
	}
	e := new(email.Email)
	err = unmarshal(buf, e)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (c *JMAPCache) PutEmail(id jmap.ID, e *email.Email) error {
	buf, err := marshal(e)
	if err != nil {
		return err
	}
	return c.put(emailKey(id), buf)
}

func (c *JMAPCache) DeleteEmail(id jmap.ID) error {
	return c.delete(emailKey(id))
}

func emailKey(id jmap.ID) string {
	return "email/" + string(id)
}
