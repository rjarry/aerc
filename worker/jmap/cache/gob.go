package cache

import (
	"bytes"
	"encoding/gob"

	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

type jmapObject interface {
	*email.Email |
		*email.QueryResponse |
		*mailbox.Mailbox |
		*FolderContents |
		*IDList
}

func marshal[T jmapObject](obj T) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(obj)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshal[T jmapObject](data []byte, obj T) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(obj)
}
