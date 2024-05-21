package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
)

func (c *JMAPCache) GetThread(id jmap.ID) (*thread.Thread, error) {
	buf, err := c.get(threadKey(id))
	if err != nil {
		return nil, err
	}
	e := new(thread.Thread)
	err = unmarshal(buf, e)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (c *JMAPCache) PutThread(id jmap.ID, e *thread.Thread) error {
	buf, err := marshal(e)
	if err != nil {
		return err
	}
	return c.put(threadKey(id), buf)
}

func (c *JMAPCache) DeleteThread(id jmap.ID) error {
	return c.delete(threadKey(id))
}

func threadKey(id jmap.ID) string {
	return "thread/" + string(id)
}
