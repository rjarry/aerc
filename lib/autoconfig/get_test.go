package autoconfig

import (
	"context"
	"net"
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
			address: "gmail.com",
			correctConfig: &Config{
				Found: ProtocolIMAP,
				IMAP: Credentials{
					Encryption: EncryptionTLS,
					Address:    "imap.gmail.com",
					Port:       993,
					Username:   "john@gmail.com",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "smtp.gmail.com",
					Port:       587,
					Username:   "john@gmail.com",
				},
			},
		},
		{
			address: "fastmail.com",
			correctConfig: &Config{
				Found: ProtocolJMAP,
				JMAP: Credentials{
					Encryption: EncryptionTLS,
					Address:    "api.fastmail.com",
					Port:       443,
					Username:   "john@fastmail.com",
				},
				IMAP: Credentials{
					Encryption: EncryptionTLS,
					Address:    "imap.fastmail.com",
					Port:       993,
					Username:   "john@fastmail.com",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "smtp.fastmail.com",
					Port:       587,
					Username:   "john@fastmail.com",
				},
			},
		},
		{
			address: "gmx.de",
			correctConfig: &Config{
				Found: ProtocolIMAP,
				IMAP: Credentials{
					Encryption: EncryptionTLS,
					Address:    "imap.gmx.net",
					Port:       993,
					Username:   "john@gmx.de",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.gmx.net",
					Port:       587,
					Username:   "john@gmx.de",
				},
			},
		},
		{
			address: "moritz.sh",
			correctConfig: &Config{
				Found: ProtocolIMAP,
				IMAP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.moritz.sh",
					Port:       143,
					Username:   "john@moritz.sh",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.moritz.sh",
					Port:       587,
					Username:   "john@moritz.sh",
				},
			},
		},
		{
			address:       "timeout",
			correctConfig: nil,
		},
	}

	lookupSRV = customLookupSRV
	httpGet = autoconfigTestGet
	mozillaGet = mozillaTestHTTP
	netDial = mxTestDialer
	defer func() {
		lookupSRV = net.LookupSRV
		httpGet = http.Get
		mozillaGet = http.Get
		netDial = net.Dial
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
