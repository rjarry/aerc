package config

import (
	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"

	"path"
	"unicode"
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
	ConfigPath string
	Name       string
	Source     string
	Folders    []string
	Params     map[string]string
}

type AercConfig struct {
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

func LoadConfig(root *string) (*AercConfig, error) {
	var (
		err  error
		file *ini.File
	)
	if root == nil {
		_root := path.Join(xdg.ConfigHome(), "aerc")
		root = &_root
	}
	if file, err = ini.Load(path.Join(*root, "aerc.conf")); err != nil {
		return nil, err
	}
	file.NameMapper = mapName
	config := &AercConfig{
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
	if ui, err := file.GetSection("ui"); err != nil {
		ui.MapTo(config.Ui)
	}
	return config, nil
}
