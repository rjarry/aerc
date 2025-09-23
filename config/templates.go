package config

import (
	"path"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

type TemplateConfig struct {
	TemplateDirs []string `ini:"template-dirs" delim:":"`
	NewMessage   string   `ini:"new-message" default:"new_message"`
	QuotedReply  string   `ini:"quoted-reply" default:"quoted_reply"`
	Forwards     string   `ini:"forwards" default:"forward_as_body"`
}

var templatesConfig atomic.Pointer[TemplateConfig]

func Templates() *TemplateConfig {
	return templatesConfig.Load()
}

func parseTemplates(file *ini.File) (*TemplateConfig, error) {
	conf := new(TemplateConfig)
	if err := MapToStruct(file.Section("templates"), conf, true); err != nil {
		return nil, err
	}

	// append default paths to template-dirs
	for _, dir := range SearchDirs {
		conf.TemplateDirs = append(
			conf.TemplateDirs, path.Join(dir, "templates"),
		)
	}

	// we want to fail during startup if the templates are not ok
	// hence we do dummy executes here
	if err := checkTemplate(conf.NewMessage, conf.TemplateDirs); err != nil {
		return nil, err
	}
	if err := checkTemplate(conf.QuotedReply, conf.TemplateDirs); err != nil {
		return nil, err
	}
	if err := checkTemplate(conf.Forwards, conf.TemplateDirs); err != nil {
		return nil, err
	}

	log.Debugf("aerc.conf: [templates] %#v", conf)

	return conf, nil
}

func checkTemplate(filename string, dirs []string) error {
	var data dummyData
	_, err := templates.ParseTemplateFromFile(filename, dirs, &data)
	return err
}

// only for validation
type dummyData struct{}

var (
	addr1 = mail.Address{Name: "John Foo", Address: "foo@bar.org"}
	addr2 = mail.Address{Name: "John Bar", Address: "bar@foo.org"}
)

func (d *dummyData) Account() string                 { return "work" }
func (d *dummyData) AccountBackend() string          { return "maildir" }
func (d *dummyData) AccountFrom() *mail.Address      { return &addr1 }
func (d *dummyData) Signature() string               { return "" }
func (d *dummyData) Folder() string                  { return "INBOX" }
func (d *dummyData) To() []*mail.Address             { return []*mail.Address{&addr1} }
func (d *dummyData) Cc() []*mail.Address             { return nil }
func (d *dummyData) Bcc() []*mail.Address            { return nil }
func (d *dummyData) From() []*mail.Address           { return []*mail.Address{&addr2} }
func (d *dummyData) Peer() []*mail.Address           { return d.From() }
func (d *dummyData) ReplyTo() []*mail.Address        { return nil }
func (d *dummyData) Date() time.Time                 { return time.Now() }
func (d *dummyData) DateAutoFormat(time.Time) string { return "" }
func (d *dummyData) Header(string) string            { return "" }
func (d *dummyData) ThreadPrefix() string            { return "└─>" }
func (d *dummyData) ThreadCount() int                { return 0 }
func (d *dummyData) ThreadUnread() int               { return 0 }
func (d *dummyData) ThreadFolded() bool              { return false }
func (d *dummyData) ThreadContext() bool             { return true }
func (d *dummyData) ThreadOrphan() bool              { return true }
func (d *dummyData) Subject() string                 { return "Re: [PATCH] hey" }
func (d *dummyData) SubjectBase() string             { return "[PATCH] hey" }
func (d *dummyData) Attach(string) string            { return "" }
func (d *dummyData) Number() int                     { return 0 }
func (d *dummyData) Labels() []string                { return nil }
func (d *dummyData) Filename() string                { return "" }
func (d *dummyData) Filenames() []string             { return nil }
func (d *dummyData) Flags() []string                 { return nil }
func (d *dummyData) IsReplied() bool                 { return true }
func (d *dummyData) HasAttachment() bool             { return true }
func (d *dummyData) IsRecent() bool                  { return false }
func (d *dummyData) IsUnread() bool                  { return false }
func (d *dummyData) IsFlagged() bool                 { return false }
func (d *dummyData) IsDraft() bool                   { return false }
func (d *dummyData) IsMarked() bool                  { return false }
func (d *dummyData) IsForwarded() bool               { return false }
func (d *dummyData) MessageId() string               { return "123456789@foo.org" }
func (d *dummyData) Size() int                       { return 420 }
func (d *dummyData) OriginalText() string            { return "Blah blah blah" }
func (d *dummyData) OriginalDate() time.Time         { return time.Now() }
func (d *dummyData) OriginalFrom() []*mail.Address   { return d.From() }
func (d *dummyData) OriginalMIMEType() string        { return "text/plain" }
func (d *dummyData) OriginalHeader(string) string    { return "" }
func (d *dummyData) Recent(...string) int            { return 1 }
func (d *dummyData) Unread(...string) int            { return 3 }
func (d *dummyData) Exists(...string) int            { return 14 }
func (d *dummyData) RUE(...string) string            { return "1/3/14" }
func (d *dummyData) Connected() bool                 { return false }
func (d *dummyData) ConnectionInfo() string          { return "" }
func (d *dummyData) ContentInfo() string             { return "" }
func (d *dummyData) StatusInfo() string              { return "" }
func (d *dummyData) TrayInfo() string                { return "" }
func (d *dummyData) PendingKeys() string             { return "" }
func (d *dummyData) Role() string                    { return "inbox" }

func (d *dummyData) Style(string, string) string               { return "" }
func (d *dummyData) StyleSwitch(string, ...models.Case) string { return "" }

func (d *dummyData) StyleMap([]string, ...models.Case) []string { return []string{} }
