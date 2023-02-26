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
var (
	shareDir   string
	libexecDir string
)

func buildDefaultDirs() []string {
	var defaultDirs []string

	prefixes := []string{
		xdg.ConfigHome(),
		"~/.local/libexec",
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

	// Add custom buildtime dirs
	if libexecDir != "" && libexecDir != "/usr/local/libexec/aerc" {
		libexecDir, err := homedir.Expand(libexecDir)
		if err == nil {
			defaultDirs = append(defaultDirs, libexecDir)
		}
	}
	if shareDir != "" && shareDir != "/usr/local/share/aerc" {
		shareDir, err := homedir.Expand(shareDir)
		if err == nil {
			defaultDirs = append(defaultDirs, shareDir)
		}
	}

	// Add fixed fallback locations
	defaultDirs = append(defaultDirs, "/usr/local/libexec/aerc")
	defaultDirs = append(defaultDirs, "/usr/local/share/aerc")
	defaultDirs = append(defaultDirs, "/usr/libexec/aerc")
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

func LoadConfigFromFile(root *string, accts []string) error {
	if root == nil {
		_root := path.Join(xdg.ConfigHome(), "aerc")
		root = &_root
	}
	filename := path.Join(*root, "aerc.conf")

	// if it doesn't exist copy over the template, then load
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("%s not found, installing the system default", filename)
		if err := installTemplate(*root, "aerc.conf"); err != nil {
			return err
		}
	}

	file, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		return err
	}
	file.NameMapper = mapName

	if err := parseGeneral(file); err != nil {
		return err
	}
	if err := parseFilters(file); err != nil {
		return err
	}
	if err := parseCompose(file); err != nil {
		return err
	}
	if err := parseConverters(file); err != nil {
		return err
	}
	if err := parseViewer(file); err != nil {
		return err
	}
	if err := parseStatusline(file); err != nil {
		return err
	}
	if err := parseOpeners(file); err != nil {
		return err
	}
	if err := parseTriggers(file); err != nil {
		return err
	}
	if err := parseUi(file); err != nil {
		return err
	}
	if err := parseTemplates(file); err != nil {
		return err
	}
	if err := parseAccounts(*root, accts); err != nil {
		return err
	}
	if err := parseBinds(*root); err != nil {
		return err
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

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// warning message related to configuration (deprecation, etc.)
type Warning struct {
	Title string
	Body  string
}

var Warnings []Warning
