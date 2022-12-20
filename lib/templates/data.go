package templates

import (
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

type TemplateData struct {
	msg *mail.Header
	// Only available when replying with a quote
	parent *models.OriginalMail
}

func NewTemplateData(
	msg *mail.Header, parent *models.OriginalMail,
) *TemplateData {
	return &TemplateData{
		msg:    msg,
		parent: parent,
	}
}

func (d *TemplateData) To() []*mail.Address {
	to, _ := d.msg.AddressList("to")
	return to
}

func (d *TemplateData) Cc() []*mail.Address {
	to, _ := d.msg.AddressList("cc")
	return to
}

func (d *TemplateData) Bcc() []*mail.Address {
	to, _ := d.msg.AddressList("bcc")
	return to
}

func (d *TemplateData) From() []*mail.Address {
	to, _ := d.msg.AddressList("from")
	return to
}

func (d *TemplateData) Date() time.Time {
	return time.Now()
}

func (d *TemplateData) Subject() string {
	subject, err := d.msg.Text("subject")
	if err != nil {
		subject = d.msg.Get("subject")
	}
	return subject
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
	return NewTemplateData(h, &original)
}
