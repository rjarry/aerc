package autoconfig

import (
	"context"
	"fmt"
	"net"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

type portEncryption struct {
	port int
	enc  encryption
}

// guessMailserver tries to guess mailserver configuration based on commonly
// used settings
//
// It tries to find imap, smtp, and mail.domain.com and attempts to guess the
// encryption based on a TCP ping to the relevant port.
func guessMailserver(ctx context.Context, localpart, domain string, result chan *Config) {
	defer log.PanicHandler()
	defer close(result)

	res := make(chan *Config)
	go func(res chan *Config) {
		defer log.PanicHandler()
		defer close(res)

		var imapConfig, smtpConfig *Credentials
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer log.PanicHandler()
			defer wg.Done()
			imapConfig = guessServername(
				domain,
				[]string{"imap", "mail"},
				[]portEncryption{
					{143, EncryptionSTARTTLS},
					{993, EncryptionTLS},
				},
			)
		}()

		go func() {
			defer log.PanicHandler()
			defer wg.Done()
			smtpConfig = guessServername(
				domain,
				[]string{"smtp", "mail"},
				[]portEncryption{
					{587, EncryptionSTARTTLS},
					{465, EncryptionTLS},
				},
			)
		}()
		wg.Wait()

		if imapConfig == nil || smtpConfig == nil {
			return
		}
		log.Debugf("successfully guessed server: %#v %#v", imapConfig, smtpConfig)

		imapConfig.Username = localpart + "@" + domain
		smtpConfig.Username = localpart + "@" + domain

		res <- &Config{
			Found: ProtocolIMAP,
			IMAP:  *imapConfig,
			SMTP:  *smtpConfig,
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

func guessServername(
	domain string,
	subdomains []string,
	portenc []portEncryption,
) *Credentials {
	for _, subdomain := range subdomains {
		cred := tryPort(subdomain+"."+domain, portenc)
		if cred != nil {
			return cred
		}
	}

	return nil
}

func tryPort(host string, ports []portEncryption) *Credentials {
	var res Credentials

	for _, pe := range ports {
		res.Address = host
		res.Port = pe.port
		res.Encryption = pe.enc
		c, err := netDial("tcp", fmt.Sprintf("%s:%d", host, pe.port))
		if err != nil {
			continue
		}
		c.Close()
		return &res
	}

	return nil
}

var netDial = net.Dial
