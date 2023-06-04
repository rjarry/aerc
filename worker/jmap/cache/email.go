package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
)

func (c *JMAPCache) GetEmail(id jmap.ID) (*email.Email, error) {
	buf, err := c.get(emailey(id))
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
	return c.put(emailey(id), buf)
}

func (c *JMAPCache) DeleteEmail(id jmap.ID) error {
	return c.delete(emailey(id))
}

func emailey(id jmap.ID) string {
	return "email/" + string(id)
}
