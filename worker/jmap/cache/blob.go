package cache

import (
	"os"
	"path"

	"git.sr.ht/~rockorager/go-jmap"
)

func (c *JMAPCache) GetBlob(id jmap.ID) ([]byte, error) {
	fpath := c.blobPath(id)
	if fpath == "" {
		return nil, notfound
	}
	return os.ReadFile(fpath)
}

func (c *JMAPCache) PutBlob(id jmap.ID, buf []byte) error {
	fpath := c.blobPath(id)
	if fpath == "" {
		return nil
	}
	_ = os.MkdirAll(path.Dir(fpath), 0o700)
	return os.WriteFile(fpath, buf, 0o600)
}

func (c *JMAPCache) DeleteBlob(id jmap.ID) error {
	fpath := c.blobPath(id)
	if fpath == "" {
		return nil
	}
	defer func() {
		_ = os.Remove(path.Dir(fpath))
	}()
	return os.Remove(fpath)
}

func (c *JMAPCache) blobPath(id jmap.ID) string {
	if c.blobsDir == "" {
		return ""
	}
	name := string(id)
	sub := name[len(name)-2:]
	return path.Join(c.blobsDir, sub, name)
}
