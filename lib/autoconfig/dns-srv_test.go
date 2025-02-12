package autoconfig

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDNSSRV(t *testing.T) {
	tests := []struct {
		address       string
		correctConfig *Config
	}{
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
	}

	lookupSRV = customLookupSRV
	defer func() {
		lookupSRV = net.LookupSRV
	}()
	for _, test := range tests {
		test := test
		t.Run(test.address, func(t *testing.T) {
			result := make(chan *Config)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go getFromProviderDNS(ctx, "john", test.address, result)

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

func customLookupSRV(service, proto, name string) (cname string, addrs []*net.SRV, err error) {
	switch name {
	case "gmail.com":
		switch service {
		case "imap":
			return "_imap.gmail.com", []*net.SRV{
				{
					Target: ".",
				},
			}, nil
		case "imaps":
			return "_simap.gmail.com", []*net.SRV{
				{
					Target:   "imap.gmail.com.",
					Port:     993,
					Priority: 5,
				},
			}, nil
		case "submission":
			return "_submission._tcp.gmail.com", []*net.SRV{
				{
					Target:   "smtp.gmail.com.",
					Port:     587,
					Priority: 5,
				},
			}, nil
		}
	case "fastmail.com":
		switch service {
		case "jmap":
			return "_jmap._tcp.fastmail.com", []*net.SRV{
				{
					Target: "api.fastmail.com.",
					Port:   443,
					Weight: 1,
				},
			}, nil
		case "imap":
			return "_imap._tcp.fastmail.com", []*net.SRV{
				{
					Target: ".",
				},
			}, nil
		case "imaps":
			return "_jmap._tcp.fastmail.com", []*net.SRV{
				{
					Target: "imap.fastmail.com.",
					Port:   993,
					Weight: 1,
				},
			}, nil
		case "submission":
			return "_submission._tcp.fastmail.com", []*net.SRV{
				{
					Target: "smtp.fastmail.com.",
					Port:   587,
					Weight: 1,
				},
			}, nil
		}
	case "timeout":
		<-time.After(5 * time.Minute)
		return "_._tcp.", []*net.SRV{
			{},
		}, nil
	}
	return "", nil, fmt.Errorf("lookup _%s._tcp.%s on 127.0.0.53:53: no such host", service, name)
}
