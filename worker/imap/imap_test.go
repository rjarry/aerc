package imap

import (
	"testing"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
)

func TestTranslateEnvelope(t *testing.T) {
	date, _ := time.Parse("2010-01-31", "1992-10-24")
	givenAddress := imap.Address{
		PersonalName: "PERSONAL_NAME",
		AtDomainList: "AT_DOMAIN_LIST",
		MailboxName:  "MAILBOX_NAME",
		HostName:     "HOST_NAME",
	}
	givenMessageID := " \r\n\r  \t <initial-message-id@with-leading-space>\t\r"
	given := imap.Envelope{
		Date:      date,
		Subject:   "Test Subject",
		From:      []*imap.Address{&givenAddress},
		ReplyTo:   []*imap.Address{&givenAddress},
		To:        []*imap.Address{&givenAddress},
		Cc:        []*imap.Address{&givenAddress},
		Bcc:       []*imap.Address{&givenAddress},
		MessageId: givenMessageID,
		InReplyTo: givenMessageID,
	}
	expectedMessageID := "initial-message-id@with-leading-space"
	expectedAddress := mail.Address{
		Name:    "PERSONAL_NAME",
		Address: "MAILBOX_NAME@HOST_NAME",
	}
	expected := models.Envelope{
		Date:      date,
		Subject:   "Test Subject",
		From:      []*mail.Address{&expectedAddress},
		ReplyTo:   []*mail.Address{&expectedAddress},
		To:        []*mail.Address{&expectedAddress},
		Cc:        []*mail.Address{&expectedAddress},
		Bcc:       []*mail.Address{&expectedAddress},
		MessageId: expectedMessageID,
		InReplyTo: expectedMessageID,
	}
	assert.Equal(t, &expected, translateEnvelope(&given))
}
