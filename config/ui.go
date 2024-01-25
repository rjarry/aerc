package config

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"text/template"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
)

type UIConfig struct {
	IndexColumns    []*ColumnDef `ini:"index-columns" parse:"ParseIndexColumns" default:"date<20,name<17,flags>4,subject<*"`
	ColumnSeparator string       `ini:"column-separator" default:"  "`

	DirListLeft  *template.Template `ini:"dirlist-left" default:"{{.Folder}}"`
	DirListRight *template.Template `ini:"dirlist-right" default:"{{if .Unread}}{{humanReadable .Unread}}/{{end}}{{if .Exists}}{{humanReadable .Exists}}{{end}}"`

	AutoMarkRead                  bool          `ini:"auto-mark-read" default:"true"`
	TimestampFormat               string        `ini:"timestamp-format" default:"2006-01-02 03:04 PM"`
	ThisDayTimeFormat             string        `ini:"this-day-time-format"`
	ThisWeekTimeFormat            string        `ini:"this-week-time-format"`
	ThisYearTimeFormat            string        `ini:"this-year-time-format"`
	MessageViewTimestampFormat    string        `ini:"message-view-timestamp-format"`
	MessageViewThisDayTimeFormat  string        `ini:"message-view-this-day-time-format"`
	MessageViewThisWeekTimeFormat string        `ini:"message-view-this-week-time-format"`
	MessageViewThisYearTimeFormat string        `ini:"message-view-this-year-time-format"`
	PinnedTabMarker               string        "ini:\"pinned-tab-marker\" default:\"`\""
	SidebarWidth                  int           `ini:"sidebar-width" default:"20"`
	EmptyMessage                  string        `ini:"empty-message" default:"(no messages)"`
	EmptyDirlist                  string        `ini:"empty-dirlist" default:"(no folders)"`
	EmptySubject                  string        `ini:"empty-subject" default:"(no subject)"`
	MouseEnabled                  bool          `ini:"mouse-enabled"`
	ThreadingEnabled              bool          `ini:"threading-enabled"`
	ForceClientThreads            bool          `ini:"force-client-threads"`
	ClientThreadsDelay            time.Duration `ini:"client-threads-delay" default:"50ms"`
	ThreadContext                 bool          `ini:"show-thread-context"`
	FuzzyComplete                 bool          `ini:"fuzzy-complete"`
	NewMessageBell                bool          `ini:"new-message-bell" default:"true"`
	Spinner                       string        `ini:"spinner" default:"[..]    , [..]   ,  [..]  ,   [..] ,    [..],   [..] ,  [..]  , [..]   "`
	SpinnerDelimiter              string        `ini:"spinner-delimiter" default:","`
	SpinnerInterval               time.Duration `ini:"spinner-interval" default:"200ms"`
	IconUnencrypted               string        `ini:"icon-unencrypted"`
	IconEncrypted                 string        `ini:"icon-encrypted" default:"[e]"`
	IconSigned                    string        `ini:"icon-signed" default:"[s]"`
	IconSignedEncrypted           string        `ini:"icon-signed-encrypted"`
	IconUnknown                   string        `ini:"icon-unknown" default:"[s?]"`
	IconInvalid                   string        `ini:"icon-invalid" default:"[s!]"`
	IconAttachment                string        `ini:"icon-attachment" default:"a"`
	IconReplied                   string        `ini:"icon-replied" default:"r"`
	IconNew                       string        `ini:"icon-new" default:"N"`
	IconOld                       string        `ini:"icon-old" default:"O"`
	IconDraft                     string        `ini:"icon-draft" default:"d"`
	IconFlagged                   string        `ini:"icon-flagged" default:"!"`
	IconMarked                    string        `ini:"icon-marked" default:"*"`
	IconDeleted                   string        `ini:"icon-deleted" default:"X"`
	DirListDelay                  time.Duration `ini:"dirlist-delay" default:"200ms"`
	DirListTree                   bool          `ini:"dirlist-tree"`
	DirListCollapse               int           `ini:"dirlist-collapse"`
	Sort                          []string      `ini:"sort" delim:" "`
	NextMessageOnDelete           bool          `ini:"next-message-on-delete" default:"true"`
	CompletionDelay               time.Duration `ini:"completion-delay" default:"250ms"`
	CompletionMinChars            int           `ini:"completion-min-chars" default:"1" parse:"ParseCompletionMinChars"`
	CompletionPopovers            bool          `ini:"completion-popovers" default:"true"`
	StyleSetDirs                  []string      `ini:"stylesets-dirs" delim:":"`
	StyleSetName                  string        `ini:"styleset-name" default:"default"`
	style                         StyleSet
	// customize border appearance
	BorderCharVertical   rune `ini:"border-char-vertical" default:" " type:"rune"`
	BorderCharHorizontal rune `ini:"border-char-horizontal" default:" " type:"rune"`

	ReverseOrder       bool `ini:"reverse-msglist-order"`
	ReverseThreadOrder bool `ini:"reverse-thread-order"`
	SortThreadSiblings bool `ini:"sort-thread-siblings"`

	// Tab Templates
	TabTitleAccount  *template.Template `ini:"tab-title-account" default:"{{.Account}}"`
	TabTitleComposer *template.Template `ini:"tab-title-composer" default:"{{.Subject}}"`
	TabTitleViewer   *template.Template `ini:"tab-title-viewer" default:"{{if .Subject}}{{.Subject}}{{else}}(no subject){{end}}"`

	// private
	contextualUis    []*UiConfigContext
	contextualCounts map[uiContextType]int
	contextualCache  map[uiContextKey]*UIConfig
}

