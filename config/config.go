package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
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
	CompletionMinChars            int           `ini:"completion-min-chars"`
	CompletionPopovers            bool          `ini:"completion-popovers"`
	StyleSetDirs                  []string      `ini:"stylesets-dirs" delim:":"`
	StyleSetName                  string        `ini:"styleset-name"`
	style                         StyleSet
	// customize border appearance
	BorderCharVertical   rune `ini:"-"`
	BorderCharHorizontal rune `ini:"-"`

	ReverseOrder       bool `ini:"reverse-msglist-order"`
	ReverseThreadOrder bool `ini:"reverse-thread-order"`
	SortThreadSiblings bool `ini:"sort-thread-siblings"`
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
		Bindings: defaultBindsConfig(),

		ContextualBinds: []BindingConfigContext{},

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
			CompletionMinChars:  1,
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

	if err := config.parseAccounts(*root, accts); err != nil {
		return nil, err
	}
	if err := config.parseBinds(*root); err != nil {
		return nil, err
	}

	return config, nil
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
