package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
	"github.com/google/shlex"
	"github.com/imdario/mergo"
	"github.com/kyoh86/xdg"
	"github.com/mitchellh/go-homedir"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/logging"
)

type GeneralConfig struct {
	DefaultSavePath    string `ini:"default-save-path"`
	PgpProvider        string `ini:"pgp-provider"`
	UnsafeAccountsConf bool   `ini:"unsafe-accounts-conf"`
}

type UIConfig struct {
	AutoMarkRead                  bool          `ini:"auto-mark-read"`
	IndexFormat                   string        `ini:"index-format"`
	TimestampFormat               string        `ini:"timestamp-format"`
	ThisDayTimeFormat             string        `ini:"this-day-time-format"`
	ThisWeekTimeFormat            string        `ini:"this-week-time-format"`
	ThisYearTimeFormat            string        `ini:"this-year-time-format"`
	MessageViewTimestampFormat    string        `ini:"message-view-timestamp-format"`
	MessageViewThisDayTimeFormat  string        `ini:"message-view-this-day-time-format"`
	MessageViewThisWeekTimeFormat string        `ini:"message-view-this-week-time-format"`
	MessageViewThisYearTimeFormat string        `ini:"message-view-this-year-time-format"`
	ShowHeaders                   []string      `delim:","`
	RenderAccountTabs             string        `ini:"render-account-tabs"`
	PinnedTabMarker               string        `ini:"pinned-tab-marker"`
	SidebarWidth                  int           `ini:"sidebar-width"`
	PreviewHeight                 int           `ini:"preview-height"`
	EmptyMessage                  string        `ini:"empty-message"`
	EmptyDirlist                  string        `ini:"empty-dirlist"`
	MouseEnabled                  bool          `ini:"mouse-enabled"`
	ThreadingEnabled              bool          `ini:"threading-enabled"`
	ForceClientThreads            bool          `ini:"force-client-threads"`
	ClientThreadsDelay            time.Duration `ini:"client-threads-delay"`
	FuzzyComplete                 bool          `ini:"fuzzy-complete"`
	NewMessageBell                bool          `ini:"new-message-bell"`
	Spinner                       string        `ini:"spinner"`
	SpinnerDelimiter              string        `ini:"spinner-delimiter"`
	IconUnencrypted               string        `ini:"icon-unencrypted"`
	IconEncrypted                 string        `ini:"icon-encrypted"`
	IconSigned                    string        `ini:"icon-signed"`
	IconSignedEncrypted           string        `ini:"icon-signed-encrypted"`
	IconUnknown                   string        `ini:"icon-unknown"`
	IconInvalid                   string        `ini:"icon-invalid"`
	DirListFormat                 string        `ini:"dirlist-format"`
	DirListDelay                  time.Duration `ini:"dirlist-delay"`
	DirListTree                   bool          `ini:"dirlist-tree"`
	DirListCollapse               int           `ini:"dirlist-collapse"`
	Sort                          []string      `delim:" "`
	NextMessageOnDelete           bool          `ini:"next-message-on-delete"`
	CompletionDelay               time.Duration `ini:"completion-delay"`
	CompletionPopovers            bool          `ini:"completion-popovers"`
	StyleSetDirs                  []string      `ini:"stylesets-dirs" delim:":"`
	StyleSetName                  string        `ini:"styleset-name"`
	style                         StyleSet
	// customize border appearance
	BorderCharVertical   rune `ini:"-"`
	BorderCharHorizontal rune `ini:"-"`
}

type ContextType int

const (
	UI_CONTEXT_FOLDER ContextType = iota
	UI_CONTEXT_ACCOUNT
	UI_CONTEXT_SUBJECT
	BIND_CONTEXT_ACCOUNT
	BIND_CONTEXT_FOLDER
)

type UIConfigContext struct {
	ContextType ContextType
	Regex       *regexp.Regexp
	UiConfig    UIConfig
}

