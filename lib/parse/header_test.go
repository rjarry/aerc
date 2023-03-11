package parse_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
)

func TestMsgIDList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "valid",
			input:    "<1q@az> (cmt)\r\n <2w@sx> (khld)",
			expected: []string{"1q@az", "2w@sx"},
		},
		{
			name:     "comma",
			input:    "<3e@dc>, <4r@fv>,\t<5t@gb>",
			expected: []string{"3e@dc", "4r@fv", "5t@gb"},
		},
		{
			name:     "other non-CFWS separators",
			input:    "<6y@>, <hn@7u>\n <> <jm@8i>",
			expected: []string{"hn@7u", "jm@8i"},
		},
	}

	for _, test := range tests {
		var h mail.Header
		h.Set("References", test.input)
		t.Run(test.name, func(t *testing.T) {
			actual := parse.MsgIDList(&h, "References")
			assert.Equal(t, test.expected, actual)
		})
	}
}
