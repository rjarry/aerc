package autoconfig

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigRetrieval(t *testing.T) {
	tests := []struct {
		address       string
		correctConfig *Config
	}{
		{
			address: "mailbox.org",
			correctConfig: &Config{
				Found: ProtocolIMAP,
				IMAP: Credentials{
					Encryption: EncryptionTLS,
					Address:    "imap.mailbox.org",
					Port:       993,
					Username:   "john@mailbox.org",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "smtp.mailbox.org",
					Port:       587,
					Username:   "john@mailbox.org",
				},
			},
		},
		{
			address: "poldi1405.srht.site",
			correctConfig: &Config{
				Found: ProtocolIMAP,
				IMAP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.example.com",
					Port:       143,
					Username:   "your-username",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.example.com",
					Port:       587,
					Username:   "your-username",
				},
			},
		},
		{
			address:       "timeout",
			correctConfig: nil,
		},
	}

	httpGet = autoconfigTestGet
	defer func() {
		httpGet = http.Get
	}()

	for _, test := range tests {
		test := test
		t.Run(test.address, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			res := GetConfig(ctx, "john@"+test.address)
			if test.correctConfig == nil {
				if res != nil {
					t.Fatalf("expected no result, but got %v", res)
				}
				return
			}
			assert.Equal(t, test.correctConfig, res)
		})
	}
}