const (
	FILTER_MIMETYPE = iota
	FILTER_HEADER
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
	Archive           string
	CopyTo            string
	Default           string
	Postpone          string
	From              string
	Aliases           string
	Name              string
	Source            string
	Folders           []string
	FoldersExclude    []string
	Params            map[string]string
	Outgoing          RemoteConfig
	SignatureFile     string
	SignatureCmd      string
	EnableFoldersSort bool     `ini:"enable-folders-sort"`
	FoldersSort       []string `ini:"folders-sort" delim:","`
	AddressBookCmd    string   `ini:"address-book-cmd"`
	SendAsUTC         bool     `ini:"send-as-utc"`

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

	// AuthRes
	TrustedAuthRes []string `ini:"trusted-authres" delim:","`
}

type BindingConfig struct {
	Global                 *KeyBindings
	AccountWizard          *KeyBindings
	Compose                *KeyBindings
	ComposeEditor          *KeyBindings
	ComposeReview          *KeyBindings
	MessageList            *KeyBindings
	MessageView            *KeyBindings
	MessageViewPassthrough *KeyBindings
	Terminal               *KeyBindings
}

type BindingConfigContext struct {
	ContextType ContextType
	Regex       *regexp.Regexp
	Bindings    *KeyBindings
	BindContext string
}

type ComposeConfig struct {
	Editor              string         `ini:"editor"`
	HeaderLayout        [][]string     `ini:"-"`
	AddressBookCmd      string         `ini:"address-book-cmd"`
	ReplyToSelf         bool           `ini:"reply-to-self"`
	NoAttachmentWarning *regexp.Regexp `ini:"-"`
}

type FilterConfig struct {
	FilterType int
	Filter     string
	Command    string
	Header     string
	Regex      *regexp.Regexp
}

type ViewerConfig struct {
	Pager          string
	Alternatives   []string
	ShowHeaders    bool       `ini:"show-headers"`
	AlwaysShowMime bool       `ini:"always-show-mime"`
	ParseHttpLinks bool       `ini:"parse-http-links"`
	HeaderLayout   [][]string `ini:"-"`
	KeyPassthrough bool       `ini:"-"`
}

type StatuslineConfig struct {
	RenderFormat string `ini:"render-format"`
	Separator    string
	DisplayMode  string `ini:"display-mode"`
}

type TriggersConfig struct {
	NewEmail       string `ini:"new-email"`
	ExecuteCommand func(command []string) error
}

type TemplateConfig struct {
	TemplateDirs []string `ini:"template-dirs" delim:":"`
	NewMessage   string   `ini:"new-message"`
	QuotedReply  string   `ini:"quoted-reply"`
	Forwards     string   `ini:"forwards"`
}

type AercConfig struct {
	Bindings        BindingConfig
	ContextualBinds []BindingConfigContext
	Compose         ComposeConfig
	Ini             *ini.File        `ini:"-"`
	Accounts        []AccountConfig  `ini:"-"`
	Filters         []FilterConfig   `ini:"-"`
	Viewer          ViewerConfig     `ini:"-"`
	Statusline      StatuslineConfig `ini:"-"`
	Triggers        TriggersConfig   `ini:"-"`
	Ui              UIConfig
	ContextualUis   []UIConfigContext
	General         GeneralConfig
	Templates       TemplateConfig
	Openers         map[string][]string
}

// Input: TimestampFormat
// Output: timestamp-format
func mapName(raw string) string {
	newstr := make([]rune, 0, len(raw))
	for i, chr := range raw {
		if isUpper := 'A' <= chr && chr <= 'Z'; isUpper {
			if i > 0 {
				newstr = append(newstr, '-')
			}
		}
		newstr = append(newstr, unicode.ToLower(chr))
	}
	return string(newstr)
}

