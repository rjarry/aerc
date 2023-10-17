package rfc822_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/models"
)

func TestMessageInfoParser(t *testing.T) {
	rootDir := "testdata/message/valid"
	msgFiles, err := os.ReadDir(rootDir)
	die(err)

	for _, fi := range msgFiles {
		if fi.IsDir() {
			continue
		}

		p := fi.Name()
		t.Run(p, func(t *testing.T) {
			m := newMockRawMessageFromPath(filepath.Join(rootDir, p))
			mi, err := rfc822.MessageInfo(m)
			if err != nil {
				t.Fatal("Failed to create MessageInfo with:", err)
			}

			if perr := mi.Error; perr != nil {
				t.Fatal("Expected no parsing error, but got:", mi.Error)
			}
		})
	}
}

func TestMessageInfoHandledError(t *testing.T) {
	rootDir := "testdata/message/invalid"
	msgFiles, err := os.ReadDir(rootDir)
	die(err)

	for _, fi := range msgFiles {
		if fi.IsDir() {
			continue
		}

		p := fi.Name()
		t.Run(p, func(t *testing.T) {
			m := newMockRawMessageFromPath(filepath.Join(rootDir, p))
			mi, err := rfc822.MessageInfo(m)
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
func (m *mockRawMessage) ModelFlags() (models.Flags, error) { return 0, nil }
func (m *mockRawMessage) Labels() ([]string, error)         { return nil, nil }
func (m *mockRawMessage) UID() uint32                       { return 0 }

func die(err error) {
	if err != nil {
		panic(err)
	}
}
