package widgets

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/danwakefield/fnmatch"
	"github.com/gdamore/tcell/v2"
	"github.com/google/shlex"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/auth"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
)

var ansi = regexp.MustCompile("\x1B\\[[0-?]*[ -/]*[@-~]")

var _ ProvidesMessages = (*MessageViewer)(nil)

type MessageViewer struct {
	ui.Invalidatable
	acct     *AccountView
	conf     *config.AercConfig
	err      error
	grid     *ui.Grid
	switcher *PartSwitcher
	msg      lib.MessageView
	uiConfig *config.UIConfig
}

type PartSwitcher struct {
	ui.Invalidatable
	parts          []*PartViewer
	selected       int
	showHeaders    bool
	alwaysShowMime bool

	height int
	mv     *MessageViewer
}

func NewMessageViewer(acct *AccountView,
	conf *config.AercConfig, msg lib.MessageView,
) *MessageViewer {
	hf := HeaderLayoutFilter{
		layout: HeaderLayout(conf.Viewer.HeaderLayout),
		keep: func(msg *models.MessageInfo, header string) bool {
			return fmtHeader(msg, header, "2") != ""
		},
	}
	layout := hf.forMessage(msg.MessageInfo())
	header, headerHeight := layout.grid(
		func(header string) ui.Drawable {
			hv := &HeaderView{
				conf: conf,
				Name: header,
				Value: fmtHeader(msg.MessageInfo(), header,
					acct.UiConfig().TimestampFormat),
				uiConfig: acct.UiConfig(),
			}
			showInfo := false
			if i := strings.IndexRune(header, '+'); i > 0 {
				header = header[:i]
				hv.Name = header
				showInfo = true
			}
			if parser := auth.New(header); parser != nil {
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

	if msg.MessageDetails() != nil || conf.Ui.IconUnencrypted != "" {
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
	err := createSwitcher(acct, switcher, conf, msg)
	if err != nil {
		return &MessageViewer{
			err:      err,
			grid:     grid,
			msg:      msg,
			uiConfig: acct.UiConfig(),
		}
	}

	borderStyle := acct.UiConfig().GetStyle(config.STYLE_BORDER)
	borderChar := acct.UiConfig().BorderCharHorizontal

	grid.AddChild(header).At(0, 0)
	if msg.MessageDetails() != nil || conf.Ui.IconUnencrypted != "" {
		grid.AddChild(NewPGPInfo(msg.MessageDetails(), acct.UiConfig())).At(1, 0)
		grid.AddChild(ui.NewFill(borderChar, borderStyle)).At(2, 0)
		grid.AddChild(switcher).At(3, 0)
	} else {
		grid.AddChild(ui.NewFill(borderChar, borderStyle)).At(1, 0)
		grid.AddChild(switcher).At(2, 0)
	}

	mv := &MessageViewer{
		acct:     acct,
		conf:     conf,
		grid:     grid,
		msg:      msg,
		switcher: switcher,
		uiConfig: acct.UiConfig(),
	}
	switcher.mv = mv

	return mv
}

func fmtHeader(msg *models.MessageInfo, header string, timefmt string) string {
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
		return msg.Envelope.Date.Local().Format(timefmt)
	case "Subject":
		return msg.Envelope.Subject
	case "Labels":
		return strings.Join(msg.Labels, ", ")
	default:
		return msg.RFC822Headers.Get(header)
	}
}

func enumerateParts(acct *AccountView, conf *config.AercConfig,
	msg lib.MessageView, body *models.BodyStructure,
	index []int,
) ([]*PartViewer, error) {
	var parts []*PartViewer
	for i, part := range body.Parts {
		curindex := append(index, i+1) //nolint:gocritic // intentional append to different slice
		if part.MIMEType == "multipart" {
			// Multipart meta-parts are faked
			pv := &PartViewer{part: part}
			parts = append(parts, pv)
			subParts, err := enumerateParts(
				acct, conf, msg, part, curindex)
			if err != nil {
				return nil, err
			}
			parts = append(parts, subParts...)
			continue
		}
		pv, err := NewPartViewer(acct, conf, msg, part, curindex)
		if err != nil {
			return nil, err
		}
		parts = append(parts, pv)
	}
	return parts, nil
}

func createSwitcher(acct *AccountView, switcher *PartSwitcher,
	conf *config.AercConfig, msg lib.MessageView,
) error {
	var err error
	switcher.selected = -1
	switcher.showHeaders = conf.Viewer.ShowHeaders
	switcher.alwaysShowMime = conf.Viewer.AlwaysShowMime

	if len(msg.BodyStructure().Parts) == 0 {
		switcher.selected = 0
		pv, err := NewPartViewer(acct, conf, msg, msg.BodyStructure(), nil)
		if err != nil {
			return err
		}
		switcher.parts = []*PartViewer{pv}
		pv.OnInvalidate(func(_ ui.Drawable) {
			switcher.Invalidate()
		})
	} else {
		switcher.parts, err = enumerateParts(acct, conf, msg,
			msg.BodyStructure(), []int{})
		if err != nil {
			return err
		}
		selectedPriority := -1
		logging.Infof("Selecting best message from %v", conf.Viewer.Alternatives)
		for i, pv := range switcher.parts {
			pv.OnInvalidate(func(_ ui.Drawable) {
				switcher.Invalidate()
			})
			// Switch to user's preferred mimetype
			if switcher.selected == -1 && pv.part.MIMEType != "multipart" {
				switcher.selected = i
			}
			mime := strings.ToLower(pv.part.MIMEType) +
				"/" + strings.ToLower(pv.part.MIMESubType)
			for idx, m := range conf.Viewer.Alternatives {
				if m != mime {
					continue
				}
				priority := len(conf.Viewer.Alternatives) - idx
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
	if mv.err != nil {
		style := mv.acct.UiConfig().GetStyle(config.STYLE_DEFAULT)
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
		ctx.Printf(0, 0, style, "%s", mv.err.Error())
		return
	}
	mv.grid.Draw(ctx)
}

func (mv *MessageViewer) MouseEvent(localX int, localY int, event tcell.Event) {
	if mv.err != nil {
		return
	}
	mv.grid.MouseEvent(localX, localY, event)
}

func (mv *MessageViewer) Invalidate() {
	mv.grid.Invalidate()
}

func (mv *MessageViewer) OnInvalidate(fn func(d ui.Drawable)) {
	mv.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(mv)
	})
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

func (mv *MessageViewer) MarkedMessages() ([]uint32, error) {
	return mv.acct.MarkedMessages()
}

func (mv *MessageViewer) ToggleHeaders() {
	switcher := mv.switcher
	switcher.Cleanup()
	mv.conf.Viewer.ShowHeaders = !mv.conf.Viewer.ShowHeaders
	err := createSwitcher(mv.acct, switcher, mv.conf, mv.msg)
	if err != nil {
		logging.Errorf("cannot create switcher: %v", err)
	}
	switcher.Invalidate()
}

func (mv *MessageViewer) ToggleKeyPassthrough() bool {
	mv.conf.Viewer.KeyPassthrough = !mv.conf.Viewer.KeyPassthrough
	return mv.conf.Viewer.KeyPassthrough
}

func (mv *MessageViewer) SelectedMessagePart() *PartInfo {
	switcher := mv.switcher
	part := switcher.parts[switcher.selected]

	return &PartInfo{
		Index: part.index,
		Msg:   part.msg.MessageInfo(),
		Part:  part.part,
		Links: part.links,
	}
}

func (mv *MessageViewer) AttachmentParts() []*PartInfo {
	var attachments []*PartInfo

	for _, p := range mv.switcher.parts {
		if p.part.Disposition == "attachment" {
			pi := &PartInfo{
				Index: p.index,
				Msg:   p.msg.MessageInfo(),
				Part:  p.part,
			}
			attachments = append(attachments, pi)
		}
	}

	return attachments
}

func (mv *MessageViewer) PreviousPart() {
	switcher := mv.switcher
	for {
		switcher.selected--
		if switcher.selected < 0 {
			switcher.selected = len(switcher.parts) - 1
		}
		if switcher.parts[switcher.selected].part.MIMEType != "multipart" {
			break
		}
	}
	mv.Invalidate()
}

func (mv *MessageViewer) NextPart() {
	switcher := mv.switcher
	for {
		switcher.selected++
		if switcher.selected >= len(switcher.parts) {
			switcher.selected = 0
		}
		if switcher.parts[switcher.selected].part.MIMEType != "multipart" {
			break
		}
	}
	mv.Invalidate()
}

func (mv *MessageViewer) Bindings() string {
	if mv.conf.Viewer.KeyPassthrough {
		return "view::passthrough"
	} else {
		return "view"
	}
}

func (mv *MessageViewer) Close() error {
	mv.switcher.Cleanup()
	return nil
}

func (mv *MessageViewer) UpdateScreen() {
	if mv.switcher == nil {
		return
	}
	parts := mv.switcher.parts
	selected := mv.switcher.selected
	if selected < 0 {
		return
	}
	if len(parts) > 0 && selected < len(parts) {
		if part := parts[selected]; part != nil {
			part.UpdateScreen()
		}
	}
}

func (ps *PartSwitcher) Invalidate() {
	ps.DoInvalidate(ps)
}

func (ps *PartSwitcher) Focus(focus bool) {
	if ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.Focus(focus)
	}
}

func (ps *PartSwitcher) Event(event tcell.Event) bool {
	return ps.parts[ps.selected].Event(event)
}

func (ps *PartSwitcher) Draw(ctx *ui.Context) {
	height := len(ps.parts)
	if height == 1 && !ps.alwaysShowMime {
		ps.parts[ps.selected].Draw(ctx)
		return
	}
	// TODO: cap height and add scrolling for messages with many parts
	ps.height = ctx.Height()
	y := ctx.Height() - height
	for i, part := range ps.parts {
		style := ps.mv.uiConfig.GetStyle(config.STYLE_DEFAULT)
		if ps.selected == i {
			style = ps.mv.uiConfig.GetStyleSelected(config.STYLE_DEFAULT)
		}
		ctx.Fill(0, y+i, ctx.Width(), 1, ' ', style)
		name := fmt.Sprintf("%s/%s",
			strings.ToLower(part.part.MIMEType),
			strings.ToLower(part.part.MIMESubType))
		if filename, ok := part.part.DispositionParams["filename"]; ok {
			name += fmt.Sprintf(" (%s)", filename)
		} else if filename, ok := part.part.Params["name"]; ok {
			// workaround golang not supporting RFC2231 besides ASCII and UTF8
			name += fmt.Sprintf(" (%s)", filename)
		}
		ctx.Printf(len(part.index)*2, y+i, style, "%s", name)
	}
	ps.parts[ps.selected].Draw(ctx.Subcontext(
		0, 0, ctx.Width(), ctx.Height()-height))
}

func (ps *PartSwitcher) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		switch event.Buttons() {
		case tcell.Button1:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y && ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			for i := range ps.parts {
				if localY != y+i {
					continue
				}
				if ps.parts[i].part.MIMEType == "multipart" {
					continue
				}
				if ps.parts[ps.selected].term != nil {
					ps.parts[ps.selected].term.Focus(false)
				}
				ps.selected = i
				ps.Invalidate()
				if ps.parts[ps.selected].term != nil {
					ps.parts[ps.selected].term.Focus(true)
				}
			}
		case tcell.WheelDown:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y && ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(false)
			}
			ps.mv.NextPart()
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(true)
			}
		case tcell.WheelUp:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y && ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(false)
			}
			ps.mv.PreviousPart()
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(true)
			}
		}
	}
}

