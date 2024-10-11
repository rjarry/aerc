package app

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/danwakefield/fnmatch"
	"github.com/emersion/go-message/textproto"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/auth"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/go-opt/v2"
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/align"

	// Image support
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// All imported image types need to be explicitly stated here. We want to check
// if we _can_ display something before we download it
var supportedImageTypes = []string{
	"image/jpeg",
	"image/png",
	"image/bmp",
	"image/tiff",
	"image/webp",
}

var _ ProvidesMessages = (*MessageViewer)(nil)

type MessageViewer struct {
	acct     *AccountView
	grid     *ui.Grid
	switcher *PartSwitcher
	msg      lib.MessageView
	uiConfig *config.UIConfig
}

func NewMessageViewer(
	acct *AccountView, msg lib.MessageView,
) (*MessageViewer, error) {
	if msg == nil {
		return &MessageViewer{acct: acct}, nil
	}
	hf := HeaderLayoutFilter{
		layout: HeaderLayout(config.Viewer.HeaderLayout),
		keep: func(msg *models.MessageInfo, header string) bool {
			return fmtHeader(msg, header, "2", "3", "4", "5") != ""
		},
	}
	layout := hf.forMessage(msg.MessageInfo())
	header, headerHeight := layout.grid(
		func(header string) ui.Drawable {
			hv := &HeaderView{
				Name: header,
				Value: fmtHeader(
					msg.MessageInfo(),
					header,
					acct.UiConfig().MessageViewTimestampFormat,
					acct.UiConfig().MessageViewThisDayTimeFormat,
					acct.UiConfig().MessageViewThisWeekTimeFormat,
					acct.UiConfig().MessageViewThisYearTimeFormat,
				),
				uiConfig: acct.UiConfig(),
			}
			showInfo := false
			if i := strings.IndexRune(header, '+'); i > 0 {
				header = header[:i]
				hv.Name = header
				showInfo = true
			}
			if parser := auth.New(header); parser != nil && msg.MessageInfo().Error == nil {
				details, err := parser(msg.MessageInfo().RFC822Headers, acct.AccountConfig().TrustedAuthRes)
				if err != nil {
					hv.Value = err.Error()
				} else {
					hv.ValueField = NewAuthInfo(details, showInfo, acct.UiConfig())
				}
				hv.Invalidate()
			}
			return hv
		},
	)

	rows := []ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(headerHeight)},
	}

	if msg.MessageDetails() != nil || acct.UiConfig().IconUnencrypted != "" {
		height := 1
		if msg.MessageDetails() != nil && msg.MessageDetails().IsSigned && msg.MessageDetails().IsEncrypted {
			height = 2
		}
		rows = append(rows, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(height)})
	}

	rows = append(rows, []ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}...)

	grid := ui.NewGrid().Rows(rows).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	switcher := &PartSwitcher{}
	err := createSwitcher(acct, switcher, msg)
	if err != nil {
		return nil, err
	}

	borderStyle := acct.UiConfig().GetStyle(config.STYLE_BORDER)
	borderChar := acct.UiConfig().BorderCharHorizontal

	grid.AddChild(header).At(0, 0)
	if msg.MessageDetails() != nil || acct.UiConfig().IconUnencrypted != "" {
		grid.AddChild(NewPGPInfo(msg.MessageDetails(), acct.UiConfig())).At(1, 0)
		grid.AddChild(ui.NewFill(borderChar, borderStyle)).At(2, 0)
		grid.AddChild(switcher).At(3, 0)
	} else {
		grid.AddChild(ui.NewFill(borderChar, borderStyle)).At(1, 0)
		grid.AddChild(switcher).At(2, 0)
	}

	mv := &MessageViewer{
		acct:     acct,
		grid:     grid,
		msg:      msg,
		switcher: switcher,
		uiConfig: acct.UiConfig(),
	}
	switcher.uiConfig = mv.uiConfig

	return mv, nil
}

