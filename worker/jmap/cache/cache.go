package cache

import (
	"errors"
	"os"
	"path"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/mitchellh/go-homedir"
	"github.com/syndtr/goleveldb/leveldb"
)

type JMAPCache struct {
	mem      map[string][]byte
	file     *leveldb.DB
	blobsDir string
}

func NewJMAPCache(state, blobs bool, accountName string) *JMAPCache {
	c := new(JMAPCache)
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir, err = homedir.Expand("~/.cache")
		if err != nil {
			log.Errorf("homedir.Expand: %s", err)
			cacheDir = ""
		}
	}
	if state && cacheDir != "" {
		dir := path.Join(cacheDir, "aerc", accountName, "state")
		_ = os.MkdirAll(dir, 0o700)
		c.file, err = leveldb.OpenFile(dir, nil)
		if err != nil {
			log.Errorf("failed to open goleveldb: %s", err)
			c.mem = make(map[string][]byte)
		}
	} else {
		c.mem = make(map[string][]byte)
	}
	if blobs && cacheDir != "" {
		c.blobsDir = path.Join(cacheDir, "aerc", accountName, "blobs")
	}
	return c
}

var notfound = errors.New("key not found")

func (c *JMAPCache) get(key string) ([]byte, error) {
	switch {
	case c.file != nil:
		return c.file.Get([]byte(key), nil)
	case c.mem != nil:
		value, ok := c.mem[key]
		if !ok {
			return nil, notfound
		}
		return value, nil
	}
	panic("jmap cache with no backend")
}

func (c *JMAPCache) put(key string, value []byte) error {
	switch {
	case c.file != nil:
		return c.file.Put([]byte(key), value, nil)
	case c.mem != nil:
		c.mem[key] = value
		return nil
	}
	panic("jmap cache with no backend")
}

func (c *JMAPCache) delete(key string) error {
	switch {
	case c.file != nil:
		return c.file.Delete([]byte(key), nil)
	case c.mem != nil:
		delete(c.mem, key)
		return nil
	}
	panic("jmap cache with no backend")
}