func (ps *PartSwitcher) Cleanup() {
	for _, partViewer := range ps.parts {
		partViewer.Cleanup()
	}
}

func (mv *MessageViewer) Event(event tcell.Event) bool {
	return mv.switcher.Event(event)
}

func (mv *MessageViewer) Focus(focus bool) {
	mv.switcher.Focus(focus)
}

type PartViewer struct {
	ui.Invalidatable
	conf        *config.AercConfig
	acctConfig  *config.AccountConfig
	err         error
	fetched     bool
	filter      *exec.Cmd
	index       []int
	msg         lib.MessageView
	pager       *exec.Cmd
	pagerin     io.WriteCloser
	part        *models.BodyStructure
	showHeaders bool
	sink        io.WriteCloser
	source      io.Reader
	term        *Terminal
	grid        *ui.Grid
	uiConfig    *config.UIConfig

	links []string
}

func NewPartViewer(acct *AccountView, conf *config.AercConfig,
	msg lib.MessageView, part *models.BodyStructure,
	index []int,
) (*PartViewer, error) {
	var (
		filter  *exec.Cmd
		pager   *exec.Cmd
		pipe    io.WriteCloser
		pagerin io.WriteCloser
		term    *Terminal
	)
	cmd, err := shlex.Split(conf.Viewer.Pager)
	if err != nil {
		return nil, err
	}

	pager = exec.Command(cmd[0], cmd[1:]...)

	info := msg.MessageInfo()
	for _, f := range conf.Filters {
		mime := strings.ToLower(part.MIMEType) +
			"/" + strings.ToLower(part.MIMESubType)
		switch f.FilterType {
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
		}
		if filter != nil {
			break
		}
	}
	if filter != nil {
		if pipe, err = filter.StdinPipe(); err != nil {
			return nil, err
		}
		if pagerin, _ = pager.StdinPipe(); err != nil {
			return nil, err
		}
		if term, err = NewTerminal(pager); err != nil {
			return nil, err
		}
	}

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(3)}, // Message
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	pv := &PartViewer{
		conf:        conf,
		acctConfig:  acct.AccountConfig(),
		filter:      filter,
		index:       index,
		msg:         msg,
		pager:       pager,
		pagerin:     pagerin,
		part:        part,
		showHeaders: conf.Viewer.ShowHeaders,
		sink:        pipe,
		term:        term,
		grid:        grid,
		uiConfig:    acct.UiConfig(),
	}

	if term != nil {
		term.OnStart = func() {
			pv.attemptCopy()
		}
		term.OnInvalidate(func(_ ui.Drawable) {
			pv.Invalidate()
		})
	}

	return pv, nil
}

