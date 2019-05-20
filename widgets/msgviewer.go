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
	conf     *config.AercConfig
	err      error
	msg      *types.MessageInfo
	grid     *ui.Grid
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

func NewMessageViewer(conf *config.AercConfig, store *lib.MessageStore,
	msg *types.MessageInfo) *MessageViewer {

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

	for i, part := range msg.BodyStructure.Parts {
		fmt.Println(i, part.MIMEType, part.MIMESubType)
	}

	// TODO: add multipart switcher and configure additional parts
	pv, err := NewPartViewer(conf, msg, 0)
	if err != nil {
		goto handle_error
	}
	body.AddChild(pv).At(0, 0).Span(1, 2)

	grid.AddChild(headers).At(0, 0)
	grid.AddChild(body).At(1, 0)

	store.FetchBodyPart(msg.Uid, 0, pv.SetSource)

	return &MessageViewer{
		grid:  grid,
		msg:   msg,
		parts: []*PartViewer{pv},
	}

handle_error:
	return &MessageViewer{
		err:  err,
		grid: grid,
		msg:  msg,
	}
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

func (mv *MessageViewer) Event(event tcell.Event) bool {
	// What is encapsulation even
	if mv.parts[mv.selected].term != nil {
		return mv.parts[mv.selected].term.Event(event)
	}
	return false
}

func (mv *MessageViewer) Focus(focus bool) {
	if mv.parts[mv.selected].term != nil {
		mv.parts[mv.selected].term.Focus(focus)
	}
}

type PartViewer struct {
	err     error
	filter  *exec.Cmd
	index   string
	msg     *types.MessageInfo
	pager   *exec.Cmd
	pagerin io.WriteCloser
	part    *imap.BodyStructure
	sink    io.WriteCloser
	source  io.Reader
	term    *Terminal
}

func NewPartViewer(conf *config.AercConfig,
	msg *types.MessageInfo, index int) (*PartViewer, error) {
	var (
		part *imap.BodyStructure
	)
	// TODO: Find IMAP index, which may differ
	if len(msg.BodyStructure.Parts) != 0 {
		part = msg.BodyStructure.Parts[index]
	} else {
		part = msg.BodyStructure
	}

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
	} else {
		if pipe, err = pager.StdinPipe(); err != nil {
			return nil, err
		}
	}
	if term, err = NewTerminal(pager); err != nil {
		return nil, err
	}

	pv := &PartViewer{
		filter:  filter,
		pager:   pager,
		pagerin: pagerin,
		part:    part,
		sink:    pipe,
		term:    term,
	}

	term.OnStart = func() {
		pv.attemptCopy()
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

func (pv *PartViewer) OnInvalidate(fn func(ui.Drawable)) {
	pv.term.OnInvalidate(func(_ ui.Drawable) {
		fn(pv)
	})
}

func (pv *PartViewer) Invalidate() {
	pv.term.Invalidate()
}

func (pv *PartViewer) Draw(ctx *ui.Context) {
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

type MultipartView struct {
	ui.Invalidatable
}

func (mpv *MultipartView) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	ctx.Fill(0, 0, ctx.Width(), 1, ' ', tcell.StyleDefault.Reverse(true))
	ctx.Printf(0, 0, tcell.StyleDefault.Reverse(true), "text/plain")
	ctx.Printf(0, 1, tcell.StyleDefault, "text/html")
	ctx.Printf(0, 2, tcell.StyleDefault, "application/pgp-si…")
}

func (mpv *MultipartView) Invalidate() {
	mpv.DoInvalidate(mpv)
}
