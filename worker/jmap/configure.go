package jmap

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/worker/jmap/cache"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (w *JMAPWorker) handleConfigure(msg *types.Configure) error {
	w.config.cacheState = parseBool(msg.Config.Params["cache-state"])
	w.config.cacheBlobs = parseBool(msg.Config.Params["cache-blobs"])
	w.config.useLabels = parseBool(msg.Config.Params["use-labels"])
	w.cache = cache.NewJMAPCache(
		w.config.cacheState, w.config.cacheBlobs, msg.Config.Name)

	u, err := url.Parse(msg.Config.Source)
	if err != nil {
		return err
	}

	if strings.HasSuffix(u.Scheme, "+oauthbearer") {
		w.config.oauth = true
	} else {
		if u.User == nil {
			return fmt.Errorf("user:password not specified")
		} else if u.User.Username() == "" {
			return fmt.Errorf("username not specified")
		} else if _, ok := u.User.Password(); !ok {
			return fmt.Errorf("password not specified")
		}
	}

	u.RawQuery = ""
	u.Fragment = ""
	w.config.user = u.User
	u.User = nil
	u.Scheme = "https"

	w.config.endpoint = u.String()
	w.config.account = msg.Config
	w.config.allMail = msg.Config.Params["all-mail"]
	if w.config.allMail == "" {
		w.config.allMail = "All mail"
	}
	if ping, ok := msg.Config.Params["server-ping"]; ok {
		dur, err := time.ParseDuration(ping)
		if err != nil {
			return fmt.Errorf("server-ping: %w", err)
		}
		w.config.serverPing = dur
	}

	return nil
}

func parseBool(val string) bool {
	switch strings.ToLower(val) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	}
	return false
}