type uiContextType int

const (
	uiContextFolder uiContextType = iota
	uiContextAccount
)

type UiConfigContext struct {
	ContextType uiContextType
	Regex       *regexp.Regexp
	UiConfig    *UIConfig
	Section     ini.Section
}

type uiContextKey struct {
	ctxType uiContextType
	value   string
}

var Ui = &UIConfig{
	contextualCounts: make(map[uiContextType]int),
	contextualCache:  make(map[uiContextKey]*UIConfig),
}

var uiContextualSectionRe = regexp.MustCompile(`^ui:(account|folder|subject)([~=])(.+)$`)

func parseUi(file *ini.File) error {
	if err := Ui.parse(file.Section("ui")); err != nil {
		return err
	}

	for _, section := range file.Sections() {
		var err error
		groups := uiContextualSectionRe.FindStringSubmatch(section.Name())
		if groups == nil {
			continue
		}
		ctx, separator, value := groups[1], groups[2], groups[3]

		uiSubConfig := UIConfig{}
		if err = uiSubConfig.parse(section); err != nil {
			return err
		}
		contextualUi := UiConfigContext{
			UiConfig: &uiSubConfig,
			Section:  *section,
		}

		switch ctx {
		case "account":
			contextualUi.ContextType = uiContextAccount
		case "folder":
			contextualUi.ContextType = uiContextFolder
		}
		if separator == "=" {
			value = "^" + regexp.QuoteMeta(value) + "$"
		}
		contextualUi.Regex, err = regexp.Compile(value)
		if err != nil {
			return err
		}

		Ui.contextualUis = append(Ui.contextualUis, &contextualUi)
		Ui.contextualCounts[contextualUi.ContextType]++
	}

	// append default paths to styleset-dirs
	for _, dir := range SearchDirs {
		Ui.StyleSetDirs = append(
			Ui.StyleSetDirs, path.Join(dir, "stylesets"),
		)
	}

	if err := Ui.loadStyleSet(Ui.StyleSetDirs); err != nil {
		return err
	}

	for _, contextualUi := range Ui.contextualUis {
		if contextualUi.UiConfig.StyleSetName == "" &&
			len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			continue // no need to do anything if nothing is overridden
		}
		// fill in the missing part from the base
		if contextualUi.UiConfig.StyleSetName == "" {
			contextualUi.UiConfig.StyleSetName = Ui.StyleSetName
		} else if len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			contextualUi.UiConfig.StyleSetDirs = Ui.StyleSetDirs
		}
		// since at least one of them has changed, load the styleset
		if err := contextualUi.UiConfig.loadStyleSet(
			contextualUi.UiConfig.StyleSetDirs); err != nil {
			return err
		}
	}

	log.Debugf("aerc.conf: [ui] %#v", Ui)

	return nil
}

func (config *UIConfig) parse(section *ini.Section) error {
	if err := MapToStruct(section, config, section.Name() == "ui"); err != nil {
		return err
	}

	if config.MessageViewTimestampFormat == "" {
		config.MessageViewTimestampFormat = config.TimestampFormat
	}

	return nil
}