func (pv *PartViewer) SetSource(reader io.Reader) {
	pv.source = reader
	pv.attemptCopy()
}

func (pv *PartViewer) UpdateScreen() {
	if pv.term != nil {
		pv.term.Invalidate()
	}
}

func (pv *PartViewer) attemptCopy() {
	if pv.source == nil || pv.pager == nil || pv.pager.Process == nil {
		return
	}
	if pv.filter != nil {
		pv.copyFilterOutToPager() // delayed until we write to the sink
	}
	go func() {
		defer logging.PanicHandler()

		pv.writeMailHeaders()
		if strings.EqualFold(pv.part.MIMEType, "text") {
			// if the content is plain we can strip ansi control chars
			pv.copySourceToSinkStripAnsi()
		} else {
			// if it's binary we have to rely on the filter to be sane
			_, err := io.Copy(pv.sink, pv.source)
			if err != nil {
				logging.Warnf("failed to copy: %w", err)
			}
		}
		pv.sink.Close()
	}()
}

func (pv *PartViewer) writeMailHeaders() {
	info := pv.msg.MessageInfo()
	if pv.showHeaders && info.RFC822Headers != nil {
		// header need to bypass the filter, else we run into issues
		// with the filter messing with newlines etc.
		// hence all writes in this block go directly to the pager
		fields := info.RFC822Headers.Fields()
		for fields.Next() {
			var value string
			var err error
			if value, err = fields.Text(); err != nil {
				// better than nothing, use the non decoded version
				value = fields.Value()
			}
			field := fmt.Sprintf(
				"%s: %s\n", fields.Key(), value)
			_, err = pv.pagerin.Write([]byte(field))
			if err != nil {
				logging.Errorf("failed to write to stdin of pager: %v", err)
			}
		}
		// virtual header
		if len(info.Labels) != 0 {
			labels := fmtHeader(info, "Labels", "")
			_, err := pv.pagerin.Write([]byte(fmt.Sprintf("Labels: %s\n", labels)))
			if err != nil {
				logging.Errorf("failed to write to stdin of pager: %v", err)
			}
		}
		_, err := pv.pagerin.Write([]byte{'\n'})
		if err != nil {
			logging.Errorf("failed to write to stdin of pager: %v", err)
		}
	}
}

