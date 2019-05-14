package config

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"
)

type UIConfig struct {
	IndexFormat       string   `ini:"index-format"`
	TimestampFormat   string   `ini:"timestamp-format"`
	ShowHeaders       []string `delim:","`
	LoadingFrames     []string `delim:","`
	RenderAccountTabs string   `ini:"render-account-tabs"`
	SidebarWidth      int      `ini:"sidebar-width"`
	PreviewHeight     int      `ini:"preview-height"`
	EmptyMessage      string   `ini:"empty-message"`
}

const (
	FILTER_MIMETYPE = iota
	FILTER_HEADER
)

type AccountConfig struct {
	Default  string
	From     string
	Name     string
	Source   string
	Folders  []string
	Params   map[string]string
	Outgoing string
}

type BindingConfig struct {
	Global        *KeyBindings
	Compose       *KeyBindings
	ComposeEditor *KeyBindings
	ComposeReview *KeyBindings
	MessageList   *KeyBindings
	MessageView   *KeyBindings
	Terminal      *KeyBindings
}

type ComposeConfig struct {
	Editor string `ini:"editor"`
}

type FilterConfig struct {
	FilterType int
	Filter     string
	Command    string
	Header     string
	Regex      *regexp.Regexp
}

type ViewerConfig struct {
	Pager        string
	Alternatives []string
}

type AercConfig struct {
	Bindings BindingConfig
	Compose  ComposeConfig
	Ini      *ini.File       `ini:"-"`
	Accounts []AccountConfig `ini:"-"`
	Filters  []FilterConfig  `ini:"-"`
	Viewer   ViewerConfig    `ini:"-"`
	Ui       UIConfig
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

func loadAccountConfig(path string) ([]AccountConfig, error) {
	file, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	file.NameMapper = mapName

	var accounts []AccountConfig
	for _, _sec := range file.SectionStrings() {
		if _sec == "DEFAULT" {
			continue
		}
		sec := file.Section(_sec)
		account := AccountConfig{
			Default: "INBOX",
			Name:    _sec,
			Params:  make(map[string]string),
		}
		if err = sec.MapTo(&account); err != nil {
			return nil, err
		}
		for key, val := range sec.KeysHash() {
			if key == "folders" {
				account.Folders = strings.Split(val, ",")
			} else if key == "outgoing" {
				account.Outgoing = val
			} else if key == "from" {
				account.From = val
			} else if key != "name" {
				account.Params[key] = val
			}
		}
		if account.Source == "" {
			return nil, fmt.Errorf("Expected source for account %s", _sec)
		}
		accounts = append(accounts, account)
	}
	if len(accounts) == 0 {
		err = errors.New("No accounts configured in accounts.conf")
		return nil, err
	}
	return accounts, nil
}

func LoadConfig(root *string) (*AercConfig, error) {
	if root == nil {
		_root := path.Join(xdg.ConfigHome(), "aerc")
		root = &_root
	}
	file, err := ini.Load(path.Join(*root, "aerc.conf"))
	if err != nil {
		return nil, err
	}
	file.NameMapper = mapName
	config := &AercConfig{
		Bindings: BindingConfig{
			Global:        NewKeyBindings(),
			Compose:       NewKeyBindings(),
			ComposeEditor: NewKeyBindings(),
			ComposeReview: NewKeyBindings(),
			MessageList:   NewKeyBindings(),
			MessageView:   NewKeyBindings(),
			Terminal:      NewKeyBindings(),
		},
		Ini: file,

		Ui: UIConfig{
			IndexFormat:     "%4C %Z %D %-17.17n %s",
			TimestampFormat: "%F %l:%M %p",
			ShowHeaders: []string{
				"From", "To", "Cc", "Bcc", "Subject", "Date",
			},
			LoadingFrames: []string{
				"[..]  ", " [..] ", "  [..]", " [..] ",
			},
			RenderAccountTabs: "auto",
			SidebarWidth:      20,
			PreviewHeight:     12,
			EmptyMessage:      "(no messages)",
		},
	}
	if filters, err := file.GetSection("filters"); err == nil {
		// TODO: Parse the filter more finely, e.g. parse the regex
		for _, match := range filters.KeyStrings() {
			cmd := filters.KeysHash()[match]
			filter := FilterConfig{
				Command: cmd,
				Filter:  match,
			}
			if strings.Contains(match, ",~") {
				filter.FilterType = FILTER_HEADER
				header := filter.Filter[:strings.Index(filter.Filter, ",")]
				regex := filter.Filter[strings.Index(filter.Filter, "~")+1:]
				filter.Header = strings.ToLower(header)
				filter.Regex, err = regexp.Compile(regex)
				if err != nil {
					panic(err)
				}
			} else if strings.ContainsRune(match, ',') {
				filter.FilterType = FILTER_HEADER
				header := filter.Filter[:strings.Index(filter.Filter, ",")]
				value := filter.Filter[strings.Index(filter.Filter, ",")+1:]
				filter.Header = strings.ToLower(header)
				filter.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
			} else {
				filter.FilterType = FILTER_MIMETYPE
			}
			config.Filters = append(config.Filters, filter)
		}
	}
	if viewer, err := file.GetSection("viewer"); err == nil {
		if err := viewer.MapTo(&config.Viewer); err != nil {
			return nil, err
		}
		for key, val := range viewer.KeysHash() {
			switch key {
			case "alternatives":
				config.Viewer.Alternatives = strings.Split(val, ",")
			}
		}
	}
	if compose, err := file.GetSection("compose"); err == nil {
		if err := compose.MapTo(&config.Compose); err != nil {
			return nil, err
		}
	}
	if ui, err := file.GetSection("ui"); err == nil {
		if err := ui.MapTo(&config.Ui); err != nil {
			return nil, err
		}
	}
	accountsPath := path.Join(*root, "accounts.conf")
	if accounts, err := loadAccountConfig(accountsPath); err != nil {
		return nil, err
	} else {
		config.Accounts = accounts
	}
	binds, err := ini.Load(path.Join(*root, "binds.conf"))
	if err != nil {
		return nil, err
	}
	groups := map[string]**KeyBindings{
		"default":  &config.Bindings.Global,
		"compose":  &config.Bindings.Compose,
		"messages": &config.Bindings.MessageList,
		"terminal": &config.Bindings.Terminal,
		"view":     &config.Bindings.MessageView,

		"compose::editor": &config.Bindings.ComposeEditor,
		"compose::review": &config.Bindings.ComposeReview,
	}
	for _, name := range binds.SectionStrings() {
		sec, err := binds.GetSection(name)
		if err != nil {
			return nil, err
		}
		group, ok := groups[strings.ToLower(name)]
		if !ok {
			return nil, errors.New("Unknown keybinding group " + name)
		}
		bindings := NewKeyBindings()
		for key, value := range sec.KeysHash() {
			if key == "$ex" {
				strokes, err := ParseKeyStrokes(value)
				if err != nil {
					return nil, err
				}
				if len(strokes) != 1 {
					return nil, errors.New(
						"Error: only one keystroke supported for $ex")
				}
				bindings.ExKey = strokes[0]
				continue
			}
			if key == "$noinherit" {
				if value == "false" {
					continue
				}
				if value != "true" {
					return nil, errors.New(
						"Error: expected 'true' or 'false' for $noinherit")
				}
				bindings.Globals = false
			}
			binding, err := ParseBinding(key, value)
			if err != nil {
				return nil, err
			}
			bindings.Add(binding)
		}
		*group = MergeBindings(bindings, *group)
	}
	// Globals can't inherit from themselves
	config.Bindings.Global.Globals = false
	return config, nil
}