func fmtHeader(msg *models.MessageInfo, header string,
	timefmt string, todayFormat string, thisWeekFormat string, thisYearFormat string,
) string {
	if msg == nil || msg.Envelope == nil {
		return "error: no envelope for this message"
	}

	if v := auth.New(header); v != nil {
		return "Fetching.."
	}

	switch header {
	case "From":
		return format.FormatAddresses(msg.Envelope.From)
	case "To":
		return format.FormatAddresses(msg.Envelope.To)
	case "Cc":
		return format.FormatAddresses(msg.Envelope.Cc)
	case "Bcc":
		return format.FormatAddresses(msg.Envelope.Bcc)
	case "Date":
		return format.DummyIfZeroDate(
			msg.Envelope.Date.Local(),
			timefmt,
			todayFormat,
			thisWeekFormat,
			thisYearFormat,
		)
	case "Subject":
		return msg.Envelope.Subject
	case "Labels":
		return strings.Join(msg.Labels, ", ")
	default:
		return msg.RFC822Headers.Get(header)
	}
}

func enumerateParts(
	acct *AccountView, msg lib.MessageView,
	body *models.BodyStructure, index []int,
) ([]*PartViewer, error) {
	var parts []*PartViewer
	for i, part := range body.Parts {
		curindex := append(index, i+1) //nolint:gocritic // intentional append to different slice
		if part.MIMEType == "multipart" {
			// Multipart meta-parts are faked
			pv := &PartViewer{part: part}
			parts = append(parts, pv)
			subParts, err := enumerateParts(
				acct, msg, part, curindex)
			if err != nil {
				return nil, err
			}
			parts = append(parts, subParts...)
			continue
		}
		pv, err := NewPartViewer(acct, msg, part, curindex)
		if err != nil {
			return nil, err
		}
		parts = append(parts, pv)
	}
	return parts, nil
}

func createSwitcher(
	acct *AccountView, switcher *PartSwitcher, msg lib.MessageView,
) error {
	var err error
	switcher.selected = -1

	if msg.MessageInfo().Error != nil {
		return fmt.Errorf("could not view message: %w", msg.MessageInfo().Error)
	}

	if len(msg.BodyStructure().Parts) == 0 {
		switcher.selected = 0
		pv, err := NewPartViewer(acct, msg, msg.BodyStructure(), nil)
		if err != nil {
			return err
		}
		switcher.parts = []*PartViewer{pv}
	} else {
		switcher.parts, err = enumerateParts(acct, msg,
			msg.BodyStructure(), []int{})
		if err != nil {
			return err
		}
		selectedPriority := -1
		log.Tracef("Selecting best message from %v", config.Viewer.Alternatives)
		for i, pv := range switcher.parts {
			// Switch to user's preferred mimetype
			if switcher.selected == -1 && pv.part.MIMEType != "multipart" {
				switcher.selected = i
			}
			mime := pv.part.FullMIMEType()
			for idx, m := range config.Viewer.Alternatives {
				if m != mime {
					continue
				}
				priority := len(config.Viewer.Alternatives) - idx
				if priority > selectedPriority {
					selectedPriority = priority
					switcher.selected = i
				}
			}
		}
	}
	return nil
}

func (mv *MessageViewer) Draw(ctx *ui.Context) {
	if mv.switcher == nil {
		style := mv.acct.UiConfig().GetStyle(config.STYLE_DEFAULT)
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
		ctx.Printf(0, 0, style, "%s", "(no message selected)")
		return
	}
	mv.grid.Draw(ctx)
}

func (mv *MessageViewer) MouseEvent(localX int, localY int, event vaxis.Event) {
	if mv.switcher == nil {
		return
	}
	mv.grid.MouseEvent(localX, localY, event)
}

func (mv *MessageViewer) Invalidate() {
	ui.Invalidate()
}

func (mv *MessageViewer) Terminal() *Terminal {
	if mv.switcher == nil {
		return nil
	}

	nparts := len(mv.switcher.parts)
	if nparts == 0 || mv.switcher.selected < 0 || mv.switcher.selected >= nparts {
		return nil
	}

	pv := mv.switcher.parts[mv.switcher.selected]
	if pv == nil {
		return nil
	}

	return pv.term
}

func (mv *MessageViewer) Store() *lib.MessageStore {
	return mv.msg.Store()
}

func (mv *MessageViewer) SelectedAccount() *AccountView {
	return mv.acct
}

func (mv *MessageViewer) MessageView() lib.MessageView {
	return mv.msg
}