func (pv *PartViewer) hyperlinks(r io.Reader) (reader io.Reader) {
	if !pv.conf.Viewer.ParseHttpLinks {
		return r
	}
	reader, pv.links = parse.HttpLinks(r)
	return reader
}

func (pv *PartViewer) copyFilterOutToPager() {
	stdout, _ := pv.filter.StdoutPipe()
	stderr, _ := pv.filter.StderrPipe()
	err := pv.filter.Start()
	if err != nil {
		logging.Warnf("failed to start filter: %v", err)
	}
	ch := make(chan interface{})
	go func() {
		defer logging.PanicHandler()

		_, err := io.Copy(pv.pagerin, stdout)
		if err != nil {
			pv.err = err
			pv.Invalidate()
		}
		stdout.Close()
		ch <- nil
	}()
	go func() {
		defer logging.PanicHandler()

		_, err := io.Copy(pv.pagerin, stderr)
		if err != nil {
			pv.err = err
			pv.Invalidate()
		}
		stderr.Close()
		ch <- nil
	}()
	go func() {
		defer logging.PanicHandler()

		<-ch
		<-ch
		err := pv.filter.Wait()
		if err != nil {
			logging.Warnf("failed to wait for the filter process: %v", err)
		}
		pv.pagerin.Close()
		// If the pager command doesn't keep the terminal running, we
		// risk not drawing the screen until user input unless we
		// invalidate after writing
		pv.Invalidate()
	}()
}

