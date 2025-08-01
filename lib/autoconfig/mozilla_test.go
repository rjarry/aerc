package autoconfig

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMozilla(t *testing.T) {
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
					Encryption: EncryptionTLS,
					Address:    "smtp.gmail.com",
					Port:       465,
					Username:   "john@gmail.com",
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
	}

	mozillaGet = mozillaTestHTTP
	defer func() {
		mozillaGet = http.Get
	}()
	for _, test := range tests {
		t.Run(test.address, func(t *testing.T) {
			result := make(chan *Config)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go getFromMozilla(ctx, "john", test.address, result)

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

func mozillaTestHTTP(url string) (*http.Response, error) {
	switch url {
	case "https://autoconfig.thunderbird.net/v1.1/gmail.com":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(gmailCfg)),
		}, nil
	case "https://autoconfig.thunderbird.net/v1.1/gmx.de":
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(gmxCfg)),
		}, nil
	case "https://autoconfig.thunderbird.net/v1.1/timeout":
		<-time.After(10 * time.Second)
		return nil, nil
	default:
		return nil, errors.New("not prepared")
	}
}

const (
	gmailCfg = `<clientConfig version="1.1">
  <emailProvider id="googlemail.com">
    <domain>gmail.com</domain>
    <domain>googlemail.com</domain>
    <!-- MX, for Google Apps -->
    <domain>google.com</domain>
    <!-- HACK. Only add ISPs with 100000+ users here -->
    <domain>jazztel.es</domain>

    <displayName>Google Mail</displayName>
    <displayShortName>GMail</displayShortName>

    <incomingServer type="imap">
      <hostname>imap.gmail.com</hostname>
      <port>993</port>
      <socketType>SSL</socketType>
      <username>%EMAILADDRESS%</username>
      <authentication>OAuth2</authentication>
      <authentication>password-cleartext</authentication>
    </incomingServer>
    <incomingServer type="pop3">
      <hostname>pop.gmail.com</hostname>
      <port>995</port>
      <socketType>SSL</socketType>
      <username>%EMAILADDRESS%</username>
      <authentication>OAuth2</authentication>
      <authentication>password-cleartext</authentication>
      <pop3>
        <leaveMessagesOnServer>true</leaveMessagesOnServer>
      </pop3>
    </incomingServer>
    <outgoingServer type="smtp">
      <hostname>smtp.gmail.com</hostname>
      <port>465</port>
      <socketType>SSL</socketType>
      <username>%EMAILADDRESS%</username>
      <authentication>OAuth2</authentication>
      <authentication>password-cleartext</authentication>
    </outgoingServer>

    <documentation url="http://mail.google.com/support/bin/answer.py?answer=13273">
      <descr>How to enable IMAP/POP3 in GMail</descr>
    </documentation>
    <documentation url="http://mail.google.com/support/bin/topic.py?topic=12806">
      <descr>How to configure email clients for IMAP</descr>
    </documentation>
    <documentation url="http://mail.google.com/support/bin/topic.py?topic=12805">
      <descr>How to configure email clients for POP3</descr>
    </documentation>
    <documentation url="http://mail.google.com/support/bin/answer.py?answer=86399">
      <descr>How to configure TB 2.0 for POP3</descr>
    </documentation>
  </emailProvider>

  <oAuth2>
    <issuer>accounts.google.com</issuer>
    <!-- https://developers.google.com/identity/protocols/oauth2/scopes -->
    <scope>https://mail.google.com/ https://www.googleapis.com/auth/contacts https://www.googleapis.com/auth/calendar https://www.googleapis.com/auth/carddav</scope>
    <authURL>https://accounts.google.com/o/oauth2/auth</authURL>
    <tokenURL>https://www.googleapis.com/oauth2/v3/token</tokenURL>
  </oAuth2>

  <enable visiturl="https://mail.google.com/mail/?ui=2&amp;shva=1#settings/fwdandpop">
    <instruction>You need to enable IMAP access</instruction>
  </enable>

  <webMail>
    <loginPage url="https://accounts.google.com/ServiceLogin?service=mail&amp;continue=http://mail.google.com/mail/"/>
    <loginPageInfo url="https://accounts.google.com/ServiceLogin?service=mail&amp;continue=http://mail.google.com/mail/">
      <username>%EMAILADDRESS%</username>
      <usernameField id="Email"/>
      <passwordField id="Passwd"/>
      <loginButton id="signIn"/>
    </loginPageInfo>
  </webMail>

</clientConfig>
`
	gmxCfg = `<clientConfig version="1.1">
  <emailProvider id="gmx.net">
    <domain>gmx.net</domain>
    <domain>gmx.de</domain>
    <domain>gmx.at</domain>
    <domain>gmx.ch</domain>
    <domain>gmx.eu</domain>
    <domain>gmx.biz</domain>
    <domain>gmx.org</domain>
    <domain>gmx.info</domain>
    <!-- see also other domains below -->
    <!-- gmx.com is same company, but different access servers -->
    <displayName>GMX Freemail</displayName>
    <displayShortName>GMX</displayShortName>
    <!-- imap officially costs money, but actually works with freemail accounts, too -->
    <incomingServer type="imap">
      <hostname>imap.gmx.net</hostname>
      <port>993</port>
      <socketType>SSL</socketType>
      <!-- Kundennummer (customer no) and email address should both work -->
      <username>%EMAILADDRESS%</username>
      <authentication>password-cleartext</authentication>
    </incomingServer>
    <incomingServer type="imap">
      <hostname>imap.gmx.net</hostname>
      <port>143</port>
      <socketType>STARTTLS</socketType>
      <username>%EMAILADDRESS%</username>
      <authentication>password-cleartext</authentication>
    </incomingServer>
    <incomingServer type="pop3">
      <hostname>pop.gmx.net</hostname>
      <port>995</port>
      <socketType>SSL</socketType>
      <!-- see above -->
      <username>%EMAILADDRESS%</username>
      <authentication>password-cleartext</authentication>
    </incomingServer>
    <incomingServer type="pop3">
      <hostname>pop.gmx.net</hostname>
      <port>110</port>
      <socketType>STARTTLS</socketType>
      <username>%EMAILADDRESS%</username>
      <authentication>password-cleartext</authentication>
    </incomingServer>
    <outgoingServer type="smtp">
      <hostname>mail.gmx.net</hostname>
      <port>465</port>
      <socketType>SSL</socketType>
      <!-- see above -->
      <username>%EMAILADDRESS%</username>
      <authentication>password-cleartext</authentication>
    </outgoingServer>
    <outgoingServer type="smtp">
      <hostname>mail.gmx.net</hostname>
      <port>587</port>
      <socketType>STARTTLS</socketType>
      <username>%EMAILADDRESS%</username>
      <authentication>password-cleartext</authentication>
    </outgoingServer>
  </emailProvider>
</clientConfig>
`
)
