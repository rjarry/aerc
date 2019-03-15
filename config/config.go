package config

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"
)

type UIConfig struct {
	IndexFormat       string
	TimestampFormat   string
	ShowHeaders       []string `delim:","`
	LoadingFrames     []string `delim:","`
	RenderAccountTabs string
	SidebarWidth      int
	PreviewHeight     int
	EmptyMessage      string
}

type AccountConfig struct {
	Name    string
	Source  string
	Folders []string
	Params  map[string]string
}

type AercConfig struct {
	Lbinds   *KeyBindings
	Ini      *ini.File       `ini:"-"`
	Accounts []AccountConfig `ini:"-"`
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
		account := AccountConfig{Name: _sec}
		if err = sec.MapTo(&account); err != nil {
			return nil, err
		}
		for key, val := range sec.KeysHash() {
			if key == "source" {
				account.Source = val
			} else if key == "folders" {
				account.Folders = strings.Split(val, ",")
			} else if key != "name" {
				account.Params[key] = val
			}
		}
		if account.Source == "" {
			return nil, fmt.Errorf("Expected source for account %s", _sec)
		}
		accounts = append(accounts, account)
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
		Lbinds: NewKeyBindings(),
		Ini:    file,

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
	if ui, err := file.GetSection("ui"); err == nil {
		ui.MapTo(config.Ui)
	}
	if lbinds, err := file.GetSection("lbinds"); err == nil {
		for key, value := range lbinds.KeysHash() {
			binding, err := ParseBinding(key, value)
			if err != nil {
				return nil, err
			}
			config.Lbinds.Add(binding)
		}
	}
	accountsPath := path.Join(*root, "accounts.conf")
	if accounts, err := loadAccountConfig(accountsPath); err != nil {
		return nil, err
	} else {
		config.Accounts = accounts
	}
	return config, nil
}