func (pv *PartViewer) copySourceToSinkStripAnsi() {
	scanner := bufio.NewScanner(pv.hyperlinks(pv.source))
	// some people send around huge html without any newline in between
	// this did overflow the default 64KB buffer of bufio.Scanner.
	// If something can't fit in a GB there's no hope left
	scanner.Buffer(nil, 1024*1024*1024)
	for scanner.Scan() {
		text := scanner.Text()
		text = ansi.ReplaceAllString(text, "")
		_, err := io.WriteString(pv.sink, text+"\n")
		if err != nil {
			logging.Warnf("failed write ", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read line: %v\n", err)
	}
}

var noFilterConfiguredCommands = [][]string{
	{":open<enter>", "Open using the system handler"},
	{":save<space>", "Save to file"},
	{":pipe<space>", "Pipe to shell command"},
}

func newNoFilterConfigured(pv *PartViewer) *ui.Grid {
	bindings := pv.conf.MergeContextualBinds(
		pv.conf.Bindings.MessageView,
		config.BIND_CONTEXT_ACCOUNT,
		pv.acctConfig.Name,
		"view",
	)

	var actions []string

	for _, command := range noFilterConfiguredCommands {
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

	uiConfig := pv.conf.Ui

	noFilter := fmt.Sprintf(`No filter configured for this mimetype ('%s/%s')
What would you like to do?`, pv.part.MIMEType, pv.part.MIMESubType)
	grid.AddChild(ui.NewText(noFilter,
		uiConfig.GetStyle(config.STYLE_TITLE))).At(0, 0)
	for i, action := range actions {
		grid.AddChild(ui.NewText(action,
			uiConfig.GetStyle(config.STYLE_DEFAULT))).At(i+1, 0)
	}

	return grid
}

func (pv *PartViewer) Invalidate() {
	pv.DoInvalidate(pv)
}

func (pv *PartViewer) Draw(ctx *ui.Context) {
	style := pv.uiConfig.GetStyle(config.STYLE_DEFAULT)
	if pv.filter == nil {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
		newNoFilterConfigured(pv).Draw(ctx)
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
}

func (pv *PartViewer) Cleanup() {
	if pv.term != nil {
		pv.term.Close(nil)
		pv.term = nil
	}
}

func (pv *PartViewer) Event(event tcell.Event) bool {
	if pv.term != nil {
		return pv.term.Event(event)
	}
	return false
}

type HeaderView struct {
	ui.Invalidatable
	conf       *config.AercConfig
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
	hv.DoInvalidate(hv)
}