func (mv *MessageViewer) SelectedMessage() (*models.MessageInfo, error) {
	if mv.msg == nil {
		return nil, errors.New("no message selected")
	}
	return mv.msg.MessageInfo(), nil
}

func (mv *MessageViewer) MarkedMessages() ([]models.UID, error) {
	return mv.acct.MarkedMessages()
}

func (mv *MessageViewer) ToggleHeaders() {
	if mv.switcher == nil {
		return
	}
	switcher := mv.switcher
	switcher.Cleanup()
	config.Viewer.ShowHeaders = !config.Viewer.ShowHeaders
	err := createSwitcher(mv.acct, switcher, mv.msg)
	if err != nil {
		log.Errorf("cannot create switcher: %v", err)
	}
	switcher.Invalidate()
}

func (mv *MessageViewer) ToggleKeyPassthrough() bool {
	config.Viewer.KeyPassthrough = !config.Viewer.KeyPassthrough
	return config.Viewer.KeyPassthrough
}

func (mv *MessageViewer) SelectedMessagePart() *PartInfo {
	if mv.switcher == nil {
		return nil
	}
	part := mv.switcher.SelectedPart()
	return &PartInfo{
		Index: part.index,
		Msg:   part.msg.MessageInfo(),
		Part:  part.part,
		Links: part.links,
	}
}

func (mv *MessageViewer) AttachmentParts(all bool) []*PartInfo {
	if mv.switcher == nil {
		return nil
	}
	return mv.switcher.AttachmentParts(all)
}

func (mv *MessageViewer) PreviousPart() {
	if mv.switcher == nil {
		return
	}
	mv.switcher.PreviousPart()
	mv.Invalidate()
}

func (mv *MessageViewer) NextPart() {
	if mv.switcher == nil {
		return
	}
	mv.switcher.NextPart()
	mv.Invalidate()
}

func (mv *MessageViewer) Bindings() string {
	if config.Viewer.KeyPassthrough {
		return "view::passthrough"
	} else {
		return "view"
	}
}

func (mv *MessageViewer) Close() {
	if mv.switcher != nil {
		mv.switcher.Cleanup()
	}
}

func (mv *MessageViewer) Event(event vaxis.Event) bool {
	if mv.switcher != nil {
		return mv.switcher.Event(event)
	}
	return false
}

func (mv *MessageViewer) Focus(focus bool) {
	if mv.switcher != nil {
		mv.switcher.Focus(focus)
	}
}

func (mv *MessageViewer) Show(visible bool) {
	if mv.switcher != nil {
		mv.switcher.Show(visible)
	}
}

type PartViewer struct {
	acctConfig *config.AccountConfig
	err        error
	fetched    bool
	filter     *exec.Cmd
	index      []int
	msg        lib.MessageView
	pager      *exec.Cmd
	pagerin    io.WriteCloser
	part       *models.BodyStructure
	source     io.Reader
	term       *Terminal
	grid       *ui.Grid
	noFilter   *ui.Grid
	uiConfig   *config.UIConfig
	copying    int32
	inlineImg  bool
	image      image.Image
	graphic    vaxis.Image
	width      int
	height     int

	links []string
}

const copying int32 = 1

