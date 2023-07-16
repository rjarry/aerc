package xgmext_test

import (
	"bytes"
	"testing"

	"git.sr.ht/~rjarry/aerc/worker/imap/extensions/xgmext"
	"github.com/emersion/go-imap"
)

func TestXGMEXT_Search(t *testing.T) {
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
