package imap

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rjarry/aerc/worker/middleware"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"golang.org/x/oauth2"
)

func (w *IMAPWorker) handleConfigure(msg *types.Configure) error {
	w.config.name = msg.Config.Name
	u, err := url.Parse(msg.Config.Source)
	if err != nil {
		return err
	}

	w.config.scheme = u.Scheme
	if strings.HasSuffix(w.config.scheme, "+insecure") {
		w.config.scheme = strings.TrimSuffix(w.config.scheme, "+insecure")
		w.config.insecure = true
	}

	if strings.HasSuffix(w.config.scheme, "+oauthbearer") {
		w.config.scheme = strings.TrimSuffix(w.config.scheme, "+oauthbearer")
		w.config.oauthBearer.Enabled = true
		q := u.Query()

		oauth2 := &oauth2.Config{}
		if q.Get("token_endpoint") != "" {
			oauth2.ClientID = q.Get("client_id")
			oauth2.ClientSecret = q.Get("client_secret")
			oauth2.Scopes = []string{q.Get("scope")}
			oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
		}
		w.config.oauthBearer.OAuth2 = oauth2
	}

	if strings.HasSuffix(w.config.scheme, "+xoauth2") {
		w.config.scheme = strings.TrimSuffix(w.config.scheme, "+xoauth2")
		w.config.xoauth2.Enabled = true
		q := u.Query()

		oauth2 := &oauth2.Config{}
		if q.Get("token_endpoint") != "" {
			oauth2.ClientID = q.Get("client_id")
			oauth2.ClientSecret = q.Get("client_secret")
			oauth2.Scopes = []string{q.Get("scope")}
			oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
		}
		w.config.xoauth2.OAuth2 = oauth2
	}

	w.config.addr = u.Host
	if !strings.ContainsRune(w.config.addr, ':') {
		w.config.addr += ":" + w.config.scheme
	}

	w.config.user = u.User
	w.config.folders = msg.Config.Folders
	w.config.headers = msg.Config.Headers
	w.config.headersExclude = msg.Config.HeadersExclude

	w.config.idle_timeout = 10 * time.Second
	w.config.idle_debounce = 10 * time.Millisecond

	w.config.connection_timeout = 90 * time.Second
	w.config.keepalive_period = 0 * time.Second
	w.config.keepalive_probes = 3
	w.config.keepalive_interval = 3

	w.config.reconnect_maxwait = 30 * time.Second

	w.config.cacheEnabled = false
	w.config.cacheMaxAge = 30 * 24 * time.Hour // 30 days
	w.config.expungePolicy = ExpungePolicyAuto

	for key, value := range msg.Config.Params {
		switch key {
		case "idle-timeout":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid idle-timeout value %v: %w",
					value, err)
			}
			w.config.idle_timeout = val
		case "idle-debounce":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid idle-debounce value %v: %w",
					value, err)
			}
			w.config.idle_debounce = val
		case "reconnect-maxwait":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid reconnect-maxwait value %v: %w",
					value, err)
			}
			w.config.reconnect_maxwait = val
		case "connection-timeout":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid connection-timeout value %v: %w",
					value, err)
			}
			w.config.connection_timeout = val
		case "keepalive-period":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid keepalive-period value %v: %w",
					value, err)
			}
			w.config.keepalive_period = val
		case "keepalive-probes":
			val, err := strconv.Atoi(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid keepalive-probes value %v: %w",
					value, err)
			}
			w.config.keepalive_probes = val
		case "keepalive-interval":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid keepalive-interval value %v: %w",
					value, err)
			}
			w.config.keepalive_interval = int(val.Seconds())
		case "cache-headers":
			cache, err := strconv.ParseBool(value)
			if err != nil {
				// Return an error here because the user tried to set header
				// caching, and we want them to know they didn't set it right -
				// one way or the other
				return fmt.Errorf("invalid cache-headers value %v: %w", value, err)
			}
			w.config.cacheEnabled = cache
		case "cache-max-age":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf("invalid cache-max-age value %v: %w", value, err)
			}
			w.config.cacheMaxAge = val
		case "expunge-policy":
			switch value {
			case "auto":
				w.config.expungePolicy = ExpungePolicyAuto
			case "low-to-high":
				w.config.expungePolicy = ExpungePolicyLowToHigh
			case "stable":
				w.config.expungePolicy = ExpungePolicyStable
			default:
				return fmt.Errorf("invalid expunge-policy value %v", value)
			}
		}
	}
	if w.config.cacheEnabled {
		w.initCacheDb(msg.Config.Name)
	}
	w.idler = newIdler(w.config, w.worker, w.executeIdle)
	w.observer = newObserver(w.config, w.worker)

	if name, ok := msg.Config.Params["folder-map"]; ok {
		file := xdg.ExpandHome(name)
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		fmap, order, err := lib.ParseFolderMap(bufio.NewReader(f))
		if err != nil {
			return err
		}
		w.worker = middleware.NewFolderMapper(w.worker, fmap, order)
	}

	return nil
}