func NewPartViewer(
	acct *AccountView, msg lib.MessageView, part *models.BodyStructure,
	curindex []int,
) (*PartViewer, error) {
	var (
		filter  *exec.Cmd
		pager   *exec.Cmd
		pagerin io.WriteCloser
		term    *Terminal
	)
	pagerCmd, err := CmdFallbackSearch(config.PagerCmds(), false)
	if err != nil {
		acct.PushError(fmt.Errorf("could not start pager: %w", err))
		return nil, err
	}
	cmd := opt.SplitArgs(pagerCmd)
	pager = exec.Command(cmd[0], cmd[1:]...)

	info := msg.MessageInfo()
	mime := part.FullMIMEType()

	for _, f := range config.Filters {
		switch f.Type {
		case config.FILTER_MIMETYPE:
			if fnmatch.Match(f.Filter, mime, 0) {
				filter = exec.Command("sh", "-c", f.Command)
			}
		case config.FILTER_HEADER:
			var header string
			switch f.Header {
			case "subject":
				header = info.Envelope.Subject
			case "from":
				header = format.FormatAddresses(info.Envelope.From)
			case "to":
				header = format.FormatAddresses(info.Envelope.To)
			case "cc":
				header = format.FormatAddresses(info.Envelope.Cc)
			default:
				header = msg.MessageInfo().RFC822Headers.Get(f.Header)
			}
			if f.Regex.Match([]byte(header)) {
				filter = exec.Command("sh", "-c", f.Command)
			}
		case config.FILTER_FILENAME:
			if f.Regex.Match([]byte(part.DispositionParams["filename"])) {
				filter = exec.Command("sh", "-c", f.Command)
				log.Tracef("command %v", f.Command)
			}
		}
		if filter != nil {
			break
		}
	}
	var noFilter *ui.Grid
	if filter != nil {
		path, _ := os.LookupEnv("PATH")
		var paths []string
		for _, dir := range config.SearchDirs {
			paths = append(paths, dir+"/filters")
		}
		paths = append(paths, path)
		path = strings.Join(paths, ":")
		filter.Env = os.Environ()
		filter.Env = append(filter.Env, fmt.Sprintf("PATH=%s", path))
		filter.Env = append(filter.Env,
			fmt.Sprintf("AERC_MIME_TYPE=%s", mime))
		filter.Env = append(filter.Env,
			fmt.Sprintf("AERC_FILENAME=%s", part.FileName()))
		if flowed, ok := part.Params["format"]; ok {
			filter.Env = append(filter.Env,
				fmt.Sprintf("AERC_FORMAT=%s", flowed))
		}
		filter.Env = append(filter.Env,
			fmt.Sprintf("AERC_SUBJECT=%s", info.Envelope.Subject))
		filter.Env = append(filter.Env, fmt.Sprintf("AERC_FROM=%s",
			format.FormatAddresses(info.Envelope.From)))
		filter.Env = append(filter.Env, fmt.Sprintf("AERC_STYLESET=%s",
			acct.UiConfig().StyleSetPath()))
		if config.General.EnableOSC8 {
			filter.Env = append(filter.Env, "AERC_OSC8_URLS=1")
		}
		log.Debugf("<%s> part=%v %s: %v | %v",
			info.Envelope.MessageId, curindex, mime, filter, pager)
		if pagerin, err = pager.StdinPipe(); err != nil {
			return nil, err
		}
		if term, err = NewTerminal(pager); err != nil {
			return nil, err
		}
	} else {
		noFilter = newNoFilterConfigured(acct.Name(), part)
	}

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(3)}, // Message
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	index := make([]int, len(curindex))
	copy(index, curindex)

	pv := &PartViewer{
		acctConfig: acct.AccountConfig(),
		filter:     filter,
		index:      index,
		msg:        msg,
		pager:      pager,
		pagerin:    pagerin,
		part:       part,
		term:       term,
		grid:       grid,
		noFilter:   noFilter,
		uiConfig:   acct.UiConfig(),
	}

	if term != nil {
		term.OnStart = func() {
			if term.ctx != nil {
				filter.Env = append(filter.Env, fmt.Sprintf("COLUMNS=%d", term.ctx.Window().Width))
				filter.Env = append(filter.Env, fmt.Sprintf("LINES=%d", term.ctx.Window().Height))
			}
			pv.attemptCopy()
		}
	}

	return pv, nil
}

func (pv *PartViewer) SetSource(reader io.Reader) {
	pv.source = reader
	switch pv.inlineImg {
	case true:
		pv.decodeImage()
	default:
		pv.attemptCopy()
	}
}

func (pv *PartViewer) decodeImage() {
	atomic.StoreInt32(&pv.copying, copying)
	go func() {
		defer log.PanicHandler()
		defer pv.Invalidate()
		defer atomic.StoreInt32(&pv.copying, 0)
		img, _, err := image.Decode(pv.source)
		if err != nil {
			log.Errorf("error decoding image: %v", err)
			return
		}
		pv.image = img
	}()
}

