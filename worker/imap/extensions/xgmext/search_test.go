package xgmext_test

import (
	"bytes"
	"testing"

	"git.sr.ht/~rjarry/aerc/worker/imap/extensions/xgmext"
	"github.com/emersion/go-imap"
)

func TestXGMEXT_ThreadIDSearch(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want string
	}{
		{
			name: "search for single id",
			ids:  []string{"1234"},
			want: "* SEARCH CHARSET UTF-8 X-GM-THRID 1234\r\n",
		},
		{
			name: "search for multiple id",
			ids:  []string{"1234", "5678", "2345"},
			want: "* SEARCH CHARSET UTF-8 OR OR X-GM-THRID 1234 X-GM-THRID 5678 X-GM-THRID 2345\r\n",
		},
	}
	for _, test := range tests {
		cmd := xgmext.NewThreadIDSearch(test.ids).Command()
		var buf bytes.Buffer
		err := cmd.WriteTo(imap.NewWriter(&buf))
		if err != nil {
			t.Errorf("failed to write command: %v", err)
		}
		if got := buf.String(); got != test.want {
			t.Errorf("test '%s' failed: got: '%s', but wanted: '%s'",
				test.name, got, test.want)
		}
	}
}

func TestXGMEXT_RawSearch(t *testing.T) {
	tests := []struct {
		name   string
		search string
		want   string
	}{
		{
			name:   "search messages from mailing list",
			search: "list:info@example.com",
			want:   "* SEARCH CHARSET UTF-8 X-GM-RAW list:info@example.com\r\n",
		},
		{
			name:   "search for an exact phrase",
			search: "\"good morning\"",
			want:   "* SEARCH CHARSET UTF-8 X-GM-RAW \"good morning\"\r\n",
		},
		{
			name:   "group multiple search terms together",
			search: "subject:(dinner movie)",
			want:   "* SEARCH CHARSET UTF-8 X-GM-RAW subject:(dinner movie)\r\n",
		},
	}
	for _, test := range tests {
		cmd := xgmext.NewRawSearch(test.search).Command()
		var buf bytes.Buffer
		err := cmd.WriteTo(imap.NewWriter(&buf))
		if err != nil {
			t.Errorf("failed to write command: %v", err)
		}
		if got := buf.String(); got != test.want {
			t.Errorf("test '%s' failed: got: '%s', but wanted: '%s'",
				test.name, got, test.want)
		}
	}
}
