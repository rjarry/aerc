package mboxer

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func createMailboxContainer(path string) (*mailboxContainer, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	mbdata := &mailboxContainer{mailboxes: make(map[string]*container)}

	openMboxFile := func(path string, r io.Reader) error {
		// read mbox file
		messages, err := Read(r)
		if err != nil {
			return err
		}
		_, name := filepath.Split(path)
		name = strings.TrimSuffix(name, ".mbox")
		mbdata.mailboxes[name] = &container{filename: path, messages: messages}
		return nil
	}

	if fileInfo.IsDir() {
		files, err := filepath.Glob(filepath.Join(path, "*.mbox"))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				continue
			}
			if err := openMboxFile(file, f); err != nil {
				return nil, err
			}
			f.Close()
		}
	} else {
		if err := openMboxFile(path, file); err != nil {
			return nil, err
		}
	}

	return mbdata, nil
}
