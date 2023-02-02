package state

import (
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/models"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-message/mail"
)

type TemplateData struct {
	// only available when composing/replying/forwarding
	headers *mail.Header
	// only available when replying with a quote
	parent *models.OriginalMail
	// only available for the message list
	info   *models.MessageInfo
	marked bool
	msgNum int

	// message list threading
	ThreadSameSubject bool
	ThreadPrefix      string

	// selected account
	account     *config.AccountConfig
	myAddresses map[string]bool
	folder      string // selected folder name
	folders     []string
	getRUEcount func(string) (int, int, int)
}

// only used for compose/reply/forward
func (d *TemplateData) SetHeaders(h *mail.Header, o *models.OriginalMail) {
	d.headers = h
	d.parent = o
}

// only used for message list templates
func (d *TemplateData) SetInfo(info *models.MessageInfo, num int, marked bool) {
	d.info = info
	d.msgNum = num
	d.marked = marked
}

func (d *TemplateData) SetAccount(acct *config.AccountConfig) {
	d.account = acct
	d.myAddresses = map[string]bool{acct.From.Address: true}
	for _, addr := range acct.Aliases {
		d.myAddresses[addr.Address] = true
	}
}

func (d *TemplateData) SetFolder(folder string) {
	d.folder = folder
}

func (d *TemplateData) SetRUE(folders []string, cb func(string) (int, int, int)) {
	d.folders = folders
	d.getRUEcount = cb
}

func (d *TemplateData) Account() string {
	if d.account != nil {
		return d.account.Name
	}
	return ""
}

func (d *TemplateData) Folder() string {
	return d.folder
}

func (d *TemplateData) ui() *config.UIConfig {
	return config.Ui.ForAccount(d.Account()).ForFolder(d.folder)
}

func (d *TemplateData) To() []*mail.Address {
	var to []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		to = d.info.Envelope.To
	case d.headers != nil:
		to, _ = d.headers.AddressList("to")
	}
	return to
}

func (d *TemplateData) Cc() []*mail.Address {
	var cc []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		cc = d.info.Envelope.Cc
	case d.headers != nil:
		cc, _ = d.headers.AddressList("cc")
	}
	return cc
}

func (d *TemplateData) Bcc() []*mail.Address {
	var bcc []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		bcc = d.info.Envelope.Bcc
	case d.headers != nil:
		bcc, _ = d.headers.AddressList("bcc")
	}
	return bcc
}

func (d *TemplateData) From() []*mail.Address {
	var from []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		from = d.info.Envelope.From
	case d.headers != nil:
		from, _ = d.headers.AddressList("from")
	}
	return from
}

func (d *TemplateData) Peer() []*mail.Address {
	var from, to []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		from = d.info.Envelope.From
		to = d.info.Envelope.To
	case d.headers != nil:
		from, _ = d.headers.AddressList("from")
		to, _ = d.headers.AddressList("to")
	}
	for _, addr := range from {
		if d.myAddresses[addr.Address] {
			return to
		}
	}
	return from
}

func (d *TemplateData) ReplyTo() []*mail.Address {
	var replyTo []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		replyTo = d.info.Envelope.ReplyTo
	case d.headers != nil:
		replyTo, _ = d.headers.AddressList("reply-to")
	}
	return replyTo
}

func (d *TemplateData) Date() time.Time {
	var date time.Time
	switch {
	case d.info != nil && d.info.Envelope != nil:
		date = d.info.Envelope.Date
	case d.info != nil:
		date = d.info.InternalDate
	default:
		date = time.Now()
	}
	return date
}

func (d *TemplateData) DateAutoFormat(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	ui := d.ui()
	year := date.Year()
	day := date.YearDay()
	now := time.Now()
	thisYear := now.Year()
	thisDay := now.YearDay()
	fmt := ui.TimestampFormat
	if year == thisYear {
		switch {
		case day == thisDay && ui.ThisDayTimeFormat != "":
			fmt = ui.ThisDayTimeFormat
		case day > thisDay-7 && ui.ThisWeekTimeFormat != "":
			fmt = ui.ThisWeekTimeFormat
		case ui.ThisYearTimeFormat != "":
			fmt = ui.ThisYearTimeFormat
		}
	}
	return date.Format(fmt)
}