func (pv *PartViewer) attemptCopy() {
	if pv.source == nil ||
		pv.filter == nil ||
		atomic.LoadInt32(&pv.copying) == copying {
		return
	}
	atomic.StoreInt32(&pv.copying, copying)
	pv.writeMailHeaders()
	if strings.EqualFold(pv.part.MIMEType, "text") {
		pv.source = parse.StripAnsi(pv.hyperlinks(pv.source))
	}
	pv.filter.Stdin = pv.source
	pv.filter.Stdout = pv.pagerin
	pv.filter.Stderr = pv.pagerin
	err := pv.filter.Start()
	if err != nil {
		log.Errorf("error running filter: %v", err)
		return
	}
	go func() {
		defer log.PanicHandler()
		defer atomic.StoreInt32(&pv.copying, 0)
		err = pv.filter.Wait()
		if err != nil {
			log.Errorf("error waiting for filter: %v", err)
			return
		}
		err = pv.pagerin.Close()
		if err != nil {
			log.Errorf("error closing pager pipe: %v", err)
			return
		}
	}()
}

func (pv *PartViewer) writeMailHeaders() {
	info := pv.msg.MessageInfo()
	if config.Viewer.ShowHeaders && info.RFC822Headers != nil {
		var file io.WriteCloser

		for _, f := range config.Filters {
			if f.Type != config.FILTER_HEADERS {
				continue
			}
			log.Debugf("<%s> piping headers in filter: %s",
				info.Envelope.MessageId, f.Command)
			filter := exec.Command("sh", "-c", f.Command)
			if pv.filter != nil {
				// inherit from filter env
				filter.Env = pv.filter.Env
			}

			stdin, err := filter.StdinPipe()
			if err == nil {
				filter.Stdout = pv.pagerin
				filter.Stderr = pv.pagerin
				err := filter.Start()
				if err == nil {
					//nolint:errcheck // who cares?
					defer filter.Wait()
					file = stdin
				} else {
					log.Errorf(
						"failed to start header filter: %v",
						err)
				}
			} else {
				log.Errorf("failed to create pipe: %v", err)
			}
			break
		}
		if file == nil {
			file = pv.pagerin
		} else {
			defer file.Close()
		}

		var buf bytes.Buffer
		err := textproto.WriteHeader(&buf, info.RFC822Headers.Header.Header)
		if err != nil {
			log.Errorf("failed to format headers: %v", err)
		}
		_, err = file.Write(bytes.TrimRight(buf.Bytes(), "\r\n"))
		if err != nil {
			log.Errorf("failed to write headers: %v", err)
		}

		// virtual header
		if len(info.Labels) != 0 {
			labels := fmtHeader(info, "Labels", "", "", "", "")
			_, err := file.Write([]byte(fmt.Sprintf("\r\nLabels: %s", labels)))
			if err != nil {
				log.Errorf("failed to write to labels: %v", err)
			}
		}
		_, err = file.Write([]byte{'\r', '\n', '\r', '\n'})
		if err != nil {
			log.Errorf("failed to write empty line: %v", err)
		}
	}
}

func (pv *PartViewer) hyperlinks(r io.Reader) (reader io.Reader) {
	if !config.Viewer.ParseHttpLinks {
		return r
	}
	reader, pv.links = parse.HttpLinks(r)
	return reader
}

var noFilterConfiguredCommands = [][]string{
	{":open<enter>", "Open using the system handler"},
	{":save<space>", "Save to file"},
	{":pipe<space>", "Pipe to shell command"},
}

func newNoFilterConfigured(account string, part *models.BodyStructure) *ui.Grid {
	bindings := config.Binds.MessageView.ForAccount(account)

	var actions []string

	configured := noFilterConfiguredCommands
	if strings.Contains(strings.ToLower(part.MIMEType), "message") {
		configured = append(configured, []string{
			":eml<Enter>", "View message attachment",
		})
	}

	for _, command := range configured {
		cmd := command[0]
		name := command[1]
		strokes, _ := config.ParseKeyStrokes(cmd)
		var inputs []string
		for _, input := range bindings.GetReverseBindings(strokes) {
			inputs = append(inputs, config.FormatKeyStrokes(input))
		}
		actions = append(actions, fmt.Sprintf("  %-6s  %-29s  %s",
			strings.Join(inputs, ", "), name, cmd))
	}

	spec := []ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(2)},
	}
	for i := 0; i < len(actions)-1; i++ {
		spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
	}
	// make the last element fill remaining space
	spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)})

	grid := ui.NewGrid().Rows(spec).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	uiConfig := config.Ui.ForAccount(account)

	noFilter := fmt.Sprintf(`No filter configured for this mimetype ('%s')
What would you like to do?`, part.FullMIMEType())
	grid.AddChild(ui.NewText(noFilter,
		uiConfig.GetStyle(config.STYLE_TITLE))).At(0, 0)
	for i, action := range actions {
		grid.AddChild(ui.NewText(action,
			uiConfig.GetStyle(config.STYLE_DEFAULT))).At(i+1, 0)
	}

	return grid
}

