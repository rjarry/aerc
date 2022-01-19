package lib

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"git.sr.ht/~rjarry/aerc/models"
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
	path string
}

func newMockRawMessageFromPath(p string) *mockRawMessage {
	return &mockRawMessage{
		path: p,
	}
}

func (m *mockRawMessage) NewReader() (io.ReadCloser, error) {
	return os.Open(m.path)
}
func (m *mockRawMessage) ModelFlags() ([]models.Flag, error) { return nil, nil }
func (m *mockRawMessage) Labels() ([]string, error)          { return nil, nil }
func (m *mockRawMessage) UID() uint32                        { return 0 }

func die(err error) {
	if err != nil {
		panic(err)
	}
}
