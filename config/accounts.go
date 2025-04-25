package config

import (
	"bytes"
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

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

var (
	EnablePinentry  func()
	DisablePinentry func()
	SetPinentryEnv  func(*exec.Cmd)
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
		usePinentry := EnablePinentry != nil &&
			DisablePinentry != nil &&
			SetPinentryEnv != nil

		cmd := exec.Command("sh", "-c", c.PasswordCmd)
		cmd.Stdin = os.Stdin

		buf := new(bytes.Buffer)
		cmd.Stderr = buf

		if usePinentry {
			EnablePinentry()
			defer DisablePinentry()
			SetPinentryEnv(cmd)
		}

		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %v: %w",
				buf.String(), err)
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
	Name    string
	Backend string
	// backend specific
	Params map[string]string

	Archive           string          `ini:"archive" default:"Archive"`
	CopyTo            []string        `ini:"copy-to" delim:","`
	CopyToReplied     bool            `ini:"copy-to-replied" default:"false"`
	StripBcc          bool            `ini:"strip-bcc" default:"true"`
	Default           string          `ini:"default" default:"INBOX"`
	Postpone          string          `ini:"postpone" default:"Drafts"`
	From              *mail.Address   `ini:"from"`
	UseEnvelopeFrom   bool            `ini:"use-envelope-from" default:"false"`
	OriginalToHeader  string          `ini:"original-to-header"`
	Aliases           []*mail.Address `ini:"aliases"`
	Source            string          `ini:"source" parse:"ParseSource"`
	Folders           []string        `ini:"folders" delim:","`
	FoldersExclude    []string        `ini:"folders-exclude" delim:","`
	Headers           []string        `ini:"headers" delim:","`
	HeadersExclude    []string        `ini:"headers-exclude" delim:","`
	Outgoing          RemoteConfig    `ini:"outgoing" parse:"ParseOutgoing"`
	SignatureFile     string          `ini:"signature-file"`
	SignatureCmd      string          `ini:"signature-cmd"`
	EnableFoldersSort bool            `ini:"enable-folders-sort" default:"true"`
	FoldersSort       []string        `ini:"folders-sort" delim:","`
	AddressBookCmd    string          `ini:"address-book-cmd"`
	SendAsUTC         bool            `ini:"send-as-utc" default:"false"`
	SendWithHostname  bool            `ini:"send-with-hostname" default:"false"`
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
	PgpAttachKey            bool   `ini:"pgp-attach-key"`
	PgpOpportunisticEncrypt bool   `ini:"pgp-opportunistic-encrypt"`
	PgpErrorLevel           int    `ini:"pgp-error-level" parse:"ParsePgpErrorLevel" default:"warn"`
	PgpSelfEncrypt          bool   `ini:"pgp-self-encrypt"`

	// AuthRes
	TrustedAuthRes []string `ini:"trusted-authres" delim:","`
}

const (
	PgpErrorLevelNone = iota
	PgpErrorLevelWarn
	PgpErrorLevelError
)

var Accounts []*AccountConfig

func parseAccountsFromFile(root string, accts []string, filename string) error {
	log.Debugf("Parsing accounts configuration from %s", filename)

	file, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		return err
	}

	starttls_warned := false
	var globals *ini.Section
	for _, _sec := range file.SectionStrings() {
		if _sec == "DEFAULT" {
			globals = file.Section(_sec)
			continue
		}
		if len(accts) > 0 && !contains(accts, _sec) {
			continue
		}
		sec := file.Section(_sec)
		for key, val := range globals.KeysHash() {
			if !sec.HasKey(key) {
				_, _ = sec.NewKey(key, val)
			}
		}

		account, err := ParseAccountConfig(_sec, sec)
		if err != nil {
			log.Errorf("failed to load account [%s]: %s", _sec, err)
			Warnings = append(Warnings, Warning{
				Title: "accounts.conf: error",
				Body: fmt.Sprintf(
					"Failed to load account [%s]:\n\n%s",
					_sec, err,
				),
			})
			continue
		}
		if _, ok := account.Params["smtp-starttls"]; ok && !starttls_warned {
			Warnings = append(Warnings, Warning{
				Title: "accounts.conf: smtp-starttls is deprecated",
				Body: `
SMTP connections now use STARTTLS by default and the smtp-starttls setting is ignored.

If you want to disable STARTTLS, append +insecure to the schema.
`,
			})
			starttls_warned = true
		}

		log.Debugf("accounts.conf: [%s] from = %s", account.Name, account.From)
		Accounts = append(Accounts, account)
	}
	if len(accts) > 0 {
		// Sort accounts struct to match the specified order, if we
		// have one
		var acctnames []string
		for _, acc := range Accounts {
			acctnames = append(acctnames, acc.Name)
		}
		var sortaccts []string
		for _, acc := range accts {
			if contains(acctnames, acc) {
				sortaccts = append(sortaccts, acc)
			} else {
				log.Errorf("account [%s] not found", acc)
			}
		}

		idx := make(map[string]int)
		for i, acct := range sortaccts {
			idx[acct] = i
		}
		sort.Slice(Accounts, func(i, j int) bool {
			return idx[Accounts[i].Name] < idx[Accounts[j].Name]
		})
	}

	return nil
}

func parseAccounts(root string, accts []string, filename string) error {
	if filename == "" {
		filename = path.Join(root, "accounts.conf")
		err := checkConfigPerms(filename)
		if errors.Is(err, os.ErrNotExist) {
			// No config triggers account configuration wizard
			return nil
		} else if err != nil {
			return err
		}
	}

	if err := parseAccountsFromFile(root, accts, filename); err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}

	return nil
}

func ParseAccountConfig(name string, section *ini.Section) (*AccountConfig, error) {
	account := AccountConfig{
		Name:   name,
		Params: make(map[string]string),
	}
	if err := MapToStruct(section, &account, true); err != nil {
		return nil, err
	}
	for key, val := range section.KeysHash() {
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
	if account.Source == "" {
		return nil, fmt.Errorf("missing 'source' parameter")
	}

	account.Backend = parseBackend(account.Source)
	if account.From == nil {
		return nil, fmt.Errorf("missing 'from' parameter")
	}
	if len(account.Headers) > 0 {
		defaults := []string{
			"date",
			"subject",
			"from",
			"sender",
			"reply-to",
			"to",
			"cc",
			"bcc",
			"in-reply-to",
			"message-id",
			"references",
		}
		account.Headers = append(account.Headers, defaults...)
	}
	return &account, nil
}

func parseBackend(source string) string {
	u, err := url.Parse(source)
	if err != nil {
		return ""
	}
	if strings.HasPrefix(u.Scheme, "imap") {
		return "imap"
	}
	if strings.HasPrefix(u.Scheme, "maildir") {
		return "maildir"
	}
	if strings.HasPrefix(u.Scheme, "jmap") {
		return "jmap"
	}
	return u.Scheme
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
	if err != nil {
		return err
	}

	perms := info.Mode().Perm()
	if perms&0o44 != 0 && !General.UnsafeAccountsConf {
		// group or others have read access
		fmt.Fprintf(os.Stderr, "The file %v has too open permissions.\n", filename)
		fmt.Fprintln(os.Stderr, "This is a security issue (it contains passwords).")
		fmt.Fprintf(os.Stderr, "To fix it, run `chmod 600 %v`\n", filename)
		return errors.New("account.conf permissions too lax")
	}
	return nil
}