func loadAccountConfig(path string, accts []string) ([]AccountConfig, error) {
	file, err := ini.Load(path)
	if err != nil {
		// No config triggers account configuration wizard
		return nil, nil
	}
	file.NameMapper = mapName

	var accounts []AccountConfig
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
		}
		if err = sec.MapTo(&account); err != nil {
			return nil, err
		}
		for key, val := range sec.KeysHash() {
			switch key {
			case "folders":
				folders := strings.Split(val, ",")
				sort.Strings(folders)
				account.Folders = folders
			case "folders-exclude":
				folders := strings.Split(val, ",")
				sort.Strings(folders)
				account.FoldersExclude = folders
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
					return nil, fmt.Errorf(
						"%s=%s %w", key, val, err)
				}
				account.Outgoing.CacheCmd = cache
			case "from":
				account.From = val
			case "aliases":
				account.Aliases = val
			case "copy-to":
				account.CopyTo = val
			case "archive":
				account.Archive = val
			case "enable-folders-sort":
				account.EnableFoldersSort, _ = strconv.ParseBool(val)
			case "pgp-key-id":
				account.PgpKeyId = val
			case "pgp-auto-sign":
				account.PgpAutoSign, _ = strconv.ParseBool(val)
			case "pgp-opportunistic-encrypt":
				account.PgpOpportunisticEncrypt, _ = strconv.ParseBool(val)
			case "address-book-cmd":
				account.AddressBookCmd = val
			default:
				if key != "name" {
					account.Params[key] = val
				}
			}
		}
		if account.Source == "" {
			return nil, fmt.Errorf("Expected source for account %s", _sec)
		}
		if account.From == "" {
			return nil, fmt.Errorf("Expected from for account %s", _sec)
		}

		source, err := sourceRemoteConfig.ConnectionString()
		if err != nil {
			return nil, fmt.Errorf("Invalid source credentials for %s: %w", _sec, err)
		}
		account.Source = source

		_, err = account.Outgoing.parseValue()
		if err != nil {
			return nil, fmt.Errorf("Invalid outgoing credentials for %s: %w", _sec, err)
		}

		accounts = append(accounts, account)
	}
	if len(accts) > 0 {
		// Sort accounts struct to match the specified order, if we
		// have one
		if len(accounts) != len(accts) {
			return nil, errors.New("account(s) not found")
		}
		sort.Slice(accounts, func(i, j int) bool {
			return strings.ToLower(accts[i]) < strings.ToLower(accts[j])
		})
	}
	return accounts, nil
}

// Set at build time
var shareDir string

func buildDefaultDirs() []string {
	var defaultDirs []string

	prefixes := []string{
		xdg.ConfigHome(),
		xdg.DataHome(),
	}

	// Add XDG_CONFIG_HOME and XDG_DATA_HOME
	for _, v := range prefixes {
		if v != "" {
			v, err := homedir.Expand(v)
			if err != nil {
				log.Println(err)
			}
			defaultDirs = append(defaultDirs, path.Join(v, "aerc"))
		}
	}

	// Add custom buildtime shareDir
	if shareDir != "" && shareDir != "/usr/local/share/aerc" {
		shareDir, err := homedir.Expand(shareDir)
		if err == nil {
			defaultDirs = append(defaultDirs, shareDir)
		}
	}

	// Add fixed fallback locations
	defaultDirs = append(defaultDirs, "/usr/local/share/aerc")
	defaultDirs = append(defaultDirs, "/usr/share/aerc")

	return defaultDirs
}

var SearchDirs = buildDefaultDirs()

func installTemplate(root, name string) error {
	var err error
	if _, err = os.Stat(root); os.IsNotExist(err) {
		err = os.MkdirAll(root, 0o755)
		if err != nil {
			return err
		}
	}
	var data []byte
	for _, dir := range SearchDirs {
		data, err = os.ReadFile(path.Join(dir, name))
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(root, name), data, 0o644)
	if err != nil {
		return err
	}
	return nil
}

