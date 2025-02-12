package autoconfig

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

// getFromProviderDNS retrieves the config from the provider using SRV DNS records (RFC2782)
func getFromProviderDNS(ctx context.Context, localpart, domain string, result chan *Config) {
	defer log.PanicHandler()
	defer close(result)

	res := make(chan *Config)
	go func(res chan *Config) {
		defer log.PanicHandler()
		defer close(res)

		type Serverconf struct {
			Hostname string
			Port     int
		}

		configs := map[string]*Serverconf{
			"jmap":       {},
			"imap":       {},
			"imaps":      {},
			"submission": {},
			"smtps":      {},
		}

		var wg sync.WaitGroup
		for key, conf := range configs {
			wg.Add(1)
			go func(service string, conf *Serverconf) {
				defer log.PanicHandler()
				defer wg.Done()

				_, srvList, err := lookupSRV(service, "tcp", domain)
				if err != nil {
					return
				}
				srv := getHighestSRV(srvList)
				srv.Target = strings.TrimRight(srv.Target, ".")
				if srv.Target == "" {
					return
				}
				if srv.Port == 0 {
					return
				}
				*conf = Serverconf{
					Hostname: srv.Target,
					Port:     int(srv.Port),
				}
			}(key, conf)
		}
		wg.Wait()

		cfg := &Config{}
		cfg.Found = ProtocolIMAP

		var increds Credentials
		var outcreds Credentials

		increds.Username = fmt.Sprintf("%s@%s", localpart, domain)
		outcreds.Username = fmt.Sprintf("%s@%s", localpart, domain)

		if configs["imap"].Hostname == "" &&
			configs["imaps"].Hostname == "" &&
			configs["jmap"].Hostname == "" {
			return
		}

		if configs["jmap"].Hostname != "" {
			cfg.Found = ProtocolJMAP
			increds.Address = configs["jmap"].Hostname
			increds.Port = configs["jmap"].Port
			increds.Encryption = EncryptionTLS
			cfg.JMAP = increds
		}

		increds.Encryption = EncryptionSTARTTLS
		if configs["imap"].Hostname == "" {
			configs["imap"] = configs["imaps"]
			increds.Encryption = EncryptionTLS
		}
		increds.Address = configs["imap"].Hostname
		increds.Port = configs["imap"].Port

		outcreds.Encryption = EncryptionSTARTTLS
		if configs["submission"].Hostname == "" {
			configs["submission"] = configs["smtps"]
			outcreds.Encryption = EncryptionTLS
		}
		outcreds.Address = configs["submission"].Hostname
		outcreds.Port = configs["submission"].Port

		cfg.IMAP = increds
		cfg.SMTP = outcreds

		log.Debugf("found SRV config: %#v", cfg)
		res <- cfg
	}(res)

	select {
	case r := <-res:
		result <- r
	case <-ctx.Done():
	}
}

func getHighestSRV(list []*net.SRV) *net.SRV {
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Priority < list[j].Priority
	})

	var max int
	for i := range list {
		if list[i].Weight > list[max].Weight {
			max = i
		}
	}
	return list[max]
}

var lookupSRV = net.LookupSRV
