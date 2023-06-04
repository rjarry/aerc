package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
)

func (c *JMAPCache) GetSession() (*jmap.Session, error) {
	buf, err := c.get(sessionKey)
	if err != nil {
		return nil, err
	}
	s := new(jmap.Session)
	err = unmarshal(buf, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *JMAPCache) PutSession(s *jmap.Session) error {
	buf, err := marshal(s)
	if err != nil {
		return err
	}
	return c.put(sessionKey, buf)
}

func (c *JMAPCache) DeleteSession() error {
	return c.delete(sessionKey)
}

const sessionKey = "session"
