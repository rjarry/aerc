package config

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
	"github.com/imdario/mergo"
)

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

func defaultUiConfig() UIConfig {
	return UIConfig{
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
	}
}

func (config *AercConfig) parseUi(file *ini.File) error {
	if ui, err := file.GetSection("ui"); err == nil {
		if err := config.Ui.parse(ui); err != nil {
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
		if err := uiSubConfig.parse(uiSection); err != nil {
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

	// append default paths to styleset-dirs
	for _, dir := range SearchDirs {
		config.Ui.StyleSetDirs = append(
			config.Ui.StyleSetDirs, path.Join(dir, "stylesets"),
		)
	}

	if err := config.Ui.loadStyleSet(config.Ui.StyleSetDirs); err != nil {
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

	log.Debugf("aerc.conf: [ui] %#v", config.Ui)

	return nil
}

func (config *UIConfig) parse(section *ini.Section) error {
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

func (ui *UIConfig) loadStyleSet(styleSetDirs []string) error {
	ui.style = NewStyleSet()
	err := ui.style.LoadStyleSet(ui.StyleSetName, styleSetDirs)
	if err != nil {
		return fmt.Errorf("Unable to load default styleset: %w", err)
	}

	return nil
}

func (config *AercConfig) mergeContextualUi(baseUi UIConfig,
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
			log.Warnf("merge ui failed: %v", err)
		}
		if contextualUi.UiConfig.StyleSetName != "" {
			baseUi.style = contextualUi.UiConfig.style
		}
		return baseUi
	}

	return baseUi
}

func (uiConfig *UIConfig) GetStyle(so StyleObject) tcell.Style {
	return uiConfig.style.Get(so)
}

func (uiConfig *UIConfig) GetStyleSelected(so StyleObject) tcell.Style {
	return uiConfig.style.Selected(so)
}

func (uiConfig *UIConfig) GetComposedStyle(base StyleObject,
	styles []StyleObject,
) tcell.Style {
	return uiConfig.style.Compose(base, styles)
}

func (uiConfig *UIConfig) GetComposedStyleSelected(
	base StyleObject, styles []StyleObject,
) tcell.Style {
	return uiConfig.style.ComposeSelected(base, styles)
}

func (config *AercConfig) GetUiConfig(params map[ContextType]string) *UIConfig {
	baseUi := config.Ui

	for k, v := range params {
		baseUi = config.mergeContextualUi(baseUi, k, v)
	}

	return &baseUi
}

func (config *AercConfig) GetContextualUIConfigs() []UIConfigContext {
	return config.ContextualUis
}
