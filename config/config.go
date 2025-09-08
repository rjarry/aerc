package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"github.com/go-ini/ini"
)

// Set at build time
var (
	shareDir   string
	libexecDir string
)

func buildDefaultDirs() []string {
	var defaultDirs []string

	prefixes := []string{
		xdg.ConfigPath(),
		"~/.local/libexec",
		xdg.DataPath(),
	}

	// Add XDG_CONFIG_HOME and XDG_DATA_HOME
	for _, v := range prefixes {
		if v != "" {
			defaultDirs = append(defaultDirs, xdg.ExpandHome(v, "aerc"))
		}
	}

	// Trim null chars inserted post-build by systems like Conda
	shareDir := strings.TrimRight(shareDir, "\x00")
	libexecDir := strings.TrimRight(libexecDir, "\x00")

	// Add custom buildtime dirs
	if libexecDir != "" && libexecDir != "/usr/local/libexec/aerc" {
		defaultDirs = append(defaultDirs, xdg.ExpandHome(libexecDir))
	}
	if shareDir != "" && shareDir != "/usr/local/share/aerc" {
		defaultDirs = append(defaultDirs, xdg.ExpandHome(shareDir))
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

func parseConf(filename string) error {
	file, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		return err
	}
	general, err := parseGeneral(file)
	if err != nil {
		return err
	}
	filters, err := parseFilters(file)
	if err != nil {
		return err
	}
	compose, err := parseCompose(file)
	if err != nil {
		return err
	}
	converters, err := parseConverters(file)
	if err != nil {
		return err
	}
	viewer, err := parseViewer(file)
	if err != nil {
		return err
	}
	statusline, err := parseStatusline(file)
	if err != nil {
		return err
	}
	openers, err := parseOpeners(file)
	if err != nil {
		return err
	}
	hooks, err := parseHooks(file)
	if err != nil {
		return err
	}
	ui, err := parseUi(file)
	if err != nil {
		return err
	}
	templates, err := parseTemplates(file)
	if err != nil {
		return err
	}

	// config parse successful, atomically change all items at once
	generalConfig.Store(general)
	filtersConfig.Store(&filters)
	composeConfig.Store(compose)
	convertersConfig.Store(&converters)
	viewerConfig.Store(viewer)
	statuslineConfig.Store(statusline)
	openersConfig.Store(&openers)
	hooksConfig.Store(hooks)
	uiConfig.Store(ui)
	templatesConfig.Store(templates)

	return nil
}

func init() {
	// store empty values to ensure unit-tests pass without configuration
	generalConfig.Store(&GeneralConfig{})
	filtersConfig.Store(nil)
	composeConfig.Store(&ComposeConfig{})
	convertersConfig.Store(nil)
	viewerConfig.Store(&ViewerConfig{})
	statuslineConfig.Store(&StatuslineConfig{})
	openersConfig.Store(nil)
	hooksConfig.Store(&HooksConfig{})
	ui := &UIConfig{}
	ui.style.Store(&StyleSet{})
	uiConfig.Store(ui)
	templatesConfig.Store(&TemplateConfig{})
}

func LoadConfigFromFile(
	root *string, accts []string, filename, bindPath, acctPath string,
) error {
	if root == nil {
		_root := xdg.ConfigPath("aerc")
		root = &_root
	}
	if filename == "" {
		filename = path.Join(*root, "aerc.conf")
		// if it doesn't exist copy over the template, then load
		if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("%s not found, installing the system default\n", filename)
			if err := installTemplate(*root, "aerc.conf"); err != nil {
				return err
			}
		}
	}
	SetConfFilename(filename)
	if err := parseConf(filename); err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}
	if err := parseAccounts(*root, accts, acctPath); err != nil {
		return err
	}
	if err := parseBinds(*root, bindPath); err != nil {
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
	return slices.Contains(list, v)
}

// warning message related to configuration (deprecation, etc.)
type Warning struct {
	Title string
	Body  string
}

var Warnings []Warning