func (d *TemplateData) Header(name string) string {
	var h *mail.Header
	switch {
	case d.headers != nil:
		h = d.headers
	case d.info != nil && d.info.RFC822Headers != nil:
		h = d.info.RFC822Headers
	default:
		return ""
	}
	text, err := h.Text(name)
	if err != nil {
		text = h.Get(name)
	}
	return text
}

func (d *TemplateData) Subject() string {
	var subject string
	switch {
	case d.info != nil && d.info.Envelope != nil:
		subject = d.info.Envelope.Subject
	case d.headers != nil:
		subject = d.Header("subject")
	}
	if d.ThreadSameSubject {
		subject = ""
	}
	return d.ThreadPrefix + subject
}

func (d *TemplateData) SubjectBase() string {
	base, _ := sortthread.GetBaseSubject(d.Subject())
	return base
}

func (d *TemplateData) Number() int {
	return d.msgNum
}

func (d *TemplateData) Labels() []string {
	if d.info == nil {
		return nil
	}
	return d.info.Labels
}

func (d *TemplateData) Flags() []string {
	var flags []string
	if d.info == nil {
		return flags
	}

	switch {
	case d.info.Flags.Has(models.SeenFlag | models.AnsweredFlag):
		flags = append(flags, "r") // message has been replied to
	case d.info.Flags.Has(models.SeenFlag):
		break
	case d.info.Flags.Has(models.RecentFlag):
		flags = append(flags, "N") // message is new
	default:
		flags = append(flags, "O") // message is old
	}
	if d.info.Flags.Has(models.DeletedFlag) {
		flags = append(flags, "D")
	}
	if d.info.BodyStructure != nil {
		for _, bS := range d.info.BodyStructure.Parts {
			if strings.ToLower(bS.Disposition) == "attachment" {
				flags = append(flags, d.ui().IconAttachment)
				break
			}
		}
	}
	if d.info.Flags.Has(models.FlaggedFlag) {
		flags = append(flags, "!")
	}
	if d.marked {
		flags = append(flags, "*")
	}
	return flags
}

func (d *TemplateData) MessageId() string {
	if d.info == nil || d.info.Envelope == nil {
		return ""
	}
	return d.info.Envelope.MessageId
}

func (d *TemplateData) Size() int {
	if d.info == nil || d.info.Envelope == nil {
		return 0
	}
	return int(d.info.Size)
}

func (d *TemplateData) OriginalText() string {
	if d.parent == nil {
		return ""
	}
	return d.parent.Text
}

func (d *TemplateData) OriginalDate() time.Time {
	if d.parent == nil {
		return time.Time{}
	}
	return d.parent.Date
}

func (d *TemplateData) OriginalFrom() []*mail.Address {
	if d.parent == nil || d.parent.RFC822Headers == nil {
		return nil
	}
	from, _ := d.parent.RFC822Headers.AddressList("from")
	return from
}

func (d *TemplateData) OriginalMIMEType() string {
	if d.parent == nil {
		return ""
	}
	return d.parent.MIMEType
}

func (d *TemplateData) OriginalHeader(name string) string {
	if d.parent == nil || d.parent.RFC822Headers == nil {
		return ""
	}
	text, err := d.parent.RFC822Headers.Text(name)
	if err != nil {
		text = d.parent.RFC822Headers.Get(name)
	}
	return text
}

func (d *TemplateData) rue(folders ...string) (int, int, int) {
	var recent, unread, exists int
	if d.getRUEcount != nil {
		if len(folders) == 0 {
			folders = d.folders
		}
		for _, dir := range folders {
			r, u, e := d.getRUEcount(dir)
			recent += r
			unread += u
			exists += e
		}
	}
	return recent, unread, exists
}

func (d *TemplateData) Recent(folders ...string) int {
	r, _, _ := d.rue(folders...)
	return r
}

func (d *TemplateData) Unread(folders ...string) int {
	_, u, _ := d.rue(folders...)
	return u
}

func (d *TemplateData) Exists(folders ...string) int {
	_, _, e := d.rue(folders...)
	return e
}
