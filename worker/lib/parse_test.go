package lib

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"git.sr.ht/~sircmpwn/aerc/models"
)

func TestMessageInfoHandledError(t *testing.T) {
	rootDir := "testdata/message/invalid"
	msgFiles, err := ioutil.ReadDir(rootDir)
	die(err)

	for _, fi := range msgFiles {
		if fi.IsDir() {
			continue
		}

		p := fi.Name()
		t.Run(p, func(t *testing.T) {
			m := newMockRawMessageFromPath(filepath.Join(rootDir, p))
			mi, err := MessageInfo(m)
			if err != nil {
				t.Fatal(err)
			}

			if perr := mi.Error; perr == nil {
				t.Fatal("Expected MessageInfo.Error, got none")
			}
		})
	}
}

type mockRawMessage struct {
	body []byte
}

func newMockRawMessage(body []byte) *mockRawMessage {
	return &mockRawMessage{
		body: body,
	}
}

func newMockRawMessageFromPath(p string) *mockRawMessage {
	b, err := ioutil.ReadFile(p)
	die(err)
	return newMockRawMessage(b)
}

func (m *mockRawMessage) NewReader() (io.Reader, error) {
	return bytes.NewReader(m.body), nil
}
func (m *mockRawMessage) ModelFlags() ([]models.Flag, error) { return nil, nil }
func (m *mockRawMessage) Labels() ([]string, error)          { return nil, nil }
func (m *mockRawMessage) UID() uint32                        { return 0 }

func die(err error) {
	if err != nil {
		panic(err)
	}
}
