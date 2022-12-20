package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

type RemoteConfig struct {
	Value       string
	PasswordCmd string
	CacheCmd    bool
	cache       string
}

func (c *RemoteConfig) parseValue() (*url.URL, error) {
	return url.Parse(c.Value)
}

func (c *RemoteConfig) ConnectionString() (string, error) {
	if c.Value == "" || c.PasswordCmd == "" {
		return c.Value, nil
	}

	u, err := c.parseValue()
	if err != nil {
		return "", err
	}

	// ignore the command if a password is specified
	if _, exists := u.User.Password(); exists {
		return c.Value, nil
	}

	// don't attempt to parse the command if the url is a path (ie /usr/bin/sendmail)
	if !u.IsAbs() {
		return c.Value, nil
	}

	pw := c.cache

	if pw == "" {
		cmd := exec.Command("sh", "-c", c.PasswordCmd)
		cmd.Stdin = os.Stdin
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		pw = strings.TrimSpace(string(output))
	}
	u.User = url.UserPassword(u.User.Username(), pw)
	if c.CacheCmd {
		c.cache = pw
	}

	return u.String(), nil
}

type AccountConfig struct {
	Archive           string            `ini:"archive"`
	CopyTo            string            `ini:"copy-to"`
	Default           string            `ini:"default"`
	Postpone          string            `ini:"postpone"`
	From              *mail.Address     `ini:"-"`
	Aliases           []*mail.Address   `ini:"-"`
	Name              string            `ini:"-"`
	Source            string            `ini:"-"`
	Folders           []string          `ini:"folders" delim:","`
	FoldersExclude    []string          `ini:"folders-exclude" delim:","`
	Params            map[string]string `ini:"-"`
	Outgoing          RemoteConfig      `ini:"-"`
	SignatureFile     string            `ini:"signature-file"`
	SignatureCmd      string            `ini:"signature-cmd"`
	EnableFoldersSort bool              `ini:"enable-folders-sort"`
	FoldersSort       []string          `ini:"folders-sort" delim:","`
	AddressBookCmd    string            `ini:"address-book-cmd"`
	SendAsUTC         bool              `ini:"send-as-utc"`
	LocalizedRe       *regexp.Regexp    `ini:"-"`

	// CheckMail
	CheckMail        time.Duration `ini:"check-mail"`
	CheckMailCmd     string        `ini:"check-mail-cmd"`
	CheckMailTimeout time.Duration `ini:"check-mail-timeout"`
	CheckMailInclude []string      `ini:"check-mail-include"`
	CheckMailExclude []string      `ini:"check-mail-exclude"`

	// PGP Config
	PgpKeyId                string `ini:"pgp-key-id"`
	PgpAutoSign             bool   `ini:"pgp-auto-sign"`
	PgpOpportunisticEncrypt bool   `ini:"pgp-opportunistic-encrypt"`
	PgpErrorLevel           int    `ini:"pgp-error-level"`

	// AuthRes
	TrustedAuthRes []string `ini:"trusted-authres" delim:","`
}

const (
	PgpErrorLevelNone = iota
	PgpErrorLevelWarn
	PgpErrorLevelError
)

var Accounts []*AccountConfig