func (config *AercConfig) LoadConfig(file *ini.File) error {
	if filters, err := file.GetSection("filters"); err == nil {
		// TODO: Parse the filter more finely, e.g. parse the regex
		for _, match := range filters.KeyStrings() {
			cmd := filters.KeysHash()[match]
			filter := FilterConfig{
				Command: cmd,
				Filter:  match,
			}
			switch {
			case strings.Contains(match, ",~"):
				filter.FilterType = FILTER_HEADER
				header := filter.Filter[:strings.Index(filter.Filter, ",")] //nolint:gocritic // guarded by strings.Contains
				regex := filter.Filter[strings.Index(filter.Filter, "~")+1:]
				filter.Header = strings.ToLower(header)
				filter.Regex, err = regexp.Compile(regex)
				if err != nil {
					return err
				}
			case strings.ContainsRune(match, ','):
				filter.FilterType = FILTER_HEADER
				header := filter.Filter[:strings.Index(filter.Filter, ",")] //nolint:gocritic // guarded by strings.Contains
				value := filter.Filter[strings.Index(filter.Filter, ",")+1:]
				filter.Header = strings.ToLower(header)
				filter.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
				if err != nil {
					return err
				}
			default:
				filter.FilterType = FILTER_MIMETYPE
			}
			config.Filters = append(config.Filters, filter)
		}
	}
	if openers, err := file.GetSection("openers"); err == nil {
		for mimeType, command := range openers.KeysHash() {
			mimeType = strings.ToLower(mimeType)
			if args, err := shlex.Split(command); err != nil {
				return err
			} else {
				config.Openers[mimeType] = args
			}
		}
	}
	if viewer, err := file.GetSection("viewer"); err == nil {
		if err := viewer.MapTo(&config.Viewer); err != nil {
			return err
		}
		for key, val := range viewer.KeysHash() {
			switch key {
			case "alternatives":
				config.Viewer.Alternatives = strings.Split(val, ",")
			case "header-layout":
				config.Viewer.HeaderLayout = parseLayout(val)
			}
		}
	}
	if statusline, err := file.GetSection("statusline"); err == nil {
		if err := statusline.MapTo(&config.Statusline); err != nil {
			return err
		}
	}
	if compose, err := file.GetSection("compose"); err == nil {
		if err := compose.MapTo(&config.Compose); err != nil {
			return err
		}
		for key, val := range compose.KeysHash() {
			if key == "header-layout" {
				config.Compose.HeaderLayout = parseLayout(val)
			}

			if key == "no-attachment-warning" && len(val) > 0 {
				re, err := regexp.Compile("(?im)" + val)
				if err != nil {
					return fmt.Errorf(
						"Invalid no-attachment-warning '%s': %w",
						val, err,
					)
				}

				config.Compose.NoAttachmentWarning = re
			}
		}
	}

	if ui, err := file.GetSection("ui"); err == nil {
		if err := parseUiConfig(ui, &config.Ui); err != nil {
			return err
		}
	}

	for _, sectionName := range file.SectionStrings() {
		if !strings.Contains(sectionName, "ui:") {
			continue
		}

		uiSection, err := file.GetSection(sectionName)
		if err != nil {
			return err
		}
		uiSubConfig := UIConfig{}
		if err := parseUiConfig(uiSection, &uiSubConfig); err != nil {
			return err
		}
		contextualUi := UIConfigContext{
			UiConfig: uiSubConfig,
		}

		var index int
		switch {
		case strings.Contains(sectionName, "~"):
			index = strings.Index(sectionName, "~")
			regex := string(sectionName[index+1:])
			contextualUi.Regex, err = regexp.Compile(regex)
			if err != nil {
				return err
			}
		case strings.Contains(sectionName, "="):
			index = strings.Index(sectionName, "=")
			value := string(sectionName[index+1:])
			contextualUi.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Invalid Ui Context regex in %s", sectionName)
		}

		switch sectionName[3:index] {
		case "account":
			contextualUi.ContextType = UI_CONTEXT_ACCOUNT
		case "folder":
			contextualUi.ContextType = UI_CONTEXT_FOLDER
		case "subject":
			contextualUi.ContextType = UI_CONTEXT_SUBJECT
		default:
			return fmt.Errorf("Unknown Contextual Ui Section: %s", sectionName)
		}
		config.ContextualUis = append(config.ContextualUis, contextualUi)
	}
	if triggers, err := file.GetSection("triggers"); err == nil {
		if err := triggers.MapTo(&config.Triggers); err != nil {
			return err
		}
	}
	if templatesSec, err := file.GetSection("templates"); err == nil {
		if err := templatesSec.MapTo(&config.Templates); err != nil {
			return err
		}
		templateDirs := templatesSec.Key("template-dirs").String()
		if templateDirs != "" {
			config.Templates.TemplateDirs = strings.Split(templateDirs, ":")
		}
	}

	// append default paths to template-dirs and styleset-dirs
	for _, dir := range SearchDirs {
		config.Ui.StyleSetDirs = append(
			config.Ui.StyleSetDirs, path.Join(dir, "stylesets"),
		)
		config.Templates.TemplateDirs = append(
			config.Templates.TemplateDirs, path.Join(dir, "templates"),
		)
	}

	// we want to fail during startup if the templates are not ok
	// hence we do dummy executes here
	t := config.Templates
	if err := templates.CheckTemplate(t.NewMessage, t.TemplateDirs); err != nil {
		return err
	}
	if err := templates.CheckTemplate(t.QuotedReply, t.TemplateDirs); err != nil {
		return err
	}
	if err := templates.CheckTemplate(t.Forwards, t.TemplateDirs); err != nil {
		return err
	}
	if err := config.Ui.loadStyleSet(
		config.Ui.StyleSetDirs); err != nil {
		return err
	}

	for idx, contextualUi := range config.ContextualUis {
		if contextualUi.UiConfig.StyleSetName == "" &&
			len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			continue // no need to do anything if nothing is overridden
		}
		// fill in the missing part from the base
		if contextualUi.UiConfig.StyleSetName == "" {
			config.ContextualUis[idx].UiConfig.StyleSetName = config.Ui.StyleSetName
		} else if len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			config.ContextualUis[idx].UiConfig.StyleSetDirs = config.Ui.StyleSetDirs
		}
		// since at least one of them has changed, load the styleset
		if err := config.ContextualUis[idx].UiConfig.loadStyleSet(
			config.ContextualUis[idx].UiConfig.StyleSetDirs); err != nil {
			return err
		}
	}

	return nil
}

