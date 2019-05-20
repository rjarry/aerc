package widgets

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/danwakefield/fnmatch"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell"
	"github.com/google/shlex"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type MessageViewer struct {
	ui.Invalidatable
	conf     *config.AercConfig
	err      error
	grid     *ui.Grid
	msg      *types.MessageInfo
	switcher *PartSwitcher
	store    *lib.MessageStore
}

type PartSwitcher struct {
	ui.Invalidatable
	parts    []*PartViewer
	selected int
}

func formatAddresses(addrs []*imap.Address) string {
	val := bytes.Buffer{}
	for i, addr := range addrs {
		if addr.PersonalName != "" {
			val.WriteString(fmt.Sprintf("%s <%s@%s>",
				addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			val.WriteString(fmt.Sprintf("%s@%s",
				addr.MailboxName, addr.HostName))
		}
		if i != len(addrs)-1 {
			val.WriteString(", ")
		}
	}
	return val.String()
}

func NewMessageViewer(conf *config.AercConfig,
	store *lib.MessageStore, msg *types.MessageInfo) *MessageViewer {

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 3}, // TODO: Based on number of header rows
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	// TODO: let user specify additional headers to show by default
	headers := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 1},
		{ui.SIZE_EXACT, 1},
		{ui.SIZE_EXACT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_WEIGHT, 1},
	})
	headers.AddChild(
		&HeaderView{
			Name:  "From",
			Value: formatAddresses(msg.Envelope.From),
		}).At(0, 0)
	headers.AddChild(
		&HeaderView{
			Name:  "To",
			Value: formatAddresses(msg.Envelope.To),
		}).At(0, 1)
	headers.AddChild(
		&HeaderView{
			Name:  "Subject",
			Value: msg.Envelope.Subject,
		}).At(1, 0).Span(1, 2)
	headers.AddChild(ui.NewFill(' ')).At(2, 0).Span(1, 2)

	body := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_EXACT, 20},
	})

	var (
		err error
		mv  *MessageViewer
	)

	switcher := &PartSwitcher{}
	if len(msg.BodyStructure.Parts) == 0 {
		pv, err := NewPartViewer(conf, store, msg, msg.BodyStructure, []int{1})
		if err != nil {
			goto handle_error
		}
		switcher.parts = []*PartViewer{pv}
		pv.OnInvalidate(func(_ ui.Drawable) {
			switcher.Invalidate()
		})
	} else {
		switcher.parts, err = enumerateParts(conf, store,
			msg, msg.BodyStructure, []int{})
		if err != nil {
			goto handle_error
		}
		switcher.selected = -1
		for i, pv := range switcher.parts {
			pv.OnInvalidate(func(_ ui.Drawable) {
				switcher.Invalidate()
			})
			// TODO: switch to user's preferred mimetype, if configured
			if switcher.selected == -1 && pv.part.MIMEType != "multipart" {
				switcher.selected = i
			}
		}
	}

	grid.AddChild(headers).At(0, 0)
	grid.AddChild(body).At(1, 0)

	mv = &MessageViewer{
		grid:     grid,
		msg:      msg,
		store:    store,
		switcher: switcher,
	}

	body.AddChild(mv.switcher).At(0, 0).Span(1, 2)
	return mv

handle_error:
	return &MessageViewer{
		err:  err,
		grid: grid,
		msg:  msg,
	}
}

func enumerateParts(conf *config.AercConfig, store *lib.MessageStore,
	msg *types.MessageInfo, body *imap.BodyStructure,
	index []int) ([]*PartViewer, error) {

	var parts []*PartViewer
	for i, part := range body.Parts {
		curindex := append(index, i+1)
		if part.MIMEType == "multipart" {
			// Multipart meta-parts are faked
			pv := &PartViewer{part: part}
			parts = append(parts, pv)
			subParts, err := enumerateParts(
				conf, store, msg, part, curindex)
			if err != nil {
				return nil, err
			}
			parts = append(parts, subParts...)
			continue
		}
		pv, err := NewPartViewer(conf, store, msg, part, curindex)
		if err != nil {
			return nil, err
		}
		parts = append(parts, pv)
	}
	return parts, nil
}

func (mv *MessageViewer) Draw(ctx *ui.Context) {
	if mv.err != nil {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
		ctx.Printf(0, 0, tcell.StyleDefault, "%s", mv.err.Error())
		return
	}
	mv.grid.Draw(ctx)
}

func (mv *MessageViewer) Invalidate() {
	mv.grid.Invalidate()
}

func (mv *MessageViewer) OnInvalidate(fn func(d ui.Drawable)) {
	mv.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(mv)
	})
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

func (ps *PartSwitcher) Invalidate() {
	ps.DoInvalidate(ps)
}

func (ps *PartSwitcher) Focus(focus bool) {
	if ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.Focus(focus)
	}
}

func (ps *PartSwitcher) Event(event tcell.Event) bool {
	if ps.parts[ps.selected].term != nil {
		return ps.parts[ps.selected].term.Event(event)
	}
	return false
}

