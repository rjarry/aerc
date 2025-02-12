package autoconfig

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGuess(t *testing.T) {
	tests := []struct {
		address       string
		correctConfig *Config
	}{
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
	}

	netDial = mxTestDialer
	defer func() {
		netDial = net.Dial
	}()
	for _, test := range tests {
		test := test
		t.Run(test.address, func(t *testing.T) {
			result := make(chan *Config)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			go guessMailserver(ctx, "john", test.address, result)

			select {
			case res := <-result:
				if res == nil {
					t.Log("no result")
					t.FailNow()
				}
				assert.Equal(t, test.correctConfig, res)
			case <-ctx.Done():
				t.Error("retrieval timed out!")
			}
		})
	}
}

func mxTestDialer(_ string, address string) (net.Conn, error) {
	switch address {
	case "mail.moritz.sh:143":
		return &net.UnixConn{}, nil
	case "mail.moritz.sh:587":
		return &net.UnixConn{}, nil
	case "imap.mailbox.org:143", "imap.mailbox.org:993", "mail.mailbox.org:143", "mail.mailbox.org:993", "smtp.mailbox.org:587", "smtp.mailbox.org:465", "smtp.poldi1405.srht.site:587", "smtp.poldi1405.srht.site:465", "mail.mailbox.org:587", "mail.mailbox.org:465", "imap.poldi1405.srht.site:143", "imap.poldi1405.srht.site:993", "mail.poldi1405.srht.site:143", "mail.poldi1405.srht.site:993", "mail.poldi1405.srht.site:587", "mail.poldi1405.srht.site:465", "imap.gmail.com:143", "imap.gmail.com:993", "smtp.gmail.com:587", "smtp.gmail.com:465", "mail.gmail.com:143", "mail.gmail.com:993", "mail.gmail.com:587", "mail.gmail.com:465", "imap.fastmail.com:143", "imap.fastmail.com:993", "smtp.gmx.de:587", "smtp.gmx.de:465", "mail.gmx.de:587", "mail.gmx.de:465", "mail.fastmail.com:143", "mail.fastmail.com:993", "mail.fastmail.com:587", "mail.fastmail.com:465", "smtp.fastmail.com:587", "smtp.fastmail.com:465", "imap.gmx.de:143", "imap.gmx.de:993", "mail.gmx.de:143", "mail.gmx.de:993", "imap.moritz.sh:143", "imap.moritz.sh:993", "smtp.moritz.sh:587", "smtp.moritz.sh:465", "smtp.poldrack.dev:587", "smtp.poldrack.dev:465", "imap.poldrack.dev:143", "imap.poldrack.dev:993", "mail.poldrack.dev:143", "mail.poldrack.dev:587", "mail.poldrack.dev:465", "mail.poldrack.dev:993", "imap.timeout:143", "imap.timeout:993", "smtp.timeout:587", "smtp.timeout:465", "mail.timeout:143", "mail.timeout:993", "mail.timeout:587", "mail.timeout:465":
		return nil, fmt.Errorf("unprepared address %q", address)
	}

	panic(address)

	return nil, fmt.Errorf("unprepared address %q", address)
}