func (*UIConfig) ParseIndexColumns(section *ini.Section, key *ini.Key) ([]*ColumnDef, error) {
	if !section.HasKey("column-date") {
		_, _ = section.NewKey("column-date", `{{.DateAutoFormat .Date.Local}}`)
	}
	if !section.HasKey("column-name") {
		_, _ = section.NewKey("column-name", `{{index (.From | names) 0}}`)
	}
	if !section.HasKey("column-flags") {
		_, _ = section.NewKey("column-flags", `{{.Flags | join ""}}`)
	}
	if !section.HasKey("column-subject") {
		_, _ = section.NewKey("column-subject",
			`{{.ThreadPrefix}}{{if .ThreadFolded}}{{printf "{%d}" .ThreadCount}}{{end}}{{.Subject}}`)
	}
	return ParseColumnDefs(key, section)
}

const MANUAL_COMPLETE = math.MaxInt

func (*UIConfig) ParseCompletionMinChars(section *ini.Section, key *ini.Key) (int, error) {
	if key.String() == "manual" {
		return MANUAL_COMPLETE, nil
	}
	return key.Int()
}

func (ui *UIConfig) loadStyleSet(styleSetDirs []string) error {
	ui.style = NewStyleSet()
	err := ui.style.LoadStyleSet(ui.StyleSetName, styleSetDirs)
	if err != nil {
		return fmt.Errorf("Unable to load %q styleset: %w",
			ui.StyleSetName, err)
	}

	return nil
}

func (base *UIConfig) mergeContextual(
	contextType uiContextType, s string,
) *UIConfig {
	for _, contextualUi := range base.contextualUis {
		if contextualUi.ContextType != contextType {
			continue
		}
		if !contextualUi.Regex.Match([]byte(s)) {
			continue
		}
		ui := *base
		err := ui.parse(&contextualUi.Section)
		if err != nil {
			log.Warnf("merge ui failed: %v", err)
		}
		ui.contextualCache = make(map[uiContextKey]*UIConfig)
		ui.contextualCounts = base.contextualCounts
		ui.contextualUis = base.contextualUis
		if contextualUi.UiConfig.StyleSetName != "" {
			ui.style = contextualUi.UiConfig.style
		}
		return &ui
	}
	return base
}

func (uiConfig *UIConfig) GetUserStyle(name string) tcell.Style {
	return uiConfig.style.UserStyle(name)
}

func (uiConfig *UIConfig) GetStyle(so StyleObject) tcell.Style {
	return uiConfig.style.Get(so, nil)
}

func (uiConfig *UIConfig) GetStyleSelected(so StyleObject) tcell.Style {
	return uiConfig.style.Selected(so, nil)
}

func (uiConfig *UIConfig) GetComposedStyle(base StyleObject,
	styles []StyleObject,
) tcell.Style {
	return uiConfig.style.Compose(base, styles, nil)
}

func (uiConfig *UIConfig) GetComposedStyleSelected(
	base StyleObject, styles []StyleObject,
) tcell.Style {
	return uiConfig.style.ComposeSelected(base, styles, nil)
}

func (uiConfig *UIConfig) MsgComposedStyle(
	base StyleObject, styles []StyleObject, h *mail.Header,
) tcell.Style {
	return uiConfig.style.Compose(base, styles, h)
}

func (uiConfig *UIConfig) MsgComposedStyleSelected(
	base StyleObject, styles []StyleObject, h *mail.Header,
) tcell.Style {
	return uiConfig.style.ComposeSelected(base, styles, h)
}

func (uiConfig *UIConfig) StyleSetPath() string {
	return uiConfig.style.path
}

func (base *UIConfig) contextual(ctxType uiContextType, value string) *UIConfig {
	if base.contextualCounts[ctxType] == 0 {
		// shortcut if no contextual ui for that type
		return base
	}
	key := uiContextKey{ctxType: ctxType, value: value}
	c, found := base.contextualCache[key]
	if !found {
		c = base.mergeContextual(ctxType, value)
		base.contextualCache[key] = c
	}
	return c
}

func (base *UIConfig) ForAccount(account string) *UIConfig {
	return base.contextual(uiContextAccount, account)
}

func (base *UIConfig) ForFolder(folder string) *UIConfig {
	return base.contextual(uiContextFolder, folder)
}
