package templates

import (
	"fmt"
	"strings"
	"time"

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

	// account config
	myAddresses map[string]bool
	account     string
	folder      string // selected folder name

	// ui config
	timeFmt         string
	thisDayTimeFmt  string
	thisWeekTimeFmt string
	thisYearTimeFmt string
	iconAttachment  string
}

func NewTemplateData(
	from *mail.Address,
	aliases []*mail.Address,
	account string,
	folder string,
	timeFmt string,
	thisDayTimeFmt string,
	thisWeekTimeFmt string,
	thisYearTimeFmt string,
	iconAttachment string,
) *TemplateData {
	myAddresses := map[string]bool{from.Address: true}
	for _, addr := range aliases {
		myAddresses[addr.Address] = true
	}
	return &TemplateData{
		myAddresses:     myAddresses,
		account:         account,
		folder:          folder,
		timeFmt:         timeFmt,
		thisDayTimeFmt:  thisDayTimeFmt,
		thisWeekTimeFmt: thisWeekTimeFmt,
		thisYearTimeFmt: thisYearTimeFmt,
		iconAttachment:  iconAttachment,
	}
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

func (d *TemplateData) Account() string {
	return d.account
}

func (d *TemplateData) Folder() string {
	return d.folder
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
	year := date.Year()
	day := date.YearDay()
	now := time.Now()
	thisYear := now.Year()
	thisDay := now.YearDay()
	fmt := d.timeFmt
	if year == thisYear {
		switch {
		case day == thisDay && d.thisDayTimeFmt != "":
			fmt = d.thisDayTimeFmt
		case day > thisDay-7 && d.thisWeekTimeFmt != "":
			fmt = d.thisWeekTimeFmt
		case d.thisYearTimeFmt != "":
			fmt = d.thisYearTimeFmt
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

func (d *TemplateData) Number() string {
	return fmt.Sprintf("%d", d.msgNum)
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
				flags = append(flags, d.iconAttachment)
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

func (d *TemplateData) Size() uint32 {
	if d.info == nil || d.info.Envelope == nil {
		return 0
	}
	return d.info.Size
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

// DummyData provides dummy data to test template validity
func DummyData() *TemplateData {
	from := &mail.Address{
		Name:    "John Doe",
		Address: "john@example.com",
	}
	to := &mail.Address{
		Name:    "Alice Doe",
		Address: "alice@example.com",
	}
	h := &mail.Header{}
	h.SetAddressList("from", []*mail.Address{from})
	h.SetAddressList("to", []*mail.Address{to})

	oh := &mail.Header{}
	oh.SetAddressList("from", []*mail.Address{to})
	oh.SetAddressList("to", []*mail.Address{from})

	original := models.OriginalMail{
		Date:          time.Now(),
		From:          from.String(),
		Text:          "This is only a test text",
		MIMEType:      "text/plain",
		RFC822Headers: oh,
	}
	data := NewTemplateData(
		to,
		nil,
		"account",
		"folder",
		"2006 Jan 02, 15:04 GMT-0700",
		"15:04",
		"Monday 15:04",
		"Jan 02",
		"a",
	)
	data.SetHeaders(h, &original)

	info := &models.MessageInfo{
		BodyStructure: &models.BodyStructure{
			MIMEType:          "text",
			MIMESubType:       "plain",
			Params:            make(map[string]string),
			Description:       "",
			Encoding:          "",
			Parts:             []*models.BodyStructure{},
			Disposition:       "",
			DispositionParams: make(map[string]string),
		},
		Envelope: &models.Envelope{
			Date:      time.Date(1981, 6, 23, 16, 52, 0, 0, time.UTC),
			Subject:   "[PATCH aerc 2/3] foo: baz bar buz",
			From:      []*mail.Address{from},
			ReplyTo:   []*mail.Address{},
			To:        []*mail.Address{to},
			Cc:        []*mail.Address{},
			Bcc:       []*mail.Address{},
			MessageId: "",
			InReplyTo: "",
		},
		Flags:         models.FlaggedFlag,
		Labels:        []string{"inbox", "patch"},
		InternalDate:  time.Now(),
		RFC822Headers: nil,
		Refs:          []string{},
		Size:          65512,
		Uid:           12345,
		Error:         nil,
	}
	data.SetInfo(info, 42, true)

	return data
}
