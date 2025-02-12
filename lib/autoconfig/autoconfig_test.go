package autoconfig

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAutoconfig(t *testing.T) {
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

			result := make(chan *Config)
			go getFromAutoconfig(ctx, "john", test.address, result)
			res := <-result
			assert.Equal(t, test.correctConfig, res)
		})
	}
}

func autoconfigTestGet(url string) (*http.Response, error) {
	switch url {
	case "https://autoconfig.mailbox.org/mail/config-v1.1.xml?emailaddress=john%40mailbox.org":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(mailboxAutoconf)),
		}, nil
	case "https://poldi1405.srht.site/.well-known/autoconfig/mail/config-v1.1.xml?emailaddress=john%40poldi1405.srht.site":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(srhtAutoconf)),
		}, nil
	case "https://timeout/.well-known/autoconfig/mail/config-v1.1.xml?emailaddress=john%40timeout":
		<-time.After(5 * time.Minute)
		return nil, nil
	default:
		return nil, fmt.Errorf("%q not prepared", url)
	}
}

const (
	mailboxAutoconf = `<?xml version="1.0" encoding="UTF-8"?>

<clientConfig version="1.1">
  <emailProvider id="mailbox.org">
    <domain>mailbox.org</domain>
    <displayName>mailbox.org -- damit Privates privat bleibt</displayName>
    <displayShortName>mailbox.org</displayShortName>

    <incomingServer type="imap">
      <hostname>imap.mailbox.org</hostname>
      <port>993</port>
      <socketType>SSL</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </incomingServer>
    <incomingServer type="imap">
      <hostname>imap.mailbox.org</hostname>
      <port>143</port>
      <socketType>STARTTLS</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </incomingServer>

    <incomingServer type="pop3">
      <hostname>pop3.mailbox.org</hostname>
      <port>995</port>
      <socketType>SSL</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </incomingServer>
    <incomingServer type="pop3">
      <hostname>pop3.mailbox.org</hostname>
      <port>110</port>
      <socketType>STARTTLS</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </incomingServer>

    <outgoingServer type="smtp">
      <hostname>smtp.mailbox.org</hostname>
      <port>465</port>
      <socketType>SSL</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </outgoingServer>
    <outgoingServer type="smtp">
      <hostname>smtp.mailbox.org</hostname>
      <port>587</port>
      <socketType>STARTTLS</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </outgoingServer>


    <documentation url="http://www.mailbox.org/">
      <descr lang="de">FAQ und Support-Datenbank</descr>
      <descr lang="en">Frequently Asked Questions (FAQ)</descr>
    </documentation>
  </emailProvider>
</clientConfig>

`
	srhtAutoconf = `<clientConfig version="1.1">
	<emailProvider id="example.com">
		<domain>example.com</domain>
		<displayName>Not valid</displayName>
		<displayShortName>Not valid</displayShortName>
		<incomingServer type="imap">
			<hostname>mail.example.com</hostname>
			<port>143</port>
			<socketType>STARTTLS</socketType>
			<username>your-username</username>
			<authentication>password-cleartext</authentication>
		</incomingServer>
		<outgoingServer type="smtp">
			<hostname>mail.example.com</hostname>
			<port>587</port>
			<socketType>STARTTLS</socketType>
			<username>your-username</username>
			<authentication>password-cleartext</authentication>
		</outgoingServer>
	</emailProvider>
</clientConfig>
`
)
