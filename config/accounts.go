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
	Name string
	// backend specific
	Params map[string]string

	Archive           string          `ini:"archive" default:"Archive"`
	CopyTo            string          `ini:"copy-to"`
	Default           string          `ini:"default" default:"INBOX"`
	Postpone          string          `ini:"postpone" default:"Drafts"`
	From              *mail.Address   `ini:"from"`
	Aliases           []*mail.Address `ini:"aliases"`
	Source            string          `ini:"source" parse:"ParseSource"`
	Folders           []string        `ini:"folders" delim:","`
	FoldersExclude    []string        `ini:"folders-exclude" delim:","`
	Outgoing          RemoteConfig    `ini:"outgoing" parse:"ParseOutgoing"`
	SignatureFile     string          `ini:"signature-file"`
	SignatureCmd      string          `ini:"signature-cmd"`
	EnableFoldersSort bool            `ini:"enable-folders-sort" default:"true"`
	FoldersSort       []string        `ini:"folders-sort" delim:","`
	AddressBookCmd    string          `ini:"address-book-cmd"`
	SendAsUTC         bool            `ini:"send-as-utc" default:"false"`
	LocalizedRe       *regexp.Regexp  `ini:"subject-re-pattern" default:"(?i)^((AW|RE|SV|VS|ODP|R): ?)+"`

	// CheckMail
	CheckMail        time.Duration `ini:"check-mail"`
	CheckMailCmd     string        `ini:"check-mail-cmd"`
	CheckMailTimeout time.Duration `ini:"check-mail-timeout" default:"10s"`
	CheckMailInclude []string      `ini:"check-mail-include"`
	CheckMailExclude []string      `ini:"check-mail-exclude"`

	// PGP Config
	PgpKeyId                string `ini:"pgp-key-id"`
	PgpAutoSign             bool   `ini:"pgp-auto-sign"`
	PgpOpportunisticEncrypt bool   `ini:"pgp-opportunistic-encrypt"`
	PgpErrorLevel           int    `ini:"pgp-error-level" parse:"ParsePgpErrorLevel" default:"warn"`

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

	for _, _sec := range file.SectionStrings() {
		if _sec == "DEFAULT" {
			continue
		}
		if len(accts) > 0 && !contains(accts, _sec) {
			continue
		}
		sec := file.Section(_sec)
		account := AccountConfig{
			Name:   _sec,
			Params: make(map[string]string),
		}
		if err = MapToStruct(sec, &account, true); err != nil {
			return err
		}
		for key, val := range sec.KeysHash() {
			backendSpecific := true
			typ := reflect.TypeOf(account)
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				if field.Tag.Get("ini") == key {
					backendSpecific = false
					break
				}
			}
			if backendSpecific {
				account.Params[key] = val
			}
		}
		if _, ok := account.Params["smtp-starttls"]; ok {
			Warnings = append(Warnings, Warning{
				Title: "accounts.conf: smtp-starttls is deprecated",
				Body: `
SMTP connections now use STARTTLS by default and the smtp-starttls setting is ignored.

If you want to disable STARTTLS, append +insecure to the schema.
`,
			})
		}
		if account.Source == "" {
			return fmt.Errorf("Expected source for account %s", _sec)
		}
		if account.From == nil {
			return fmt.Errorf("Expected from for account %s", _sec)
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

func (a *AccountConfig) ParseSource(sec *ini.Section, key *ini.Key) (string, error) {
	var remote RemoteConfig
	remote.Value = key.String()
	if k, err := sec.GetKey("source-cred-cmd"); err == nil {
		remote.PasswordCmd = k.String()
	}
	return remote.ConnectionString()
}

func (a *AccountConfig) ParseOutgoing(sec *ini.Section, key *ini.Key) (RemoteConfig, error) {
	var remote RemoteConfig
	remote.Value = key.String()
	if k, err := sec.GetKey("outgoing-cred-cmd"); err == nil {
		remote.PasswordCmd = k.String()
	}
	if k, err := sec.GetKey("outgoing-cred-cmd-cache"); err == nil {
		cache, err := k.Bool()
		if err != nil {
			return remote, err
		}
		remote.CacheCmd = cache
	}
	_, err := remote.parseValue()
	return remote, err
}

func (a *AccountConfig) ParsePgpErrorLevel(sec *ini.Section, key *ini.Key) (int, error) {
	var level int
	var err error
	switch strings.ToLower(key.String()) {
	case "none":
		level = PgpErrorLevelNone
	case "warn":
		level = PgpErrorLevelWarn
	case "error":
		level = PgpErrorLevelError
	default:
		err = fmt.Errorf("unknown level: %s", key.String())
	}
	return level, err
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