func parseAccounts(root string, accts []string) error {
	filename := path.Join(root, "accounts.conf")
	if !General.UnsafeAccountsConf {
		if err := checkConfigPerms(filename); err != nil {
			return err
		}
	}

	log.Debugf("Parsing accounts configuration from %s", filename)

	file, err := ini.Load(filename)
	if err != nil {
		// No config triggers account configuration wizard
		return nil
	}
	file.NameMapper = mapName

	for _, _sec := range file.SectionStrings() {
		if _sec == "DEFAULT" {
			continue
		}
		if len(accts) > 0 && !contains(accts, _sec) {
			continue
		}
		sec := file.Section(_sec)
		sourceRemoteConfig := RemoteConfig{}
		account := AccountConfig{
			Archive:           "Archive",
			Default:           "INBOX",
			Postpone:          "Drafts",
			Name:              _sec,
			Params:            make(map[string]string),
			EnableFoldersSort: true,
			CheckMailTimeout:  10 * time.Second,
			PgpErrorLevel:     PgpErrorLevelWarn,
			// localizedRe contains a list of known translations for the common Re:
			LocalizedRe: regexp.MustCompile(`(?i)^((AW|RE|SV|VS|ODP|R): ?)+`),
		}
		if err = sec.MapTo(&account); err != nil {
			return err
		}
		for key, val := range sec.KeysHash() {
			switch key {
			case "source":
				sourceRemoteConfig.Value = val
			case "source-cred-cmd":
				sourceRemoteConfig.PasswordCmd = val
			case "outgoing":
				account.Outgoing.Value = val
			case "outgoing-cred-cmd":
				account.Outgoing.PasswordCmd = val
			case "outgoing-cred-cmd-cache":
				cache, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("%s=%s %w", key, val, err)
				}
				account.Outgoing.CacheCmd = cache
			case "from":
				addr, err := mail.ParseAddress(val)
				if err != nil {
					return fmt.Errorf("%s=%s %w", key, val, err)
				}
				account.From = addr
			case "aliases":
				addrs, err := mail.ParseAddressList(val)
				if err != nil {
					return fmt.Errorf("%s=%s %w", key, val, err)
				}
				account.Aliases = addrs
			case "subject-re-pattern":
				re, err := regexp.Compile(val)
				if err != nil {
					return fmt.Errorf("%s=%s %w", key, val, err)
				}
				account.LocalizedRe = re
			case "pgp-error-level":
				switch strings.ToLower(val) {
				case "none":
					account.PgpErrorLevel = PgpErrorLevelNone
				case "warn":
					account.PgpErrorLevel = PgpErrorLevelWarn
				case "error":
					account.PgpErrorLevel = PgpErrorLevelError
				default:
					return fmt.Errorf("unknown pgp-error-level: %s", val)
				}
			default:
				backendSpecific := true
				typ := reflect.TypeOf(account)
				for i := 0; i < typ.NumField(); i++ {
					field := typ.Field(i)
					switch field.Tag.Get("ini") {
					case key:
						fallthrough
					case "source":
						fallthrough
					case "source-cred-cmd":
						fallthrough
					case "outgoing":
						fallthrough
					case "outgoing-cred-cmd":
						fallthrough
					case "outgoing-cred-cmd-cache":
						fallthrough
					case "subject-re-pattern":
						fallthrough
					case "pgp-error-level":
						backendSpecific = false
					}
				}
				if backendSpecific {
					account.Params[key] = val
				}
			}
		}
		source, err := sourceRemoteConfig.ConnectionString()
		if err != nil {
			return fmt.Errorf("Invalid source credentials for %s: %w", _sec, err)
		}
		account.Source = source

		if account.Source == "" {
			return fmt.Errorf("Expected source for account %s", _sec)
		}
		if account.From == nil {
			return fmt.Errorf("Expected from for account %s", _sec)
		}

		_, err = account.Outgoing.parseValue()
		if err != nil {
			return fmt.Errorf("Invalid outgoing credentials for %s: %w", _sec, err)
		}

		log.Debugf("accounts.conf: [%s] from = %s", account.Name, account.From)
		Accounts = append(Accounts, &account)
	}
	if len(accts) > 0 {
		// Sort accounts struct to match the specified order, if we
		// have one
		if len(Accounts) != len(accts) {
			return errors.New("account(s) not found")
		}
		sort.Slice(Accounts, func(i, j int) bool {
			return strings.ToLower(accts[i]) < strings.ToLower(accts[j])
		})
	}

	return nil
}

// checkConfigPerms checks for too open permissions
// printing the fix on stdout and returning an error
func checkConfigPerms(filename string) error {
	info, err := os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil // disregard absent files
	}
	if err != nil {
		return err
	}

	perms := info.Mode().Perm()
	// group or others have read access
	if perms&0o44 != 0 {
		fmt.Fprintf(os.Stderr, "The file %v has too open permissions.\n", filename)
		fmt.Fprintln(os.Stderr, "This is a security issue (it contains passwords).")
		fmt.Fprintf(os.Stderr, "To fix it, run `chmod 600 %v`\n", filename)
		return errors.New("account.conf permissions too lax")
	}
	return nil
}
