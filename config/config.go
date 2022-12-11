package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"
	"github.com/mitchellh/go-homedir"
)

type AercConfig struct {
	Bindings   BindingConfig
	Compose    ComposeConfig
	Converters map[string]string
	Accounts   []AccountConfig  `ini:"-"`
	Filters    []FilterConfig   `ini:"-"`
	Viewer     ViewerConfig     `ini:"-"`
	Statusline StatuslineConfig `ini:"-"`
	Triggers   TriggersConfig   `ini:"-"`
	Ui         UIConfig
	General    GeneralConfig
	Templates  TemplateConfig
	Openers    map[string][]string
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

func LoadConfigFromFile(root *string, accts []string) (*AercConfig, error) {
	if root == nil {
		_root := path.Join(xdg.ConfigHome(), "aerc")
		root = &_root
	}
	filename := path.Join(*root, "aerc.conf")

	// if it doesn't exist copy over the template, then load
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("%s not found, installing the system default", filename)
		if err := installTemplate(*root, "aerc.conf"); err != nil {
			return nil, err
		}
	}

	file, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		return nil, err
	}
	file.NameMapper = mapName
	config := &AercConfig{
		Bindings:   defaultBindsConfig(),
		General:    defaultGeneralConfig(),
		Ui:         defaultUiConfig(),
		Viewer:     defaultViewerConfig(),
		Statusline: defaultStatuslineConfig(),
		Compose:    defaultComposeConfig(),
		Converters: make(map[string]string),
		Templates:  defaultTemplatesConfig(),
		Openers:    make(map[string][]string),
	}

	if err := config.parseGeneral(file); err != nil {
		return nil, err
	}
	if err := config.parseFilters(file); err != nil {
		return nil, err
	}
	if err := config.parseCompose(file); err != nil {
		return nil, err
	}
	if err := config.parseConverters(file); err != nil {
		return nil, err
	}
	if err := config.parseViewer(file); err != nil {
		return nil, err
	}
	if err := config.parseStatusline(file); err != nil {
		return nil, err
	}
	if err := config.parseOpeners(file); err != nil {
		return nil, err
	}
	if err := config.parseTriggers(file); err != nil {
		return nil, err
	}
	if err := config.parseUi(file); err != nil {
		return nil, err
	}
	if err := config.parseTemplates(file); err != nil {
		return nil, err
	}
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
