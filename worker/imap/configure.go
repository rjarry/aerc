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
)

func (w *IMAPWorker) handleConfigure(msg *types.Configure) error {
	w.config.name = msg.Config.Name
	u, err := url.Parse(msg.Config.Source)
	if err != nil {
		return err
	}

	w.config.provider = w.providerFromURL(u.Host)
	w.config.url = u

	w.config.folders = msg.Config.Folders
	w.config.headers = msg.Config.Headers
	w.config.headersExclude = msg.Config.HeadersExclude

	w.config.checkMail = msg.Config.CheckMail

	w.config.connection_timeout = 90 * time.Second
	w.config.keepalive_period = 0 * time.Second
	w.config.keepalive_probes = 3
	w.config.keepalive_interval = 3

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
			w.idler.timeout = val
		case "idle-debounce":
			val, err := time.ParseDuration(value)
			if err != nil || val < 0 {
				return fmt.Errorf(
					"invalid idle-debounce value %v: %w",
					value, err)
			}
			w.idler.debounce = val
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
		case "debug-log-path":
			w.config.debugLogPath = value
		}
	}
	if w.config.cacheEnabled {
		w.initCacheDb(msg.Config.Name)
	}

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

func (w *IMAPWorker) providerFromURL(url string) imapProvider {
	isValidURLPrefix := func(url string, prefix string) bool {
		if !strings.HasPrefix(url, prefix) {
			return false
		}
		if len(url) > len(prefix) && url[len(prefix)] != ':' {
			// URL is not of the form "$prefix:$port"
			return false
		}
		return true
	}
	switch {
	case isValidURLPrefix(url, "imap.gmail.com"):
		return GMail
	case isValidURLPrefix(url, "127.0.0.1"):
		return Proton
	case isValidURLPrefix(url, "outlook.office365.com"):
		return Office365
	case isValidURLPrefix(url, "imap.zoho.com"):
		return Zoho
	case isValidURLPrefix(url, "imap.fastmail.com"):
		return FastMail
	case isValidURLPrefix(url, "imap.mail.me.com"):
		return iCloud
	default:
		return Unknown
	}
}
