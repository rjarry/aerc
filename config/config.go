package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-ini/ini"
	"github.com/google/shlex"
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

	// append default paths to template-dirs
	for _, dir := range SearchDirs {
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

		Ui:            defaultUiConfig(),
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
	if err := config.parseUi(file); err != nil {
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

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
