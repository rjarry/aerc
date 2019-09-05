package widgets

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/danwakefield/fnmatch"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell"
	"github.com/google/shlex"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/models"
)

var ansi = regexp.MustCompile("^\x1B\\[[0-?]*[ -/]*[@-~]")

type MessageViewer struct {
	ui.Invalidatable
	acct     *AccountView
	conf     *config.AercConfig
	err      error
	grid     *ui.Grid
	msg      *models.MessageInfo
	switcher *PartSwitcher
	store    *lib.MessageStore
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

func NewMessageViewer(acct *AccountView, conf *config.AercConfig,
	store *lib.MessageStore, msg *models.MessageInfo) *MessageViewer {

	layout := HeaderLayout(conf.Viewer.HeaderLayout).forMessage(msg)
	header, headerHeight := layout.grid(
		func(header string) ui.Drawable {
			return &HeaderView{
				Name:  header,
				Value: fmtHeader(msg, header),
			}
		},
	)

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, headerHeight},
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	switcher := &PartSwitcher{}
	err := createSwitcher(switcher, conf, store, msg)
	if err != nil {
		return &MessageViewer{
			err:  err,
			grid: grid,
			msg:  msg,
		}
	}

	grid.AddChild(header).At(0, 0)
	grid.AddChild(switcher).At(1, 0)

	mv := &MessageViewer{
		acct:     acct,
		conf:     conf,
		grid:     grid,
		msg:      msg,
		store:    store,
		switcher: switcher,
	}
	switcher.mv = mv

	return mv
}

func fmtHeader(msg *models.MessageInfo, header string) string {
	switch header {
	case "From":
		return models.FormatAddresses(msg.Envelope.From)
	case "To":
		return models.FormatAddresses(msg.Envelope.To)
	case "Cc":
		return models.FormatAddresses(msg.Envelope.Cc)
	case "Bcc":
		return models.FormatAddresses(msg.Envelope.Bcc)
	case "Date":
		return msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM")
	case "Subject":
		return msg.Envelope.Subject
	default:
		return msg.RFC822Headers.Get(header)
	}
}

func enumerateParts(conf *config.AercConfig, store *lib.MessageStore,
	msg *models.MessageInfo, body *models.BodyStructure,
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

func createSwitcher(switcher *PartSwitcher, conf *config.AercConfig,
	store *lib.MessageStore, msg *models.MessageInfo) error {

	var err error
	switcher.selected = -1
	switcher.showHeaders = conf.Viewer.ShowHeaders
	switcher.alwaysShowMime = conf.Viewer.AlwaysShowMime

	if len(msg.BodyStructure.Parts) == 0 {
		switcher.selected = 0
		pv, err := NewPartViewer(conf, store, msg, msg.BodyStructure, []int{1})
		if err != nil {
			return err
		}
		switcher.parts = []*PartViewer{pv}
		pv.OnInvalidate(func(_ ui.Drawable) {
			switcher.Invalidate()
		})
	} else {
		switcher.parts, err = enumerateParts(conf, store,
			msg, msg.BodyStructure, []int{})
		if err != nil {
			return err
		}
		selectedPriority := -1
		fmt.Printf("Selecting best message from %v\n", conf.Viewer.Alternatives)
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
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
		ctx.Printf(0, 0, tcell.StyleDefault, "%s", mv.err.Error())
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
	return mv.store
}

func (mv *MessageViewer) SelectedAccount() *AccountView {
	return mv.acct
}

func (mv *MessageViewer) SelectedMessage() (*models.MessageInfo, error) {
	if mv.msg == nil {
		return nil, errors.New("no message selected")
	}
	return mv.msg, nil
}

func (mv *MessageViewer) ToggleHeaders() {
	switcher := mv.switcher
	mv.conf.Viewer.ShowHeaders = !mv.conf.Viewer.ShowHeaders
	err := createSwitcher(
		switcher, mv.conf, mv.store, mv.msg)
	if err != nil {
		mv.acct.Logger().Printf(
			"warning: error during create switcher - %v", err)
	}
	switcher.Invalidate()
}

func (mv *MessageViewer) SelectedMessagePart() *PartInfo {
	switcher := mv.switcher
	part := switcher.parts[switcher.selected]

	return &PartInfo{
		Index: part.index,
		Msg:   part.msg,
		Part:  part.part,
		Store: part.store,
	}
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
	if height == 1 && !ps.alwaysShowMime {
		ps.parts[ps.selected].Draw(ctx)
		return
	}
	// TODO: cap height and add scrolling for messages with many parts
	ps.height = ctx.Height()
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

func (ps *PartSwitcher) MouseEvent(localX int, localY int, event tcell.Event) {
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			for i, _ := range ps.parts {
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
			if localY < y {
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
			if localY < y {
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

func (mv *MessageViewer) Event(event tcell.Event) bool {
	return mv.switcher.Event(event)
}

func (mv *MessageViewer) Focus(focus bool) {
	mv.switcher.Focus(focus)
}

type PartViewer struct {
	ui.Invalidatable
	err         error
	fetched     bool
	filter      *exec.Cmd
	index       []int
	msg         *models.MessageInfo
	pager       *exec.Cmd
	pagerin     io.WriteCloser
	part        *models.BodyStructure
	showHeaders bool
	sink        io.WriteCloser
	source      io.Reader
	store       *lib.MessageStore
	term        *Terminal
}

func NewPartViewer(conf *config.AercConfig,
	store *lib.MessageStore, msg *models.MessageInfo,
	part *models.BodyStructure,
	index []int) (*PartViewer, error) {

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
				header = models.FormatAddresses(msg.Envelope.From)
			case "to":
				header = models.FormatAddresses(msg.Envelope.To)
			case "cc":
				header = models.FormatAddresses(msg.Envelope.Cc)
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
		filter:      filter,
		index:       index,
		msg:         msg,
		pager:       pager,
		pagerin:     pagerin,
		part:        part,
		showHeaders: conf.Viewer.ShowHeaders,
		sink:        pipe,
		store:       store,
		term:        term,
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
			stderr, _ := pv.filter.StderrPipe()
			pv.filter.Start()
			ch := make(chan interface{})
			go func() {
				_, err := io.Copy(pv.pagerin, stdout)
				if err != nil {
					pv.err = err
					pv.Invalidate()
				}
				stdout.Close()
				ch <- nil
			}()
			go func() {
				_, err := io.Copy(pv.pagerin, stderr)
				if err != nil {
					pv.err = err
					pv.Invalidate()
				}
				stderr.Close()
				ch <- nil
			}()
			go func() {
				<-ch
				<-ch
				pv.pagerin.Close()
			}()
		}
		go func() {
			if pv.showHeaders && pv.msg.RFC822Headers != nil {
				fields := pv.msg.RFC822Headers.Fields()
				for fields.Next() {
					field := fmt.Sprintf(
						"%s: %s\n", fields.Key(), fields.Value())
					pv.sink.Write([]byte(field))
				}
				pv.sink.Write([]byte{'\n'})
			}

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
			if pv.part.MIMEType == "text" {
				scanner := bufio.NewScanner(part.Body)
				for scanner.Scan() {
					text := scanner.Text()
					text = ansi.ReplaceAllString(text, "")
					io.WriteString(pv.sink, text+"\n")
				}
			} else {
				io.Copy(pv.sink, part.Body)
			}
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
		ctx.Printf(0, 0, tcell.StyleDefault.Foreground(tcell.ColorRed),
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
