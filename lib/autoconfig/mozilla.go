package autoconfig

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

// getFromMozilla retrieves the config from Mozillas public database
func getFromMozilla(
	ctx context.Context,
	localpart, domain string,
	result chan *Config,
) {
	defer log.PanicHandler()
	defer close(result)

	res := make(chan *Config)
	go func(res chan *Config) {
		defer log.PanicHandler()
		defer close(res)
		var cc ClientConfig

		u, err := url.Parse(
			fmt.Sprintf("https://autoconfig.thunderbird.net/v1.1/%s", domain),
		)
		if err != nil {
			return
		}

		response, err := mozillaGet(u.String())
		if err != nil {
			return
		}
		log.Debugf("found config in Mozilla dataset")

		err = xml.NewDecoder(response.Body).Decode(&cc)
		if err != nil {
			return
		}
		// IMAP sanity check
		var incoming *IncomingServer
		for i := range cc.EmailProvider.IncomingServer {
			providerType := cc.EmailProvider.IncomingServer[i].Type
			if strings.ToLower(providerType) != "imap" {
				continue
			}
			incoming = &cc.EmailProvider.IncomingServer[i]
			break
		}
		if incoming == nil {
			// no imap server found
			return
		}
		var incomingPort int
		if incomingPort, err = strconv.Atoi(incoming.Port); err != nil {
			return
		}
		inenc := EncryptionSTARTTLS
		switch strings.ToLower(incoming.SocketType) {
		case "plain":
			inenc = EncryptionInsecure
		case "ssl":
			inenc = EncryptionTLS
		}
		if strings.ToLower(incoming.Username) == "%emailaddress%" {
			incoming.Username = localpart + "@" + domain
		}

		var outport int
		retrievedPort := cc.EmailProvider.OutgoingServer.Port
		if outport, err = strconv.Atoi(retrievedPort); err != nil {
			return
		}
		outenc := EncryptionSTARTTLS
		switch strings.ToLower(cc.EmailProvider.OutgoingServer.SocketType) {
		case "plain":
			outenc = EncryptionInsecure
		case "ssl":
			outenc = EncryptionTLS
		}
		username := cc.EmailProvider.OutgoingServer.Username
		if strings.ToLower(username) == "%emailaddress%" {
			cc.EmailProvider.OutgoingServer.Username = localpart + "@" + domain
		}

		res <- &Config{
			Found: ProtocolIMAP,
			IMAP: Credentials{
				Encryption: inenc,
				Address:    incoming.Hostname,
				Port:       incomingPort,
				Username:   incoming.Username,
			},
			SMTP: Credentials{
				Encryption: outenc,
				Address:    cc.EmailProvider.OutgoingServer.Hostname,
				Port:       outport,
				Username:   cc.EmailProvider.OutgoingServer.Username,
			},
		}
	}(res)

	select {
	case r, next := <-res:
		if next {
			result <- r
		}
	case <-ctx.Done():
	}
}

var mozillaGet = http.Get
