package config

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

type UIConfig struct {
	IndexColumns    []*ColumnDef `ini:"index-columns" parse:"ParseIndexColumns" default:"flags:4,name<20%,subject,date>="`
	ColumnSeparator string       `ini:"column-separator" default:"  "`

	DirListLeft  *template.Template `ini:"dirlist-left" default:"{{.Folder}}"`
	DirListRight *template.Template `ini:"dirlist-right" default:"{{if .Unread}}{{humanReadable .Unread}}{{end}}"`

	AutoMarkRead                  bool          `ini:"auto-mark-read" default:"true"`
	TimestampFormat               string        `ini:"timestamp-format" default:"2006 Jan 02"`
	ThisDayTimeFormat             string        `ini:"this-day-time-format" default:"15:04"`
	ThisWeekTimeFormat            string        `ini:"this-week-time-format" default:"Jan 02"`
	ThisYearTimeFormat            string        `ini:"this-year-time-format" default:"Jan 02"`
	MessageViewTimestampFormat    string        `ini:"message-view-timestamp-format" default:"2006 Jan 02, 15:04 GMT-0700"`
	MessageViewThisDayTimeFormat  string        `ini:"message-view-this-day-time-format"`
	MessageViewThisWeekTimeFormat string        `ini:"message-view-this-week-time-format"`
	MessageViewThisYearTimeFormat string        `ini:"message-view-this-year-time-format"`
	PinnedTabMarker               string        "ini:\"pinned-tab-marker\" default:\"`\""
	SidebarWidth                  int           `ini:"sidebar-width" default:"22"`
	MessageListSplit              SplitParams   `ini:"message-list-split" parse:"ParseSplit"`
	EmptyMessage                  string        `ini:"empty-message" default:"(no messages)"`
	EmptyDirlist                  string        `ini:"empty-dirlist" default:"(no folders)"`
	EmptySubject                  string        `ini:"empty-subject" default:"(no subject)"`
	MouseEnabled                  bool          `ini:"mouse-enabled"`
	ThreadingEnabled              bool          `ini:"threading-enabled"`
	ForceClientThreads            bool          `ini:"force-client-threads"`
	ThreadingBySubject            bool          `ini:"threading-by-subject"`
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
	IconForwarded                 string        `ini:"icon-forwarded" default:"f"`
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
	MsglistScrollOffset           int           `ini:"msglist-scroll-offset" default:"0"`
	DialogPosition                string        `ini:"dialog-position" default:"center" parse:"ParseDialogPosition"`
	DialogWidth                   int           `ini:"dialog-width" default:"50" parse:"ParseDialogDimensions"`
	DialogHeight                  int           `ini:"dialog-height" default:"50" parse:"ParseDialogDimensions"`
	StyleSetDirs                  []string      `ini:"stylesets-dirs" delim:":"`
	StyleSetName                  string        `ini:"styleset-name" default:"default"`
	style                         StyleSet
	// customize border appearance
	BorderCharVertical   rune `ini:"border-char-vertical" default:"│" type:"rune"`
	BorderCharHorizontal rune `ini:"border-char-horizontal" default:"─" type:"rune"`

	SelectLast         bool `ini:"select-last-message" default:"false"`
	ReverseOrder       bool `ini:"reverse-msglist-order"`
	ReverseThreadOrder bool `ini:"reverse-thread-order"`
	SortThreadSiblings bool `ini:"sort-thread-siblings"`

	ThreadPrefixTip                string `ini:"thread-prefix-tip" default:">"`
	ThreadPrefixIndent             string `ini:"thread-prefix-indent" default:" "`
	ThreadPrefixStem               string `ini:"thread-prefix-stem" default:"│"`
	ThreadPrefixLimb               string `ini:"thread-prefix-limb" default:""`
	ThreadPrefixFolded             string `ini:"thread-prefix-folded" default:"+"`
	ThreadPrefixUnfolded           string `ini:"thread-prefix-unfolded" default:""`
	ThreadPrefixFirstChild         string `ini:"thread-prefix-first-child" default:""`
	ThreadPrefixHasSiblings        string `ini:"thread-prefix-has-siblings" default:"├─"`
	ThreadPrefixLone               string `ini:"thread-prefix-lone" default:""`
	ThreadPrefixOrphan             string `ini:"thread-prefix-orphan" default:""`
	ThreadPrefixLastSibling        string `ini:"thread-prefix-last-sibling" default:"└─"`
	ThreadPrefixDummy              string `ini:"thread-prefix-dummy" default:"┬─"`
	ThreadPrefixLastSiblingReverse string `ini:"thread-prefix-last-sibling-reverse" default:"┌─"`
	ThreadPrefixFirstChildReverse  string `ini:"thread-prefix-first-child-reverse" default:""`
	ThreadPrefixOrphanReverse      string `ini:"thread-prefix-orphan-reverse" default:""`
	ThreadPrefixDummyReverse       string `ini:"thread-prefix-dummy-reverse" default:"┴─"`

	// Tab Templates
	TabTitleAccount  *template.Template `ini:"tab-title-account" default:"{{.Account}}"`
	TabTitleComposer *template.Template `ini:"tab-title-composer" default:"{{if .To}}to:{{index (.To | shortmboxes) 0}} {{end}}{{.SubjectBase}}"`
	TabTitleViewer   *template.Template `ini:"tab-title-viewer" default:"{{.Subject}}"`

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

var Ui = defaultUIConfig()

func defaultUIConfig() *UIConfig {
	return &UIConfig{
		contextualCounts: make(map[uiContextType]int),
		contextualCache:  make(map[uiContextKey]*UIConfig),
	}
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

	if err := Ui.LoadStyle(); err != nil {
		return err
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
		_, _ = section.NewKey("column-subject", `{{.ThreadPrefix}}{{.Subject}}`)
	}
	return ParseColumnDefs(key, section)
}

type SplitDirection int

const (
	SPLIT_NONE SplitDirection = iota
	SPLIT_HORIZONTAL
	SPLIT_VERTICAL
)

type SplitParams struct {
	Direction SplitDirection
	Size      int
}

func (*UIConfig) ParseSplit(section *ini.Section, key *ini.Key) (p SplitParams, err error) {
	re := regexp.MustCompile(`^\s*(v(?:ert(?:ical)?)?|h(?:oriz(?:ontal)?)?)?\s+(\d+)\s*$`)
	match := re.FindStringSubmatch(key.String())
	if len(match) != 3 {
		err = fmt.Errorf("bad option value")
		return
	}
	p.Direction = SPLIT_HORIZONTAL
	switch match[1] {
	case "v", "vert", "vertical":
		p.Direction = SPLIT_VERTICAL
	case "h", "horiz", "horizontal":
		p.Direction = SPLIT_HORIZONTAL
	}
	size, e := strconv.ParseUint(match[2], 10, 32)
	if e != nil {
		err = e
		return
	}
	p.Size = int(size)
	return
}

func (*UIConfig) ParseDialogPosition(section *ini.Section, key *ini.Key) (string, error) {
	match, _ := regexp.MatchString(`^\s*(top|center|bottom)\s*$`, key.String())
	if !(match) {
		return "", fmt.Errorf("bad option value")
	}
	return key.String(), nil
}

const (
	DIALOG_MIN_PROPORTION = 10
	DIALOG_MAX_PROPORTION = 100
)

func (*UIConfig) ParseDialogDimensions(section *ini.Section, key *ini.Key) (int, error) {
	value, err := key.Int()
	if value < DIALOG_MIN_PROPORTION || value > DIALOG_MAX_PROPORTION || err != nil {
		return 0, fmt.Errorf("value out of range")
	}
	return value, nil
}

const MANUAL_COMPLETE = math.MaxInt

func (*UIConfig) ParseCompletionMinChars(section *ini.Section, key *ini.Key) (int, error) {
	if key.String() == "manual" {
		return MANUAL_COMPLETE, nil
	}
	return key.Int()
}

func (ui *UIConfig) ClearCache() {
	for k := range ui.contextualCache {
		delete(ui.contextualCache, k)
	}
}

func (ui *UIConfig) LoadStyle() error {
	if err := ui.loadStyleSet(ui.StyleSetDirs); err != nil {
		return err
	}

	for _, contextualUi := range ui.contextualUis {
		if contextualUi.UiConfig.StyleSetName == "" &&
			len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			continue // no need to do anything if nothing is overridden
		}
		// fill in the missing part from the base
		if contextualUi.UiConfig.StyleSetName == "" {
			contextualUi.UiConfig.StyleSetName = ui.StyleSetName
		} else if len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			contextualUi.UiConfig.StyleSetDirs = ui.StyleSetDirs
		}
		// since at least one of them has changed, load the styleset
		if err := contextualUi.UiConfig.loadStyleSet(
			contextualUi.UiConfig.StyleSetDirs); err != nil {
			return err
		}
	}

	return nil
}