func parseUiConfig(section *ini.Section, config *UIConfig) error {
	if err := section.MapTo(config); err != nil {
		return err
	}

	if key, err := section.GetKey("border-char-vertical"); err == nil {
		chars := []rune(key.String())
		if len(chars) != 1 {
			return fmt.Errorf("%v must be one and only one character", key)
		}
		config.BorderCharVertical = chars[0]
	}
	if key, err := section.GetKey("border-char-horizontal"); err == nil {
		chars := []rune(key.String())
		if len(chars) != 1 {
			return fmt.Errorf("%v must be one and only one character", key)
		}
		config.BorderCharHorizontal = chars[0]
	}

	// Values with type=time.Duration must be explicitly set. If these
	// values are given a default in the struct passed to ui.MapTo, which
	// they are, a zero-value in the config won't overwrite the default.
	if key, err := section.GetKey("dirlist-delay"); err == nil {
		dur, err := key.Duration()
		if err != nil {
			return err
		}
		config.DirListDelay = dur
	}
	if key, err := section.GetKey("completion-delay"); err == nil {
		dur, err := key.Duration()
		if err != nil {
			return err
		}
		config.CompletionDelay = dur
	}

	if config.MessageViewTimestampFormat == "" {
		config.MessageViewTimestampFormat = config.TimestampFormat
	}
	if config.MessageViewThisDayTimeFormat == "" {
		config.MessageViewThisDayTimeFormat = config.TimestampFormat
	}
	if config.MessageViewThisWeekTimeFormat == "" {
		config.MessageViewThisWeekTimeFormat = config.TimestampFormat
	}
	if config.MessageViewThisDayTimeFormat == "" {
		config.MessageViewThisDayTimeFormat = config.TimestampFormat
	}

	return nil
}

func validatePgpProvider(section *ini.Section) error {
	m := map[string]bool{
		"gpg":      true,
		"internal": true,
	}
	for key, val := range section.KeysHash() {
		if key == "pgp-provider" {
			if !m[strings.ToLower(val)] {
				return fmt.Errorf("%v must be either 'gpg' or 'internal'", key)
			}
		}
	}
	return nil
}

