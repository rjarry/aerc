package templates

import (
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

type TemplateData struct {
	To      []*mail.Address
	Cc      []*mail.Address
	Bcc     []*mail.Address
	From    []*mail.Address
	Date    time.Time
	Subject string
	// Only available when replying with a quote
	OriginalText     string
	OriginalFrom     []*mail.Address
	OriginalDate     time.Time
	OriginalMIMEType string
}

func ParseTemplateData(h *mail.Header, original models.OriginalMail) TemplateData {
	// we ignore errors as this shouldn't fail the sending / replying even if
	// something is wrong with the message we reply to
	to, _ := h.AddressList("to")
	cc, _ := h.AddressList("cc")
	bcc, _ := h.AddressList("bcc")
	from, _ := h.AddressList("from")
	subject, err := h.Text("subject")
	if err != nil {
		subject = h.Get("subject")
	}

	td := TemplateData{
		To:               to,
		Cc:               cc,
		Bcc:              bcc,
		From:             from,
		Date:             time.Now(),
		Subject:          subject,
		OriginalText:     original.Text,
		OriginalDate:     original.Date,
		OriginalMIMEType: original.MIMEType,
	}
	if original.RFC822Headers != nil {
		origFrom, _ := original.RFC822Headers.AddressList("from")
		td.OriginalFrom = origFrom
	}
	return td
}

// DummyData provides dummy data to test template validity
func DummyData() interface{} {
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
	return ParseTemplateData(h, original)
}
