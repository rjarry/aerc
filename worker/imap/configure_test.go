package imap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderFromURL(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		url      string
		provider imapProvider
	}{
		// Positive tests.
		{
			url:      "imap.gmail.com",
			provider: GMail,
		},
		{
			url:      "127.0.0.1",
			provider: Proton,
		},
		{
			url:      "outlook.office365.com",
			provider: Office365,
		},
		{
			url:      "imap.zoho.com",
			provider: Zoho,
		},
		{
			url:      "imap.fastmail.com",
			provider: FastMail,
		},
		{
			url:      "imap.mail.me.com",
			provider: iCloud,
		},
		// Negative tests
		{
			url:      "imp.gmail.com",
			provider: Unknown,
		},
		{
			url:      "127.0.0.10",
			provider: Unknown,
		},
		{
			url:      "outlok.office365.com",
			provider: Unknown,
		},
		{
			url:      "imap.zorro.com",
			provider: Unknown,
		},
		{
			url:      "imap.slowmail.com",
			provider: Unknown,
		},
	}
	for _, test := range tests {
		imapw := &IMAPWorker{}
		assert.Equal(test.provider, imapw.providerFromURL(test.url))
		assert.Equal(test.provider, imapw.providerFromURL(test.url+":1982"))
		assert.Equal(Unknown, imapw.providerFromURL("a"+test.url))
	}
}