func LoadConfigFromFile(root *string, accts []string) (*AercConfig, error) {
	if root == nil {
		_root := path.Join(xdg.ConfigHome(), "aerc")
		root = &_root
	}
	filename := path.Join(*root, "aerc.conf")

	// if it doesn't exist copy over the template, then load
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		logging.Debugf("%s not found, installing the system default", filename)
		if err := installTemplate(*root, "aerc.conf"); err != nil {
			return nil, err
		}
	}

	logging.Infof("Parsing configuration from %s", filename)

	file, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		return nil, err
	}
	file.NameMapper = mapName
	config := &AercConfig{
		Bindings: BindingConfig{
			Global:                 NewKeyBindings(),
			AccountWizard:          NewKeyBindings(),
			Compose:                NewKeyBindings(),
			ComposeEditor:          NewKeyBindings(),
			ComposeReview:          NewKeyBindings(),
			MessageList:            NewKeyBindings(),
			MessageView:            NewKeyBindings(),
			MessageViewPassthrough: NewKeyBindings(),
			Terminal:               NewKeyBindings(),
		},

		ContextualBinds: []BindingConfigContext{},

		Ini: file,

		General: GeneralConfig{
			PgpProvider:        "internal",
			UnsafeAccountsConf: false,
		},

		Ui: UIConfig{
			AutoMarkRead:       true,
			IndexFormat:        "%-20.20D %-17.17n %Z %s",
			TimestampFormat:    "2006-01-02 03:04 PM",
			ThisDayTimeFormat:  "",
			ThisWeekTimeFormat: "",
			ThisYearTimeFormat: "",
			ShowHeaders: []string{
				"From", "To", "Cc", "Bcc", "Subject", "Date",
			},
			RenderAccountTabs:   "auto",
			PinnedTabMarker:     "`",
			SidebarWidth:        20,
			PreviewHeight:       12,
			EmptyMessage:        "(no messages)",
			EmptyDirlist:        "(no folders)",
			MouseEnabled:        false,
			ClientThreadsDelay:  50 * time.Millisecond,
			NewMessageBell:      true,
			FuzzyComplete:       false,
			Spinner:             "[..]    , [..]   ,  [..]  ,   [..] ,    [..],   [..] ,  [..]  , [..]   ",
			SpinnerDelimiter:    ",",
			IconUnencrypted:     "",
			IconSigned:          "[s]",
			IconEncrypted:       "[e]",
			IconSignedEncrypted: "",
			IconUnknown:         "[s?]",
			IconInvalid:         "[s!]",
			DirListFormat:       "%n %>r",
			DirListDelay:        200 * time.Millisecond,
			NextMessageOnDelete: true,
			CompletionDelay:     250 * time.Millisecond,
			CompletionPopovers:  true,
			StyleSetDirs:        []string{},
			StyleSetName:        "default",
			// border defaults
			BorderCharVertical:   ' ',
			BorderCharHorizontal: ' ',
		},

		ContextualUis: []UIConfigContext{},

		Viewer: ViewerConfig{
			Pager:        "less -R",
			Alternatives: []string{"text/plain", "text/html"},
			ShowHeaders:  false,
			HeaderLayout: [][]string{
				{"From", "To"},
				{"Cc", "Bcc"},
				{"Date"},
				{"Subject"},
			},
			ParseHttpLinks: true,
		},

		Statusline: StatuslineConfig{
			RenderFormat: "[%a] %S %>%T",
			Separator:    " | ",
			DisplayMode:  "",
		},

		Compose: ComposeConfig{
			HeaderLayout: [][]string{
				{"To", "From"},
				{"Subject"},
			},
			ReplyToSelf: true,
		},

		Templates: TemplateConfig{
			TemplateDirs: []string{},
			NewMessage:   "new_message",
			QuotedReply:  "quoted_reply",
			Forwards:     "forward_as_body",
		},

		Openers: make(map[string][]string),
	}

	// These bindings are not configurable
	config.Bindings.AccountWizard.ExKey = KeyStroke{
		Key: tcell.KeyCtrlE,
	}
	quit, _ := ParseBinding("<C-q>", ":quit<Enter>")
	config.Bindings.AccountWizard.Add(quit)

	if err = config.LoadConfig(file); err != nil {
		return nil, err
	}

	if ui, err := file.GetSection("general"); err == nil {
		if err := ui.MapTo(&config.General); err != nil {
			return nil, err
		}
		if err := validatePgpProvider(ui); err != nil {
			return nil, err
		}
	}

	logging.Debugf("aerc.conf: [general] %#v", config.General)
	logging.Debugf("aerc.conf: [ui] %#v", config.Ui)
	logging.Debugf("aerc.conf: [statusline] %#v", config.Statusline)
	logging.Debugf("aerc.conf: [viewer] %#v", config.Viewer)
	logging.Debugf("aerc.conf: [compose] %#v", config.Compose)
	logging.Debugf("aerc.conf: [filters] %#v", config.Filters)
	logging.Debugf("aerc.conf: [openers] %#v", config.Openers)
	logging.Debugf("aerc.conf: [triggers] %#v", config.Triggers)
	logging.Debugf("aerc.conf: [templates] %#v", config.Templates)

	filename = path.Join(*root, "accounts.conf")
	if !config.General.UnsafeAccountsConf {
		if err := checkConfigPerms(filename); err != nil {
			return nil, err
		}
	}

	accountsPath := path.Join(*root, "accounts.conf")
	logging.Infof("Parsing accounts configuration from %s", accountsPath)
	if accounts, err := loadAccountConfig(accountsPath, accts); err != nil {
		return nil, err
	} else {
		config.Accounts = accounts
	}

	for _, acct := range config.Accounts {
		logging.Debugf("accounts.conf: [%s] from = %s", acct.Name, acct.From)
	}

	filename = path.Join(*root, "binds.conf")
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		logging.Debugf("%s not found, installing the system default", filename)
		if err := installTemplate(*root, "binds.conf"); err != nil {
			return nil, err
		}
	}
	logging.Infof("Parsing key bindings configuration from %s", filename)
	binds, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}

	baseGroups := map[string]**KeyBindings{
		"default":           &config.Bindings.Global,
		"compose":           &config.Bindings.Compose,
		"messages":          &config.Bindings.MessageList,
		"terminal":          &config.Bindings.Terminal,
		"view":              &config.Bindings.MessageView,
		"view::passthrough": &config.Bindings.MessageViewPassthrough,
		"compose::editor":   &config.Bindings.ComposeEditor,
		"compose::review":   &config.Bindings.ComposeReview,
	}

	// Base Bindings
	for _, sectionName := range binds.SectionStrings() {
		// Handle :: delimeter
		baseSectionName := strings.ReplaceAll(sectionName, "::", "////")
		sections := strings.Split(baseSectionName, ":")
		baseOnly := len(sections) == 1
		baseSectionName = strings.ReplaceAll(sections[0], "////", "::")

		group, ok := baseGroups[strings.ToLower(baseSectionName)]
		if !ok {
			return nil, errors.New("Unknown keybinding group " + sectionName)
		}

		if baseOnly {
			err = config.LoadBinds(binds, baseSectionName, group)
			if err != nil {
				return nil, err
			}
		}
	}

	config.Bindings.Global.Globals = false
	for _, contextBind := range config.ContextualBinds {
		if contextBind.BindContext == "default" {
			contextBind.Bindings.Globals = false
		}
	}
	logging.Debugf("binds.conf: %#v", config.Bindings)

	return config, nil
}