func (pv *PartViewer) Invalidate() {
	ui.Invalidate()
}

func (pv *PartViewer) Draw(ctx *ui.Context) {
	style := pv.uiConfig.GetStyle(config.STYLE_DEFAULT)
	switch {
	case pv.filter == nil && canInline(pv.part.FullMIMEType()) && pv.err == nil:
		pv.inlineImg = true
	case pv.filter == nil:
		// No filter, can't inline, and/or we attempted to inline an image
		// and resulted in an error (maybe because of a bad encoding or
		// the terminal doesn't support any graphics protocol).
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
		pv.noFilter.Draw(ctx)
		return
	}
	if !pv.fetched {
		pv.msg.FetchBodyPart(pv.index, pv.SetSource)
		pv.fetched = true
	}
	if pv.err != nil {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
		ctx.Printf(0, 0, style, "%s", pv.err.Error())
		return
	}
	if pv.term != nil {
		pv.term.Draw(ctx)
	}
	if pv.image != nil && (pv.resized(ctx) || pv.graphic == nil) {
		// This path should only occur on resizes or the first pass
		// after the image is downloaded and could be slow due to
		// encoding the image to either sixel or uploading via the kitty
		// protocol. Generally it's pretty fast since we will only ever
		// be downsizing images
		vx := ctx.Window().Vx
		if pv.graphic == nil {
			var err error
			pv.graphic, err = vx.NewImage(pv.image)
			if err != nil {
				log.Errorf("Couldn't create image: %v", err)
				return
			}
		}
		pv.graphic.Resize(pv.width, pv.height)
	}
	if pv.graphic != nil {
		w, h := pv.graphic.CellSize()
		win := align.Center(ctx.Window(), w, h)
		pv.graphic.Draw(win)
	}
}

func (pv *PartViewer) Cleanup() {
	if pv.term != nil {
		pv.term.Close()
	}
	if pv.graphic != nil {
		pv.graphic.Destroy()
	}
}

func (pv *PartViewer) resized(ctx *ui.Context) bool {
	w := ctx.Width()
	h := ctx.Height()
	if pv.width != w || pv.height != h {
		pv.width = w
		pv.height = h
		return true
	}
	return false
}

func (pv *PartViewer) Event(event vaxis.Event) bool {
	if pv.term != nil {
		return pv.term.Event(event)
	}
	return false
}

type HeaderView struct {
	Name       string
	Value      string
	ValueField ui.Drawable
	uiConfig   *config.UIConfig
}

func (hv *HeaderView) Draw(ctx *ui.Context) {
	name := hv.Name
	size := runewidth.StringWidth(name + ":")
	lim := ctx.Width() - size - 1
	if lim <= 0 || ctx.Height() <= 0 {
		return
	}
	value := runewidth.Truncate(" "+hv.Value, lim, "…")

	vstyle := hv.uiConfig.GetStyle(config.STYLE_DEFAULT)
	hstyle := hv.uiConfig.GetStyle(config.STYLE_HEADER)

	// TODO: Make this more robust and less dumb
	if hv.Name == "PGP" {
		vstyle = hv.uiConfig.GetStyle(config.STYLE_SUCCESS)
	}

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', vstyle)
	ctx.Printf(0, 0, hstyle, "%s:", name)
	if hv.ValueField == nil {
		ctx.Printf(size, 0, vstyle, "%s", value)
	} else {
		hv.ValueField.Draw(ctx.Subcontext(size, 0, lim, 1))
	}
}

func (hv *HeaderView) Invalidate() {
	ui.Invalidate()
}

func canInline(mime string) bool {
	for _, ext := range supportedImageTypes {
		if mime == ext {
			return true
		}
	}
	return false
}
