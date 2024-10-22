package cache

import (
	"git.sr.ht/~rockorager/go-jmap"
)

func (c *JMAPCache) GetThread(id jmap.ID) ([]jmap.ID, error) {
	buf, err := c.get(threadKey(id))
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

func (c *JMAPCache) PutThread(id jmap.ID, list []jmap.ID) error {
	buf, err := marshal(&IDList{IDs: list})
	if err != nil {
		return err
	}
	return c.put(threadKey(id), buf)
}

func (c *JMAPCache) DeleteThread(id jmap.ID) error {
	return c.delete(mailboxKey(id))
}

func threadKey(id jmap.ID) string {
	return "thread/" + string(id)
}
