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
	}

	// panic(address)

	return nil, fmt.Errorf("unprepared address %q", address)
}