func (ps *PartSwitcher) Draw(ctx *ui.Context) {
	height := len(ps.parts)
	if height == 1 {
		ps.parts[ps.selected].Draw(ctx)
		return
	}
	// TODO: cap height and add scrolling for messages with many parts
	y := ctx.Height() - height
	for i, part := range ps.parts {
		style := tcell.StyleDefault.Reverse(ps.selected == i)
		ctx.Fill(0, y+i, ctx.Width(), 1, ' ', style)
		name := fmt.Sprintf("%s/%s",
			strings.ToLower(part.part.MIMEType),
			strings.ToLower(part.part.MIMESubType))
		if filename, ok := part.part.DispositionParams["filename"]; ok {
			name += fmt.Sprintf(" (%s)", filename)
		}
		ctx.Printf(len(part.index)*2, y+i, style, "%s", name)
	}
	ps.parts[ps.selected].Draw(ctx.Subcontext(
		0, 0, ctx.Width(), ctx.Height()-height))
}

func (mv *MessageViewer) Event(event tcell.Event) bool {
	return mv.switcher.Event(event)
}

func (mv *MessageViewer) Focus(focus bool) {
	mv.switcher.Focus(focus)
}

type PartViewer struct {
	ui.Invalidatable
	err     error
	fetched bool
	filter  *exec.Cmd
	index   []int
	msg     *types.MessageInfo
	pager   *exec.Cmd
	pagerin io.WriteCloser
	part    *imap.BodyStructure
	sink    io.WriteCloser
	source  io.Reader
	store   *lib.MessageStore
	term    *Terminal
}

func NewPartViewer(conf *config.AercConfig,
	store *lib.MessageStore, msg *types.MessageInfo,
	part *imap.BodyStructure, index []int) (*PartViewer, error) {

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
				header = msg.Envelope.Subject
			case "from":
				header = formatAddresses(msg.Envelope.From)
			case "to":
				header = formatAddresses(msg.Envelope.To)
			case "cc":
				header = formatAddresses(msg.Envelope.Cc)
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

	pv := &PartViewer{
		filter:  filter,
		index:   index,
		msg:     msg,
		pager:   pager,
		pagerin: pagerin,
		part:    part,
		sink:    pipe,
		store:   store,
		term:    term,
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

func (pv *PartViewer) attemptCopy() {
	if pv.source != nil && pv.pager.Process != nil {
		header := message.Header{}
		header.SetText("Content-Transfer-Encoding", pv.part.Encoding)
		header.SetContentType(pv.part.MIMEType, pv.part.Params)
		header.SetText("Content-Description", pv.part.Description)
		if pv.filter != nil {
			stdout, _ := pv.filter.StdoutPipe()
			pv.filter.Start()
			go func() {
				_, err := io.Copy(pv.pagerin, stdout)
				if err != nil {
					pv.err = err
					pv.Invalidate()
				}
				pv.pagerin.Close()
				stdout.Close()
			}()
		}
		go func() {
			entity, err := message.New(header, pv.source)
			if err != nil {
				pv.err = err
				pv.Invalidate()
				return
			}
			reader := mail.NewReader(entity)
			part, err := reader.NextPart()
			if err != nil {
				pv.err = err
				pv.Invalidate()
				return
			}
			io.Copy(pv.sink, part.Body)
			pv.sink.Close()
		}()
	}
}

func (pv *PartViewer) Invalidate() {
	pv.DoInvalidate(pv)
}

func (pv *PartViewer) Draw(ctx *ui.Context) {
	if pv.filter == nil {
		// TODO: Let them download it directly or something
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
		ctx.Printf(0, 0, tcell.StyleDefault,
			"No filter configured for this mimetype")
		return
	}
	if !pv.fetched {
		pv.store.FetchBodyPart(pv.msg.Uid, pv.index, pv.SetSource)
		pv.fetched = true
	}
	if pv.err != nil {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
		ctx.Printf(0, 0, tcell.StyleDefault, "%s", pv.err.Error())
		return
	}
	pv.term.Draw(ctx)
}

type HeaderView struct {
	ui.Invalidatable
	Name  string
	Value string
}

func (hv *HeaderView) Draw(ctx *ui.Context) {
	name := hv.Name
	size := runewidth.StringWidth(name)
	lim := ctx.Width() - size - 1
	value := runewidth.Truncate(" "+hv.Value, lim, "…")
	var (
		hstyle tcell.Style
		vstyle tcell.Style
	)
	// TODO: Make this more robust and less dumb
	if hv.Name == "PGP" {
		vstyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		hstyle = tcell.StyleDefault.Bold(true)
	} else {
		vstyle = tcell.StyleDefault
		hstyle = tcell.StyleDefault.Bold(true)
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', vstyle)
	ctx.Printf(0, 0, hstyle, name)
	ctx.Printf(size, 0, vstyle, value)
}

func (hv *HeaderView) Invalidate() {
	hv.DoInvalidate(hv)
}
