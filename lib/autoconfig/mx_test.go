package autoconfig

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMX(t *testing.T) {
	tests := []struct {
		address       string
		correctConfig *Config
	}{
		{
			address: "poldrack.dev",
			correctConfig: &Config{
				Found: ProtocolIMAP,
				IMAP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.moritz.sh",
					Port:       143,
					Username:   "john@poldrack.dev",
				},
				SMTP: Credentials{
					Encryption: EncryptionSTARTTLS,
					Address:    "mail.moritz.sh",
					Port:       587,
					Username:   "john@poldrack.dev",
				},
			},
		},
	}

	netDial = mxTestDialer
	lookupMX = mxTestLookup
	defer func() {
		netDial = net.Dial
		lookupMX = net.LookupMX
	}()
	for _, test := range tests {
		test := test
		t.Run(test.address, func(t *testing.T) {
			result := make(chan *Config)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go guessMX(ctx, "john", test.address, result)

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

func mxTestLookup(address string) ([]*net.MX, error) {
	switch address {
	case "poldrack.dev":
		return []*net.MX{
			{Host: "mail.moritz.sh", Pref: 1},
		}, nil
	default:
		return nil, errors.New("unknown address")
	}
}
