package autoconfig

import (
	"context"
	"net/mail"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

// GetConfig attempts to retrieve the settings for the given mail address.
func GetConfig(ctx context.Context, email string) *Config {
	log.Debugf("looking up configuration for %q", email)
	mail, err := mail.ParseAddress(email)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(mail.Address, "@", 2)
	localpart := parts[0]
	domain := parts[1]

	resultList := make(chan chan *Config, 5)

	ProviderSRV := make(chan *Config, 1)
	go getFromProviderDNS(ctx, localpart, domain, ProviderSRV)
	resultList <- ProviderSRV

	ProviderHTTP := make(chan *Config, 1)
	go getFromAutoconfig(ctx, localpart, domain, ProviderHTTP)
	resultList <- ProviderHTTP

	ProviderMozilla := make(chan *Config, 1)
	go getFromMozilla(ctx, localpart, domain, ProviderMozilla)
	resultList <- ProviderMozilla

	ProviderGuess := make(chan *Config, 1)
	go guessMailserver(ctx, localpart, domain, ProviderGuess)
	resultList <- ProviderGuess

	ProviderMX := make(chan *Config, 1)
	go guessMX(ctx, localpart, domain, ProviderMX)
	resultList <- ProviderMX

	close(resultList)

	for reschan := range resultList {
		conf := <-reschan
		if conf != nil {
			return conf
		}
	}

	return nil
}