func LoadBindingSection(sec *ini.Section) (*KeyBindings, error) {
	bindings := NewKeyBindings()
	for key, value := range sec.KeysHash() {
		if key == "$ex" {
			strokes, err := ParseKeyStrokes(value)
			if err != nil {
				return nil, err
			}
			if len(strokes) != 1 {
				return nil, errors.New("Invalid binding")
			}
			bindings.ExKey = strokes[0]
			continue
		}
		if key == "$noinherit" {
			if value == "false" {
				continue
			}
			if value != "true" {
				return nil, errors.New("Invalid binding")
			}
			bindings.Globals = false
			continue
		}
		binding, err := ParseBinding(key, value)
		if err != nil {
			return nil, err
		}
		bindings.Add(binding)
	}
	return bindings, nil
}

func (config *AercConfig) LoadBinds(binds *ini.File, baseName string, baseGroup **KeyBindings) error {
	if sec, err := binds.GetSection(baseName); err == nil {
		binds, err := LoadBindingSection(sec)
		if err != nil {
			return err
		}
		*baseGroup = MergeBindings(binds, *baseGroup)
	}

	for _, sectionName := range binds.SectionStrings() {
		if !strings.Contains(sectionName, baseName+":") ||
			strings.Contains(sectionName, baseName+"::") {
			continue
		}

		bindSection, err := binds.GetSection(sectionName)
		if err != nil {
			return err
		}

		binds, err := LoadBindingSection(bindSection)
		if err != nil {
			return err
		}

		contextualBind := BindingConfigContext{
			Bindings:    binds,
			BindContext: baseName,
		}

		var index int
		if strings.Contains(sectionName, "=") {
			index = strings.Index(sectionName, "=")
			value := string(sectionName[index+1:])
			contextualBind.Regex, err = regexp.Compile(value)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Invalid Bind Context regex in %s", sectionName)
		}

		switch sectionName[len(baseName)+1 : index] {
		case "account":
			acctName := sectionName[index+1:]
			valid := false
			for _, acctConf := range config.Accounts {
				matches := contextualBind.Regex.FindString(acctConf.Name)
				if matches != "" {
					valid = true
				}
			}
			if !valid {
				logging.Warnf("binds.conf: unexistent account: %s", acctName)
				continue
			}
			contextualBind.ContextType = BIND_CONTEXT_ACCOUNT
		case "folder":
			// No validation needed. If the folder doesn't exist, the binds
			// never get used
			contextualBind.ContextType = BIND_CONTEXT_FOLDER
		default:
			return fmt.Errorf("Unknown Context Bind Section: %s", sectionName)
		}
		config.ContextualBinds = append(config.ContextualBinds, contextualBind)
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

func parseLayout(layout string) [][]string {
	rows := strings.Split(layout, ",")
	l := make([][]string, len(rows))
	for i, r := range rows {
		l[i] = strings.Split(r, "|")
	}
	return l
}

func (ui *UIConfig) loadStyleSet(styleSetDirs []string) error {
	ui.style = NewStyleSet()
	err := ui.style.LoadStyleSet(ui.StyleSetName, styleSetDirs)
	if err != nil {
		return fmt.Errorf("Unable to load default styleset: %w", err)
	}

	return nil
}

func (config AercConfig) mergeContextualUi(baseUi UIConfig,
	contextType ContextType, s string,
) UIConfig {
	for _, contextualUi := range config.ContextualUis {
		if contextualUi.ContextType != contextType {
			continue
		}

		if !contextualUi.Regex.Match([]byte(s)) {
			continue
		}

		err := mergo.Merge(&baseUi, contextualUi.UiConfig, mergo.WithOverride)
		if err != nil {
			logging.Warnf("merge ui failed: %v", err)
		}
		if contextualUi.UiConfig.StyleSetName != "" {
			baseUi.style = contextualUi.UiConfig.style
		}
		return baseUi
	}

	return baseUi
}

func (config AercConfig) GetUiConfig(params map[ContextType]string) *UIConfig {
	baseUi := config.Ui

	for k, v := range params {
		baseUi = config.mergeContextualUi(baseUi, k, v)
	}

	return &baseUi
}

func (config *AercConfig) GetContextualUIConfigs() []UIConfigContext {
	return config.ContextualUis
}

func (uiConfig UIConfig) GetStyle(so StyleObject) tcell.Style {
	return uiConfig.style.Get(so)
}

func (uiConfig UIConfig) GetStyleSelected(so StyleObject) tcell.Style {
	return uiConfig.style.Selected(so)
}

func (uiConfig UIConfig) GetComposedStyle(base StyleObject,
	styles []StyleObject,
) tcell.Style {
	return uiConfig.style.Compose(base, styles)
}

func (uiConfig UIConfig) GetComposedStyleSelected(base StyleObject, styles []StyleObject) tcell.Style {
	return uiConfig.style.ComposeSelected(base, styles)
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
