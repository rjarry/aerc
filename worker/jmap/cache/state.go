package cache

func (c *JMAPCache) GetMailboxState() (string, error) {
	buf, err := c.get(mailboxStateKey)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (c *JMAPCache) PutMailboxState(state string) error {
	return c.put(mailboxStateKey, []byte(state))
}

func (c *JMAPCache) GetThreadState() (string, error) {
	buf, err := c.get(threadStateKey)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (c *JMAPCache) PutThreadState(state string) error {
	return c.put(threadStateKey, []byte(state))
}

func (c *JMAPCache) GetEmailState() (string, error) {
	buf, err := c.get(emailStateKey)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (c *JMAPCache) PutEmailState(state string) error {
	return c.put(emailStateKey, []byte(state))
}

const (
	mailboxStateKey = "state/mailbox"
	emailStateKey   = "state/email"
	threadStateKey  = "state/thread"
)
