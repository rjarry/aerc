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

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageViewer struct {
	conf    *config.AercConfig
	err     error
	filter  *exec.Cmd
	msg     *types.MessageInfo
	pager   *exec.Cmd
	source  io.Reader
	pagerin io.WriteCloser
	sink    io.WriteCloser
	grid    *ui.Grid
	term    *Terminal
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

	var (
		filter  *exec.Cmd
		pager   *exec.Cmd
		pipe    io.WriteCloser
		pagerin io.WriteCloser
		term    *Terminal
		viewer  *MessageViewer
	)
	cmd, err := shlex.Split(conf.Viewer.Pager)
	if err != nil {
		goto handle_error
	}
	pager = exec.Command(cmd[0], cmd[1:]...)

	for _, f := range conf.Filters {
		mime := strings.ToLower(msg.BodyStructure.MIMEType) +
			"/" + strings.ToLower(msg.BodyStructure.MIMESubType)
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
		pipe, _ = filter.StdinPipe()
		pagerin, _ = pager.StdinPipe()
	} else {
		pipe, _ = pager.StdinPipe()
	}

	term, _ = NewTerminal(pager)
	// TODO: configure multipart view. I left a spot for it in the grid
	body.AddChild(term).At(0, 0).Span(1, 2)

	grid.AddChild(headers).At(0, 0)
	grid.AddChild(body).At(1, 0)

	viewer = &MessageViewer{
		filter:  filter,
		grid:    grid,
		msg:     msg,
		pager:   pager,
		pagerin: pagerin,
		sink:    pipe,
		term:    term,
	}

	store.FetchBodyPart(msg.Uid, 0, func(reader io.Reader) {
		viewer.source = reader
		viewer.attemptCopy()
	})

	term.OnStart = func() {
		viewer.attemptCopy()
	}

	return viewer

handle_error:
	viewer = &MessageViewer{
		err:  err,
		grid: grid,
		msg:  msg,
	}
	return viewer
}

func (mv *MessageViewer) attemptCopy() {
	if mv.source != nil && mv.pager.Process != nil {
		header := message.Header{}
		header.SetText("Content-Transfer-Encoding",
			mv.msg.BodyStructure.Encoding)
		header.SetContentType(
			mv.msg.BodyStructure.MIMEType, mv.msg.BodyStructure.Params)
		header.SetText("Content-Description", mv.msg.BodyStructure.Description)
		if mv.filter != nil {
			stdout, _ := mv.filter.StdoutPipe()
			mv.filter.Start()
			go func() {
				_, err := io.Copy(mv.pagerin, stdout)
				if err != nil {
					mv.err = err
					mv.Invalidate()
				}
				mv.pagerin.Close()
				stdout.Close()
			}()
		}
		go func() {
			entity, err := message.New(header, mv.source)
			if err != nil {
				mv.err = err
				mv.Invalidate()
				return
			}
			reader := mail.NewReader(entity)
			part, err := reader.NextPart()
			if err != nil {
				mv.err = err
				mv.Invalidate()
				return
			}
			io.Copy(mv.sink, part.Body)
			mv.sink.Close()
		}()
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
	if mv.term != nil {
		return mv.term.Event(event)
	}
	return false
}

func (mv *MessageViewer) Focus(focus bool) {
	if mv.term != nil {
		mv.term.Focus(focus)
	}
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
