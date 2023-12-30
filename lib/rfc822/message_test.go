package rfc822

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/require"
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
			mi, err := MessageInfo(m)
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

func TestParseMessageDate(t *testing.T) {
	// we use different times for "Date" and "Received" fields to make sure the right one is parsed
	tests := []struct {
		date     string
		received string
		utc      time.Time
	}{
		{
			date:     "Fri, 22 Dec 2023 11:19:01 +0000",
			received: "from aaa.bbb.com for <user@host.com>; Fri, 22 Dec 2023 06:19:02 -0500 (EST)",
			utc:      time.Date(2023, time.December, 22, 11, 19, 1, 0, time.UTC),
		},
		{
			date:     "Fri, 29 Dec 2023 14:06:37 +0100",
			received: "from somewhere.com for a@b.c; Fri, 30 Dec 2023 4:06:43 +1300",
			utc:      time.Date(2023, time.December, 29, 13, 6, 37, 0, time.UTC),
		},
		{
			date:     "Fri, 29 Dec 2023 00:51:00 EST",
			received: "by hostname.com; Fri, 29 Dec 2023 00:51:33 -0500 (EST)",
			utc:      time.Date(2023, time.December, 29, 5, 51, 33, 0, time.UTC),
		},
	}

	for _, test := range tests {
		h := mail.Header{}
		h.SetText("Date", test.date)
		h.SetText("Received", test.received)
		res, err := parseDate(&h)
		require.Nil(t, err)
		require.Equal(t, test.utc, res.UTC())
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
