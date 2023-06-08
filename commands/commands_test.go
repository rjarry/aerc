package commands

import (
	"reflect"
	"testing"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

func TestExecuteCommand_expand(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"prompt", "Really quit? ", "quit"},
			want: []string{"prompt", "Really quit? ", "quit"},
		},
		{
			args: []string{"{{", "print", "\"hello\"", "}}"},
			want: []string{"hello"},
		},
		{
			args: []string{"prompt", "Really quit  ? ", "  quit "},
			want: []string{"prompt", "Really quit  ? ", "  quit "},
		},
		{
			args: []string{
				"prompt", "Really quit? ", "{{",
				"print", "\"quit\"", "}}",
			},
			want: []string{"prompt", "Really quit? ", "quit"},
		},
		{
			args: []string{
				"prompt", "Really quit? ", "{{",
				"if", "1", "}}", "quit", "{{end}}",
			},
			want: []string{"prompt", "Really quit? ", "quit"},
		},
	}

	var data dummyData

	for i, test := range tests {
		got, err := expand(&data, test.args)
		if err != nil {
			t.Errorf("test %d failed with err: %v", i, err)
		} else if !reflect.DeepEqual(got, test.want) {
			t.Errorf("test %d failed: "+
				"got: %v, but want: %v", i, got, test.want)
		}
	}
}

// only for validation
type dummyData struct{}

var (
	addr1 = mail.Address{Name: "John Foo", Address: "foo@bar.org"}
	addr2 = mail.Address{Name: "John Bar", Address: "bar@foo.org"}
)

func (d *dummyData) Account() string                           { return "work" }
func (d *dummyData) Folder() string                            { return "INBOX" }
func (d *dummyData) To() []*mail.Address                       { return []*mail.Address{&addr1} }
func (d *dummyData) Cc() []*mail.Address                       { return nil }
func (d *dummyData) Bcc() []*mail.Address                      { return nil }
func (d *dummyData) From() []*mail.Address                     { return []*mail.Address{&addr2} }
func (d *dummyData) Peer() []*mail.Address                     { return d.From() }
func (d *dummyData) ReplyTo() []*mail.Address                  { return nil }
func (d *dummyData) Date() time.Time                           { return time.Now() }
func (d *dummyData) DateAutoFormat(time.Time) string           { return "" }
func (d *dummyData) Header(string) string                      { return "" }
func (d *dummyData) ThreadPrefix() string                      { return "└─>" }
func (d *dummyData) Subject() string                           { return "Re: [PATCH] hey" }
func (d *dummyData) SubjectBase() string                       { return "[PATCH] hey" }
func (d *dummyData) Number() int                               { return 0 }
func (d *dummyData) Labels() []string                          { return nil }
func (d *dummyData) Flags() []string                           { return nil }
func (d *dummyData) IsReplied() bool                           { return true }
func (d *dummyData) HasAttachment() bool                       { return true }
func (d *dummyData) IsRecent() bool                            { return false }
func (d *dummyData) IsUnread() bool                            { return false }
func (d *dummyData) IsFlagged() bool                           { return false }
func (d *dummyData) IsMarked() bool                            { return false }
func (d *dummyData) MessageId() string                         { return "123456789@foo.org" }
func (d *dummyData) Size() int                                 { return 420 }
func (d *dummyData) OriginalText() string                      { return "Blah blah blah" }
func (d *dummyData) OriginalDate() time.Time                   { return time.Now() }
func (d *dummyData) OriginalFrom() []*mail.Address             { return d.From() }
func (d *dummyData) OriginalMIMEType() string                  { return "text/plain" }
func (d *dummyData) OriginalHeader(string) string              { return "" }
func (d *dummyData) Recent(...string) int                      { return 1 }
func (d *dummyData) Unread(...string) int                      { return 3 }
func (d *dummyData) Exists(...string) int                      { return 14 }
func (d *dummyData) RUE(...string) string                      { return "1/3/14" }
func (d *dummyData) Connected() bool                           { return false }
func (d *dummyData) ConnectionInfo() string                    { return "" }
func (d *dummyData) ContentInfo() string                       { return "" }
func (d *dummyData) StatusInfo() string                        { return "" }
func (d *dummyData) TrayInfo() string                          { return "" }
func (d *dummyData) PendingKeys() string                       { return "" }
func (d *dummyData) Role() string                              { return "inbox" }
func (d *dummyData) Style(string, string) string               { return "" }
func (d *dummyData) StyleSwitch(string, ...models.Case) string { return "" }

func (d *dummyData) StyleMap([]string, ...models.Case) []string { return []string{} }
