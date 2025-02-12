package autoconfig

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

func guessMX(ctx context.Context, localpart, domain string, result chan *Config) {
	defer log.PanicHandler()
	defer close(result)

	res := make(chan *Config)
	go func(res chan *Config) {
		defer log.PanicHandler()
		defer close(res)

		var imapConfig, smtpConfig *Credentials
		var wg sync.WaitGroup

		records, err := lookupMX(domain)
		if err != nil || len(records) == 0 {
			return
		}

		sort.Slice(records, func(a, b int) bool { return records[a].Pref < records[b].Pref })

		mailserver := records[0].Host
		mailserver = strings.TrimSuffix(mailserver, ".")

		wg.Add(2)

		go func() {
			defer log.PanicHandler()
			defer wg.Done()
			imapConfig = tryPort(mailserver, []portEncryption{{143, EncryptionSTARTTLS}, {993, EncryptionTLS}})
		}()

		go func() {
			defer log.PanicHandler()
			defer wg.Done()
			smtpConfig = tryPort(mailserver, []portEncryption{{587, EncryptionSTARTTLS}, {465, EncryptionTLS}})
		}()
		wg.Wait()

		if imapConfig == nil || smtpConfig == nil {
			return
		}
		log.Debugf("found MX records: %v %v", imapConfig, smtpConfig)

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

var lookupMX = net.LookupMX