func (ui *UIConfig) loadStyleSet(styleSetDirs []string) error {
	ui.style = NewStyleSet()
	err := ui.style.LoadStyleSet(ui.StyleSetName, styleSetDirs)
	if err != nil {
		if ui.style.path == "" {
			ui.style.path = ui.StyleSetName
		}
		return fmt.Errorf("%s: %w", ui.style.path, err)
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

func (uiConfig *UIConfig) GetUserStyle(name string) vaxis.Style {
	return uiConfig.style.UserStyle(name)
}

func (uiConfig *UIConfig) GetStyle(so StyleObject) vaxis.Style {
	return uiConfig.style.Get(so, nil)
}

func (uiConfig *UIConfig) GetStyleSelected(so StyleObject) vaxis.Style {
	return uiConfig.style.Selected(so, nil)
}

func (uiConfig *UIConfig) GetComposedStyle(base StyleObject,
	styles []StyleObject,
) vaxis.Style {
	return uiConfig.style.Compose(base, styles, nil)
}

func (uiConfig *UIConfig) GetComposedStyleSelected(
	base StyleObject, styles []StyleObject,
) vaxis.Style {
	return uiConfig.style.ComposeSelected(base, styles, nil)
}

func (uiConfig *UIConfig) MsgComposedStyle(
	base StyleObject, styles []StyleObject, h *mail.Header,
) vaxis.Style {
	return uiConfig.style.Compose(base, styles, h)
}

func (uiConfig *UIConfig) MsgComposedStyleSelected(
	base StyleObject, styles []StyleObject, h *mail.Header,
) vaxis.Style {
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
